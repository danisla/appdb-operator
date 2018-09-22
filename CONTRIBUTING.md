## Development

This project uses the following build tools:

- [dep](https://github.com/golang/dep)
- [skaffold](https://github.com/GoogleContainerTools/skaffold)
- [kustomize](https://github.com/kubernetes-sigs/kustomize)

1. Clone the repository into your `GOPATH`:

```
mkdir -p ${GOPATH}/src/github.com/danisla/
cd ${GOPATH}/src/github.com/danisla/
git clone https://github.com/danisla/appdb-operator.git
```

Add your fork as another git remote:

```
FORK_URI=git@github.com:YOUR_GITHUB_USER/appdb-operator.git
git remote add fork ${FORK_URI}
```

2. Modify the skaffold and kustomize image to a docker registry you can push to:

In skaffold.yaml:

```
build:
  artifacts:
  - imageName: YOUR_REGISTRY/appdb-operator
```

> Replace `YOUR_REGISTRY` with something you can push to. 

In `manifests/dev/image.yaml`:

```
spec:
  template:
    spec:
      containers:
      - name: appdb-operator
        image: YOUR_REGISTRY/appdb-operator
```

> Replace `YOUR_REGISTRY` with something you can push to.

3. Install the metacontroller:

```
make install-metacontroller
```

4. Install terraform-operator

If developing for a database instance that uses Terraform, install the [terraform-operator](https://github.com/danisla/terraform-operator).

```
make install-terraform-operator
```

5. Install provider credentials:

```
make -e GOOGLE_CREDENTIALS_SA_KEY=~/.tf-google-sa-key.json credentials
```

6. Install go dependencies:

```
dep ensure
```

7. Run in cluster with skaffold:

```
skaffold dev
```

## Building the release container image

1. Build image using container builder in current project:

```
make image
```

## Submitting a pull request

1. Push changes to a branch in your fork.

```
git checkout -b my-new-feature
git add .
git commit -m "my new feature"
git push fork my-new-feature
```

2. Submit a Github pull request from your branch in your fork to the master branch.
