package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDBOperatorState represents the string mapping of the possible controller states. See the const definition below for enumerated states.
type AppDBOperatorState string

// AppDBOperatorStatus is the status structure for the custom resource
type AppDBOperatorStatus struct {
	LastAppliedSig    string                 `json:"lastAppliedSig"`
	StateCurrent      AppDBOperatorState     `json:"stateCurrent"`
	Provisioning      string                 `json:"provisioning"`
	CloudSQLDB        *AppDBCloudSQLDBStatus `json:"cloudSQLDB"`
	CredentialsSecret string                 `json:"credentialsSecret"`
}

// AppDBCloudSQLDBStatus is the status structure for the CloudSQL driver
type AppDBCloudSQLDBStatus struct {
	TFApplyName    string
	TFApplyPodName string
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
	AppDBInstance string   `json:"appDBInstance,omitempty"`
	DBName        string   `json:"dbName,omitempty"`
	Users         []string `json:"users,omitempty"`
}
