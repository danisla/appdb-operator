package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func makeStatus(parent *appdbv1.AppDBInstance, children *AppDBInstanceChildren) *appdbv1.AppDBInstanceOperatorStatus {
	status := appdbv1.AppDBInstanceOperatorStatus{
		StateCurrent: StateNone,
	}

	changed := false
	sig := calcParentSig(parent.Spec, "")

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

	if parent.Status.CloudSQL != nil && changed == false {
		status.CloudSQL = parent.Status.CloudSQL
	}

	if parent.Status.Provisioning != "" && changed == false {
		status.Provisioning = parent.Status.Provisioning
	}

	return &status
}

func calcParentSig(spec interface{}, addStr string) string {
	hasher := sha1.New()
	data, err := json.Marshal(&spec)
	if err != nil {
		log.Printf("[ERROR] Failed to convert parent spec to JSON, this is a bug.\n")
		return ""
	}
	hasher.Write([]byte(data))
	hasher.Write([]byte(addStr))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func changeDetected(parent *appdbv1.AppDBInstance, children *AppDBInstanceChildren, status *appdbv1.AppDBInstanceOperatorStatus) bool {
	changed := false

	if status.StateCurrent == StateIdle {

		// Changed if parent spec changes
		if status.LastAppliedSig != "" && status.LastAppliedSig != calcParentSig(parent.Spec, "") {
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
