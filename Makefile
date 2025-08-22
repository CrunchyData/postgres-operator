
PGMONITOR_DIR ?= hack/tools/pgmonitor
PGMONITOR_VERSION ?= v5.2.1
QUERIES_CONFIG_DIR ?= hack/tools/queries

BUILDAH ?= buildah
GO ?= go
GO_TEST ?= $(GO) test

# Ensure modules imported by `postgres-operator` and `controller-gen` are compatible
# by managing them together in the main module.
CONTROLLER ?= $(GO) tool sigs.k8s.io/controller-tools/cmd/controller-gen

# Run tests using the latest tools.
CHAINSAW ?= $(GO) run github.com/kyverno/chainsaw@latest
CHAINSAW_TEST ?= $(CHAINSAW) test
ENVTEST ?= $(GO) run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
KUTTL ?= $(GO) run github.com/kudobuilder/kuttl/cmd/kubectl-kuttl@latest
KUTTL_TEST ?= $(KUTTL) test

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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

.PHONY: notes
notes: ## List known issues and future considerations
	command -v rg > /dev/null && rg '(BUGS|FIXME|NOTE|TODO)[(][^)]+[)]' || grep -Ern '(BUGS|FIXME|NOTE|TODO)[(][^)]+[)]' *

.PHONY: clean
clean: ## Clean resources
clean: clean-deprecated
	rm -f bin/postgres-operator
	rm -rf licenses/*/
	[ ! -d testing/kuttl/e2e-generated ] || rm -r testing/kuttl/e2e-generated
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
	@# CRDs used to require patching
	[ ! -d build/crd ] || rm -r build/crd
	@# Old testing directories
	[ ! -d testing/kuttl/e2e-generated-other ] || rm -r testing/kuttl/e2e-generated-other
	@# Tools used to be downloaded directly
	[ ! -f hack/tools/controller-gen ] || rm hack/tools/controller-gen
	[ ! -f hack/tools/setup-envtest ] || rm hack/tools/setup-envtest


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
deploy-dev: get-pgmonitor
deploy-dev: createnamespaces
	kubectl apply --server-side -k ./config/dev
	hack/create-kubeconfig.sh postgres-operator pgo
	env \
		QUERIES_CONFIG_DIR='$(QUERIES_CONFIG_DIR)' \
		CRUNCHY_DEBUG="$${CRUNCHY_DEBUG:-true}" \
		PGO_FEATURE_GATES="$${PGO_FEATURE_GATES:-AllAlpha=true}" \
		CHECK_FOR_UPGRADES="$${CHECK_FOR_UPGRADES:-false}" \
		KUBECONFIG=hack/.kube/postgres-operator/pgo \
		PGO_NAMESPACE='postgres-operator' \
		PGO_INSTALLER='deploy-dev' \
		PGO_INSTALLER_ORIGIN='postgres-operator-repo' \
		BUILD_SOURCE='build-postgres-operator' \
		$(shell kubectl kustomize ./config/dev | \
			sed -ne '/^kind: Deployment/,/^---/ { \
				/RELATED_IMAGE_/ { N; s,.*\(RELATED_[^[:space:]]*\).*value:[[:space:]]*\([^[:space:]]*\),\1="\2",; p; }; \
			}') \
		$(foreach v,$(filter RELATED_IMAGE_%,$(.VARIABLES)),$(v)="$($(v))") \
		$(GO) run ./cmd/postgres-operator

##@ Build

.PHONY: build
build: ## Build a postgres-operator image
	$(BUILDAH) build --tag localhost/postgres-operator \
		--label org.opencontainers.image.authors='Crunchy Data' \
		--label org.opencontainers.image.description='Crunchy PostgreSQL Operator' \
		--label org.opencontainers.image.revision='$(shell git rev-parse HEAD)' \
		--label org.opencontainers.image.source='https://github.com/CrunchyData/postgres-operator' \
		--label org.opencontainers.image.title='Crunchy PostgreSQL Operator' \
		.

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
check-envtest: get-pgmonitor
	@$(ENVTEST_USE) --print=overview && echo
	source <($(ENVTEST_USE) --print=env) && PGO_NAMESPACE="postgres-operator" QUERIES_CONFIG_DIR="$(CURDIR)/${QUERIES_CONFIG_DIR}" \
		$(GO_TEST) -count=1 -cover ./...

# The "PGO_TEST_TIMEOUT_SCALE" environment variable (default: 1) can be set to a
# positive number that extends test timeouts. The following runs tests with
# timeouts that are 20% longer than normal:
# make check-envtest-existing PGO_TEST_TIMEOUT_SCALE=1.2
.PHONY: check-envtest-existing
check-envtest-existing: ## Run check using envtest and an existing kube api
check-envtest-existing: get-pgmonitor
check-envtest-existing: createnamespaces
	kubectl apply --server-side -k ./config/dev
	USE_EXISTING_CLUSTER=true PGO_NAMESPACE="postgres-operator" QUERIES_CONFIG_DIR="$(CURDIR)/${QUERIES_CONFIG_DIR}" \
		$(GO_TEST) -count=1 -cover -p=1 ./...
	kubectl delete -k ./config/dev

# Expects operator to be running
#
# Chainsaw runs with a single kubectl context named "chainsaw".
# If you experience `cluster "minikube" does not exist`, try `MINIKUBE_PROFILE=chainsaw`.
#
# https://kyverno.github.io/chainsaw/latest/operations/script#kubeconfig
#
.PHONY: check-chainsaw
check-chainsaw:
	$(CHAINSAW_TEST) --config testing/chainsaw/e2e/config.yaml --values testing/chainsaw/e2e/values.yaml testing/chainsaw/e2e

# Expects operator to be running
#
# KUTTL runs with a single kubectl context named "cluster".
# If you experience `cluster "minikube" does not exist`, try `MINIKUBE_PROFILE=cluster`.
#
.PHONY: check-kuttl
check-kuttl: ## Run kuttl end-to-end tests
check-kuttl: ## example command: make check-kuttl KUTTL_TEST='
	${KUTTL_TEST} \
		--config testing/kuttl/kuttl-test.yaml

.PHONY: generate-kuttl
generate-kuttl: export KUTTL_PG_UPGRADE_FROM_VERSION ?= 16
generate-kuttl: export KUTTL_PG_UPGRADE_TO_VERSION ?= 17
generate-kuttl: export KUTTL_PG_VERSION ?= 16
generate-kuttl: export KUTTL_POSTGIS_VERSION ?= 3.4
generate-kuttl: export KUTTL_PSQL_IMAGE ?= registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi9-17.5-2520
generate-kuttl: export KUTTL_TEST_DELETE_NAMESPACE ?= kuttl-test-delete-namespace
generate-kuttl: ## Generate kuttl tests
	[ ! -d testing/kuttl/e2e-generated ] || rm -r testing/kuttl/e2e-generated
	bash -ceu ' \
	render() { envsubst '"'"' \
		$$KUTTL_PG_UPGRADE_FROM_VERSION $$KUTTL_PG_UPGRADE_TO_VERSION \
		$$KUTTL_PG_VERSION $$KUTTL_POSTGIS_VERSION $$KUTTL_PSQL_IMAGE \
		$$KUTTL_TEST_DELETE_NAMESPACE'"'"'; }; \
	while [ $$# -gt 0 ]; do \
		source="$${1}" target="$${1/e2e/e2e-generated}"; \
		mkdir -p "$${target%/*}"; render < "$${source}" > "$${target}"; \
		shift; \
	done' - testing/kuttl/e2e/*/*.yaml testing/kuttl/e2e/*/*/*.yaml

##@ Generate

.PHONY: check-generate
check-generate: ## Check everything generated is also committed
check-generate: generate
	git diff --exit-code -- config/crd
	git diff --exit-code -- config/rbac
	git diff --exit-code -- internal/collector
	git diff --exit-code -- pkg/apis

.PHONY: generate
generate: ## Generate everything
generate: generate-collector
generate: generate-crd
generate: generate-deepcopy
generate: generate-rbac

.PHONY: generate-crd
generate-crd: ## Generate Custom Resource Definitions (CRDs)
	$(CONTROLLER) \
		crd:crdVersions='v1' \
		paths='./pkg/apis/...' \
		output:dir='config/crd/bases' # {directory}/{group}_{plural}.yaml

.PHONY: generate-collector
generate-collector: ## Generate OTel Collector files
	$(GO) generate ./internal/collector

.PHONY: generate-deepcopy
generate-deepcopy: ## Generate DeepCopy functions
	$(CONTROLLER) \
		object:headerFile='hack/boilerplate.go.txt' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...'

.PHONY: generate-rbac
generate-rbac: ## Generate RBAC
	$(CONTROLLER) \
		rbac:roleName='postgres-operator' \
		paths='./cmd/...' paths='./internal/...' \
		output:dir='config/rbac' # {directory}/role.yaml
