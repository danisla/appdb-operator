TEST_CLOUDSQL_ARTIFACTS := db1-cloudsql-appdbinstance.yaml
TEST_APPDB_ARTIFACTS := db1-appdb.yaml

TEST_ARTIFACTS := $(TEST_CLOUDSQL_ARTIFACTS) $(TEST_APPDB_ARTIFACTS)

GOOGLE_CREDENTIALS_SA_KEY := $(HOME)/.tf-google-sa-key.json
GOOGLE_PROVIDER_SECRET_NAME := tf-provider-google

project:
	$(eval PROJECT := $(shell gcloud config get-value project))

backend_bucket: project
	$(eval BACKEND_BUCKET := $(PROJECT)-terraform-operator)

define TEST_CLOUDSQL
apiVersion: ctl.isla.solutions/v1
kind: AppDBInstance
metadata:
  name: {{NAME}}
spec:
  driver:
    cloudSQL:
      region: us-central1
      database_version: MYSQL_5_6
      tier: db-f1-micro

      proxy:
        image: gcr.io/cloudsql-docker/gce-proxy:1.11
        replicas: 3
endef

define TEST_APPDB
apiVersion: ctl.isla.solutions/v1
kind: AppDB
metadata:
  name: {{NAME}}
spec:
  appDBInstance: {{APPDB_INSTANCE}}
  users:
    rw:
    - {{RW_USER}}
    ro:
    - {{RO_USER}}
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

export TEST_APPDB
tests/db%-appdb.yaml:
	@mkdir -p tests
	@echo "$${TEST_APPDB}" | \
	sed -e "s/{{NAME}}/db$*/g" \
	    -e "s/{{APPDB_INSTANCE}}/job$*-cloudsql/g" \
	    -e "s/{{RW_USER}}/db$*-writer/g" \
	    -e "s/{{RO_USER}}/db$*-reader/g" \
	> $@

### END Tests with CloudSQL instance ###


test-artifacts: $(addprefix tests/,$(TEST_ARTIFACTS))

test: $(addprefix tests/,$(TEST_ARTIFACTS))
	-@for f in $^; do kubectl apply -f $$f; done

test-stop: $(addprefix tests/,$(TEST_ARTIFACTS))
	-@for f in $^; do kubectl delete -f $$f; done

test-clean: $(addprefix tests/,$(TEST_ARTIFACTS))
	rm -f $^