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
	if err = parent.Spec.Verify(); err != nil {
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
	conditions := parent.MakeConditions(tNow)
	conditionOrder := parent.GetConditionOrder()

	// Resources used in multiple conditions.
	var appdbi appdbv1.AppDBInstance
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
		case appdbv1.ConditionTypeSpecAppDBInstanceReady:
			newStatus, appdbi = reconcileAppDBIReady(condition, parent, &status, children, &desiredChildren)

		case appdbv1.ConditionTypeDBCreateComplete:
			newStatus, tfapply = reconcileDBCreateComplete(condition, parent, &status, children, &desiredChildren, appdbi)

		case appdbv1.ConditionTypeCredentialsSecretCreated:
			newStatus = reconcileSecretCreated(condition, parent, &status, children, &desiredChildren, appdbi, tfapply)

		case appdbv1.ConditionTypeSnapshotLoadComplete:
			newStatus = reconcileSnapshotLoadComplete(condition, parent, &status, children, &desiredChildren, appdbi)

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
