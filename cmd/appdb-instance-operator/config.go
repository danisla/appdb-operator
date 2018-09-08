package main

import (
	"log"
	"os"

	"cloud.google.com/go/compute/metadata"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config is the configuration structure used by the controller.
type Config struct {
	Project                      string
	ProjectNum                   string
	clientset                    *kubernetes.Clientset
	CloudSQLProxyImage           string
	CLoudSQLProxyImagePullPolicy corev1.PullPolicy
}

func (c *Config) loadAndValidate() error {
	var err error

	if c.Project == "" {
		log.Printf("[INFO] Fetching Project ID from Compute metadata API...")
		c.Project, err = metadata.ProjectID()
		if err != nil {
			return err
		}
	}

	if c.ProjectNum == "" {
		log.Printf("[INFO] Fetching Numeric Project ID from Compute metadata API...")
		c.ProjectNum, err = metadata.NumericProjectID()
		if err != nil {
			return err
		}
	}

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return err
	}
	c.clientset = clientset

	// CLOUD_SQL_PROXY_IMAGE is optional
	if image, ok := os.LookupEnv("CLOUD_SQL_PROXY_IMAGE"); ok == true {
		c.CloudSQLProxyImage = image
	}

	// CLOUD_SQL_PROXY_IMAGE_PULL_POLICY is optional
	if pullPolicy, ok := os.LookupEnv("CLOUD_SQL_PROXY_IMAGE_PULL_POLICY"); ok == true {
		c.CLoudSQLProxyImagePullPolicy = corev1.PullPolicy(pullPolicy)
	}

	return nil
}
