package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func reconcileSnapshotLoadComplete(condition *appdbv1.AppDBCondition, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}, appdbi appdbv1.AppDBInstance) appdbv1.ConditionStatus {
	newStatus := appdbv1.ConditionFalse
	jobName := fmt.Sprintf("appdb-%s-%s-load", appdbi.GetName(), parent.GetName())
	loadURL := parent.Spec.LoadURL
	if len(loadURL) >= 5 && loadURL[0:5] != "gs://" {
		// Relative url to bucket.
		loadURL = fmt.Sprintf("gs://%s/%s", tfDriverConfig.BackendBucket, parent.Spec.LoadURL)
	}
	job := makeLoadJob(jobName, parent.GetNamespace(), appdbi.Status.CloudSQL.InstanceName, loadURL, parent.Spec.DBName, parent.Spec.Users[0], appdbi.Status.CloudSQL.ServiceAccountEmail)
	if currJob, ok := children.Jobs[job.GetName()]; ok == true {
		// Wait for load job to complete.
		if currJob.Status.Succeeded == 1 {
			// load complete.
			newStatus = appdbv1.ConditionTrue
			children.claimChildAndGetCurrent(job, desiredChildren)
		} else if currJob.Status.Failed == *currJob.Spec.BackoffLimit {
			// Requeue job
			parent.Log("INFO", "Recreating SQL Load job")
		} else {
			children.claimChildAndGetCurrent(job, desiredChildren)
		}
	} else {
		// Create job
		children.claimChildAndGetCurrent(job, desiredChildren)
		parent.Log("INFO", "Created SQL load job from snapshot %s: %s", loadURL, job.GetName())
	}

	return newStatus
}
