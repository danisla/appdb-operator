package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func sync(parentType ParentType, parent *appdbv1.AppDB, children *AppDBChildren) (*appdbv1.AppDBOperatorStatus, *[]interface{}, error) {
	status := makeStatus(parent, children)
	currState := status.StateCurrent
	desiredChildren := make([]interface{}, 0)
	nextState := currState[0:1] + currState[1:] // string copy of currState

	var err error
	switch currState {
	case StateNone, StateIdle, StateWaitComplete:
		nextState, err = stateIdle(parentType, parent, status, children, &desiredChildren)

	case StateAppDBInstancePending:
		nextState, err = stateAppDBInstancePending(parentType, parent, status)

	case StateCloudSQLDBPending:
		nextState, err = stateCloudSQLDBRunning(parentType, parent, status, children, &desiredChildren)
	}

	if err != nil {
		return status, &desiredChildren, err
	}

	if status.StateCurrent != StateNone {
		// Claim the terraformapplys.
		for _, o := range children.TerraformApplys {
			desiredChildren = append(desiredChildren, o)
		}

		// Claim the secrets.
		for _, o := range children.Secrets {
			desiredChildren = append(desiredChildren, o)
		}
	}

	// Advance the state
	if status.StateCurrent != nextState {
		myLog(parent, "INFO", fmt.Sprintf("State %s -> %s", status.StateCurrent, nextState))
	}
	status.StateCurrent = nextState

	return status, &desiredChildren, nil
}
