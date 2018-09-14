package types

import (
	"fmt"
	"log"

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
	// ConditionTypeAppDBInstanceReady is True when the AppDBInstance resource is available and ready.
	ConditionTypeAppDBInstanceReady AppDBConditionType = "AppDBInstanceReady"
	// ConditionTypeDBCreateComplete is True when the DB create driver action is complete.
	ConditionTypeDBCreateComplete AppDBConditionType = "DBCreateComplete"
	// ConditionTypeSnapshotLoadComplete is True when the Job for loading SQL data has been created and is complete.
	ConditionTypeSnapshotLoadComplete AppDBConditionType = "SnapshotLoadComplete"
	// ConditionTypeCredentialsSecretCreated is True when the secret containing the database credentials and info has been created.
	ConditionTypeCredentialsSecretCreated AppDBConditionType = "CredentialsSecretCreated"
	// ConditionTypeAppDBReady means that all prior conditions are Ready
	ConditionTypeAppDBReady AppDBConditionType = "Ready"
)

type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

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

// AppDBSpec is the top level structure of the spec body
type AppDBSpec struct {
	AppDBInstance string   `json:"appDBInstance,omitempty"`
	DBName        string   `json:"dbName,omitempty"`
	Users         []string `json:"users,omitempty"`
	LoadURL       string   `json:"loadURL,omitempty"`
}
