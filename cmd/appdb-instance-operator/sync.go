package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	"github.com/jinzhu/copier"
)

func sync(parentType ParentType, parent *appdbv1.AppDBInstance, children *AppDBInstanceChildren) (*appdbv1.AppDBInstanceOperatorStatus, *[]interface{}, error) {
	var status appdbv1.AppDBInstanceOperatorStatus
	copier.Copy(&status, &parent.Status)

	desiredTFApplys := make(map[string]bool, 0)
	desiredDeployments := make(map[string]bool, 0)
	desiredServices := make(map[string]bool, 0)
	desiredChildren := make([]interface{}, 0)

	if parent.Spec.Driver.CloudSQLTerraform != nil {

		tfApplyName := parent.Name

		if tfapply, ok := children.TerraformApplys[tfApplyName]; ok == true {
			mySig := calcParentSig(parent.Spec, "")
			tfapplySig := tfapply.Annotations["appdb-parent-sig"]

			if mySig == tfapplySig {

				status.CloudSQL.TFApplyPodName = tfapply.Status.PodName

				if tfapply.Status.PodStatus == "COMPLETED" {
					status.Provisioning = appdbv1.ProvisioningStatusComplete

					if nameVar, ok := tfapply.Status.TFOutput["name"]; ok == false {
						myLog(parent, "ERROR", fmt.Sprintf("Output variable 'name' not found in status of TerraformApply: %s", tfapply.ObjectMeta.Name))
					} else {
						status.CloudSQL.InstanceName = nameVar.Value
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
			// Create new TerraformApply to provision DB instance.
			tfapply, err := makeCloudSQLTerraform(tfApplyName, parent)
			if err != nil {
				myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformApply spec for CloudSQL: %v", err))
				return &status, &desiredChildren, nil
			}

			desiredTFApplys[tfApplyName] = true
			desiredChildren = append(desiredChildren, tfapply)

			myLog(parent, "INFO", fmt.Sprintf("Creating TerraformApply: %s", tfApplyName))

			status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
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

		// Claim new deployments else claim existing.
		for _, o := range children.Deployments {
			if desiredDeployments[o.Name] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new services else claim existing.
		for _, o := range children.Services {
			if desiredServices[o.Name] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}
	} else {
		myLog(parent, "WARN", "Unsupported AppDBInstance driver")
	}

	return &status, &desiredChildren, nil
}
