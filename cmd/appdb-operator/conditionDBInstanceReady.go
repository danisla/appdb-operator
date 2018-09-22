package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func reconcileAppDBIReady(condition *appdbv1.AppDBCondition, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}) (appdbv1.ConditionStatus, appdbv1.AppDBInstance) {
	newStatus := appdbv1.ConditionFalse
	appdbi, err := getAppDBInstance(parent.GetNamespace(), parent.Spec.AppDBInstance)
	if err == nil {
		if status.AppDBInstanceSig != "" && status.AppDBInstanceSig != calcParentSig(appdbi.Spec, "") {
			// AppDBInstance spec changed.
			condition.Reason = fmt.Sprintf("AppDBInstance/%s change detected", appdbi.GetName())
		} else {
			if appdbi.Status.Provisioning == appdbv1.ProvisioningStatusComplete {
				newStatus = appdbv1.ConditionTrue
				status.AppDBInstanceSig = calcParentSig(appdbi.Spec, "")
			}
			condition.Reason = fmt.Sprintf("AppDBInstance/%s: %s", appdbi.GetName(), appdbi.Status.Provisioning)
		}
	} else {
		condition.Reason = fmt.Sprintf("AppDBInstance/%s: Not found", parent.Spec.AppDBInstance)
	}

	return newStatus, appdbi
}
