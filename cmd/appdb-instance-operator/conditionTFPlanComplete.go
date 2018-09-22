package main

import (
	"fmt"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
)

func reconcileTFPlanComplete(condition *appdbv1.AppDBInstanceCondition, parent *appdbv1.AppDBInstance, status *appdbv1.AppDBInstanceOperatorStatus, children *AppDBInstanceChildren, desiredChildren *[]interface{}) appdbv1.ConditionStatus {
	var ok bool
	var tfplan tfv1.Terraform
	newStatus := appdbv1.ConditionFalse
	tfName := makeTFName(parent)
	kind := "TerraformPlan"

	if status.CloudSQL == nil {
		status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{}
	}

	if newChild, err := makeCloudSQLTerraform(tfName, parent); err != nil {
		condition.Reason = fmt.Sprintf("Failed to make %s: %v", kind, err)
	} else {
		// Switch object kind.
		newChild.TypeMeta.Kind = kind

		if tfplan, ok = children.TerraformPlans[tfName]; ok == true {
			// Already created
			condition.Reason = fmt.Sprintf("%s/%s: %s", kind, tfName, tfplan.Status.PodStatus)

			// Update the parent status
			status.CloudSQL.TFPlanName = tfName
			status.CloudSQL.TFPlanPodName = tfplan.Status.PodName
			status.CloudSQL.TFPlanStatus = tfplan.Status.PodStatus

			if tfplan.Status.PodStatus == tfv1.PodStatusPassed {
				// Check plan
				if tfplan.Status.TFPlanDiff.Destroyed > 0 {
					condition.Reason = fmt.Sprintf("%s/%s: Contains Destroy Actions", kind, tfName)

					// Retry in 60 seconds.
					tfplanFishedAtTime, err := time.Parse(time.RFC3339, tfplan.Status.FinishedAt)
					if err != nil {
						parent.Log("WARN", fmt.Sprintf("Failed to parse %s finished at time: %v", kind, err))
						condition.Reason = fmt.Sprintf("%s/%s: Internal error", kind, tfName)
						children.claimChildAndGetCurrent(newChild, desiredChildren)
					} else {
						if time.Since(tfplanFishedAtTime).Seconds() > 60 {
							parent.Log("INFO", "Retrying %s/%s", kind, tfName)
							// Do not claim the new child, this will cause it to be deleted and recreated on the next sync.
						} else {
							children.claimChildAndGetCurrent(newChild, desiredChildren)
						}
					}
				} else {
					// plan passed with no destroy actions.
					newStatus = appdbv1.ConditionTrue
					status.CloudSQL.LastPlanSig = parent.GetSig()
					children.claimChildAndGetCurrent(newChild, desiredChildren)
				}
			} else if tfplan.Status.PodStatus == tfv1.PodStatusFailed {
				status.Provisioning = appdbv1.ProvisioningStatusFailed
				children.claimChildAndGetCurrent(newChild, desiredChildren)
			} else {
				children.claimChildAndGetCurrent(newChild, desiredChildren)
			}
		} else {
			// Not yet created.
			parent.Log("INFO", "Creating new %s/%s", kind, tfName)
			children.claimChildAndGetCurrent(newChild, desiredChildren)
		}
	}

	return newStatus
}
