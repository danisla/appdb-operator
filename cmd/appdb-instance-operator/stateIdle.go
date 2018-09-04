package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func stateIdle(parentType ParentType, parent *appdbv1.AppDBInstance, status *appdbv1.AppDBInstanceOperatorStatus, children *AppDBInstanceChildren, desiredChildren *[]interface{}) (appdbv1.AppDBInstanceOperatorState, error) {
	var err error

	if status.StateCurrent == StateIdle && !changeDetected(parent, children, status) {
		return StateIdle, nil
	}

	// Create instance using CloudSQLTerraform driver.
	// When complete, the driver is expected to set status.Provisioning to COMPLETE or FAILED.
	if parent.Spec.Driver.CloudSQLTerraform != nil {
		if status.StateCurrent != StateWaitComplete {
			// Create new instance.
			myLog(parent, "INFO", "Database driver is Cloud SQL")

			tfapply, err := makeCloudSQLTerraform(parent.Name, parent.ObjectMeta.Namespace, parent.Spec.Driver.CloudSQLTerraform)
			if err != nil {
				myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformApply spec for CloudSQL: %v", err))
				return StateIdle, nil
			}

			*desiredChildren = append(*desiredChildren, tfapply)

			myLog(parent, "INFO", fmt.Sprintf("Created CloudSQL TerraformApply: %s", parent.Name))

			status.CloudSQL = &appdbv1.AppDBInstanceCloudSQLStatus{
				TFApplyName: parent.Name,
			}

			return StateCloudSQLPending, err
		}

	} else {
		myLog(parent, "ERROR", "Unsupported driver")
		return StateIdle, err
	}

	return StateIdle, err
}
