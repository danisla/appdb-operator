package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	yaml "github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func makeCredentialsSecret(name string, namespace string, users []string, passwords []string) corev1.Secret {
	var secret corev1.Secret

	data := make(map[string]string, 0)

	for i := 0; i < len(users); i++ {
		data[users[i]] = passwords[i]
	}

	secret = corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: data,
	}

	return secret
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
