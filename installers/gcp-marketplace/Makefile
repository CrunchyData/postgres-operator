.DEFAULT_GOAL := help

DEPLOYER_IMAGE ?= registry.localhost:5000/postgres-operator-gcp-marketplace-deployer:$(PGO_VERSION)
IMAGE_BUILDER ?= buildah
MARKETPLACE_TOOLS ?= gcr.io/cloud-marketplace-tools/k8s/dev:$(MARKETPLACE_VERSION)
MARKETPLACE_VERSION ?= 0.9.4
KUBECONFIG ?= $(HOME)/.kube/config
PARAMETERS ?= {}
PGO_VERSION ?= 4.2.0

IMAGE_BUILD_ARGS = --build-arg MARKETPLACE_VERSION='$(MARKETPLACE_VERSION)' \
		   --build-arg PGO_VERSION='$(PGO_VERSION)'

MARKETPLACE_TOOLS_DEV = docker run --net=host --rm \
			--mount 'type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock,readonly' \
			--mount 'type=bind,source=$(KUBECONFIG),target=/mount/config/.kube/config,readonly' \
			'$(MARKETPLACE_TOOLS)'

# One does _not_ need to be logged in with gcloud.
.PHONY: doctor
doctor: ## Check development prerequisites
	$(MARKETPLACE_TOOLS_DEV) doctor

.PHONY: doctor-fix
doctor-fix:
	@# https://github.com/kubernetes-sigs/application/tree/master/config/crds
	kubectl 2>/dev/null get crd/applications.app.k8s.io -o jsonpath='{""}' || \
		kubectl create -f https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/crd/app-crd.yaml

.PHONY: help
help: ALIGN=14
help: ## Print this message
	@awk -F ': ## ' -- "/^[^':]+: ## /"' { printf "'$$(tput bold)'%-$(ALIGN)s'$$(tput sgr0)' %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: image
image: image-$(IMAGE_BUILDER)

.PHONY: image-buildah
image-buildah: ## Build the deployer image with Buildah
	sudo buildah bud --file Dockerfile --tag '$(DEPLOYER_IMAGE)' $(IMAGE_BUILD_ARGS) --layers ../..
	sudo buildah push '$(DEPLOYER_IMAGE)' docker-daemon:'$(DEPLOYER_IMAGE)'

.PHONY: image-docker
image-docker: ## Build the deployer image with Docker
	docker build --file Dockerfile --tag '$(DEPLOYER_IMAGE)' $(IMAGE_BUILD_ARGS) ../..

# PARAMETERS='{"OPERATOR_NAMESPACE": "", "OPERATOR_NAME": "", "OPERATOR_ADMIN_PASSWORD": ""}'
.PHONY: install
install: ## Execute the deployer image in an existing Kubernetes namespace
	$(MARKETPLACE_TOOLS_DEV) install --deployer='$(DEPLOYER_IMAGE)' --parameters='$(PARAMETERS)'

# PARAMETERS='{"OPERATOR_ADMIN_PASSWORD": ""}'
.PHONY: verify
verify: ## Execute and test the deployer image in a new (random) Kubernetes namespace then clean up
	$(MARKETPLACE_TOOLS_DEV) verify --deployer='$(DEPLOYER_IMAGE)' --parameters='$(PARAMETERS)'
