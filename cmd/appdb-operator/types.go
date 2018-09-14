package main

import (
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
	Jobs            map[string]batchv1.Job    `json:"Job.batch/v1"`
}

// Order of condition status
var conditionStatusOrder = []appdbv1.AppDBConditionType{
	appdbv1.ConditionTypeAppDBInstanceReady,
	appdbv1.ConditionTypeDBCreateComplete,
	appdbv1.ConditionTypeCredentialsSecretCreated,
	appdbv1.ConditionTypeSnapshotLoadComplete,
	appdbv1.ConditionTypeAppDBReady,
}

var conditionDependencies = map[appdbv1.AppDBConditionType][]appdbv1.AppDBConditionType{
	appdbv1.ConditionTypeDBCreateComplete: []appdbv1.AppDBConditionType{
		appdbv1.ConditionTypeAppDBInstanceReady,
	},
	appdbv1.ConditionTypeCredentialsSecretCreated: []appdbv1.AppDBConditionType{
		appdbv1.ConditionTypeAppDBInstanceReady,
		appdbv1.ConditionTypeDBCreateComplete,
	},
}
