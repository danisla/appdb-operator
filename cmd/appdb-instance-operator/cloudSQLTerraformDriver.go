package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	return string(manifest), err
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

func makeCloudSQLProxy(parent *appdbv1.AppDBInstance) (appsv1beta1.Deployment, corev1.Service, error) {
	var deploy appsv1beta1.Deployment
	var svc corev1.Service
	var err error

	// Verify required spec values
	if parent.Spec.Driver.CloudSQLTerraform.Proxy.ServiceAccount.Name == "" {
		return deploy, svc, fmt.Errorf("Missing spec.driver.cloudSQLTerraform.proxy.serviceAccount.name")
	}
	if parent.Spec.Driver.CloudSQLTerraform.Proxy.ServiceAccount.Key == "" {
		return deploy, svc, fmt.Errorf("Missing spec.driver.cloudSQLTerraform.proxy.serviceAccount.key")
	}

	name := fmt.Sprintf("%s-proxy", parent.Name)

	namespace := parent.GetNamespace()

	selector := map[string]string{"app": name}

	replicas := parent.Spec.Driver.CloudSQLTerraform.Proxy.Replicas

	saKeyContainerPath := "/var/run/secrets/cloudsql/sa-key.json"

	cmdStr := fmt.Sprintf("/cloud_sql_proxy -instances=%s=tcp:0.0.0.0:%d -credential_file=%s", parent.Status.CloudSQL.ConnectionName, parent.Status.CloudSQL.Port, saKeyContainerPath)

	deploy = appsv1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1beta1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "cloudsql-proxy",
							Image:           config.CloudSQLProxyImage,
							ImagePullPolicy: config.CLoudSQLProxyImagePullPolicy,
							Command:         strings.Split(cmdStr, " "),
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name:      "sa-key",
									MountPath: "/var/run/secrets/cloudsql",
								},
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									Name:          "sql",
									ContainerPort: parent.Status.CloudSQL.Port,
								},
							},
						},
					}, // Containers
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: "sa-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: parent.Spec.Driver.CloudSQLTerraform.Proxy.ServiceAccount.Name,
									Items: []corev1.KeyToPath{
										corev1.KeyToPath{
											Key:  parent.Spec.Driver.CloudSQLTerraform.Proxy.ServiceAccount.Key,
											Path: "sa-key.json",
										},
									},
								},
							},
						},
					}, // Volumes
				}, // PodSpec
			}, // PodTemplateSpec
		}, // DeploymentSpec
	} // Deployment

	svc = corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name: "sql",
					Port: parent.Status.CloudSQL.Port,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sql",
					},
				},
			},
			Selector: selector,
		},
	}

	return deploy, svc, err
}
