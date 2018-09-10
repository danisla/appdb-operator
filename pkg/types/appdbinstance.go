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
	DBHost       string                       `json:"dbHost"`
	DBPort       int32                        `json:"dbPort"`
	CloudSQL     *AppDBInstanceCloudSQLStatus `json:"cloudSQL"`
}

// AppDBInstanceCloudSQLStatus is the status structure for the CloudSQL driver
type AppDBInstanceCloudSQLStatus struct {
	InstanceName   string `json:"instanceName,omitempty"`
	ConnectionName string `json:"connectionName,omitempty"`
	Port           int32  `json:"port,omitempty"`
	ProxyService   string `json:"proxyService,omitempty"`
	ProxySecret    string `json:"proxySecret,omitempty"`
	TFApplyName    string `json:"tfapplyName,omitempty"`
	TFApplyPodName string `json:"tfapplyPodName,omitempty"`
	TFApplySig     string `json:"tfapplySig,omitempty"`
	TFPlanName     string `json:"tfplanName,omitempty"`
	TFPlanPodName  string `json:"tfplanPodName,omitempty"`
	TFPlanSig      string `json:"tfplanSig,omitempty"`
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
	Replicas        int32             `json:"replicas,omitempty"`
}
