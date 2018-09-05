package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DEFAULT_CLOUD_SQL_DB_SOURCE_PATH = "/config/db/main.tf"
)

func makeCloudSQLDBTerraform(tfApplyName string, parent *appdbv1.AppDB, appdbi appdbv1.AppDBInstance) (tfv1.Terraform, error) {
	var tfapply tfv1.Terraform

	manifest, err := getCloudSQLTerraformManifest(DEFAULT_CLOUD_SQL_DB_SOURCE_PATH)
	if err != nil {
		return tfapply, fmt.Errorf("Error loading cloud sql DB terraform manifest from %s: %v", DEFAULT_CLOUD_SQL_DB_SOURCE_PATH, err)
	}

	tfvars, err := makeTFVars(appdbi.Status.CloudSQL.InstanceName, parent.Spec.DBName, parent.Spec.Users)
	if err != nil {
		return tfapply, fmt.Errorf("Failed to generate tfvars from driver config: %v", err)
	}

	parentSig := calcParentSig(parent.Spec, "")

	// Create new object.
	tfapply = tfv1.Terraform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "ctl.isla.solutions/v1",
			Kind:       "TerraformApply",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tfApplyName,
			Namespace: parent.Namespace,
			Annotations: map[string]string{
				"appdb-parent-sig": parentSig,
			},
		},
		Spec: tfv1.TerraformSpec{
			Image:           tfDriverConfig.Image,
			ImagePullPolicy: tfDriverConfig.ImagePullPolicy,
			BackendBucket:   tfDriverConfig.BackendBucket,
			BackendPrefix:   tfDriverConfig.BackendPrefix,
			ProviderConfig: map[string]tfv1.TerraformSpecProviderConfig{
				"google": tfv1.TerraformSpecProviderConfig{
					SecretName: tfDriverConfig.GoogleProviderConfigSecret,
				},
			},
			Sources: []tfv1.TerraformConfigSource{
				tfv1.TerraformConfigSource{
					Embedded: manifest,
				},
			},
			TFVars: tfvars,
		},
	}

	return tfapply, nil
}

func getCloudSQLTerraformManifest(srcPath string) (string, error) {
	var manifest []byte
	var err error

	manifest, err = ioutil.ReadFile(srcPath)
	if err != nil {
		return string(manifest), err
	}

	return base64.StdEncoding.EncodeToString(manifest), err
}

func makeTFVars(instance string, dbname string, users []string) (map[string]string, error) {
	var tfvars = make(map[string]string, 0)

	tfvars["instance"] = instance

	tfvars["dbname"] = dbname

	tfvars["users"] = strings.Join(users, ",")

	return tfvars, nil
}
