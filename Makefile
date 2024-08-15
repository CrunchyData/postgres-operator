PGO_IMAGE_NAME ?= postgres-operator
PGO_IMAGE_MAINTAINER ?= Crunchy Data
PGO_IMAGE_SUMMARY ?= Crunchy PostgreSQL Operator
PGO_IMAGE_DESCRIPTION ?= $(PGO_IMAGE_SUMMARY)
PGO_IMAGE_URL ?= https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes
PGO_IMAGE_PREFIX ?= localhost

PGMONITOR_DIR ?= hack/tools/pgmonitor
PGMONITOR_VERSION ?= v4.11.0
QUERIES_CONFIG_DIR ?= hack/tools/queries

EXTERNAL_SNAPSHOTTER_DIR ?= hack/tools/external-snapshotter
EXTERNAL_SNAPSHOTTER_VERSION ?= v8.0.1

# Buildah's "build" used to be "bud". Use the alias to be compatible for a while.
BUILDAH_BUILD ?= buildah bud

GO ?= go
GO_BUILD = $(GO) build
GO_TEST ?= $(GO) test
KUTTL ?= kubectl-kuttl
KUTTL_TEST ?= $(KUTTL) test

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

.PHONY: setup
setup: ## Run Setup needed to build images
setup: get-pgmonitor

.PHONY: get-pgmonitor
get-pgmonitor:
	git -C '$(dir $(PGMONITOR_DIR))' clone https://github.com/CrunchyData/pgmonitor.git || git -C '$(PGMONITOR_DIR)' fetch origin
	@git -C '$(PGMONITOR_DIR)' checkout '$(PGMONITOR_VERSION)'
	@git -C '$(PGMONITOR_DIR)' config pull.ff only
	[ -d '${QUERIES_CONFIG_DIR}' ] || mkdir -p '${QUERIES_CONFIG_DIR}'
	cp -r '$(PGMONITOR_DIR)/postgres_exporter/common/.' '${QUERIES_CONFIG_DIR}'
	cp '$(PGMONITOR_DIR)/postgres_exporter/linux/queries_backrest.yml' '${QUERIES_CONFIG_DIR}'

.PHONY: get-external-snapshotter
get-external-snapshotter:
	git -C '$(dir $(EXTERNAL_SNAPSHOTTER_DIR))' clone https://github.com/kubernetes-csi/external-snapshotter.git || git -C '$(EXTERNAL_SNAPSHOTTER_DIR)' fetch origin
	@git -C '$(EXTERNAL_SNAPSHOTTER_DIR)' checkout '$(EXTERNAL_SNAPSHOTTER_VERSION)'
	@git -C '$(EXTERNAL_SNAPSHOTTER_DIR)' config pull.ff only

.PHONY: clean
clean: ## Clean resources
clean: clean-deprecated
	rm -f bin/postgres-operator
	rm -f config/rbac/role.yaml
	rm -rf licenses/*/
	[ ! -d testing/kuttl/e2e-generated ] || rm -r testing/kuttl/e2e-generated
	[ ! -d testing/kuttl/e2e-generated-other ] || rm -r testing/kuttl/e2e-generated-other
	rm -rf build/crd/generated build/crd/*/generated
	[ ! -f hack/tools/setup-envtest ] || rm hack/tools/setup-envtest
	[ ! -d hack/tools/envtest ] || { chmod -R u+w hack/tools/envtest && rm -r hack/tools/envtest; }
	[ ! -d hack/tools/pgmonitor ] || rm -rf hack/tools/pgmonitor
	[ ! -d hack/tools/external-snapshotter ] || rm -rf hack/tools/external-snapshotter
	[ ! -n "$$(ls hack/tools)" ] || rm -r hack/tools/*
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
	@# crunchy-postgres-exporter used to live in this repo
	[ ! -d bin/crunchy-postgres-exporter ] || rm -r bin/crunchy-postgres-exporter
	[ ! -d build/crunchy-postgres-exporter ] || rm -r build/crunchy-postgres-exporter


##@ Deployment
.PHONY: createnamespaces
createnamespaces: ## Create operator and target namespaces
	kubectl apply -k ./config/namespace

.PHONY: deletenamespaces
deletenamespaces: ## Delete operator and target namespaces
	kubectl delete -k ./config/namespace

.PHONY: install
install: ## Install the postgrescluster CRD
	kubectl apply --server-side -k ./config/crd

.PHONY: uninstall
uninstall: ## Delete the postgrescluster CRD
	kubectl delete -k ./config/crd

.PHONY: deploy
deploy: ## Deploy the PostgreSQL Operator (enables the postgrescluster controller)
	kubectl apply --server-side -k ./config/default

.PHONY: undeploy
undeploy: ## Undeploy the PostgreSQL Operator
	kubectl delete -k ./config/default

.PHONY: deploy-dev
deploy-dev: ## Deploy the PostgreSQL Operator locally
deploy-dev: PGO_FEATURE_GATES ?= "TablespaceVolumes=true,VolumeSnapshots=true"
deploy-dev: get-pgmonitor
deploy-dev: build-postgres-operator
deploy-dev: createnamespaces
	kubectl apply --server-side -k ./config/dev
	hack/create-kubeconfig.sh postgres-operator pgo
	env \
		QUERIES_CONFIG_DIR="${QUERIES_CONFIG_DIR}" \
		CRUNCHY_DEBUG=true \
		PGO_FEATURE_GATES="${PGO_FEATURE_GATES}" \
		CHECK_FOR_UPGRADES='$(if $(CHECK_FOR_UPGRADES),$(CHECK_FOR_UPGRADES),false)' \
		KUBECONFIG=hack/.kube/postgres-operator/pgo \
		PGO_NAMESPACE='postgres-operator' \
		$(shell kubectl kustomize ./config/dev | \
			sed -ne '/^kind: Deployment/,/^---/ { \
				/RELATED_IMAGE_/ { N; s,.*\(RELATED_[^[:space:]]*\).*value:[[:space:]]*\([^[:space:]]*\),\1="\2",; p; }; \
			}') \
		$(foreach v,$(filter RELATED_IMAGE_%,$(.VARIABLES)),$(v)="$($(v))") \
		bin/postgres-operator

##@ Build - Binary
.PHONY: build-postgres-operator
build-postgres-operator: ## Build the postgres-operator binary
	CGO_ENABLED=1 $(GO_BUILD) $(\
		) --ldflags '-X "main.versionString=$(PGO_VERSION)"' $(\
		) --trimpath -o bin/postgres-operator ./cmd/postgres-operator

##@ Build - Images
.PHONY: build-postgres-operator-image
build-postgres-operator-image: ## Build the postgres-operator image
build-postgres-operator-image: PGO_IMAGE_REVISION := $(shell git rev-parse HEAD)
build-postgres-operator-image: PGO_IMAGE_TIMESTAMP := $(shell date -u +%FT%TZ)
build-postgres-operator-image: build-postgres-operator
build-postgres-operator-image: build/postgres-operator/Dockerfile
	$(if $(shell (echo 'buildah version 1.24'; $(word 1,$(BUILDAH_BUILD)) --version) | sort -Vc 2>&1), \
		$(warning WARNING: old buildah does not invalidate its cache for changed labels: \
			https://github.com/containers/buildah/issues/3517))
	$(if $(IMAGE_TAG),,	$(error missing IMAGE_TAG))
	$(strip $(BUILDAH_BUILD)) \
		--tag $(BUILDAH_TRANSPORT)$(PGO_IMAGE_PREFIX)/$(PGO_IMAGE_NAME):$(IMAGE_TAG) \
		--label name='$(PGO_IMAGE_NAME)' \
		--label build-date='$(PGO_IMAGE_TIMESTAMP)' \
		--label description='$(PGO_IMAGE_DESCRIPTION)' \
		--label maintainer='$(PGO_IMAGE_MAINTAINER)' \
		--label summary='$(PGO_IMAGE_SUMMARY)' \
		--label url='$(PGO_IMAGE_URL)' \
		--label vcs-ref='$(PGO_IMAGE_REVISION)' \
		--label vendor='$(PGO_IMAGE_MAINTAINER)' \
		--label io.k8s.display-name='$(PGO_IMAGE_NAME)' \
		--label io.k8s.description='$(PGO_IMAGE_DESCRIPTION)' \
		--label io.openshift.tags="postgresql,postgres,sql,nosql,crunchy" \
		--annotation org.opencontainers.image.authors='$(PGO_IMAGE_MAINTAINER)' \
		--annotation org.opencontainers.image.vendor='$(PGO_IMAGE_MAINTAINER)' \
		--annotation org.opencontainers.image.created='$(PGO_IMAGE_TIMESTAMP)' \
		--annotation org.opencontainers.image.description='$(PGO_IMAGE_DESCRIPTION)' \
		--annotation org.opencontainers.image.revision='$(PGO_IMAGE_REVISION)' \
		--annotation org.opencontainers.image.title='$(PGO_IMAGE_SUMMARY)' \
		--annotation org.opencontainers.image.url='$(PGO_IMAGE_URL)' \
		$(if $(PGO_VERSION),$(strip \
			--label release='$(PGO_VERSION)' \
			--label version='$(PGO_VERSION)' \
			--annotation org.opencontainers.image.version='$(PGO_VERSION)' \
		)) \
		--file $< --format docker --layers .

##@ Test
.PHONY: check
check: ## Run basic go tests with coverage output
check: get-pgmonitor
	QUERIES_CONFIG_DIR="$(CURDIR)/${QUERIES_CONFIG_DIR}" $(GO_TEST) -cover ./...

# Available versions: curl -s 'https://storage.googleapis.com/kubebuilder-tools/' | grep -o '<Key>[^<]*</Key>'
# - KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT=true
.PHONY: check-envtest
check-envtest: ## Run check using envtest and a mock kube api
check-envtest: ENVTEST_USE = $(ENVTEST) --bin-dir=$(CURDIR)/hack/tools/envtest use $(ENVTEST_K8S_VERSION)
check-envtest: SHELL = bash
check-envtest: get-pgmonitor tools/setup-envtest get-external-snapshotter
	@$(ENVTEST_USE) --print=overview && echo
	source <($(ENVTEST_USE) --print=env) && PGO_NAMESPACE="postgres-operator" QUERIES_CONFIG_DIR="$(CURDIR)/${QUERIES_CONFIG_DIR}" \
		$(GO_TEST) -count=1 -cover ./...

# The "PGO_TEST_TIMEOUT_SCALE" environment variable (default: 1) can be set to a
# positive number that extends test timeouts. The following runs tests with 
# timeouts that are 20% longer than normal:
# make check-envtest-existing PGO_TEST_TIMEOUT_SCALE=1.2
.PHONY: check-envtest-existing
check-envtest-existing: ## Run check using envtest and an existing kube api
check-envtest-existing: get-pgmonitor get-external-snapshotter
check-envtest-existing: createnamespaces
	kubectl apply --server-side -k ./config/dev
	USE_EXISTING_CLUSTER=true PGO_NAMESPACE="postgres-operator" QUERIES_CONFIG_DIR="$(CURDIR)/${QUERIES_CONFIG_DIR}" \
		$(GO_TEST) -count=1 -cover -p=1 ./...
	kubectl delete -k ./config/dev

# Expects operator to be running
.PHONY: check-kuttl
check-kuttl: ## Run kuttl end-to-end tests
check-kuttl: ## example command: make check-kuttl KUTTL_TEST='
	${KUTTL_TEST} \
		--config testing/kuttl/kuttl-test.yaml

.PHONY: generate-kuttl
generate-kuttl: export KUTTL_PG_UPGRADE_FROM_VERSION ?= 15
generate-kuttl: export KUTTL_PG_UPGRADE_TO_VERSION ?= 16
generate-kuttl: export KUTTL_PG_VERSION ?= 16
generate-kuttl: export KUTTL_POSTGIS_VERSION ?= 3.4
generate-kuttl: export KUTTL_PSQL_IMAGE ?= registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-16.3-1
generate-kuttl: export KUTTL_TEST_DELETE_NAMESPACE ?= kuttl-test-delete-namespace
generate-kuttl: ## Generate kuttl tests
	[ ! -d testing/kuttl/e2e-generated ] || rm -r testing/kuttl/e2e-generated
	[ ! -d testing/kuttl/e2e-generated-other ] || rm -r testing/kuttl/e2e-generated-other
	bash -ceu ' \
	case $(KUTTL_PG_VERSION) in \
	16 ) export KUTTL_BITNAMI_IMAGE_TAG=16.0.0-debian-11-r3 ;; \
	15 ) export KUTTL_BITNAMI_IMAGE_TAG=15.0.0-debian-11-r4 ;; \
	14 ) export KUTTL_BITNAMI_IMAGE_TAG=14.5.0-debian-11-r37 ;; \
	13 ) export KUTTL_BITNAMI_IMAGE_TAG=13.8.0-debian-11-r39 ;; \
	12 ) export KUTTL_BITNAMI_IMAGE_TAG=12.12.0-debian-11-r40 ;; \
	esac; \
	render() { envsubst '"'"' \
		$$KUTTL_PG_UPGRADE_FROM_VERSION $$KUTTL_PG_UPGRADE_TO_VERSION \
		$$KUTTL_PG_VERSION $$KUTTL_POSTGIS_VERSION $$KUTTL_PSQL_IMAGE \
		$$KUTTL_BITNAMI_IMAGE_TAG $$KUTTL_TEST_DELETE_NAMESPACE'"'"'; }; \
	while [ $$# -gt 0 ]; do \
		source="$${1}" target="$${1/e2e/e2e-generated}"; \
		mkdir -p "$${target%/*}"; render < "$${source}" > "$${target}"; \
		shift; \
	done' - testing/kuttl/e2e/*/*.yaml testing/kuttl/e2e-other/*/*.yaml testing/kuttl/e2e/*/*/*.yaml testing/kuttl/e2e-other/*/*/*.yaml

##@ Generate

.PHONY: check-generate
check-generate: ## Check crd, deepcopy functions, and rbac generation
check-generate: generate-crd
check-generate: generate-deepcopy
check-generate: generate-rbac
	git diff --exit-code -- config/crd
	git diff --exit-code -- config/rbac
	git diff --exit-code -- pkg/apis

.PHONY: generate
generate: ## Generate crd, deepcopy functions, and rbac
generate: generate-crd
generate: generate-deepcopy
generate: generate-rbac

.PHONY: generate-crd
generate-crd: ## Generate Custom Resource Definitions (CRDs)
generate-crd: tools/controller-gen
	$(CONTROLLER) \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='build/crd/postgresclusters/generated' # build/crd/{plural}/generated/{group}_{plural}.yaml
	@
	$(CONTROLLER) \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='build/crd/pgupgrades/generated' # build/crd/{plural}/generated/{group}_{plural}.yaml
	@
	$(CONTROLLER) \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='build/crd/pgadmins/generated' # build/crd/{plural}/generated/{group}_{plural}.yaml
	@
	$(CONTROLLER) \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='build/crd/crunchybridgeclusters/generated' # build/crd/{plural}/generated/{group}_{plural}.yaml
	@
	kubectl kustomize ./build/crd/postgresclusters > ./config/crd/bases/postgres-operator.crunchydata.com_postgresclusters.yaml
	kubectl kustomize ./build/crd/pgupgrades > ./config/crd/bases/postgres-operator.crunchydata.com_pgupgrades.yaml
	kubectl kustomize ./build/crd/pgadmins > ./config/crd/bases/postgres-operator.crunchydata.com_pgadmins.yaml
	kubectl kustomize ./build/crd/crunchybridgeclusters > ./config/crd/bases/postgres-operator.crunchydata.com_crunchybridgeclusters.yaml

.PHONY: generate-deepcopy
generate-deepcopy: ## Generate DeepCopy functions
generate-deepcopy: tools/controller-gen
	$(CONTROLLER) \
		object:headerFile='hack/boilerplate.go.txt' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...'

.PHONY: generate-rbac
generate-rbac: ## Generate RBAC
generate-rbac: tools/controller-gen
	$(CONTROLLER) \
		rbac:roleName='generated' \
		paths='./cmd/...' paths='./internal/...' \
		output:dir='config/rbac' # ${directory}/role.yaml
	./hack/generate-rbac.sh 'config/rbac'

##@ Tools

.PHONY: tools
tools: ## Download tools like controller-gen and kustomize if necessary.

# go-get-tool will 'go install' any package $2 and install it to $1.
define go-get-tool
@[ -f '$(1)' ] || { echo Downloading '$(2)'; GOBIN='$(abspath $(dir $(1)))' $(GO) install '$(2)'; }
endef

CONTROLLER ?= hack/tools/controller-gen
tools: tools/controller-gen
tools/controller-gen:
	$(call go-get-tool,$(CONTROLLER),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0)

ENVTEST ?= hack/tools/setup-envtest
tools: tools/setup-envtest
tools/setup-envtest:
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

##@ Release

.PHONY: license licenses
license: licenses
licenses: ## Aggregate license files
	./bin/license_aggregator.sh ./cmd/...

.PHONY: release-postgres-operator-image release-postgres-operator-image-labels
release-postgres-operator-image: ## Build the postgres-operator image and all its prerequisites
release-postgres-operator-image: release-postgres-operator-image-labels
release-postgres-operator-image: licenses
release-postgres-operator-image: build-postgres-operator-image
release-postgres-operator-image-labels:
	$(if $(PGO_IMAGE_DESCRIPTION),,	$(error missing PGO_IMAGE_DESCRIPTION))
	$(if $(PGO_IMAGE_MAINTAINER),, 	$(error missing PGO_IMAGE_MAINTAINER))
	$(if $(PGO_IMAGE_NAME),,       	$(error missing PGO_IMAGE_NAME))
	$(if $(PGO_IMAGE_SUMMARY),,    	$(error missing PGO_IMAGE_SUMMARY))
	$(if $(PGO_VERSION),,			$(error missing PGO_VERSION))
