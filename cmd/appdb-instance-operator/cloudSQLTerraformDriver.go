package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DEFAULT_CLOUD_SQL_SOURCE_PATH = "/config/cloudsql.tf"
	DEFAULT_TF_PROVIDER_SECRET    = "tf-provider-google"
	DEFAULT_CLOUD_SQL_DISK_TYPE   = "PD_SSD"
)

type CloudSQLDriverConfig struct {
	Image                      string
	ImagePullPolicy            corev1.PullPolicy
	BackendBucket              string
	BackendPrefix              string
	MaxAttempts                int
	GoogleProviderConfigSecret string
}

func (c *CloudSQLDriverConfig) loadAndValidate() error {

	// TF_IMAGE is optional
	c.Image, _ = os.LookupEnv("TF_IMAGE")

	// TF_IMAGE_PULL_POLICY is optional
	if pullPolicy, ok := os.LookupEnv("TF_IMAGE_PULL_POLICY"); ok == true {
		c.ImagePullPolicy = corev1.PullPolicy(pullPolicy)
	} else {
		c.ImagePullPolicy = corev1.PullIfNotPresent
	}

	// TF_BACKEND_BUCKET is required
	if backendBucket, ok := os.LookupEnv("TF_BACKEND_BUCKET"); ok == true {
		c.BackendBucket = backendBucket
	} else {
		// Create bucket name from project name.
		c.BackendBucket = fmt.Sprintf("%s-appdb-operator", config.Project)
		log.Printf("[INFO] No TF_BACKEND_BUCKET given, using canonical bucket name: %s", c.BackendBucket)
	}

	// TF_BACKEND_PREFIX is required
	if backendPrefix, ok := os.LookupEnv("TF_BACKEND_PREFIX"); ok == true {
		c.BackendPrefix = backendPrefix
	} else {
		// Use default prefix.
		c.BackendPrefix = "terraform"
		log.Printf("[INFO] No TF_BACKEND_PREFIX given, using default prefix of: %s", c.BackendPrefix)
	}

	if maxAttempts, ok := os.LookupEnv("TF_MAX_ATTEMPTS"); ok == true {
		i, err := strconv.Atoi(maxAttempts)
		if err != nil {
			return fmt.Errorf("Invalid number for TF_MAX_ATTEMPTS: %s, must be positive integer", maxAttempts)
		}
		if i <= 0 {
			return fmt.Errorf("Invalid number for TF_MAX_ATTEMPTS: %s, must be positive integer", maxAttempts)
		}

		c.MaxAttempts = i
	} else {
		c.MaxAttempts = 4
		log.Printf("[INFO] No TF_MAX_ATTEMPTS given, using default count of: %d", c.MaxAttempts)
	}

	if googleConfigSecret, ok := os.LookupEnv("TF_GOOGLE_PROVIDER_SECRET"); ok == true {
		c.GoogleProviderConfigSecret = googleConfigSecret
	} else {
		c.GoogleProviderConfigSecret = DEFAULT_TF_PROVIDER_SECRET
		log.Printf("[INFO] No TF_GOOGLE_PROVIDER_SECRET given, using default: %s", c.GoogleProviderConfigSecret)
	}

	return nil
}

func makeCloudSQLTerraform(name, namespace string, driver *appdbv1.AppDBCloudSQLTerraformDriver) (tfv1.Terraform, error) {
	var tfapply tfv1.Terraform

	manifest, err := getCloudSQLTerraformManifest(DEFAULT_CLOUD_SQL_SOURCE_PATH)
	if err != nil {
		return tfapply, fmt.Errorf("Error loading cloud sql terraform manifest from %s: %v", DEFAULT_CLOUD_SQL_SOURCE_PATH, err)
	}

	tfvars, err := makeTFVars(name, driver)
	if err != nil {
		return tfapply, fmt.Errorf("Failed to generate tfvars from driver config: %v", err)
	}

	tfapply = tfv1.Terraform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "ctl.isla.solutions/v1",
			Kind:       "TerraformApply",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: tfv1.TerraformSpec{
			Image:           cloudSQLDriverconfig.Image,
			ImagePullPolicy: cloudSQLDriverconfig.ImagePullPolicy,
			BackendBucket:   cloudSQLDriverconfig.BackendBucket,
			BackendPrefix:   cloudSQLDriverconfig.BackendPrefix,
			ProviderConfig: map[string]tfv1.TerraformSpecProviderConfig{
				"google": tfv1.TerraformSpecProviderConfig{
					SecretName: cloudSQLDriverconfig.GoogleProviderConfigSecret,
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
	var manifest string
	var err error

	data, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return manifest, err
	}
	manifest = string(data)
	return manifest, err
}

func makeTFVars(name string, cfg *appdbv1.AppDBCloudSQLTerraformDriver) (map[string]string, error) {
	var tfvars = make(map[string]string, 0)
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
