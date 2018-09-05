package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DEFAULT_CLOUD_SQL_SOURCE_PATH = "/config/dbinstance/main.tf"
	DEFAULT_CLOUD_SQL_DISK_TYPE   = "PD_SSD"
)

func makeCloudSQLTerraform(tfApplyName string, parent *appdbv1.AppDBInstance) (tfv1.Terraform, error) {
	var tfapply tfv1.Terraform

	manifest, err := getCloudSQLTerraformManifest(DEFAULT_CLOUD_SQL_SOURCE_PATH)
	if err != nil {
		return tfapply, fmt.Errorf("Error loading cloud sql terraform manifest from %s: %v", DEFAULT_CLOUD_SQL_SOURCE_PATH, err)
	}

	tfvars, err := makeTFVars(tfApplyName, parent.Spec.Driver.CloudSQLTerraform)
	if err != nil {
		return tfapply, fmt.Errorf("Failed to generate tfvars from driver config: %v", err)
	}

	parentSig := calcParentSig(parent.Spec, "")

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

func makeTFVars(name string, cfg *appdbv1.AppDBCloudSQLTerraformDriver) (map[string]string, error) {
	var tfvars = make(map[string]string, 0)

	// Names must be unique and cannot be reused across destroys.
	// the Terraform source will create a new name using this as a prefix.
	tfvars["name"] = name

	// Marshal params to json and unmarshal as tfvars
	data, err := json.Marshal(cfg.Params)
	if err != nil {
		return tfvars, err
	}

	var paramsJSON map[string]string
	err = json.Unmarshal(data, &paramsJSON)
	if err != nil {
		return tfvars, err
	}

	for k, v := range paramsJSON {
		tfvars[k] = v
	}

	return tfvars, nil
}
