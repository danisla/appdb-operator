package main

import (
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func claimChildAndGetCurrent(newChild interface{}, children *AppDBChildren, desiredChildren *[]interface{}) interface{} {
	var currChild interface{}
	switch o := newChild.(type) {
	case appdbv1.Terraform:
		if child, ok := children.TerraformApplys[o.GetName()]; ok == true {
			currChild = child
		}
	case corev1.Secret:
		if child, ok := children.Secrets[o.GetName()]; ok == true {
			currChild = child
		}
	case appdbv1.Job:
		if child, ok := children.Jobs[o.GetName()]; ok == true {
			currChild = child
		}
	}

	*desiredChildren = append(*desiredChildren, newChild)

	return currChild
}
