package types

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	tfv1 "github.com/danisla/terraform-operator/pkg/types"
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

// Log is a conventional log method to print the parent name and kind before the log message.
func (parent *AppDBInstance) Log(level, msgfmt string, fmtargs ...interface{}) {
	log.Printf("[%s][%s][%s] %s", level, parent.Kind, parent.Name, fmt.Sprintf(msgfmt, fmtargs...))
}

// GetSig returns a hash of the current parent spec.
func (parent *AppDBInstance) GetSig() string {
	hasher := sha1.New()
	data, err := json.Marshal(&parent.Spec)
	if err != nil {
		parent.Log("ERROR", "Failed to convert parent spec to JSON, this is a bug.")
		return ""
	}
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// MakeConditions initializes a new AppDBConditions struct
func (parent *AppDBInstance) MakeConditions(initTime metav1.Time) AppDBInstanceConditions {
	conditions := make(map[AppDBInstanceConditionType]*AppDBInstanceCondition, 0)

	// Extract existing conditions from status and copy to conditions map for easier lookup.
	for _, c := range parent.GetConditionOrder() {
		// Search for condition type in conditions.
		found := false
		for _, condition := range parent.Status.Conditions {
			if condition.Type == c {
				found = true
				condition.LastProbeTime = initTime
				condition.Reason = ""
				condition.Message = ""
				conditions[c] = &condition
				break
			}
		}
		if found == false {
			// Initialize condition with unknown state
			conditions[c] = &AppDBInstanceCondition{
				Type:               c,
				Status:             ConditionUnknown,
				LastProbeTime:      initTime,
				LastTransitionTime: initTime,
			}
		}
	}

	return conditions
}

// GetConditionOrder returns an ordered slice of the conditions as the should appear in the status.
// This is dependent on which fields are provided in the parent spec.
func (parent *AppDBInstance) GetConditionOrder() []AppDBInstanceConditionType {
	desiredOrder := []AppDBInstanceConditionType{
		ConditionTypeAppDBInstanceTFPlanComplete,
		ConditionTypeAppDBInstanceTFApplyComplete,
		ConditionTypeAppDBInstanceCloudSQLProxyReady,
		ConditionTypeAppDBInstanceReady,
	}

	conditionOrder := make([]AppDBInstanceConditionType, 0)
	for _, c := range desiredOrder {
		if parent.Spec.Driver.CloudSQLTerraform == nil && (c == ConditionTypeAppDBInstanceTFPlanComplete || c == ConditionTypeAppDBInstanceTFApplyComplete) {
			// Skip Terraform driver conditions
			continue
		}
		conditionOrder = append(conditionOrder, c)
	}
	return conditionOrder
}

// AppDBInstanceOperatorStatus is the status structure for the custom resource
type AppDBInstanceOperatorStatus struct {
	Provisioning ProvisioningStatus           `json:"provisioning,omitempty"`
	DBHost       string                       `json:"dbHost,omitempty"`
	DBPort       int32                        `json:"dbPort,omitempty"`
	CloudSQL     *AppDBInstanceCloudSQLStatus `json:"cloudSQL,omitempty"`
	Conditions   []AppDBInstanceCondition     `json:"conditions,omitempty"`
}

// GetConditionOrder returns an ordered slice of the conditions as the should appear in the status.
func (status *AppDBInstanceOperatorStatus) GetConditionOrder() []AppDBInstanceConditionType {
	return []AppDBInstanceConditionType{
		ConditionTypeAppDBInstanceTFPlanComplete,
		ConditionTypeAppDBInstanceTFApplyComplete,
		ConditionTypeAppDBInstanceCloudSQLProxyReady,
		ConditionTypeAppDBInstanceReady,
	}
}

// AppDBInstanceCondition defines the format for a status condition element.
type AppDBInstanceCondition struct {
	Type               AppDBInstanceConditionType `json:"type"`
	Status             ConditionStatus            `json:"status"`
	LastProbeTime      metav1.Time                `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time                `json:"lastTransitionTime,omitempty"`
	Reason             string                     `json:"reason,omitempty"`
	Message            string                     `json:"message,omitempty"`
}

// AppDBInstanceConditions is a map of the condition types to their condition.
type AppDBInstanceConditions map[AppDBInstanceConditionType]*AppDBInstanceCondition

// AppDBInstanceConditionType is a valid value for AppDBInstanceCondition.Type
type AppDBInstanceConditionType string

// The condition type constants listed below are in the order they should roughly happen and in the order they
// exist in the status.conditions list. This gives visibility to what the operator is doing.
// Some conditions can be satisfied in parallel with others.
const (
	// ConditionTypeAppDBInstanceTFPlanComplete is True when the TerraformPlan has completed successfully.
	ConditionTypeAppDBInstanceTFPlanComplete AppDBInstanceConditionType = "TerraformPlanComplete"
	// ConditionTypeAppDBInstanceTFApplyComplete is True when the TerraformApply has completed successfully.
	ConditionTypeAppDBInstanceTFApplyComplete AppDBInstanceConditionType = "TerraformApplyComplete"
	// ConditionTypeAppDBInstanceCloudSQLProxyReady is True when the Cloud SQL Proxy has been created and all replicas are available.
	ConditionTypeAppDBInstanceCloudSQLProxyReady AppDBInstanceConditionType = "CloudSQLProxyReady"
	// ConditionTypeAppDBInstanceReady is True when all prior conditions are ready
	ConditionTypeAppDBInstanceReady AppDBInstanceConditionType = "Ready"
)

// GetDependencies returns a map of condition type names to an ordered slice of dependent condition types.
func (conditionType *AppDBInstanceConditionType) GetDependencies() []AppDBInstanceConditionType {
	switch *conditionType {
	case ConditionTypeAppDBInstanceCloudSQLProxyReady:
		return []AppDBInstanceConditionType{
			ConditionTypeAppDBInstanceTFApplyComplete,
		}
	}
	return []AppDBInstanceConditionType{}
}

// CheckConditions verifies that all given conditions have been met for the given conditionType on the receiving conditions.
func (conditions AppDBInstanceConditions) CheckConditions(conditionType AppDBInstanceConditionType) error {
	waiting := []string{}
	for _, t := range conditionType.GetDependencies() {
		condition := conditions[t]
		if condition.Status != ConditionTrue {
			waiting = append(waiting, string(t))
		}
	}

	if len(waiting) == 0 {
		return nil
	}

	return fmt.Errorf("Waiting on conditions: %s", strings.Join(waiting, ","))
}

// AppDBInstanceCloudSQLStatus is the status structure for the CloudSQL driver
type AppDBInstanceCloudSQLStatus struct {
	InstanceName        string         `json:"instanceName,omitempty"`
	ServiceAccountEmail string         `json:"serviceAccountEmail,omitempty"`
	ConnectionName      string         `json:"connectionName,omitempty"`
	Port                int32          `json:"port,omitempty"`
	ProxyService        string         `json:"proxyService,omitempty"`
	ProxySecret         string         `json:"proxySecret,omitempty"`
	TFApplyName         string         `json:"tfapplyName,omitempty"`
	TFApplyPodName      string         `json:"tfapplyPodName,omitempty"`
	TFPlanName          string         `json:"tfplanName,omitempty"`
	TFPlanPodName       string         `json:"tfplanPodName,omitempty"`
	TFPlanStatus        tfv1.PodStatus `json:"tfPlanStatus,omitempty"`
	LastPlanSig         string         `json:"lastPlanSig,omitempty"`
}

// AppDBInstanceSpec is the top level structure of the spec body
type AppDBInstanceSpec struct {
	Driver AppDBDriver `json:"driver,omitempty"`
}

// Verify checks all required fields in the spec.
func (spec *AppDBInstanceSpec) Verify() error {
	if spec.Driver.CloudSQLTerraform == nil {
		return fmt.Errorf("Missing spec.Driver.CloudSQLTerraform")
	}
	if err := spec.Driver.CloudSQLTerraform.Verify(); err != nil {
		return err
	}

	return nil
}

// AppDBDriver is the spec of the driver
type AppDBDriver struct {
	CloudSQLTerraform *AppDBCloudSQLTerraformDriver `json:"cloudSQLTerraform,omitempty"`
}

// AppDBCloudSQLTerraformDriver is the CloudSQL driver spec
type AppDBCloudSQLTerraformDriver struct {
	// Params are Terraform variable values passed through to the config as TF_VARs
	Params []DriverParam `json:"params,omitempty"`
	// Proxy is the CloudSQL proxy spec. It is optional, default values will be used if not provided.
	Proxy *CloudSQLProxySpec `json:"proxy,omitempty"`
}

// DriverParam is a key-value element for the driver spec.
type DriverParam struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// Verify checks for all required values in the terraform driver spec.
func (spec *AppDBCloudSQLTerraformDriver) Verify() error {
	if len(spec.Params) == 0 {
		return fmt.Errorf("Missing spec.driver.cloudSQLTerraform.params list of name,value pairs")
	}
	return nil
}

// CloudSQLProxySpec is the spec for a cloudsql proxy
type CloudSQLProxySpec struct {
	Image           string            `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Replicas        int32             `json:"replicas,omitempty"`
}
