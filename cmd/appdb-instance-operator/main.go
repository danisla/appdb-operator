package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	tfdriverv1 "github.com/danisla/appdb-operator/pkg/tfdriver"
	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

var (
	config         Config
	tfDriverConfig tfdriverv1.TerraformDriverConfig
)

func init() {
	config = Config{
		Project:                      "",                                      // Derived from instance metadata server
		ProjectNum:                   "",                                      // Derived from instance metadata server
		CloudSQLProxyImage:           "gcr.io/cloudsql-docker/gce-proxy:1.11", // Override with env var: CLOUD_SQL_PROXY_IMAGE
		CLoudSQLProxyImagePullPolicy: corev1.PullIfNotPresent,                 // Override with env var: CLOUD_SQL_PROXY_IMAGE_PULL_POLICY
	}

	if err := config.loadAndValidate(); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	tfDriverConfig = tfdriverv1.TerraformDriverConfig{}

	if err := tfDriverConfig.LoadAndValidate(config.Project); err != nil {
		log.Fatalf("Failed to load terraform driver config: %v", err)
	}
}

func main() {
	http.HandleFunc("/healthz", healthzHandler())
	http.HandleFunc("/", webhookHandler())

	log.Printf("[INFO] Initialized controller on port 80\n")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthzHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK\n")
	}
}

func webhookHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req SyncRequest
		var desiredStatus *appdbv1.AppDBInstanceOperatorStatus
		var desiredChildren *[]interface{}
		var parentType ParentType

		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Unsupported method\n")
			return
		}

		if os.Getenv("HTTP_DEBUG") != "" {
			log.Printf("---HTTP REQUEST %s %s ---", r.Method, r.URL.String())
			reqDump, _ := httputil.DumpRequest(r, true)
			log.Println(string(reqDump))
		}

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("[ERROR] Failed to read request body: %v", err)
			return
		}

		err = json.Unmarshal(reqBody, &req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("[ERROR] Could not parse SyncRequest: %v", err)
			return
		}

		switch req.Parent.Kind {
		case "AppDBInstance":
			parentType = ParentDBInstance
		}
		desiredStatus, desiredChildren, err = sync(parentType, &req.Parent, &req.Children)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("[ERROR] Could not sync state: %v", err)
		}

		resp := SyncResponse{
			Status:   *desiredStatus,
			Children: *desiredChildren,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("[ERROR] Could not generate SyncResponse: %v", err)
			return
		}
		w.Write(data)

		if os.Getenv("HTTP_DEBUG") != "" {
			log.Printf("---JSON RESPONSE %s %s ---", r.Method, r.URL.String())
			log.Println(string(data))
		}
	}
}
