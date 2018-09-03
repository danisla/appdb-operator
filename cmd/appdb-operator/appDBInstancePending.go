package main

import (
	"bytes"
	"fmt"
	"os/exec"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	yaml "gopkg.in/yaml.v2"
)

// Get AppDBInstance and wait for provisioning to complete.
func stateAppDBInstancePending(parentType ParentType, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus) (appdbv1.AppDBOperatorState, error) {
	var appdbi appdbv1.AppDBInstance
	var err error

	appdbi, err = getAppDBInstance(parent.ObjectMeta.Namespace, parent.Spec.AppDBInstance)
	if err != nil {
		// Wait for object creation.
		return StateAppDBInstancePending, nil
	}

	if appdbi.Status.Provisioning != appdbv1.ProvisioningStatusComplete {
		// Wait for provisioning to complete.
		return StateAppDBInstancePending, nil
	}

	return StateWaitComplete, nil
}

func getAppDBInstance(namespace string, name string) (appdbv1.AppDBInstance, error) {
	var appdbi appdbv1.AppDBInstance
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("kubectl", "get", "appdbinstance", "-n", namespace, name, "-o", "yaml")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return appdbi, fmt.Errorf("Failed to run kubectl: %s\n%v", stderr.String(), err)
	}

	err = yaml.Unmarshal(stdout.Bytes(), &appdbi)

	return appdbi, err
}
