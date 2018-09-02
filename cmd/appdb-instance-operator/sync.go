package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func sync(parentType ParentType, parent *appdbv1.AppDBInstance, children *AppDBInstanceChildren) (*appdbv1.AppDBInstanceOperatorStatus, *[]interface{}, error) {
	status := makeStatus(parent, children)
	currState := status.StateCurrent
	desiredChildren := make([]interface{}, 0)
	nextState := currState[0:1] + currState[1:] // string copy of currState

	var err error
	switch currState {
	case StateNone, StateIdle:
		nextState, err = stateIdle(parentType, parent, status, children, &desiredChildren)

	}

	if err != nil {
		return status, &desiredChildren, err
	}

	// Claim the terraformapplys.
	for _, o := range children.TerraformApplys {
		desiredChildren = append(desiredChildren, o)
	}

	// Claim the deploymenys.
	for _, o := range children.Deployments {
		desiredChildren = append(desiredChildren, o)
	}

	// Claim the services.
	for _, o := range children.Services {
		desiredChildren = append(desiredChildren, o)
	}

	// Advance the state
	if status.StateCurrent != nextState {
		myLog(parent, "INFO", fmt.Sprintf("State %s -> %s", status.StateCurrent, nextState))
	}
	status.StateCurrent = nextState

	return status, &desiredChildren, nil
}
