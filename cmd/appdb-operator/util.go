package main

import (
	"bytes"
	"fmt"
	"os/exec"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	yaml "github.com/ghodss/yaml"
)

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
