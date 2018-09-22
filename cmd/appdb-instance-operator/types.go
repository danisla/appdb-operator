package main

import (
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// ParentType represents the strign mapping to the possible parent types in the const below.
type ParentType string

const (
	ParentDBInstance = "appdbi"
)

// SyncRequest describes the payload from the CompositeController hook
type SyncRequest struct {
	Parent   appdbv1.AppDBInstance `json:"parent"`
	Children AppDBInstanceChildren `json:"children"`
}

// SyncResponse is the CompositeController response structure.
type SyncResponse struct {
	Status   appdbv1.AppDBInstanceOperatorStatus `json:"status"`
	Children []interface{}                       `json:"children"`
}

// AppDBInstanceChildren is the children definition passed by the CompositeController request for the controller.
type AppDBInstanceChildren struct {
	TerraformApplys map[string]tfv1.Terraform         `json:"Terraformapply.ctl.isla.solutions/v1"`
	TerraformPlans  map[string]tfv1.Terraform         `json:"Terraformplan.ctl.isla.solutions/v1"`
	Services        map[string]corev1.Service         `json:"Service.v1"`
	Deployments     map[string]appsv1beta1.Deployment `json:"Deployment.apps/v1beta1"`
	Secrets         map[string]corev1.Secret          `json:"Secret.v1"`
}

func (children *AppDBInstanceChildren) claimChildAndGetCurrent(newChild interface{}, desiredChildren *[]interface{}) interface{} {
	var currChild interface{}
	switch o := newChild.(type) {
	case appdbv1.Terraform:
		if child, ok := children.TerraformPlans[o.GetName()]; ok == true {
			currChild = child
		}
		if child, ok := children.TerraformApplys[o.GetName()]; ok == true {
			currChild = child
		}
	case corev1.Service:
		if child, ok := children.Services[o.GetName()]; ok == true {
			currChild = child
		}
	case appsv1beta1.Deployment:
		if child, ok := children.Deployments[o.GetName()]; ok == true {
			currChild = child
		}
	case corev1.Secret:
		if child, ok := children.Secrets[o.GetName()]; ok == true {
			currChild = child
		}
	}

	*desiredChildren = append(*desiredChildren, newChild)

	return currChild
}
