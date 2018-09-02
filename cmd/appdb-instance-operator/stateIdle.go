package main

import (
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func stateIdle(parentType ParentType, parent *appdbv1.AppDBInstance, status *appdbv1.AppDBInstanceOperatorStatus, children *AppDBInstanceChildren, desiredChildren *[]interface{}) (appdbv1.AppDBInstanceOperatorState, error) {
	var err error

	if status.StateCurrent == StateIdle && !changeDetected(parent, children, status) {
		return StateIdle, nil
	}

	return StateIdle, err
}
