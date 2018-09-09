package main

import (
	"fmt"
	"strconv"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	"github.com/jinzhu/copier"
)

func sync(parentType ParentType, parent *appdbv1.AppDBInstance, children *AppDBInstanceChildren) (*appdbv1.AppDBInstanceOperatorStatus, *[]interface{}, error) {
	var status appdbv1.AppDBInstanceOperatorStatus
	copier.Copy(&status, &parent.Status)

	desiredTFApplys := make(map[string]bool, 0)
	desiredTFPlans := make(map[string]bool, 0)
	desiredSecrets := make(map[string]bool, 0)
	desiredDeployments := make(map[string]bool, 0)
	desiredServices := make(map[string]bool, 0)
	desiredChildren := make([]interface{}, 0)

	if parent.Spec.Driver.CloudSQLTerraform != nil {

		tfApplyName := fmt.Sprintf("appdbi-%s", parent.Name)
		planRunning := false

		if tfplan, ok := children.TerraformPlans[tfApplyName]; ok == true {

			status.Provisioning = appdbv1.ProvisioningStatusPending

			if status.CloudSQL == nil {
				myLog(parent, "WARN", "Found TerraformPlan in children, but status.CloudSQL was nil, re-sync collision.")
				// Delete TerraformPlan and try again.
				desiredTFPlans[tfApplyName] = true
			} else {

				// Handle terraform plan
				mySig := calcParentSig(parent.Spec, "")
				tfplanSig := tfplan.Annotations["appdb-parent-sig"]

				if mySig == tfplanSig {
					// TODO: something this throws a nil pointer dereference... maybe a resync collision.
					status.CloudSQL.TFPlanPodName = tfplan.Status.PodName

					planRunning = true

					if tfplan.Status.PodStatus == "COMPLETED" {
						// Check plan
						if tfplan.Status.TFPlanDiff.Destroyed > 0 {
							myLog(parent, "ERROR", "TerraformPlan contains destroy actions, skipping patch.")

							// Retry in 60 seconds.
							tfplanFishedAtTime, err := time.Parse(time.RFC3339, tfplan.Status.FinishedAt)
							if err != nil {
								myLog(parent, "WARN", fmt.Sprintf("Failed to parse tfplan finished at time: %v", err))
							} else {
								if time.Since(tfplanFishedAtTime).Seconds() > 60 {
									myLog(parent, "INFO", "Retrying TerraformPlan")
									// Setting desiredTFPlans to true will cause it to be omitted during the claim phase, therefore deleting it.
									desiredTFPlans[tfApplyName] = true
								}
							}
						} else {
							myLog(parent, "INFO", "TerraformPlan contains no destroy actions, proceeding with update.")

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
											TFApplyName: tfapply.GetName(),
											TFApplySig:  calcParentSig(parent.Spec, ""),
										}

										desiredTFApplys[tfApplyName] = true
										desiredChildren = append(desiredChildren, tfapply)
									}
								} else {
									// No existing tfapply, create new one.
									status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
										TFApplyName: tfapply.GetName(),
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
		}

		if tfapply, ok := children.TerraformApplys[tfApplyName]; ok == true {
			mySig := calcParentSig(parent.Spec, "")
			tfapplySig := tfapply.Annotations["appdb-parent-sig"]

			if mySig == tfapplySig {

				status.CloudSQL.TFApplyPodName = tfapply.Status.PodName

				if tfapply.Status.PodStatus == "COMPLETED" {
					status.Provisioning = appdbv1.ProvisioningStatusComplete

					// Get the "name" output variable.
					if nameVar, ok := tfapply.Status.TFOutput["name"]; ok == false {
						myLog(parent, "ERROR", fmt.Sprintf("Output variable 'name' not found in status of TerraformApply: %s", tfapply.GetName()))
					} else {
						status.CloudSQL.InstanceName = nameVar.Value
					}

					// Get the "connection" output variable.
					if connVar, ok := tfapply.Status.TFOutput["connection"]; ok == false {
						myLog(parent, "ERROR", fmt.Sprintf("Output variable 'connection' not found in status of TerraformApply: %s", tfapply.GetName()))
					} else {
						status.CloudSQL.ConnectionName = connVar.Value
					}

					// Get the "port" output variable.
					if portVar, ok := tfapply.Status.TFOutput["port"]; ok == false {
						myLog(parent, "ERROR", fmt.Sprintf("Output variable 'port' not found in status of TerraformApply: %s", tfapply.GetName()))
					} else {
						port, err := strconv.Atoi(portVar.Value)
						if err != nil {
							myLog(parent, "ERROR", fmt.Sprintf("Output variable 'port' could not be parsed as int: %s", portVar.Value))
						}
						status.CloudSQL.Port = int32(port)
					}

					// Create the Cloud SQL Proxy
					secret, deploy, svc, err := makeCloudSQLProxy(parent, tfapply)
					if err != nil {
						myLog(parent, "ERROR", fmt.Sprintf("Failed to generate cloud sql proxy spec: %v", err))
					} else {

						// Cloud SQL Proxy Service Account Key Secret
						if _, ok := children.Secrets[secret.GetName()]; ok == false {
							myLog(parent, "INFO", fmt.Sprintf("Creating Cloud SQL Proxy secret: %s", secret.GetName()))
							desiredSecrets[secret.GetName()] = true
							desiredChildren = append(desiredChildren, secret)
						}

						// Cloud SQL Proxy Deployment
						if _, ok := children.Deployments[deploy.GetName()]; ok == false {
							myLog(parent, "INFO", fmt.Sprintf("Creating Cloud SQL Proxy deployment: %s", deploy.GetName()))
							desiredDeployments[deploy.GetName()] = true
							desiredChildren = append(desiredChildren, deploy)
						}

						// Cloud SQL Proxy Service
						if _, ok := children.Services[svc.GetName()]; ok == false {
							myLog(parent, "INFO", fmt.Sprintf("Creating Cloud SQL Proxy service: %s", svc.GetName()))
							desiredServices[svc.GetName()] = true
							desiredChildren = append(desiredChildren, svc)
						}

						status.CloudSQL.ProxyService = svc.GetName()

						status.DBHost = fmt.Sprintf("%s.%s.svc.cluster.local", svc.GetName(), svc.GetNamespace())
						status.DBPort = status.CloudSQL.Port
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

						status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
							TFPlanName: tfApplyName,
							TFPlanSig:  calcParentSig(parent.Spec, ""),
						}

						desiredTFPlans[tfApplyName] = true
						desiredChildren = append(desiredChildren, tfplan)

						myLog(parent, "INFO", fmt.Sprintf("Created TerraformPlan: %s", tfApplyName))
					}
				}
			}
		} else {
			if planRunning == false {
				// Create new TerraformPlan first before provisioning DB instance.
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

					myLog(parent, "INFO", fmt.Sprintf("Created TerraformPlan: %s", tfApplyName))
				}
			}
		}

		// Claim new terraformapplys else claim existing.
		for _, o := range children.TerraformApplys {
			if desiredTFApplys[o.GetName()] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new terraformplans else claim existing.
		for _, o := range children.TerraformPlans {
			if desiredTFPlans[o.GetName()] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new secrets else claim existing.
		for _, o := range children.Secrets {
			if desiredSecrets[o.GetName()] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new deployments else claim existing.
		for _, o := range children.Deployments {
			if desiredDeployments[o.GetName()] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}

		// Claim new services else claim existing.
		for _, o := range children.Services {
			if desiredServices[o.GetName()] == false {
				desiredChildren = append(desiredChildren, o)
			}
		}
	} else {
		myLog(parent, "WARN", "Unsupported AppDBInstance driver")
	}

	return &status, &desiredChildren, nil
}
