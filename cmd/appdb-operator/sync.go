package main

import (
	"fmt"
	"strings"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"

	"github.com/jinzhu/copier"
)

func sync(parentType ParentType, parent *appdbv1.AppDB, children *AppDBChildren) (*appdbv1.AppDBOperatorStatus, *[]interface{}, error) {
	var err error
	var status appdbv1.AppDBOperatorStatus
	copier.Copy(&status, &parent.Status)

	desiredTFApplys := make(map[string]bool, 0)
	desiredSecrets := make(map[string]bool, 0)
	desiredJobs := make(map[string]bool, 0)
	desiredChildren := make([]interface{}, 0)

	if parent.Spec.AppDBInstance == "" {
		// Must have AppDBInstance
		myLog(parent, "ERROR", "Missing appDBInstance")
		return &status, &desiredChildren, nil
	}

	if parent.Spec.DBName == "" {
		// Must have DBName
		myLog(parent, "ERROR", "Missing dbName")
		return &status, &desiredChildren, nil
	}

	if len(parent.Spec.Users) == 0 {
		// Must have at least 1 user
		myLog(parent, "ERROR", "Users list is empty")
		return &status, &desiredChildren, nil
	}

	var appdbi appdbv1.AppDBInstance
	appdbi, err = getAppDBInstance(parent.GetNamespace(), parent.Spec.AppDBInstance)
	if err != nil {
		// Wait for AppDBInstance provisioning status to COMPLETE
		myLog(parent, "INFO", fmt.Sprintf("Waiting for AppDBInstance: %s", parent.Spec.AppDBInstance))
		status.Provisioning = appdbv1.ProvisioningStatusPending
		return &status, &desiredChildren, nil
	} else if appdbi.Status.Provisioning != appdbv1.ProvisioningStatusComplete {
		// Wait for provisioning to complete.
		myLog(parent, "INFO", fmt.Sprintf("Waiting for AppDBInstance provisioning to complete: %s", parent.Spec.AppDBInstance))
		status.Provisioning = appdbv1.ProvisioningStatusPending
		return &status, &desiredChildren, nil
	} else {
		status.AppDBInstanceSig = calcParentSig(appdbi.Spec, "")
	}

	if status.AppDBInstanceSig != "" && status.AppDBInstanceSig != calcParentSig(appdbi.Spec, "") {
		// AppDBInstance changed, clear out children and start over.
		myLog(parent, "WARN", fmt.Sprintf("AppDBInstance signature changed, deleting all child resources"))
		return &status, &desiredChildren, nil
	}

	if appdbi.Spec.Driver.CloudSQLTerraform != nil {

		tfApplyName := fmt.Sprintf("appdb-%s-%s", appdbi.GetName(), parent.GetName())

		if tfapply, ok := children.TerraformApplys[tfApplyName]; ok == true {
			if status.CloudSQLDB == nil {
				myLog(parent, "WARN", "Found TerraformPlan in children, but status.CloudSQLDB was nil, re-sync collision.")
				// Delete TerraformApply and try again
				desiredTFApplys[tfApplyName] = true
			} else {

				mySig := calcParentSig(parent.Spec, "")
				tfapplySig := tfapply.Annotations["appdb-parent-sig"]

				if mySig == tfapplySig {

					status.CloudSQLDB.TFApplyPodName = tfapply.Status.PodName

					if tfapply.Status.PodStatus == "COMPLETED" {

						// Generate secret for DB credentials.
						if passwordsVar, ok := tfapply.Status.TFOutput["user_passwords"]; ok == true {

							passwords := strings.Split(passwordsVar.Value, ",")
							if len(parent.Spec.Users) != len(passwords) {
								myLog(parent, "ERROR", "passwords output from TerraformApply is different length than input users.")
							} else {
								status.CredentialsSecrets = make(map[string]string, 0)
								for i := 0; i < len(parent.Spec.Users); i++ {
									secretName := fmt.Sprintf("appdb-%s-%s-user-%d", appdbi.GetName(), parent.GetName(), i)

									secret := makeCredentialsSecret(secretName, parent.GetNamespace(), parent.Spec.Users[i], passwords[i], appdbi.Status.DBHost, appdbi.Status.DBPort)

									status.CredentialsSecrets[parent.Spec.Users[i]] = secretName

									desiredSecrets[secretName] = true
									desiredChildren = append(desiredChildren, secret)
								}
							}
						} else {
							myLog(parent, "ERROR", "No user_passwords found in output varibles of TerraformApply status")
						}

						// Optionally load snapshot from GCS.
						if parent.Spec.LoadURL != "" {
							jobName := fmt.Sprintf("appdb-%s-%s-load", appdbi.GetName(), parent.GetName())
							loadURL := parent.Spec.LoadURL
							if len(loadURL) >= 5 && loadURL[0:5] != "gs://" {
								// Relative url to bucket.
								loadURL = fmt.Sprintf("gs://%s/%s", tfDriverConfig.BackendBucket, parent.Spec.LoadURL)
							}
							job := makeLoadJob(jobName, parent.GetNamespace(), appdbi.Status.CloudSQL.InstanceName, loadURL, parent.Spec.DBName, parent.Spec.Users[0], appdbi.Status.CloudSQL.ServiceAccountEmail)
							if currJob, ok := children.Jobs[job.GetName()]; ok == true {
								// Wait for load job to complete.
								if currJob.Status.Succeeded == 1 {
									// load complete.
									status.Provisioning = appdbv1.ProvisioningStatusComplete
								} else if currJob.Status.Failed == *currJob.Spec.BackoffLimit {
									// Requeue job
									desiredJobs[job.GetName()] = true
								}
							} else {
								// Create job
								desiredJobs[job.GetName()] = true
								desiredChildren = append(desiredChildren, job)

								myLog(parent, "INFO", fmt.Sprintf("Created SQL load job from snapshot %s: %s", loadURL, job.GetName()))
							}
						} else {
							// No load job, so done.
							status.Provisioning = appdbv1.ProvisioningStatusComplete
						}

					} else if tfapply.Status.PodStatus == "FAILED" {
						status.Provisioning = appdbv1.ProvisioningStatusFailed

						// Try again in 60 seconds.
						tfapplyFishedAtTime, err := time.Parse(time.RFC3339, tfapply.Status.FinishedAt)
						if err != nil {
							myLog(parent, "WARN", fmt.Sprintf("Failed to parse tfplan finished at time: %v", err))
						} else {
							if time.Since(tfapplyFishedAtTime).Seconds() > 60 {
								myLog(parent, "INFO", "Retrying TerraformApply")
								// Setting desiredTFApplys to true will cause it to be omitted during the claim phase, therefore deleting it.
								desiredTFApplys[tfApplyName] = true
							}
						}
					} else {
						status.Provisioning = appdbv1.ProvisioningStatusPending
					}
				} else {
					// Patch tfapply with updated spec.
					myLog(parent, "INFO", fmt.Sprintf("Change detected, patching TerraformApply: %s", tfApplyName))

					tfapply, err := makeCloudSQLDBTerraform(tfApplyName, parent, appdbi)
					if err != nil {
						myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformApply spec for CloudSQL: %v", err))
					} else {
						if _, ok := children.TerraformApplys[tfApplyName]; ok == true {
							// found existing tfapply, apply changes to it.
							err = kubectlApply(parent.GetNamespace(), tfApplyName, tfapply)
							if err != nil {
								myLog(parent, "ERROR", fmt.Sprintf("Failed to kubectl apply the TerraformApply resource: %v", err))
							} else {

								status.CloudSQLDB = &appdbv1.AppDBCloudSQLDBStatus{
									TFApplyName: tfapply.GetName(),
									TFApplySig:  calcParentSig(parent.Spec, ""),
								}

								desiredTFApplys[tfApplyName] = true
								desiredChildren = append(desiredChildren, tfapply)
							}
						} else {
							myLog(parent, "WARN", "No existing TerraformApply claimed to main changes to.")
						}
					}
				}
			}
		} else {
			// Create new TerraformApply to create DB and users.
			tfapply, err := makeCloudSQLDBTerraform(tfApplyName, parent, appdbi)
			if err != nil {
				myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformApply spec for CloudSQL DB: %v", err))
				return &status, &desiredChildren, nil
			}

			desiredTFApplys[tfApplyName] = true
			desiredChildren = append(desiredChildren, tfapply)

			myLog(parent, "INFO", fmt.Sprintf("Creating TerraformApply: %s", tfApplyName))

			status.CloudSQLDB = &appdbv1.AppDBCloudSQLDBStatus{
				TFApplyName: tfapply.GetName(),
				TFApplySig:  calcParentSig(parent.Spec, ""),
			}
		}

		// Claim new terraformapplys else claim existing.
		for _, o := range children.TerraformApplys {
			if desiredTFApplys[o.Name] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new secrets else claim existing.
		for _, o := range children.Secrets {
			if desiredSecrets[o.Name] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new jobs else claim existing.
		for _, o := range children.Jobs {
			if desiredJobs[o.Name] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}
	} else {
		myLog(parent, "WARN", "Unsupported AppDBInstance driver")
	}

	return &status, &desiredChildren, nil
}
