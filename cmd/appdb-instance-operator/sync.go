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
	desiredTFPlans := make(map[string]bool, 0)
	desiredDeployments := make(map[string]bool, 0)
	desiredServices := make(map[string]bool, 0)
	desiredChildren := make([]interface{}, 0)

	if parent.Spec.Driver.CloudSQLTerraform != nil {

		tfApplyName := parent.Name
		planRunning := false

		if tfplan, ok := children.TerraformPlans[tfApplyName]; ok == true {
			// Handle terraform plan
			mySig := calcParentSig(parent.Spec, "")
			tfplanSig := tfplan.Annotations["appdb-parent-sig"]

			if mySig == tfplanSig {
				status.CloudSQL.TFPlanPodName = tfplan.Status.PodName

				planRunning = true

				if tfplan.Status.PodStatus == "COMPLETED" {
					// Check plan
					if tfplan.Status.TFPlanDiff.Destroyed > 0 {
						myLog(parent, "ERROR", "TerraformPlan contains destroy actions, skipping patch.")
					} else {
						myLog(parent, "INFO", "TerraformPlan contains no destroy actions, proceeding with patch.")

						// Setting desiredTFPlans to true will cause it to be omitted during the claim phase, therefore deleting it.
						desiredTFPlans[tfApplyName] = true

						tfapply, err := makeCloudSQLTerraform(tfApplyName, parent)
						if err != nil {
							myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformApply spec for CloudSQL: %v", err))
						} else {
							if _, ok := children.TerraformApplys[tfApplyName]; ok == true {
								// found existing tfapply, apply changes to it.
								err = kubectlApply(parent.Namespace, tfApplyName, tfapply)
								if err != nil {
									myLog(parent, "ERROR", fmt.Sprintf("Failed to kubectl apply the TerraformApply resource: %v", err))
								} else {

									status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
										TFApplyName: tfapply.ObjectMeta.Name,
										TFApplySig:  calcParentSig(parent.Spec, ""),
									}

									desiredTFApplys[tfApplyName] = true
									desiredChildren = append(desiredChildren, tfapply)
								}
							} else {
								// No existing tfapply, create new one.
								status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
									TFApplyName: tfapply.ObjectMeta.Name,
									TFApplySig:  calcParentSig(parent.Spec, ""),
								}

								desiredTFApplys[tfApplyName] = true
								desiredChildren = append(desiredChildren, tfapply)
							}
						}
					}
				} else if tfplan.Status.PodStatus == "FAILED" {
					myLog(parent, "WARN", "Failed to run TerraformPlan")
				} else {
					// Wait for plan to complete.
				}
			} else {
				myLog(parent, "WARN", "Found TerraformPlan with non-matching parent sig.")
				return &status, &desiredChildren, nil
			}
		}

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
				if planRunning == false {
					// Patch tfapply with updated spec.
					myLog(parent, "INFO", "Change detected, running TerraformPlan to preview changes.")

					// CompositeController updateStrategy is set to OnDelete, which means we cannot update the child resource from the controller.
					// Instead, just use kubectl to apply the update.

					// Verify requested change won't trigger a destroy operation.
					tfplan, err := makeCloudSQLTerraform(tfApplyName, parent)
					if err != nil {
						myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformPlan spec to check breaking changes for CloudSQL: %v", err))
					} else {
						tfplan.TypeMeta.Kind = "TerraformPlan"
						status.CloudSQL.TFPlanName = tfApplyName
						status.CloudSQL.TFPlanSig = calcParentSig(parent.Spec, "")

						desiredTFPlans[tfApplyName] = true
						desiredChildren = append(desiredChildren, tfplan)
					}
				}
			}
		} else {
			// Create new TerraformApply to provision DB instance.

			// Verify requested change won't trigger a destroy operation.
			tfplan, err := makeCloudSQLTerraform(tfApplyName, parent)
			if err != nil {
				myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformPlan spec to check breaking changes for CloudSQL: %v", err))
			} else {
				tfplan.TypeMeta.Kind = "TerraformPlan"
				status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
					TFPlanName: tfApplyName,
					TFPlanSig:  calcParentSig(parent.Spec, ""),
				}

				desiredTFPlans[tfApplyName] = true
				desiredChildren = append(desiredChildren, tfplan)
			}
		}

		// Claim new terraformapplys else claim existing.
		for _, o := range children.TerraformApplys {
			if desiredTFApplys[o.Name] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new terraformplans else claim existing.
		for _, o := range children.TerraformPlans {
			if desiredTFPlans[o.Name] == false {
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
