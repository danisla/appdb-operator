package main

import (
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func stateIdle(parentType ParentType, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}) (appdbv1.AppDBOperatorState, error) {
	var err error

	if status.StateCurrent == StateIdle && !changeDetected(parent, children, status) {
		return StateIdle, nil
	}

	return StateIdle, err
}
