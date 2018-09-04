package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func makeStatus(parent *appdbv1.AppDB, children *AppDBChildren) *appdbv1.AppDBOperatorStatus {
	status := appdbv1.AppDBOperatorStatus{
		StateCurrent: StateNone,
		CloudSQLDB:   &appdbv1.AppDBCloudSQLDBStatus{},
	}

	changed := false
	sig := calcParentSig(parent, "")

	if parent.Status.LastAppliedSig != "" {
		if parent.Status.StateCurrent == StateIdle && sig != parent.Status.LastAppliedSig {
			changed = true
			status.LastAppliedSig = ""
		} else {
			status.LastAppliedSig = parent.Status.LastAppliedSig
		}
	}

	if parent.Status.StateCurrent != "" && changed == false {
		status.StateCurrent = parent.Status.StateCurrent
	}

	if parent.Status.CloudSQLDB != nil && changed == false {
		status.CloudSQLDB = parent.Status.CloudSQLDB
	}

	if parent.Status.Provisioning != "" && changed == false {
		status.Provisioning = parent.Status.Provisioning
	}

	if parent.Status.CredentialsSecret != "" && changed == false {
		status.CredentialsSecret = parent.Status.CredentialsSecret
	}

	return &status
}

func calcParentSig(parent *appdbv1.AppDB, addStr string) string {
	hasher := sha1.New()
	data, err := json.Marshal(&parent.Spec)
	if err != nil {
		myLog(parent, "ERROR", "Failed to convert parent spec to JSON, this is a bug.")
		return ""
	}
	hasher.Write([]byte(data))
	hasher.Write([]byte(addStr))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func changeDetected(parent *appdbv1.AppDB, children *AppDBChildren, status *appdbv1.AppDBOperatorStatus) bool {
	changed := false

	if status.StateCurrent == StateIdle {

		// Changed if parent spec changes
		if status.LastAppliedSig != "" && status.LastAppliedSig != calcParentSig(parent, "") {
			myLog(parent, "DEBUG", "Changed because parent sig different")
			changed = true
		}
	}

	return changed
}

func toSha1(data interface{}) (string, error) {
	h := sha1.New()
	var b []byte
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}
