package main

import (
	"fmt"
	"strings"
	"time"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	"github.com/jinzhu/copier"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func sync(parentType ParentType, parent *appdbv1.AppDBInstance, children *AppDBInstanceChildren) (*appdbv1.AppDBInstanceOperatorStatus, *[]interface{}, error) {
	var err error
	var status appdbv1.AppDBInstanceOperatorStatus
	copier.Copy(&status, &parent.Status)

	desiredChildren := make([]interface{}, 0)

	// Current time used for updating conditions
	tNow := metav1.NewTime(time.Now())

	// Verify required top level fields.
	if err = parent.Spec.Verify(); err != nil {
		parent.Log("ERROR", "Invalid spec: %v", err)
		status.Conditions = append(status.Conditions, appdbv1.AppDBInstanceCondition{
			Type:               appdbv1.ConditionTypeAppDBInstanceReady,
			Status:             appdbv1.ConditionFalse,
			LastProbeTime:      tNow,
			LastTransitionTime: tNow,
			Reason:             "Invalid spec",
			Message:            fmt.Sprintf("%v", err),
		})
		return &status, &desiredChildren, nil
	}

	// Map of condition types to conditions, converted to list of conditions after switch statement.
	conditions := parent.MakeConditions(tNow)
	conditionOrder := parent.GetConditionOrder()

	// Resources used in multiple conditions.
	var tfapply tfv1.Terraform

	// Reconcile each condition.
	for _, conditionType := range conditionOrder {
		condition := conditions[conditionType]
		newStatus := condition.Status

		// Skip processing conditions with unmet dependencies.
		if err = conditions.CheckConditions(conditionType); err != nil {
			newStatus = appdbv1.ConditionFalse
			condition.Reason = err.Error()
			if condition.Status != newStatus {
				condition.LastTransitionTime = tNow
				condition.Status = newStatus
			}
			continue
		}

		switch conditionType {
		case appdbv1.ConditionTypeAppDBInstanceTFPlanComplete:
			newStatus = reconcileTFPlanComplete(condition, parent, &status, children, &desiredChildren)

		case appdbv1.ConditionTypeAppDBInstanceTFApplyComplete:
			newStatus, tfapply = reconcileTFApplyComplete(condition, parent, &status, children, &desiredChildren)

		case appdbv1.ConditionTypeAppDBInstanceCloudSQLProxyReady:
			newStatus = reconcileCloudSQLProxyReady(condition, parent, &status, children, &desiredChildren, tfapply)

		case appdbv1.ConditionTypeAppDBInstanceReady:
			newStatus = appdbv1.ConditionTrue
			notReady := []string{}
			for _, c := range conditionOrder {
				if c != appdbv1.ConditionTypeAppDBInstanceReady && conditions[c].Status != appdbv1.ConditionTrue {
					notReady = append(notReady, string(c))
					newStatus = appdbv1.ConditionFalse
				}
			}
			if len(notReady) > 0 {
				condition.Reason = fmt.Sprintf("Waiting for conditions: %s", strings.Join(notReady, ","))
				if status.Provisioning != appdbv1.ProvisioningStatusFailed {
					status.Provisioning = appdbv1.ProvisioningStatusPending
				}
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

	// Update status with new conditions.
	parent.SetConditionStatus(conditions)

	return &status, &desiredChildren, nil
}
