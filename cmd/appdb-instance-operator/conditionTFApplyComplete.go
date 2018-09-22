package main

import (
	"fmt"
	"strconv"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	"github.com/jinzhu/copier"
)

func reconcileTFApplyComplete(condition *appdbv1.AppDBInstanceCondition, parent *appdbv1.AppDBInstance, status *appdbv1.AppDBInstanceOperatorStatus, children *AppDBInstanceChildren, desiredChildren *[]interface{}) (appdbv1.ConditionStatus, tfv1.Terraform) {
	var ok bool
	var tfapply tfv1.Terraform
	newStatus := appdbv1.ConditionFalse
	tfName := makeTFName(parent)
	kind := "TerraformApply"

	if newChild, err := makeCloudSQLTerraform(tfName, parent); err != nil {
		condition.Reason = fmt.Sprintf("Failed to make %s: %v", kind, err)
	} else {
		if tfapply, ok = children.TerraformApplys[tfName]; ok == true {
			// Already created
			newStatus = condition.Status

			// Check to see if we should continue processing the TerraformApply or skip because the plan has not yet been verified.
			if status.CloudSQL.TFPlanStatus == tfv1.PodStatusPassed {
				condition.Reason = fmt.Sprintf("%s/%s: %s", kind, tfName, tfapply.Status.PodStatus)

				// Update the parent status
				status.CloudSQL.TFApplyName = tfName
				status.CloudSQL.TFApplyPodName = tfapply.Status.PodName

				if tfapply.Status.PodStatus == tfv1.PodStatusPassed {
					allVars := true

					// Get the "name" output variable.
					if nameVar, ok := tfapply.Status.TFOutput["name"]; ok == false {
						condition.Reason = fmt.Sprintf("%s/%s: Missing output 'name'", kind, tfName)
						allVars = false
					} else {
						status.CloudSQL.InstanceName = nameVar.Value
					}

					// Get the "connection" output variable.
					if connVar, ok := tfapply.Status.TFOutput["connection"]; ok == false {
						condition.Reason = fmt.Sprintf("%s/%s: Missing output 'connection'", kind, tfName)
						allVars = false
					} else {
						status.CloudSQL.ConnectionName = connVar.Value
					}

					// Get the "port" output variable.
					if portVar, ok := tfapply.Status.TFOutput["port"]; ok == false {
						condition.Reason = fmt.Sprintf("%s/%s: Missing output 'port'", kind, tfName)
						allVars = false
					} else {
						port, err := strconv.Atoi(portVar.Value)
						if err != nil {
							parent.Log("ERROR", fmt.Sprintf("Output variable 'port' could not be parsed as int: %s", portVar.Value))
							condition.Reason = fmt.Sprintf("%s/%s: Internal error", kind, tfName)
							allVars = false
						}
						status.CloudSQL.Port = int32(port)
					}

					// Get the serviceAccountEmail output variable
					if saEmail, ok := tfapply.Status.TFOutput["instance_sa_email"]; ok == false {
						condition.Reason = fmt.Sprintf("%s/%s: Missing output 'instance_sa_email'", kind, tfName)
						allVars = false
					} else {
						status.CloudSQL.ServiceAccountEmail = saEmail.Value
					}

					if allVars {
						newStatus = appdbv1.ConditionTrue
						children.claimChildAndGetCurrent(newChild, desiredChildren)
					} else {
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
					}
				} else if tfapply.Status.PodStatus == tfv1.PodStatusFailed {
					status.Provisioning = appdbv1.ProvisioningStatusFailed

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
				}
			} else {
				// Updated TerraformPlan is not ready to be applied, create child with existing TerraformApply spec.
				var currSpec tfv1.TerraformSpec
				var currSpecFrom tfv1.TerraformSpecFrom

				copier.Copy(&currSpec, &tfapply.Spec)
				copier.Copy(&currSpecFrom, &tfapply.SpecFrom)

				newChild.Spec = currSpec
				newChild.SpecFrom = currSpecFrom

				children.claimChildAndGetCurrent(newChild, desiredChildren)
			}
		} else {
			// Create after TerraformPlan completes.
			if status.CloudSQL != nil && status.CloudSQL.TFPlanStatus == tfv1.PodStatusPassed {
				children.claimChildAndGetCurrent(newChild, desiredChildren)
			}
		}
	}

	return newStatus, tfapply
}
