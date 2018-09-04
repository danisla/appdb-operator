package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func stateCloudSQLRunning(parentType ParentType, parent *appdbv1.AppDBInstance, status *appdbv1.AppDBInstanceOperatorStatus, children *AppDBInstanceChildren) (appdbv1.AppDBInstanceOperatorState, error) {

	status.Provisioning = appdbv1.ProvisioningStatusPending

	tfapply, ok := children.TerraformApplys[status.CloudSQL.TFApplyName]
	if ok == false {
		myLog(parent, "WARN", fmt.Sprintf("TerraformApply not found in children while in state %s", status.StateCurrent))
		return StateCloudSQLPending, nil
	}

	status.CloudSQL.TFApplyPodName = tfapply.Status.PodName

	switch tfapply.Status.PodStatus {
	case "FAILED":
		myLog(parent, "ERROR", "CloudSQL TerraformApply failed.")
		status.Provisioning = appdbv1.ProvisioningStatusFailed

		// Set the parent signature
		// If parent changes from here on, we'll go back through the idle state, creating new resources.
		status.LastAppliedSig = calcParentSig(parent, "")

		return StateWaitComplete, nil
	case "COMPLETED":
		myLog(parent, "INFO", "CloudSQL TerraformApply completed.")
		status.Provisioning = appdbv1.ProvisioningStatusComplete
		if nameVar, ok := tfapply.Status.TFOutput["name"]; ok == false {
			return StateCloudSQLPending, fmt.Errorf("Output variable 'name' not found in TerraformApply status")
		} else {
			status.CloudSQL.InstanceName = nameVar.Value
		}

		// Set the parent signature
		// If parent changes from here on, we'll go back through the idle state, creating new resources.
		status.LastAppliedSig = calcParentSig(parent, "")

		return StateWaitComplete, nil
	}

	return StateCloudSQLPending, nil
}
