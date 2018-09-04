package main

import (
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

// Default image used for sql job pod, can be overridden using spec.Image and spec.ImagePullPolicy
const (
	DEFAULT_IMAGE             = "gcr.io/cloud-solutions-group/appdb-pod:latest"
	DEFAULT_IMAGE_PULL_POLICY = corev1.PullIfNotPresent
)

// ServiceAccount installed with Controller deployment
const (
	DEFAULT_POD_SERVICE_ACCOUNT = "terraform"
)

// Default max retries for failed pods
const (
	DEFAULT_POD_MAX_ATTEMPTS = 4
)

// Pod status for reporting pass/fail status of pod
const (
	// PodStatusFailed indicates that the max attempts for retry have failed.
	PodStatusFailed = "FAILED"
	PodStatusPassed = "COMPLETED"
)

const (
	// StateNone is the inital state for a new spec.
	StateNone = appdbv1.AppDBOperatorState("NONE")
	// StateIdle means there are no more changes pending
	StateIdle = appdbv1.AppDBOperatorState("IDLE")
	// StateWaitComplete is used to indicate that a wait is complete and to transition back through the idle handler.
	StateWaitComplete = appdbv1.AppDBOperatorState("WAIT_COMPLETE")
	// StateAppDBInstancePending means the controller is waiting for the AppDBInstance to become available.
	StateAppDBInstancePending = appdbv1.AppDBOperatorState("APPDB_INSTANCE_WAITING")
	// StateCloudSQLDBPending means the controller is waiting for the database creation within the instance.
	StateCloudSQLDBPending = appdbv1.AppDBOperatorState("APPDB_PENDING")
)

// ParentType represents the strign mapping to the possible parent types in the const below.
type ParentType string

const (
	ParentDB = "appdb"
)

// SyncRequest describes the payload from the CompositeController hook
type SyncRequest struct {
	Parent   appdbv1.AppDB `json:"parent"`
	Children AppDBChildren `json:"children"`
}

// SyncResponse is the CompositeController response structure.
type SyncResponse struct {
	Status   appdbv1.AppDBOperatorStatus `json:"status"`
	Children []interface{}               `json:"children"`
}

// AppDBChildren is the children definition passed by the CompositeController request for the controller.
type AppDBChildren struct {
	TerraformApplys map[string]tfv1.Terraform `json:"Terraformapply.ctl.isla.solutions/v1"`
	Secrets         map[string]corev1.Secret  `json:"Secret.v1"`
}
