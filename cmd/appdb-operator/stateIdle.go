package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func stateIdle(parentType ParentType, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}) (appdbv1.AppDBOperatorState, error) {
	var err error

	if status.StateCurrent == StateIdle && !changeDetected(parent, children, status) {
		return StateIdle, nil
	}

	if parent.Spec.AppDBInstance == "" {
		// Must have AppDBInstance
		myLog(parent, "ERROR", "Missing appDBInstance")
		return StateIdle, nil
	}

	if parent.Spec.DBName == "" {
		// Must have DBName
		myLog(parent, "ERROR", "Missing dbName")
		return StateIdle, nil
	}

	var appdbi appdbv1.AppDBInstance
	appdbi, err = getAppDBInstance(parent.ObjectMeta.Namespace, parent.Spec.AppDBInstance)
	if err != nil {
		// Wait for AppDBInstance provisioning status to COMPLETE
		myLog(parent, "INFO", fmt.Sprintf("Waiting for AppDBInstance: %s", parent.Spec.AppDBInstance))
		return StateAppDBInstancePending, nil
	}

	myLog(parent, "DEBUG", fmt.Sprintf("AppDBInstance %s is ready", appdbi.ObjectMeta.Name))

	if appdbi.Spec.Driver.CloudSQLTerraform != nil {
		myLog(parent, "DEBUG", fmt.Sprintf("AppDBInstance %s CloudSQL name: %s", appdbi.ObjectMeta.Name, appdbi.Status.CloudSQL.InstanceName))
	}

	// Terraform driver for CloudSQL database and users
	if appdbi.Spec.Driver.CloudSQLTerraform != nil {
		if status.StateCurrent != StateWaitComplete {

			tfApplyName := fmt.Sprintf("%s-%s", appdbi.ObjectMeta.Name, parent.Name)

			// Create new TerraformApply to create DB and users.
			tfapply, err := makeCloudSQLDBTerraform(tfApplyName, parent, appdbi)
			if err != nil {
				myLog(parent, "ERROR", fmt.Sprintf("Failed to generate TerraformApply spec for CloudSQL DB: %v", err))
				return StateIdle, nil
			}

			*desiredChildren = append(*desiredChildren, tfapply)

			myLog(parent, "INFO", fmt.Sprintf("Created CloudSQL DB TerraformApply: %s", tfapply.ObjectMeta.Name))

			status.CloudSQLDB = &appdbv1.AppDBCloudSQLDBStatus{
				TFApplyName: tfapply.ObjectMeta.Name,
			}

			return StateCloudSQLDBPending, err
		}
	} else {
		myLog(parent, "WARN", "Unsupported AppDBInstance driver")
	}

	return StateIdle, err
}
