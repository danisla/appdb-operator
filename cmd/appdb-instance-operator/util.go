package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func myLog(parent *appdbv1.AppDBInstance, level, msg string) {
	log.Printf("[%s][%s][%s] %s", level, parent.Kind, parent.Name, msg)
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

func kubectlApply(namespace string, name string, spec interface{}) error {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	stdin := bytes.NewReader(specJSON)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("kubectl", "-n", namespace, "apply", "-f", "-")
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run kubectl: %s\n%v", stderr.String(), err)
	}

	return err
}
