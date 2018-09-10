# Basic App DB Operator Example

[![button](http://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/danisla/appdb-operator&working_dir=examples/basic&page=shell&tutorial=README.md)

This example demonstrates how to provision a database instance and user database using the App DB Operator.

## Set up the environment

1. Set the project, replace `YOUR_PROJECT` with your project ID:

```
PROJECT=YOUR_PROJECT
```

```
gcloud config set project ${PROJECT}
```

2. Create GKE cluster:

```
ZONE=us-central1-b
CLUSTER_VERSION=$(gcloud beta container get-server-config --zone ${ZONE} --format='value(validMasterVersions[0])')

gcloud container clusters create appdb-tutorial \
  --cluster-version ${CLUSTER_VERSION} \
  --machine-type n1-standard-4 \
  --num-nodes 3 \
  --scopes=cloud-platform \
  --zone ${ZONE}
```

## Install and configure the terraform-operator

1. Clone the repo into the `~/.kube/plugin` directory:

```
(
    mkdir -p ${HOME}/.kube/plugin
    cd ${HOME}/.kube/plugin
    git clone https://github.com/danisla/terraform-operator.git
)
```

2. Configure the plugin with your service account key and project info:

```
kubectl plugin terraform configure
```

3. Follow the prompts to configure the plugin.

## Install and configure the appdb-operator

1. Install the `appdb-operator`:

```
kubectl apply -f https://raw.githubusercontent.com/danisla/appdb-operator/master/manifests/appdb-operator-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/danisla/appdb-operator/master/manifests/appdb-operator.yaml
```

2. Create GCS bucket for remote state and snapshots:

```
gsutil mb gs://$(gcloud config get-value project)-appdb-operator
```

## Change to the example directory

```
[[ `basename $PWD` != plugin ]] && cd examples/basic
```

## Inspect the YAML spec files

1. Inspect the `example-appdbinstance.yaml` file:

```
cat example-appdbinstance.yaml
```

> Note that the `params` key value pairs are all string values. These are passed through to the Terraform manifest that creates the database and service account for the Cloud SQL Proxy.

2. Inspect the `example-appdb-sbtest.yaml` file:

```
cat example-appdb-sbtest.yaml
```

> Note that the name of the object and the `spec.dbName` can be different. This is beacuse the object name must be a DNS-1123 name and cannot container underscores or other non conforming characters.

## Apply the spec YAML files

1. Create the `AppDBInstance` and `AppDB` resources by applying the yaml spec files:

```
kubectl apply -f example-appdbinstance.yaml && \
  kubectl apply -f example-appdb-sbtest.yaml
```

3. Wait for the database instance and database provisioning to complete:

```
(until [[ $(kubectl get appdbi example -o jsonpath='{.status.provisioning}') == "COMPLETE"  ]]; do echo "Waiting for AppDBInstance..."; sleep 2; done && echo "AppDBInstance is ready") && \
  (until [[ $(kubectl get appdb sbtest -o jsonpath='{.status.provisioning}') == "COMPLETE"  ]]; do echo "Waiting for AppDB..."; sleep 2; done && echo "AppDB is ready")
```

> Note, this step will take 4-7 minutes to complete. To monitor the terraform output, tail the logs from the `appdbi-example-tfapply-0` and `appdb-example-sbtest-tfapply-0` pods.

4. Inspect the status of the `AppDBInstance` resource:

```
kubectl describe appdbi example
```

5. Inspect the status of the `AppDB` resource:

```
kubectl describe appdb sbtest
```

## Test the database connection

1. Inspect `sysbench-prepare-job.yaml` file:

```
cat sysbench-prepare-job.yaml
```

> Notice how the database host, port, and password are passed from the secret created by the appdb-operator.

2. Run the `sysbench-prepare` job:

```
kubectl apply -f sysbench-prepare-job.yaml
```

3. Wait for `sysbench-prepare` job to complete:

```
( until [[ $(kubectl get job sysbench-prepare -o jsonpath='{.status.succeeded}') -eq 1 ]]; do echo "Waiting for sysbench-prepare job..."; sleep 2; done && echo "sysbench-prepare job complete")
```

4. Inspect the logs from the job:

```
kubectl logs job/sysbench-prepare
```

## Cleanup

1. Delete the sysbench-prepare job:

```
kubectl delete job sysbench-prepare
```

2. Delete the App Database and user using the `example-appdb-tfdestroy.yaml` file:

```
kubectl apply -f example-appdb-tfdestroy.yaml && \
  ( until [[ $(kubectl get tfdestroy appdb-example-sbtest -o jsonpath='{.status.podStatus}') == "COMPLETED" ]]; do echo "Waiting for TerraformDestroy to complete..."; sleep 2; done && echo "TerraformDestroy complete" ) && \
  kubectl delete tfdestroy appdb-example-sbtest
```

3. Delete the `AppDB` resource:

```
kubectl delete appdb sbtest
```

4. Delete the App Datbase Instance using the `example-appdbinstance-tfdestroy.yaml` file:

```
kubectl apply -f example-appdbinstance-tfdestroy.yaml && \
  ( until [[ $(kubectl get tfdestroy appdbi-example -o jsonpath='{.status.podStatus}') == "COMPLETED" ]]; do echo "Waiting for TerraformDestroy to complete..."; sleep 2; done && echo "TerraformDestroy complete" ) && \
  kubectl delete tfdestroy appdbi-example
```

> Note that this step can take 2-5 minutes to complete as the Cloud SQL database is being destroyed. 

5. Delete the `AppDBInstance` resource:

```
kubectl delete appdbi example
```

6. Delete the GKE cluster:

```
ZONE=us-central1-b
gcloud container clusters delete appdb-tutorial --zone=${ZONE}
```