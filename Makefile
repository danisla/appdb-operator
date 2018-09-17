TAG = latest

all: image

image:
	gcloud builds submit --config cloudbuild.yaml --project cloud-solutions-group --substitutions=TAG_NAME=$(TAG) --machine-type=n1-highcpu-32

docker-clean:
	@docker ps --filter status=exited -q | xargs -I {} docker rm {} 2>/dev/null
	@docker ps --filter status=created -q | xargs -I {} docker rm {} 2>/dev/null
	@docker images --filter dangling=true -q | xargs -I {} docker rmi {} 2>/dev/null

install-metacontroller:
	-kubectl create clusterrolebinding $(USER)-cluster-admin-binding --clusterrole=cluster-admin --user=$(shell gcloud config get-value account)

	kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/metacontroller/master/manifests/metacontroller-rbac.yaml
	kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/metacontroller/master/manifests/metacontroller.yaml

install-terraform-operator:
	kubectl apply -f https://raw.githubusercontent.com/danisla/terraform-operator/master/manifests/terraform-operator-rbac.yaml
	kubectl apply -f https://raw.githubusercontent.com/danisla/terraform-operator/master/manifests/terraform-operator.yaml

lpods:
	kubectl -n metacontroller get pods
	
metalogs:
	kubectl -n metacontroller logs --tail=200 -f metacontroller-0

include kaniko.mk
include test.mk
