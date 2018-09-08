TEST_CLOUDSQL_ARTIFACTS := db1-cloudsql-appdbinstance.yaml db1-cloudsql-tfdestroy.yaml
TEST_APPDB_ARTIFACTS := db1-app1-appdb.yaml db1-app1-appdb-tfdestroy.yaml

TEST_ARTIFACTS := $(TEST_CLOUDSQL_ARTIFACTS) $(TEST_APPDB_ARTIFACTS)

GOOGLE_CREDENTIALS_SA_KEY := $(HOME)/.tf-google-sa-key.json
GOOGLE_PROVIDER_SECRET_NAME := tf-provider-google

project:
	$(eval PROJECT := $(shell gcloud config get-value project))

backend_bucket: project
	$(eval BACKEND_BUCKET := $(PROJECT)-appdb-operator)

define TEST_CLOUDSQL
apiVersion: ctl.isla.solutions/v1
kind: AppDBInstance
metadata:
  name: {{NAME}}
spec:
  driver:
    cloudSQLTerraform:
      params:
        region: "us-central1"
        database_version: "MYSQL_5_6"
        tier: "db-f1-micro"
        disk_size_gb: "10"
        disk_type: "PD_SSD"
      proxy:
        image: gcr.io/cloudsql-docker/gce-proxy:1.11
        replicas: 1
        serviceAccount:
          name: $(GOOGLE_PROVIDER_SECRET_NAME)
          key: GOOGLE_CREDENTIALS
endef

define TEST_CLOUDSQL_DESTROY
apiVersion: ctl.isla.solutions/v1
kind: TerraformDestroy
metadata:
  name: {{NAME}}
spec:
  backendBucket: {{BACKEND_BUCKET}}
  backendPrefix: {{BACKEND_PREFIX}}
  providerConfig:
    google:
      secretName: {{GOOGLE_PROVIDER_SECRET_NAME}}
  sources:
  - tfapply: {{SRC_TFAPPLY}}
    tfplan: {{SRC_TFAPPLY}}
  tfvarsFrom:
  - tfplan: {{SRC_TFAPPLY}}
    tfapply: {{SRC_TFAPPLY}}
endef

define TEST_APPDB
apiVersion: ctl.isla.solutions/v1
kind: AppDB
metadata:
  name: {{NAME}}
spec:
  appDBInstance: {{APPDB_INSTANCE}}
  dbName: {{NAME}}
  users:
  - {{DB_USER}}
endef

credentials: $(GOOGLE_CREDENTIALS_SA_KEY) project
	kubectl create secret generic $(GOOGLE_PROVIDER_SECRET_NAME) --from-literal=GOOGLE_PROJECT=$(PROJECT) --from-file=GOOGLE_CREDENTIALS=$(GOOGLE_CREDENTIALS_SA_KEY)

### BEGIN Tests with CloudSQL instance ###
export TEST_CLOUDSQL
tests/db%-cloudsql-appdbinstance.yaml:
	@mkdir -p tests
	@echo "$${TEST_CLOUDSQL}" | \
	sed -e "s/{{NAME}}/db$*-cloudsql/g" \
	> $@

export TEST_CLOUDSQL_DESTROY
tests/db%-cloudsql-tfdestroy.yaml: backend_bucket
	@mkdir -p tests
	@echo "$${TEST_CLOUDSQL_DESTROY}" | \
	sed -e "s/{{NAME}}/db$*-cloudsql/g" \
	    -e "s/{{SRC_TFAPPLY}}/db$*-cloudsql/g" \
      -e "s/{{BACKEND_BUCKET}}/$(BACKEND_BUCKET)/g" \
	    -e "s/{{BACKEND_PREFIX}}/terraform/g" \
	    -e "s/{{GOOGLE_PROVIDER_SECRET_NAME}}/$(GOOGLE_PROVIDER_SECRET_NAME)/g" \
	>$@

export TEST_CLOUDSQL_DESTROY
tests/db1-app%-appdb-tfdestroy.yaml: backend_bucket
	@mkdir -p tests
	@echo "$${TEST_CLOUDSQL_DESTROY}" | \
	sed -e "s/{{NAME}}/db1-cloudsql-app$*/g" \
	    -e "s/{{SRC_TFAPPLY}}/db1-cloudsql-app$*/g" \
      -e "s/{{BACKEND_BUCKET}}/$(BACKEND_BUCKET)/g" \
	    -e "s/{{BACKEND_PREFIX}}/terraform/g" \
	    -e "s/{{GOOGLE_PROVIDER_SECRET_NAME}}/$(GOOGLE_PROVIDER_SECRET_NAME)/g" \
	>$@

export TEST_APPDB
tests/db1-app%-appdb.yaml:
	@mkdir -p tests
	@echo "$${TEST_APPDB}" | \
	sed -e "s/{{NAME}}/app$*/g" \
	    -e "s/{{APPDB_INSTANCE}}/db1-cloudsql/g" \
	    -e "s/{{DB_USER}}/app$*-dbuser/g" \
	> $@

### END Tests with CloudSQL instance ###


test-artifacts: $(addprefix tests/,$(TEST_ARTIFACTS))

test: $(addprefix tests/,$(TEST_ARTIFACTS))
	-@for f in $^; do kubectl apply -f $$f; done

test-stop: $(addprefix tests/,$(TEST_ARTIFACTS))
	-@for f in $^; do kubectl delete -f $$f; done

test-clean: $(addprefix tests/,$(TEST_ARTIFACTS))
	rm -f $^