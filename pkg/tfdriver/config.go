package tfdriver

import (
	"fmt"
	"log"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

const (
	DEFAULT_TF_PROVIDER_SECRET = "tf-provider-google"
)

// TerraformDriverConfig is the Terraform driver config
type TerraformDriverConfig struct {
	Image                      string
	ImagePullPolicy            corev1.PullPolicy
	BackendBucket              string
	BackendPrefix              string
	MaxAttempts                int
	GoogleProviderConfigSecret string
}

func (c *TerraformDriverConfig) LoadAndValidate(project string) error {

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
		c.BackendBucket = fmt.Sprintf("%s-appdb-operator", project)
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
