package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
)

func reconcileCloudSQLProxyReady(condition *appdbv1.AppDBInstanceCondition, parent *appdbv1.AppDBInstance, status *appdbv1.AppDBInstanceOperatorStatus, children *AppDBInstanceChildren, desiredChildren *[]interface{}, tfapply tfv1.Terraform) appdbv1.ConditionStatus {
	newStatus := appdbv1.ConditionFalse

	reasons := []string{}

	// Create the Cloud SQL Proxy
	newSecret, newDeploy, newService, err := makeCloudSQLProxy(parent, tfapply)
	if err != nil {
		condition.Reason = fmt.Sprintf("Failed to make Cloud SQl Proxy: %v", err)
	} else {
		children.claimChildAndGetCurrent(newSecret, desiredChildren)
		children.claimChildAndGetCurrent(newDeploy, desiredChildren)
		children.claimChildAndGetCurrent(newService, desiredChildren)

		allReady := true

		// Cloud SQL Proxy Service Account Key Secret
		if secret, ok := children.Secrets[newSecret.GetName()]; ok == true {
			reasons = append(reasons, fmt.Sprintf("Secret/%s: CREATED", secret.GetName()))
			status.CloudSQL.ProxySecret = newSecret.GetName()
		} else {
			allReady = false
		}

		// Cloud SQL Proxy Deployment
		if deploy, ok := children.Deployments[newDeploy.GetName()]; ok == true {
			if deploy.Status.AvailableReplicas == deploy.Status.Replicas {
				// Deployment is ready.
				reasons = append(reasons, fmt.Sprintf("Deployment/%s: READY", deploy.GetName()))
			} else {
				reasons = append(reasons, fmt.Sprintf("Deployment/%s: %d/%d", deploy.GetName(), deploy.Status.AvailableReplicas, deploy.Status.Replicas))
				allReady = false
			}
		} else {
			allReady = false
		}

		// Cloud SQL Proxy Service
		if svc, ok := children.Services[newService.GetName()]; ok == true {
			reasons = append(reasons, fmt.Sprintf("Service/%s: CREATED", svc.GetName()))
			status.CloudSQL.ProxyService = svc.GetName()
			status.DBHost = fmt.Sprintf("%s.%s.svc.cluster.local", svc.GetName(), svc.GetNamespace())
		} else {
			allReady = false
		}

		if allReady {
			newStatus = appdbv1.ConditionTrue
		}
	}

	return newStatus
}
