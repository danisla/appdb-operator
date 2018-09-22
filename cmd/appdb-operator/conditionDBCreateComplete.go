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
		tfName := makeTFApplyName(parent, appdbi)
		kind := "TerraformApply"
		if newChild, err := makeCloudSQLDBTerraform(tfName, parent, appdbi); err != nil {
			condition.Reason = fmt.Sprintf("Failed to make %s: %v", kind, err)
		} else {
			if tfapply, ok = children.TerraformApplys[tfName]; ok == true {
				// Already created.
				status.CloudSQLDB = &appdbv1.AppDBCloudSQLDBStatus{
					TFApplyName:    tfName,
					TFApplyPodName: tfapply.Status.PodName,
					TFApplySig:     tfapply.Annotations["appdb-parent-sig"],
				}

				condition.Reason = fmt.Sprintf("%s/%s: %s", kind, tfName, tfapply.Status.PodStatus)

				if tfapply.Status.PodStatus == tfv1.PodStatusPassed {
					newStatus = appdbv1.ConditionTrue
					children.claimChildAndGetCurrent(newChild, desiredChildren)
				} else if tfapply.Status.PodStatus == tfv1.PodStatusFailed {
					condition.Reason = fmt.Sprintf("%s/%s pod failed", kind, tfName)

					// Try again in 60 seconds.
					tfapplyFishedAtTime, err := time.Parse(time.RFC3339, tfapply.Status.FinishedAt)
					if err != nil {
						parent.Log("WARN", fmt.Sprintf("Failed to parse %s finished at time: %v", kind, err))
						condition.Reason = fmt.Sprintf("%s/%s: Internal error", kind, tfName)
						children.claimChildAndGetCurrent(newChild, desiredChildren)
					} else {
						condition.Message = "Retry in 60 seconds"
						if time.Since(tfapplyFishedAtTime).Seconds() > 60 {
							parent.Log("Retrying %s/%s", kind, tfName)
							// Do not claim the new child, this will cause it to be deleted and recreated on the next sync.
						} else {
							children.claimChildAndGetCurrent(newChild, desiredChildren)
						}
					}
				} else {
					// Running
					children.claimChildAndGetCurrent(newChild, desiredChildren)
				}
			} else {
				// Not yet created.
				children.claimChildAndGetCurrent(newChild, desiredChildren)
			}
		}
	} else {
		condition.Reason = "Unsupported AppDBInstance driver."
	}

	return newStatus, tfapply
}
