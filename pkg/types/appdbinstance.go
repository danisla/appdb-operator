package types

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDBInstance is the custom resource definition structure.
type AppDBInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AppDBInstanceSpec           `json:"spec,omitempty"`
	Status            AppDBInstanceOperatorStatus `json:"status"`
}

// AppDBInstanceOperatorStatus is the status structure for the custom resource
type AppDBInstanceOperatorStatus struct {
	Provisioning string                       `json:"provisioning"`
	CloudSQL     *AppDBInstanceCloudSQLStatus `json:"cloudSQL"`
}

// AppDBInstanceCloudSQLStatus is the status structure for the CloudSQL driver
type AppDBInstanceCloudSQLStatus struct {
	InstanceName   string
	TFApplyName    string
	TFApplyPodName string
	TFApplySig     string
}

// AppDBInstanceSpec is the top level structure of the spec body
type AppDBInstanceSpec struct {
	Driver AppDBDriver `json:"driver,omitempty"`
}

// AppDBDriver is the spec of the driver
type AppDBDriver struct {
	CloudSQLTerraform *AppDBCloudSQLTerraformDriver `json:"cloudSQLTerraform,omitempty"`
}

// AppDBCloudSQLDriver is the CloudSQL driver spec
type AppDBCloudSQLTerraformDriver struct {
	Params map[string]string `json:"params,omitempty"`
	Proxy  CloudSQLProxySpec `json:"proxy,omitempty"`
}

// CloudSQLProxySpec is the spec for a cloudsql proxy
type CloudSQLProxySpec struct {
	Image           string            `json:"image",omitempty`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Replicas        int               `json:"replicas,omitempty"`
}
