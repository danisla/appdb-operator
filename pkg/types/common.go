package types

import (
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProvisioningStatus represents the string mapping to the possible status.Provisioning values. See the const definition below for enumerated states.
type ProvisioningStatus string

const (
	ProvisioningStatusPending  ProvisioningStatus = "PENDING"
	ProvisioningStatusFailed   ProvisioningStatus = "FAILED"
	ProvisioningStatusComplete ProvisioningStatus = "COMPLETE"
)

// Terraform is a copy of tfv1.Terraform with the exception of the status field.
// This is used when marshaling so that the Status field does not interfere.
type Terraform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              tfv1.TerraformSpec     `json:"spec,omitempty"`
	SpecFrom          tfv1.TerraformSpecFrom `json:"specFrom,omitempty"`
}

// Job is a copy of batchv1.Job with the exception of the status field.
// This is used when marshaling so that the Status field does not interfere.
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              batchv1.JobSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
