# AppDB Operator

This is not an official Google product.

## Intro

Implementation of a [CompositeController metacontroller](https://github.com/GoogleCloudPlatform/metacontroller) to operate the lifecycle of a stateful database.

This controller utilizes the following major components:
- [Custom Resource Definitions (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/): Used to represent the new custom resources.
- [metacontroller](https://github.com/GoogleCloudPlatform/metacontroller): Implements the CompositeController interface for the Custom Resource Definition.

## Prerequisites

1. Create GKE cluster:

```
ZONE=us-central1-b
CLUSTER_VERSION=$(gcloud beta container get-server-config --zone ${ZONE} --format='value(validMasterVersions[0])')

gcloud container clusters create dev \
  --cluster-version ${CLUSTER_VERSION} \
  --machine-type n1-standard-4 \
  --num-nodes 3 \
  --scopes=cloud-platform \
  --zone ${ZONE}
```

## Install metacontroller

1. Install metacontroller:

```
kubectl create clusterrolebinding ${USER}-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud config get-value account)

kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/metacontroller/master/manifests/metacontroller-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/metacontroller/master/manifests/metacontroller.yaml
```

## Install terraform-operator

1. Install terraform-operator:

```
kubectl apply -f https://raw.githubusercontent.com/danisla/terraform-operator/master/manifests/terraform-operator-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/danisla/terraform-operator/master/manifests/terraform-operator.yaml
```

## Install the operator

1. Install the `appdb-operator`:

```
kubectl apply -f https://raw.githubusercontent.com/danisla/appdb-operator/master/manifests/appdb-operator-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/danisla/appdb-operator/master/manifests/appdb-operator.yaml
```

2. Create GCS bucket for remote state and snapshots:

```
gsutil mb gs://$(gcloud config get-value project)-appdb-operator
```