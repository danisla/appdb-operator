package main

import (
	"fmt"
	"strings"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jinzhu/copier"
)

func sync(parentType ParentType, parent *appdbv1.AppDB, children *AppDBChildren) (*appdbv1.AppDBOperatorStatus, *[]interface{}, error) {
	var err error
	var status appdbv1.AppDBOperatorStatus
	copier.Copy(&status, &parent.Status)

	desiredChildren := make([]interface{}, 0)

	// Current time used for updating conditions
	tNow := metav1.NewTime(time.Now())

	// Verify required top level fields.
	if err = verifySpec(parent); err != nil {
		parent.Log("ERROR", "Invalid spec: %v", err)
		status.Conditions = append(status.Conditions, appdbv1.AppDBCondition{
			Type:               appdbv1.ConditionTypeAppDBReady,
			Status:             appdbv1.ConditionFalse,
			LastProbeTime:      tNow,
			LastTransitionTime: tNow,
			Reason:             "Invalid spec",
			Message:            fmt.Sprintf("%v", err),
		})
		return &status, &desiredChildren, nil
	}

	// Map of condition types to conditions, converted to list of conditions after switch statement.
	conditions := make(map[appdbv1.AppDBConditionType]*appdbv1.AppDBCondition, 0)
	conditionOrder := makeConditionOrder(parent)

	// Extract existing conditions from status and copy to conditions map for easier lookup.
	for _, c := range conditionOrder {
		// Search for condition type in conditions.
		found := false
		for _, condition := range status.Conditions {
			if condition.Type == c {
				found = true
				condition.LastProbeTime = tNow
				condition.Reason = ""
				condition.Message = ""
				conditions[c] = &condition
				break
			}
		}
		if found == false {
			// Initialize condition with unknown state
			conditions[c] = &appdbv1.AppDBCondition{
				Type:               c,
				Status:             appdbv1.ConditionUnknown,
				LastProbeTime:      tNow,
				LastTransitionTime: tNow,
			}
		}
	}

	// Resources used in multiple conditions.
	var appdbi appdbv1.AppDBInstance
	var tfapply tfv1.Terraform

	// Reconcile each condition.
	for _, conditionType := range conditionOrder {
		condition := conditions[conditionType]
		newStatus := condition.Status

		// Skip processing conditions with unmet dependencies.
		if err = checkConditions(conditionType, conditions); err != nil {
			newStatus = appdbv1.ConditionFalse
			condition.Reason = err.Error()
			if condition.Status != newStatus {
				condition.LastTransitionTime = tNow
				condition.Status = newStatus
			}
			continue
		}

		switch conditionType {
		case appdbv1.ConditionTypeAppDBInstanceReady:
			newStatus = appdbv1.ConditionFalse
			appdbi, err = getAppDBInstance(parent.GetNamespace(), parent.Spec.AppDBInstance)
			if err == nil {
				if status.AppDBInstanceSig != "" && status.AppDBInstanceSig != calcParentSig(appdbi.Spec, "") {
					// AppDBInstance spec changed.
					condition.Reason = fmt.Sprintf("AppDBInstance/%s change detected", appdbi.GetName())
				} else {
					if appdbi.Status.Provisioning == appdbv1.ProvisioningStatusComplete {
						newStatus = appdbv1.ConditionTrue
						status.AppDBInstanceSig = calcParentSig(appdbi.Spec, "")
					}
					condition.Reason = fmt.Sprintf("AppDBInstance/%s: %s", appdbi.GetName(), appdbi.Status.Provisioning)
				}
			} else {
				condition.Reason = fmt.Sprintf("AppDBInstance/%s: Not found", parent.Spec.AppDBInstance)
			}

		case appdbv1.ConditionTypeDBCreateComplete:
			if appdbi.Spec.Driver.CloudSQLTerraform != nil {
				// Terraform driver

				var ok bool
				newStatus = appdbv1.ConditionFalse
				tfApplyName := makeTFApplyName(parent, appdbi)
				tfapply, ok = children.TerraformApplys[tfApplyName]
				if ok == false {
					parent.Log("INFO", "Creating new TerraformApply/%s", tfApplyName)
					tfapply, err = makeCloudSQLDBTerraform(tfApplyName, parent, appdbi)
				}
				if err != nil {
					condition.Reason = fmt.Sprintf("Failed to make tfapply: %v", err)
				} else {
					condition.Reason = fmt.Sprintf("TerraformApply/%s: %s", tfapply.GetName(), tfapply.Status.PodStatus)

					status.CloudSQLDB = &appdbv1.AppDBCloudSQLDBStatus{
						TFApplyName:    tfapply.GetName(),
						TFApplyPodName: tfapply.Status.PodName,
						TFApplySig:     tfapply.Annotations["appdb-parent-sig"],
					}

					if status.CloudSQLDB.TFApplySig != calcParentSig(parent.Spec, "") {
						// Patch tfapply with updated spec.
						parent.Log("INFO", "Change detected, patching TerraformApply/%s", tfapply.GetName())

						tfapply, err := makeCloudSQLDBTerraform(tfApplyName, parent, appdbi)
						if err != nil {
							condition.Reason = fmt.Sprintf("Failed to make tfapply: %v", err)
						} else {
							err = kubectlApply(parent.GetNamespace(), tfApplyName, tfapply)
							if err != nil {
								condition.Reason = fmt.Sprintf("Failed to apply updated tfapply: %v", err)
							}
						}
						claimChildIfNotPresnet(tfApplyName, "TerraformApply", tfapply, children, &desiredChildren)
					} else {
						if tfapply.Status.PodStatus == tfv1.PodStatusPassed {
							newStatus = appdbv1.ConditionTrue
							claimChildIfNotPresnet(tfApplyName, "TerraformApply", tfapply, children, &desiredChildren)
						} else if tfapply.Status.PodStatus == tfv1.PodStatusFailed {
							condition.Reason = fmt.Sprintf("TerraformApply/%s pod failed", tfapply.GetName())

							// Try again in 60 seconds.
							tfapplyFishedAtTime, err := time.Parse(time.RFC3339, tfapply.Status.FinishedAt)
							if err != nil {
								condition.Reason = fmt.Sprintf("Failed to parse tfplan finished at time: %v", err)
							} else {
								condition.Message = "Retry in 60 seconds"
								if time.Since(tfapplyFishedAtTime).Seconds() > 60 {
									parent.Log("Retrying TerraformApply,%s", tfapply.GetName())
								} else {
									claimChildIfNotPresnet(tfApplyName, "TerraformApply", tfapply, children, &desiredChildren)
								}
							}
						} else {
							claimChildIfNotPresnet(tfApplyName, "TerraformApply", tfapply, children, &desiredChildren)
						}
					}
				}
			} else {
				condition.Reason = "Unsupported AppDBInstance driver."
			}
		case appdbv1.ConditionTypeCredentialsSecretCreated:
			// Generate secret for DB credentials.
			newStatus = appdbv1.ConditionFalse
			if passwordsVar, ok := tfapply.Status.TFOutput["user_passwords"]; ok == true {
				passwords := strings.Split(passwordsVar.Value, ",")
				if len(parent.Spec.Users) != len(passwords) {
					condition.Reason = fmt.Sprintf("passwords output from TerraformApply is different length than input users.")
				} else {
					status.CredentialsSecrets = make(map[string]string, 0)
					secretNames := []string{}
					for i := 0; i < len(parent.Spec.Users); i++ {
						secretName := fmt.Sprintf("appdb-%s-%s-user-%d", appdbi.GetName(), parent.GetName(), i)

						secret := makeCredentialsSecret(secretName, parent.GetNamespace(), parent.Spec.Users[i], passwords[i], parent.Spec.DBName, appdbi.Status.DBHost, appdbi.Status.DBPort)

						secretNames = append(secretNames, secretName)

						status.CredentialsSecrets[parent.Spec.Users[i]] = secretName

						claimChildIfNotPresnet(secretName, "Secret", secret, children, &desiredChildren)

						newStatus = appdbv1.ConditionTrue
					}
					condition.Reason = fmt.Sprintf("Secret/%s: CREATED", strings.Join(secretNames, ","))
				}
			} else {
				condition.Reason = "No user_passwords found in output varibles of TerraformApply status"
			}
		case appdbv1.ConditionTypeSnapshotLoadComplete:
			newStatus = appdbv1.ConditionFalse
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
					newStatus = appdbv1.ConditionTrue
					claimChildIfNotPresnet(jobName, "Job", job, children, &desiredChildren)
				} else if currJob.Status.Failed == *currJob.Spec.BackoffLimit {
					// Requeue job
					parent.Log("INFO", "Recreating SQL Load job")
				} else {
					claimChildIfNotPresnet(jobName, "Job", job, children, &desiredChildren)
				}
			} else {
				// Create job
				claimChildIfNotPresnet(jobName, "Job", job, children, &desiredChildren)
				parent.Log("INFO", "Created SQL load job from snapshot %s: %s", loadURL, job.GetName())
			}
		case appdbv1.ConditionTypeAppDBReady:
			newStatus = appdbv1.ConditionTrue
			notReady := []string{}
			for _, c := range conditionOrder {
				if c != appdbv1.ConditionTypeAppDBReady && conditions[c].Status != appdbv1.ConditionTrue {
					notReady = append(notReady, string(c))
					newStatus = appdbv1.ConditionFalse
				}
			}
			if len(notReady) > 0 {
				condition.Reason = fmt.Sprintf("Waiting for conditions: %s", strings.Join(notReady, ","))
				status.Provisioning = appdbv1.ProvisioningStatusPending
			} else {
				condition.Reason = "All conditions satisfied"
				status.Provisioning = appdbv1.ProvisioningStatusComplete
			}
		}

		if condition.Status != newStatus {
			condition.LastTransitionTime = tNow
			condition.Status = newStatus
		}
	}

	// Copy updated conditions back to status in order.
	newConditions := make([]appdbv1.AppDBCondition, 0)
	for _, c := range conditionOrder {
		newConditions = append(newConditions, *conditions[c])
	}
	status.Conditions = newConditions

	return &status, &desiredChildren, nil
}

func makeConditionOrder(parent *appdbv1.AppDB) []appdbv1.AppDBConditionType {
	conditionOrder := make([]appdbv1.AppDBConditionType, 0)
	for _, c := range conditionStatusOrder {
		if c == appdbv1.ConditionTypeSnapshotLoadComplete && parent.Spec.LoadURL == "" {
			// Skip condition.
			continue
		}
		conditionOrder = append(conditionOrder, c)
	}
	return conditionOrder
}

func claimChildIfNotPresnet(name, kind string, newChild interface{}, children *AppDBChildren, desiredChildren *[]interface{}) {
	// Check to see if item has been created in the children
	var claimChild interface{}
	switch kind {
	case "TerraformApply":
		if child, ok := children.TerraformApplys[name]; ok == false {
			claimChild = newChild
		} else {
			claimChild = child
		}
	case "Secret":
		if child, ok := children.Secrets[name]; ok == false {
			claimChild = newChild
		} else {
			claimChild = child
		}
	case "Job":
		if child, ok := children.Jobs[name]; ok == false {
			claimChild = newChild
		} else {
			claimChild = child
		}
	}
	*desiredChildren = append(*desiredChildren, claimChild)
}

func checkConditions(checkType appdbv1.AppDBConditionType, conditions map[appdbv1.AppDBConditionType]*appdbv1.AppDBCondition) error {
	waiting := []string{}

	for _, conditionType := range conditionDependencies[checkType] {
		condition := conditions[conditionType]
		if condition.Status != appdbv1.ConditionTrue {
			waiting = append(waiting, string(conditionType))
		}
	}

	if len(waiting) == 0 {
		return nil
	}

	return fmt.Errorf("Waiting on conditions: %s", strings.Join(waiting, ","))
}
