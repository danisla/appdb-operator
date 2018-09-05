package main

import (
	"fmt"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"

	"github.com/jinzhu/copier"
)

func sync(parentType ParentType, parent *appdbv1.AppDB, children *AppDBChildren) (*appdbv1.AppDBOperatorStatus, *[]interface{}, error) {
	var err error
	var status appdbv1.AppDBOperatorStatus
	copier.Copy(&status, &parent.Status)

	desiredTFApplys := make(map[string]bool, 0)
	desiredSecrets := make(map[string]bool, 0)
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

	var appdbi appdbv1.AppDBInstance
	appdbi, err = getAppDBInstance(parent.ObjectMeta.Namespace, parent.Spec.AppDBInstance)
	if err != nil {
		// Wait for AppDBInstance provisioning status to COMPLETE
		myLog(parent, "INFO", fmt.Sprintf("Waiting for AppDBInstance: %s", parent.Spec.AppDBInstance))
		return &status, &desiredChildren, nil
	} else if appdbi.Status.Provisioning != appdbv1.ProvisioningStatusComplete {
		// Wait for provisioning to complete.
		myLog(parent, "INFO", fmt.Sprintf("Waiting for AppDBInstance provisioning to complete: %s", parent.Spec.AppDBInstance))
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

		tfApplyName := fmt.Sprintf("%s-%s", appdbi.ObjectMeta.Name, parent.Name)

		if tfapply, ok := children.TerraformApplys[tfApplyName]; ok == true {
			mySig := calcParentSig(parent.Spec, "")
			tfapplySig := tfapply.Annotations["appdb-parent-sig"]

			if mySig == tfapplySig {

				status.CloudSQLDB.TFApplyPodName = tfapply.Status.PodName

				if tfapply.Status.PodStatus == "COMPLETED" {
					status.Provisioning = appdbv1.ProvisioningStatusComplete

					// Generate secret for DB credentials.
					if passwordsVar, ok := tfapply.Status.TFOutput["user_passwords"]; ok == true {
						secretName := status.CloudSQLDB.TFApplyName

						passwords := strings.Split(passwordsVar.Value, ",")
						if len(parent.Spec.Users) != len(passwords) {
							myLog(parent, "ERROR", "passwords output from TerraformApply is different length than input users.")
						} else {
							secret := makeCredentialsSecret(secretName, parent.ObjectMeta.Namespace, parent.Spec.Users, passwords)

							desiredSecrets[secretName] = true
							desiredChildren = append(desiredChildren, secret)

							status.CredentialsSecret = secretName
						}
					} else {
						myLog(parent, "ERROR", "No user_passwords found in output varibles of TerraformApply status")
					}
				} else if tfapply.Status.PodStatus == "FAILED" {
					status.Provisioning = appdbv1.ProvisioningStatusFailed
				} else {
					status.Provisioning = appdbv1.ProvisioningStatusPending
				}
			} else {
				// Patch tfapply with updated spec.
				myLog(parent, "INFO", fmt.Sprintf("Patching TerraformApply: %s", tfApplyName))

				myLog(parent, "WARN", "Patching not yet implemented!")
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
				TFApplyName: tfapply.ObjectMeta.Name,
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
	} else {
		myLog(parent, "WARN", "Unsupported AppDBInstance driver")
	}

	return &status, &desiredChildren, nil
}
