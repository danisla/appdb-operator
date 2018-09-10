package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
			Namespace: parent.GetNamespace(),
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

func makeTFVars(instance string, dbname string, users []string) (map[string]string, error) {
	var tfvars = make(map[string]string, 0)

	tfvars["instance"] = instance

	tfvars["dbname"] = dbname

	tfvars["users"] = strings.Join(users, ",")

	return tfvars, nil
}

func makeLoadSnapshotJob(jobName, namespace, instanceName, snapshotURI, dbname, user, proxySecretName string) batchv1.Job {
	var job batchv1.Job

	var parallelism int32 = 1
	var completions int32 = 1
	var deadlineSeconds int64 = 1200 // 20 minutes max to load data.
	var numRetries int32 = 4

	podSpec := makeLoadJobPodSpec(instanceName, snapshotURI, dbname, user, proxySecretName)

	job = batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Completions:           &completions,
			ActiveDeadlineSeconds: &deadlineSeconds,
			BackoffLimit:          &numRetries,
			Parallelism:           &parallelism,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: jobName,
				},
				Spec: podSpec,
			},
		},
	}
	return job
}

func makeLoadJobPodSpec(instanceName, snapshotURI, dbname, user, proxySecretName string) corev1.PodSpec {
	var spec corev1.PodSpec

	cmdStr := fmt.Sprintf("gcloud sql import %s %s --database=%s --user=%s", instanceName, snapshotURI, dbname, user)

	loadJobScript := fmt.Sprintf(`
gcloud auth activate-service-account --key-file=$GOOGLE_CREDENTIALS
gcloud config set project $GOOGLE_PROJECT
%s
`, cmdStr)

	spec = corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			corev1.Container{
				Name:  "sql-load",
				Image: "google/cloud-sdk:alpine",
				Command: []string{
					"bash",
					"-exc",
					loadJobScript,
				},
				VolumeMounts: []corev1.VolumeMount{
					corev1.VolumeMount{
						Name:      "sa-key",
						MountPath: "/var/run/secrets/cloudsql",
					},
				},
				Env: []corev1.EnvVar{
					corev1.EnvVar{
						Name:  "GOOGLE_PROJECT",
						Value: config.Project,
					},
					corev1.EnvVar{
						Name:  "GOOGLE_CREDENTIALS",
						Value: "/var/run/secrets/cloudsql/sa-key.json",
					},
				},
			}, //Container
		}, // Containers
		Volumes: []corev1.Volume{
			corev1.Volume{
				Name: "sa-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: proxySecretName,
					},
				},
			},
		}, // Volumes
	} // PodSpec

	return spec
}
