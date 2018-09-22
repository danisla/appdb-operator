package main

import (
	"fmt"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func makeConditionOrder(parent *appdbv1.AppDB) []appdbv1.AppDBConditionType {
	conditionOrder := make([]appdbv1.AppDBConditionType, 0)
	for _, c := range conditionStatusOrder {
		if c == appdbv1.ConditionTypeSnapshotLoadComplete && parent.Spec.LoadURL == "" {
			// Skip condition.
			continue
		}
		conditionOrder = append(conditionOrder, c)
	}
	return conditionOrder
}

func checkConditions(checkType appdbv1.AppDBConditionType, conditions map[appdbv1.AppDBConditionType]*appdbv1.AppDBCondition) error {
	waiting := []string{}

	for _, conditionType := range conditionDependencies[checkType] {
		condition := conditions[conditionType]
		if condition.Status != appdbv1.ConditionTrue {
			waiting = append(waiting, string(conditionType))
		}
	}

	if len(waiting) == 0 {
		return nil
	}

	return fmt.Errorf("Waiting on conditions: %s", strings.Join(waiting, ","))
}
