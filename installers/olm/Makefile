.DEFAULT_GOAL := help
.SUFFIXES:

CCP_IMAGE_PREFIX ?= registry.developers.crunchydata.com/crunchydata
CCP_PG_FULLVERSION ?= 13.3
CCP_POSTGIS_VERSION ?= 3.0
CONTAINER ?= docker
KUBECONFIG ?= $(HOME)/.kube/config
OLM_SDK_VERSION ?= 0.15.1
OLM_TOOLS ?= registry.localhost:5000/postgres-operator-olm-tools:$(OLM_SDK_VERSION)
OLM_VERSION ?= 0.15.1
PGO_BASEOS ?= centos8
PGO_IMAGE_PREFIX ?= registry.developers.crunchydata.com/crunchydata
PGO_VERSION ?= 4.7.0
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
CCP_IMAGE_TAG ?= $(PGO_BASEOS)-$(CCP_PG_FULLVERSION)-$(PGO_VERSION)
CCP_POSTGIS_IMAGE_TAG ?= $(PGO_BASEOS)-$(CCP_PG_FULLVERSION)-$(CCP_POSTGIS_VERSION)-$(PGO_VERSION)

OLM_TOOLS_BASH = $(CONTAINER) run --net=host --rm --tty $(CONTAINER_ARGS) \
		 --mount 'type=bind,source=$(KUBECONFIG),target=/root/.kube/config,ro' \
		 --mount 'type=bind,source=$(CURDIR)/..,target=/mnt/installers' \
		 --workdir '/mnt/installers/$(basename $(notdir $(CURDIR)))' \
		 '$(OLM_TOOLS)'

export CCP_IMAGE_PREFIX CCP_IMAGE_TAG CCP_POSTGIS_IMAGE_TAG
export KUBECONFIG
export OLM_SDK_VERSION
export PGO_IMAGE_PREFIX PGO_IMAGE_TAG PGO_VERSION

.PHONY: clean
clean:
	rm -rf ./package

.PHONY: catalog-source
catalog-source: ## Upload package and version bundle to a Kubernetes namespace
	@test -n '$(NAMESPACE)' || { >&2 echo Must choose a NAMESPACE; exit 1; }
	./install.sh registry '$(NAMESPACE)' olm-registry
	./install.sh catalog_source '$(NAMESPACE)' olm-catalog-source '$(NAMESPACE)' olm-registry

.PHONY: courier-verify
courier-verify:
	operator-courier verify --ui_validate_io ./package

.PHONY: docker-package
docker-package: image-tools
docker-package: ## Build package and version bundle from inside a container
	$(OLM_TOOLS_BASH) make package courier-verify

.PHONY: docker-shell
docker-shell: image-tools
docker-shell: CONTAINER_ARGS = --interactive
docker-shell: ## Start a shell inside a container with all the tools needed to build and test
	$(OLM_TOOLS_BASH)

.PHONY: docker-verify
docker-verify: image-tools
	$(OLM_TOOLS_BASH) make verify

.PHONY: help
help: ALIGN=14
help: ## Print this message
	@awk -F ': ## ' -- "/^[^':]+: ## /"' { printf "'$$(tput bold)'%-$(ALIGN)s'$$(tput sgr0)' %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: image-tools
image-tools:
	$(CONTAINER) build --file Dockerfile --tag '$(OLM_TOOLS)' --build-arg OLM_SDK_VERSION='$(OLM_SDK_VERSION)' .

.PHONY: install
install: ## Install the package in a Kubernetes namespace
	@test -n '$(NAMESPACE)' || { >&2 echo Must choose a NAMESPACE; exit 1; }
	./install.sh operator '$(NAMESPACE)' '$(NAMESPACE)'

.PHONY: install-olm
install-olm: ## Install OLM in Kubernetes
	kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/$(OLM_VERSION)/crds.yaml
	kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/$(OLM_VERSION)/olm.yaml

.PHONY: package
package: export PACKAGE_NAME := postgresql
package: ## Build package and version bundle
	./generate.sh

.PHONY: package-openshift
package-openshift: export K8S_DISTRIBUTION := openshift
package-openshift: package

.PHONY: package-redhat
package-redhat: export K8S_DISTRIBUTION := openshift
package-redhat: export PACKAGE_NAME := crunchy-postgres-operator
package-redhat: CCP_IMAGE_PREFIX := registry.connect.redhat.com/crunchydata
package-redhat: PGO_IMAGE_PREFIX := registry.connect.redhat.com/crunchydata
package-redhat: PGO_BASEOS := $(subst centos,rhel,$(PGO_BASEOS))
package-redhat:
	./generate.sh
	cd ./package && zip -r '$(PACKAGE_NAME)-$(PGO_VERSION).zip' *.yaml '$(PGO_VERSION)'

.PHONY: verify
verify: ## Install and test the package in a new (random) Kubernetes namespace then clean up
	./verify.sh
