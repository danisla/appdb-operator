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

	appDBInstance, err := getAppDBInstance(parent.ObjectMeta.Namespace, parent.Spec.AppDBInstance)
	if err != nil {
		// Wait for AppDBInstance provisioning status to COMPLETE
		myLog(parent, "INFO", fmt.Sprintf("Waiting for AppDBInstance: %s", parent.Spec.AppDBInstance))
		return StateAppDBInstancePending, nil
	}

	myLog(parent, "DEBUG", fmt.Sprintf("AppDBInstance %s is ready", appDBInstance.ObjectMeta.Name))

	if appDBInstance.Spec.Driver.CloudSQLTerraform != nil {
		myLog(parent, "DEBUG", fmt.Sprintf("AppDBInstance %s CloudSQL name: %s", appDBInstance.ObjectMeta.Name, appDBInstance.Status.CloudSQL.InstanceName))
	}

	return StateIdle, err
}
