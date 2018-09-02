package types

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDBInstanceOperatorState represents the string mapping of the possible controller states. See the const definition below for enumerated states.
type AppDBInstanceOperatorState string

// AppDBInstance is the custom resource definition structure.
type AppDBInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AppDBInstanceSpec           `json:"spec,omitempty"`
	Status            AppDBInstanceOperatorStatus `json:"status"`
}

// AppDBInstanceOperatorStatus is the status structure for the custom resource
type AppDBInstanceOperatorStatus struct {
	LastAppliedSig string                     `json:"lastAppliedSig"`
	StateCurrent   AppDBInstanceOperatorState `json:"stateCurrent"`
}

// AppDBInstanceSpec is the top level structure of the spec body
type AppDBInstanceSpec struct {
	Driver AppDBDriver `json:"driver,omitempty"`
}

// AppDBDriver is the spec of the driver
type AppDBDriver struct {
	CloudSQL AppDBCloudSQLDriver `json:"cloudSQL,omitempty"`
}

// AppDBCloudSQLDriver is the CloudSQL driver spec
type AppDBCloudSQLDriver struct {
	Region          string            `json:"region,omitempty"`
	DatabaseVersion string            `json:"databaseVersion,omitempty"`
	Tier            string            `json:"tier,omitempty"`
	Proxy           CloudSQLProxySpec `json:"proxy,omitempty"`
}

// CloudSQLProxySpec is the spec for a cloudsql proxy
type CloudSQLProxySpec struct {
	Image           string            `json:"image",omitempty`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Replicas        int               `json:"replicas,omitempty"`
}
