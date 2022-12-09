
# Default values if not already set
PGOROOT ?= $(CURDIR)
PGO_BASEOS ?= ubi8
PGO_IMAGE_PREFIX ?= crunchydata
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
PGO_VERSION ?= $(shell git describe --tags)
PGO_PG_VERSION ?= 14
PGO_PG_FULLVERSION ?= 14.6
PGO_KUBE_CLIENT ?= kubectl

# Valid values: buildah (default), docker
IMGBUILDER ?= buildah
# Determines whether or not rootless builds are enabled
IMG_ROOTLESS_BUILD ?= false
# Defines the sudo command that should be prepended to various build commands when rootless builds are
# not enabled
IMGCMDSUDO=
ifneq ("$(IMG_ROOTLESS_BUILD)", "true")
	IMGCMDSUDO=sudo --preserve-env
endif
IMGCMDSTEM=$(IMGCMDSUDO) buildah bud --layers $(SQUASH)
# Buildah's "build" used to be "bud". Use the alias to be compatible for a while.
BUILDAH_BUILD ?= buildah bud

# Default the buildah format to docker to ensure it is possible to pull the images from a docker
# repository using docker (otherwise the images may not be recognized)
export BUILDAH_FORMAT ?= docker

# Allows simplification of IMGBUILDER switching
ifeq ("$(IMGBUILDER)","docker")
        IMGCMDSTEM=docker build
endif

# set the proper packager, registry and base image based on the PGO_BASEOS configured
DOCKERBASEREGISTRY=
BASE_IMAGE_OS=
ifeq ("$(PGO_BASEOS)", "ubi8")
    BASE_IMAGE_OS=ubi8-minimal
    DOCKERBASEREGISTRY=registry.access.redhat.com/
    PACKAGER=microdnf
endif

DEBUG_BUILD ?= false
GO ?= go
GO_BUILD = $(GO_CMD) build -trimpath
GO_CMD = $(GO_ENV) $(GO)
GO_TEST ?= $(GO) test
KUTTL ?= kubectl-kuttl
KUTTL_TEST ?= $(KUTTL) test

# Disable optimizations if creating a debug build
ifeq ("$(DEBUG_BUILD)", "true")
	GO_BUILD = $(GO_CMD) build -gcflags='all=-N -l'
endif

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-formatting the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: all
all: ## Build all images
all: build-postgres-operator-image
all: build-crunchy-postgres-exporter-image

.PHONY: setup
setup: ## Run Setup needed to build images
	PGOROOT='$(PGOROOT)' ./bin/get-deps.sh
	./bin/check-deps.sh

.PHONY: clean
clean: ## Clean resources
clean: clean-deprecated
	rm -f bin/postgres-operator
	rm -f config/rbac/role.yaml
	[ ! -d testing/kuttl/e2e-generated ] || rm -r testing/kuttl/e2e-generated
	[ ! -d testing/kuttl/e2e-generated-other ] || rm -r testing/kuttl/e2e-generated-other
	rm -rf build/crd/generated build/crd/*/generated
	[ ! -f hack/tools/setup-envtest ] || hack/tools/setup-envtest --bin-dir=hack/tools/envtest cleanup
	[ ! -f hack/tools/setup-envtest ] || rm hack/tools/setup-envtest
	[ ! -d hack/tools/envtest ] || rm -r hack/tools/envtest
	[ ! -n "$$(ls hack/tools)" ] || rm hack/tools/*
	[ ! -d hack/.kube ] || rm -r hack/.kube

.PHONY: clean-deprecated
clean-deprecated: ## Clean deprecated resources
	@# packages used to be downloaded into the vendor directory
	[ ! -d vendor ] || rm -r vendor
	@# executables used to be compiled into the $GOBIN directory
	[ ! -n '$(GOBIN)' ] || rm -f $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo
	@# executables used to be in subdirectories
	[ ! -d bin/pgo-rmdata ] || rm -r bin/pgo-rmdata
	[ ! -d bin/pgo-backrest ] || rm -r bin/pgo-backrest
	[ ! -d bin/pgo-scheduler ] || rm -r bin/pgo-scheduler
	[ ! -d bin/postgres-operator ] || rm -r bin/postgres-operator
	@# keys used to be generated before install
	[ ! -d conf/pgo-backrest-repo ] || rm -r conf/pgo-backrest-repo
	[ ! -d conf/postgres-operator ] || rm -r conf/postgres-operator

##@ Deployment
.PHONY: createnamespaces
createnamespaces: ## Create operator and target namespaces
	$(PGO_KUBE_CLIENT) apply -k ./config/namespace

.PHONY: deletenamespaces
deletenamespaces: ## Delete operator and target namespaces
	$(PGO_KUBE_CLIENT) delete -k ./config/namespace

.PHONY: install
install: ## Install the postgrescluster CRD
	$(PGO_KUBE_CLIENT) apply --server-side -k ./config/crd

.PHONY: uninstall
uninstall: ## Delete the postgrescluster CRD
	$(PGO_KUBE_CLIENT) delete -k ./config/crd

.PHONY: deploy
deploy: ## Deploy the PostgreSQL Operator (enables the postgrescluster controller)
	$(PGO_KUBE_CLIENT) apply --server-side -k ./config/default

.PHONY: undeploy
undeploy: ## Undeploy the PostgreSQL Operator
	$(PGO_KUBE_CLIENT) delete -k ./config/default

.PHONY: deploy-dev
deploy-dev: ## Deploy the PostgreSQL Operator locally
deploy-dev: build-postgres-operator
deploy-dev: createnamespaces
	$(PGO_KUBE_CLIENT) apply --server-side -k ./config/dev
	hack/create-kubeconfig.sh postgres-operator pgo
	env \
		CRUNCHY_DEBUG=true \
		CHECK_FOR_UPGRADES='$(if $(CHECK_FOR_UPGRADES),$(CHECK_FOR_UPGRADES),false)' \
		KUBECONFIG=hack/.kube/postgres-operator/pgo \
		PGO_NAMESPACE='postgres-operator' \
		$(shell $(PGO_KUBE_CLIENT) kustomize ./config/dev | \
			sed -ne '/^kind: Deployment/,/^---/ { \
				/RELATED_IMAGE_/ { N; s,.*\(RELATED_[^[:space:]]*\).*value:[[:space:]]*\([^[:space:]]*\),\1="\2",; p; }; \
			}') \
		$(foreach v,$(filter RELATED_IMAGE_%,$(.VARIABLES)),$(v)="$($(v))") \
		bin/postgres-operator

##@ Build - Binary
.PHONY: build-postgres-operator
build-postgres-operator: ## Build the postgres-operator binary
	$(GO_BUILD) -ldflags '-X "main.versionString=$(PGO_VERSION)"' \
		-o bin/postgres-operator ./cmd/postgres-operator

##@ Build - Images
.PHONY: build-pgo-base-image
build-pgo-base-image: ## Build the pgo-base
build-pgo-base-image: licenses
build-pgo-base-image: $(PGOROOT)/build/pgo-base/Dockerfile
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/build/pgo-base/Dockerfile \
		-t $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) \
		--build-arg BASE_IMAGE_OS=$(BASE_IMAGE_OS) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg RELVER=$(PGO_VERSION) \
		--build-arg DOCKERBASEREGISTRY=$(DOCKERBASEREGISTRY) \
		--build-arg PACKAGER=$(PACKAGER) \
		--build-arg PG_FULL=$(PGO_PG_FULLVERSION) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		$(PGOROOT)

.PHONY: build-crunchy-postgres-exporter-image
build-crunchy-postgres-exporter-image: ## Build the crunchy-postgres-exporter image
build-crunchy-postgres-exporter-image: build-pgo-base-image
build-crunchy-postgres-exporter-image: $(PGOROOT)/build/crunchy-postgres-exporter/Dockerfile
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/build/crunchy-postgres-exporter/Dockerfile \
		-t $(PGO_IMAGE_PREFIX)/crunchy-postgres-exporter:$(PGO_IMAGE_TAG) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg BASEVER=$(PGO_VERSION) \
		--build-arg PACKAGER=$(PACKAGER) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg PREFIX=$(PGO_IMAGE_PREFIX) \
		$(PGOROOT)

.PHONY: build-postgres-operator-image
build-postgres-operator-image: ## Build the postgres-operator image
build-postgres-operator-image: build-postgres-operator
build-postgres-operator-image: $(PGOROOT)/build/postgres-operator/Dockerfile
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/build/postgres-operator/Dockerfile \
		-t $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) \
		--build-arg BASE_IMAGE_OS=$(BASE_IMAGE_OS) \
		--build-arg PACKAGER=$(PACKAGER) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg RELVER=$(PGO_VERSION) \
		--build-arg DOCKERBASEREGISTRY=$(DOCKERBASEREGISTRY) \
		--build-arg PACKAGER=$(PACKAGER) \
		--build-arg PG_FULL=$(PGO_PG_FULLVERSION) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		$(PGOROOT)

##@ Test
.PHONY: check
check: ## Run basic go tests with coverage output
	$(GO_TEST) -cover ./...

# Available versions: curl -s 'https://storage.googleapis.com/kubebuilder-tools/' | grep -o '<Key>[^<]*</Key>'
# - KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT=true
.PHONY: check-envtest
check-envtest: ## Run check using envtest and a mock kube api
check-envtest: ENVTEST_USE = hack/tools/setup-envtest --bin-dir=$(CURDIR)/hack/tools/envtest use $(ENVTEST_K8S_VERSION)
check-envtest: SHELL = bash
check-envtest:
	GOBIN='$(CURDIR)/hack/tools' $(GO) install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@$(ENVTEST_USE) --print=overview && echo
	source <($(ENVTEST_USE) --print=env) && PGO_NAMESPACE="postgres-operator" $(GO_TEST) -count=1 -cover -tags=envtest ./...

# The "PGO_TEST_TIMEOUT_SCALE" environment variable (default: 1) can be set to a
# positive number that extends test timeouts. The following runs tests with 
# timeouts that are 20% longer than normal:
# make check-envtest-existing PGO_TEST_TIMEOUT_SCALE=1.2
.PHONY: check-envtest-existing
check-envtest-existing: ## Run check using envtest and an existing kube api
check-envtest-existing: createnamespaces
	kubectl apply --server-side -k ./config/dev
	USE_EXISTING_CLUSTER=true PGO_NAMESPACE="postgres-operator" $(GO_TEST) -count=1 -cover -p=1 -tags=envtest ./...
	kubectl delete -k ./config/dev

# Expects operator to be running
.PHONY: check-kuttl
check-kuttl: ## Run kuttl end-to-end tests
check-kuttl: ## example command: make check-kuttl KUTTL_TEST='
	${KUTTL_TEST} \
		--config testing/kuttl/kuttl-test.yaml

.PHONY: generate-kuttl
generate-kuttl: export KUTTL_PG_UPGRADE_FROM_VERSION ?= 13
generate-kuttl: export KUTTL_PG_UPGRADE_TO_VERSION ?= 14
generate-kuttl: export KUTTL_PG_VERSION ?= 14
generate-kuttl: export KUTTL_POSTGIS_VERSION ?= 3.1
generate-kuttl: export KUTTL_PSQL_IMAGE ?= registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-14.6-2
generate-kuttl: ## Generate kuttl tests
	[ ! -d testing/kuttl/e2e-generated ] || rm -r testing/kuttl/e2e-generated
	[ ! -d testing/kuttl/e2e-generated-other ] || rm -r testing/kuttl/e2e-generated-other
	bash -ceu ' \
	case $(KUTTL_PG_VERSION) in \
	15 ) export KUTTL_BITNAMI_IMAGE_TAG=15.0.0-debian-11-r4 ;; \
	14 ) export KUTTL_BITNAMI_IMAGE_TAG=14.5.0-debian-11-r37 ;; \
	13 ) export KUTTL_BITNAMI_IMAGE_TAG=13.8.0-debian-11-r39 ;; \
	12 ) export KUTTL_BITNAMI_IMAGE_TAG=12.12.0-debian-11-r40 ;; \
	11 ) export KUTTL_BITNAMI_IMAGE_TAG=11.17.0-debian-11-r39 ;; \
	esac; \
	render() { envsubst '"'"'$$KUTTL_PG_UPGRADE_FROM_VERSION $$KUTTL_PG_UPGRADE_TO_VERSION $$KUTTL_PG_VERSION $$KUTTL_POSTGIS_VERSION $$KUTTL_PSQL_IMAGE $$KUTTL_BITNAMI_IMAGE_TAG'"'"'; }; \
	while [ $$# -gt 0 ]; do \
		source="$${1}" target="$${1/e2e/e2e-generated}"; \
		mkdir -p "$${target%/*}"; render < "$${source}" > "$${target}"; \
		shift; \
	done' - testing/kuttl/e2e/*/*.yaml testing/kuttl/e2e-other/*/*.yaml

##@ Generate

.PHONY: check-generate
check-generate: ## Check crd, crd-docs, deepcopy functions, and rbac generation
check-generate: generate-crd
check-generate: generate-deepcopy
check-generate: generate-rbac
	git diff --exit-code -- config/crd
	git diff --exit-code -- config/rbac
	git diff --exit-code -- pkg/apis

.PHONY: generate
generate: ## Generate crd, crd-docs, deepcopy functions, and rbac
generate: generate-crd
generate: generate-crd-docs
generate: generate-deepcopy
generate: generate-rbac

.PHONY: generate-crd
generate-crd: ## Generate crd
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='build/crd/postgresclusters/generated' # build/crd/{plural}/generated/{group}_{plural}.yaml
	@
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='build/crd/pgupgrades/generated' # build/crd/{plural}/generated/{group}_{plural}.yaml
	@
	$(PGO_KUBE_CLIENT) kustomize ./build/crd/postgresclusters > ./config/crd/bases/postgres-operator.crunchydata.com_postgresclusters.yaml
	$(PGO_KUBE_CLIENT) kustomize ./build/crd/pgupgrades > ./config/crd/bases/postgres-operator.crunchydata.com_pgupgrades.yaml

.PHONY: generate-crd-docs
generate-crd-docs: ## Generate crd-docs
	GOBIN='$(CURDIR)/hack/tools' $(GO) install fybrik.io/crdoc@v0.5.2
	./hack/tools/crdoc \
		--resources ./config/crd/bases \
		--template ./hack/api-template.tmpl \
		--output ./docs/content/references/crd.md

.PHONY: generate-deepcopy
generate-deepcopy: ## Generate deepcopy functions
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		object:headerFile='hack/boilerplate.go.txt' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...'

.PHONY: generate-rbac
generate-rbac: ## Generate rbac
	GOBIN='$(CURDIR)/hack/tools' ./hack/generate-rbac.sh \
		'./internal/...' 'config/rbac'

##@ Release

.PHONY: license licenses
license: licenses
licenses: ## Aggregate license files
	./bin/license_aggregator.sh ./cmd/...
