package types

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDBOperatorStatus is the status structure for the custom resource
type AppDBOperatorStatus struct {
	Provisioning       ProvisioningStatus     `json:"provisioning,omitempty"`
	AppDBInstanceSig   string                 `json:"appDBInstanceSig,omitempty"`
	CloudSQLDB         *AppDBCloudSQLDBStatus `json:"cloudSQLDB,omitempty"`
	CredentialsSecrets map[string]string      `json:"credentialsSecrets,omitempty"`
	Conditions         []AppDBCondition       `json:"conditions,omitempty"`
}

// AppDBCondition defines the format for a status condition element.
type AppDBCondition struct {
	Type               AppDBConditionType `json:"type"`
	Status             ConditionStatus    `json:"status"`
	LastProbeTime      metav1.Time        `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Message            string             `json:"message,omitempty"`
}

// AppDBConditions is a map of the condition types to their condition.
type AppDBConditions map[AppDBConditionType]*AppDBCondition

// AppDBCloudSQLDBStatus is the status structure for the CloudSQL driver
type AppDBCloudSQLDBStatus struct {
	TFApplyName    string `json:"tfapplyName,omitempty"`
	TFApplyPodName string `json:"tfapplyPodName,omitempty"`
	TFApplySig     string `json:"tfapplySig,omitempty"`
}

// AppDBConditionType is a valid value for AppDBCondition.Type
type AppDBConditionType string

// The condition type constants listed below are in the order they should roughly happen and in the order they
// exist in the status.conditions list. This gives visibility to what the operator is doing.
// Some conditions can be satisfied in parallel with others.
const (
	// ConditionTypeSpecAppDBInstanceReady is True when the AppDBInstance resource provided in the spec is available and ready.
	ConditionTypeSpecAppDBInstanceReady AppDBConditionType = "AppDBInstanceReady"
	// ConditionTypeDBCreateComplete is True when the DB create driver action is complete.
	ConditionTypeDBCreateComplete AppDBConditionType = "DBCreateComplete"
	// ConditionTypeSnapshotLoadComplete is True when the Job for loading SQL data has been created and is complete.
	ConditionTypeSnapshotLoadComplete AppDBConditionType = "SnapshotLoadComplete"
	// ConditionTypeCredentialsSecretCreated is True when the secret containing the database credentials and info has been created.
	ConditionTypeCredentialsSecretCreated AppDBConditionType = "CredentialsSecretCreated"
	// ConditionTypeAppDBReady means that all prior conditions are ready
	ConditionTypeAppDBReady AppDBConditionType = "Ready"
)

// GetDependencies returns a map of condition type names to an ordered slice of dependent condition types.
func (conditionType *AppDBConditionType) GetDependencies() []AppDBConditionType {
	switch *conditionType {
	case ConditionTypeDBCreateComplete:
		return []AppDBConditionType{
			ConditionTypeSpecAppDBInstanceReady,
		}
	case ConditionTypeCredentialsSecretCreated:
		return []AppDBConditionType{
			ConditionTypeDBCreateComplete,
		}
	case ConditionTypeSnapshotLoadComplete:
		return []AppDBConditionType{
			ConditionTypeCredentialsSecretCreated,
		}
	}
	return []AppDBConditionType{}
}

// CheckConditions verifies that all given conditions have been met for the given conditionType on the receiving conditions.
func (conditions AppDBConditions) CheckConditions(conditionType AppDBConditionType) error {
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

// AppDB is the custom resource definition structure.
type AppDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AppDBSpec           `json:"spec,omitempty"`
	Status            AppDBOperatorStatus `json:"status"`
}

// Log is a conventional log method to print the parent name and kind before the log message.
func (parent *AppDB) Log(level, msgfmt string, fmtargs ...interface{}) {
	log.Printf("[%s][%s][%s] %s", level, parent.Kind, parent.Name, fmt.Sprintf(msgfmt, fmtargs...))
}

// GetSig returns a hash of the current parent spec.
func (parent *AppDB) GetSig() string {
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
func (parent *AppDB) MakeConditions(initTime metav1.Time) AppDBConditions {
	conditions := make(map[AppDBConditionType]*AppDBCondition, 0)

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
			conditions[c] = &AppDBCondition{
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
func (parent *AppDB) GetConditionOrder() []AppDBConditionType {
	desiredOrder := []AppDBConditionType{
		ConditionTypeSpecAppDBInstanceReady,
		ConditionTypeDBCreateComplete,
		ConditionTypeCredentialsSecretCreated,
		ConditionTypeSnapshotLoadComplete,
		ConditionTypeAppDBReady,
	}

	conditionOrder := make([]AppDBConditionType, 0)
	for _, c := range desiredOrder {
		if c == ConditionTypeSnapshotLoadComplete && parent.Spec.LoadURL == "" {
			// Skip condition.
			continue
		}
		conditionOrder = append(conditionOrder, c)
	}
	return conditionOrder
}

// SetConditionStatus sets the ordered condition status from the conditions map.
func (parent *AppDB) SetConditionStatus(conditions AppDBConditions) {
	newConditions := make([]AppDBCondition, 0)
	for _, c := range parent.GetConditionOrder() {
		newConditions = append(newConditions, *conditions[c])
	}
	parent.Status.Conditions = newConditions
}

// AppDBSpec is the top level structure of the spec body
type AppDBSpec struct {
	AppDBInstance string   `json:"appDBInstance,omitempty"`
	DBName        string   `json:"dbName,omitempty"`
	Users         []string `json:"users,omitempty"`
	LoadURL       string   `json:"loadURL,omitempty"`
}

// Verify checks all required fields in the spec.
func (spec *AppDBSpec) Verify() error {
	if spec.AppDBInstance == "" {
		return fmt.Errorf("Missing spec.appDBInstance")
	}

	if spec.DBName == "" {
		return fmt.Errorf("Missing spec.dbName")
	}

	if len(spec.Users) == 0 {
		return fmt.Errorf("spec.users list is empty, must have at least 1 user")
	}

	return nil
}
