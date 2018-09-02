package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDBOperatorState represents the string mapping of the possible controller states. See the const definition below for enumerated states.
type AppDBOperatorState string

// AppDBOperatorStatus is the status structure for the custom resource
type AppDBOperatorStatus struct {
	LastAppliedSig string             `json:"lastAppliedSig"`
	StateCurrent   AppDBOperatorState `json:"stateCurrent"`
}

// AppDB is the custom resource definition structure.
type AppDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AppDBSpec           `json:"spec,omitempty"`
	Status            AppDBOperatorStatus `json:"status"`
}

// AppDBSpec is the top level structure of the spec body
type AppDBSpec struct {
	AppDBInstance string      `json:"appDBInstance,omitempty"`
	Users         []AppDBUser `json:"users,omitempty"`
}

// AppDBUser is the spec element for a user
type AppDBUser struct {
	RW []string `json:"rw,omitempty"`
	RO []string `json:"ro,omitempty"`
}
