package main

import (
	"fmt"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
)

func reconcileDBCreateComplete(condition *appdbv1.AppDBCondition, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}, appdbi appdbv1.AppDBInstance) (appdbv1.ConditionStatus, tfv1.Terraform) {
	newStatus := appdbv1.ConditionFalse
	var tfapply tfv1.Terraform

	if appdbi.Spec.Driver.CloudSQLTerraform != nil {
		// Terraform driver

		var ok bool
		newStatus = appdbv1.ConditionFalse
		tfApplyName := makeTFApplyName(parent, appdbi)
		if newChild, err := makeCloudSQLDBTerraform(tfApplyName, parent, appdbi); err != nil {
			condition.Reason = fmt.Sprintf("Failed to make tfapply: %v", err)
		} else {
			if tfapply, ok = children.TerraformApplys[tfApplyName]; ok == true {
				// Already created.
				status.CloudSQLDB = &appdbv1.AppDBCloudSQLDBStatus{
					TFApplyName:    tfapply.GetName(),
					TFApplyPodName: tfapply.Status.PodName,
					TFApplySig:     tfapply.Annotations["appdb-parent-sig"],
				}

				condition.Reason = fmt.Sprintf("TerraformApply/%s: %s", tfapply.GetName(), tfapply.Status.PodStatus)

				if tfapply.Status.PodStatus == tfv1.PodStatusPassed {
					newStatus = appdbv1.ConditionTrue
					claimChildAndGetCurrent(newChild, children, desiredChildren)
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
							claimChildAndGetCurrent(newChild, children, desiredChildren)
						}
					}
				} else {
					// Running
					claimChildAndGetCurrent(newChild, children, desiredChildren)
				}
			} else {
				// Not yet created.
				claimChildAndGetCurrent(newChild, children, desiredChildren)
			}
		}
	} else {
		condition.Reason = "Unsupported AppDBInstance driver."
	}

	return newStatus, tfapply
}
