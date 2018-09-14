# Load Snapshot from GCS App DB Operator Example

[![button](http://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/danisla/appdb-operator&working_dir=examples/load-snapshot&page=shell&tutorial=README.md)

This example demonstrates how to provision a database instance and user database with initial data from GCS using the App DB Operator.

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

## Copy SQL dataset to GCS bucket

1. Copy public dataset to GCS bucket:

```
(
  cd $(mktemp -d)
  curl -LO http://downloads.mysql.com/docs/world.sql.gz
  BUCKET=$(gcloud config get-value project)-appdb-operator
  gsutil cp world.sql.gz gs://${BUCKET}/snapshots/world.sql.gz
)
```

## Change to the example directory

```
[[ `basename $PWD` != load-snapshot ]] && cd examples/load-snapshot
```

## Inspect the YAML spec files

1. Inspect the `example-appdb-world.yaml` file:

```
cat example-appdb-world.yaml
```

> Note that the value `loadURL` is not prefixed with a `gs://`. In this case, the URL is relative to the backend bucked configured by the driver.

## Apply the spec YAML files

1. Create the `AppDBInstance` and `AppDB` resources by applying the yaml spec files:

```
kubectl apply -f example-appdbinstance.yaml && \
  kubectl apply -f example-appdb-world.yaml
```

2. Wait for the database instance and database provisioning to complete:

```
until kubectl wait pod/appdbi-example-tfapply-0 --for=condition=Initialized && \
  kubectl logs -f appdbi-example-tfapply-0 && \
  kubectl wait pod/appdb-example-world-tfapply-0 --for=condition=Initialized && \
  kubectl logs -f appdb-example-world-tfapply-0; do sleep 2; done
```

> Note, this step will take 4-7 minutes to complete.

## Verify the snapshot load job completed

1. Inspect the SQL load job output:

```
kubectl logs -f job/appdb-example-world-load
```

## Cleanup

1. Delete the App Database and user using the `example-appdb-tfdestroy.yaml` file:

```
until kubectl apply -f example-appdb-tfdestroy.yaml && \
  kubectl wait pod/appdb-example-world-tfdestroy-0 --for=condition=Initialized && \
  kubectl logs -f appdb-example-world-tfdestroy-0 && \
  kubectl delete tfdestroy appdb-example-world && \
  kubectl delete appdb world; do sleep 2; done
```

2. Delete the App Database Instance using the `example-appdbinstance-tfdestroy.yaml` file:

```
until kubectl apply -f example-appdbinstance-tfdestroy.yaml && \
  kubectl wait pod/appdbi-example-tfdestroy-0 --for=condition=Initialized && \
  kubectl logs -f appdbi-example-tfdestroy-0 && \
  kubectl delete tfdestroy appdbi-example && \
  kubectl delete appdbi example; do sleep 2; done
```

> Note that this step can take 2-5 minutes to complete as the Cloud SQL database is being destroyed. 

3. Delete the GKE cluster:

```
ZONE=us-central1-b
gcloud container clusters delete appdb-tutorial --zone=${ZONE}
```