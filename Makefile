
# Default values if not already set
ANSIBLE_VERSION ?= 2.9.*
PGOROOT ?= $(CURDIR)
PGO_BASEOS ?= centos8
BASE_IMAGE_OS ?= $(PGO_BASEOS)
PGO_IMAGE_PREFIX ?= crunchydata
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
PGO_VERSION ?= $(shell git describe --tags)
PGO_PG_VERSION ?= 13
PGO_PG_FULLVERSION ?= 13.4
PGO_BACKREST_VERSION ?= 2.33
PGO_KUBE_CLIENT ?= kubectl
PACKAGER ?= yum

RELTMPDIR=/tmp/release.$(PGO_VERSION)
RELFILE=/tmp/postgres-operator.$(PGO_VERSION).tar.gz

# Valid values: buildah (default), docker
IMGBUILDER ?= buildah
# Determines whether or not rootless builds are enabled
IMG_ROOTLESS_BUILD ?= false
# The utility to use when pushing/pulling to and from an image repo (e.g. docker or buildah)
IMG_PUSHER_PULLER ?= docker
# Determines whether or not images should be pushed to the local docker daemon when building with
# a tool other than docker (e.g. when building with buildah)
IMG_PUSH_TO_DOCKER_DAEMON ?= true
# Defines the sudo command that should be prepended to various build commands when rootless builds are
# not enabled
IMGCMDSUDO=
ifneq ("$(IMG_ROOTLESS_BUILD)", "true")
	IMGCMDSUDO=sudo --preserve-env
endif
IMGCMDSTEM=$(IMGCMDSUDO) buildah bud --layers $(SQUASH)
DFSET=$(PGO_BASEOS)

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
    DFSET=rhel
    DOCKERBASEREGISTRY=registry.access.redhat.com/
    PACKAGER=microdnf
endif
ifeq ("$(PGO_BASEOS)", "centos8")
    BASE_IMAGE_OS=centos8
    DFSET=centos
    DOCKERBASEREGISTRY=centos:
    PACKAGER=dnf
endif

DEBUG_BUILD ?= false
GO ?= go
GO_BUILD = $(GO_CMD) build -trimpath
GO_CMD = $(GO_ENV) $(GO)
GO_TEST ?= $(GO) test

# Disable optimizations if creating a debug build
ifeq ("$(DEBUG_BUILD)", "true")
	GO_BUILD = $(GO_CMD) build -gcflags='all=-N -l'
endif

# To build a specific image, run 'make <name>-image' (e.g. 'make postgres-operator-image')
images = postgres-operator \
	crunchy-postgres-exporter

.PHONY: all setup clean push pull release deploy


#======= Main functions =======
all: $(images:%=%-image)

setup:
	PGOROOT='$(PGOROOT)' ./bin/get-deps.sh
	./bin/check-deps.sh

#=== postgrescluster CRD ===

# Create operator and target namespaces
createnamespaces:
	$(PGO_KUBE_CLIENT) apply -k ./config/namespace

# Delete operator and target namespaces
deletenamespaces:
	$(PGO_KUBE_CLIENT) delete -k ./config/namespace

# Install the postgrescluster CRD
install:
	$(PGO_KUBE_CLIENT) apply -k ./config/crd

# Delete the postgrescluster CRD
uninstall:
	$(PGO_KUBE_CLIENT) delete -k ./config/crd

# Deploy the PostgreSQL Operator (enables the postgrescluster controller)
deploy:
	$(PGO_KUBE_CLIENT) apply -k ./config/default

# Deploy the PostgreSQL Operator locally
deploy-dev: build-postgres-operator
	$(PGO_KUBE_CLIENT) apply -k ./config/dev
	hack/create-kubeconfig.sh postgres-operator pgo
	env \
		CRUNCHY_DEBUG=true \
		KUBECONFIG=hack/.kube/postgres-operator/pgo \
		$(shell $(PGO_KUBE_CLIENT) kustomize ./config/dev | \
			sed -ne '/^kind: Deployment/,/^---/ { \
				/RELATED_IMAGE_/ { N; s,.*\(RELATED_[^[:space:]]*\).*value:[[:space:]]*\([^[:space:]]*\),\1="\2",; p; }; \
			}') \
		$(foreach v,$(filter RELATED_IMAGE_%,$(.VARIABLES)),$(v)="$($(v))") \
		bin/postgres-operator

# Undeploy the PostgreSQL Operator
undeploy:
	$(PGO_KUBE_CLIENT) delete -k ./config/default


#======= Binary builds =======
build-postgres-operator:
	$(GO_BUILD) -ldflags '-X "main.versionString=$(PGO_VERSION)"' \
		-o bin/postgres-operator ./cmd/postgres-operator

build-pgo-%:
	$(info No binary build needed for $@)

build-crunchy-postgres-exporter:
	$(info No binary build needed for $@)


#======= Image builds =======
$(PGOROOT)/build/%/Dockerfile:
	$(error No Dockerfile found for $* naming pattern: [$@])

%-img-build: pgo-base-$(IMGBUILDER) build-% $(PGOROOT)/build/%/Dockerfile
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/build/$*/Dockerfile \
		-t $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg BASEVER=$(PGO_VERSION) \
		--build-arg DFSET=$(DFSET) \
		--build-arg PACKAGER=$(PACKAGER) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg PREFIX=$(PGO_IMAGE_PREFIX) \
		$(PGOROOT)

%-img-buildah: %-img-build ;
# only push to docker daemon if variable PGO_PUSH_TO_DOCKER_DAEMON is set to "true"
ifeq ("$(IMG_PUSH_TO_DOCKER_DAEMON)", "true")
	$(IMGCMDSUDO) buildah push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)
endif

%-img-docker: %-img-build ;

%-image: %-img-$(IMGBUILDER) ;

pgo-base: pgo-base-$(IMGBUILDER)

pgo-base-build: $(PGOROOT)/build/pgo-base/Dockerfile licenses
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

pgo-base-buildah: pgo-base-build ;
# only push to docker daemon if variable PGO_PUSH_TO_DOCKER_DAEMON is set to "true"
ifeq ("$(IMG_PUSH_TO_DOCKER_DAEMON)", "true")
	$(IMGCMDSUDO) buildah push $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)
endif

pgo-base-docker: pgo-base-build


#======== Utility =======
.PHONY: check
check:
	$(GO_TEST) -cover ./...

# - KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT=true
.PHONY: check-envtest
check-envtest: hack/tools/envtest
	KUBEBUILDER_ASSETS="$(CURDIR)/$^/bin" $(GO_TEST) -count=1 -cover -tags=envtest ./...

.PHONY: check-envtest-existing
check-envtest-existing:
	${PGO_KUBE_CLIENT} apply -k ./config/dev
	USE_EXISTING_CLUSTER=true $(GO_TEST) -count=1 -cover -tags=envtest ./...
	${PGO_KUBE_CLIENT} delete -k ./config/dev


.PHONY: check-generate
check-generate: generate-crd generate-deepcopy generate-rbac
	git diff --exit-code -- config/crd
	git diff --exit-code -- config/rbac
	git diff --exit-code -- pkg/apis

clean: clean-deprecated
	rm -f bin/postgres-operator
	rm -f config/rbac/role.yaml
	[ ! -d hack/tools/envtest ] || rm -r hack/tools/envtest
	[ ! -n "$$(ls hack/tools)" ] || rm hack/tools/*
	[ ! -d hack/.kube ] || rm -r hack/.kube

clean-deprecated:
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

push: $(images:%=push-%) ;

push-%:
	$(IMG_PUSHER_PULLER) push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

pull: $(images:%=pull-%) ;

pull-%:
	$(IMG_PUSHER_PULLER) pull $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

generate: generate-crd generate-crd-docs generate-deepcopy generate-rbac

generate-crd:
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		crd:crdVersions='v1',preserveUnknownFields='false' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...' \
		output:dir='config/crd/bases' # config/crd/bases/{group}_{plural}.yaml

# TODO(cbandy): Run config/crd through Kustomize to pickup any patches there.
generate-crd-docs:
	GOBIN='$(CURDIR)/hack/tools' go install fybrik.io/crdoc@v0.4.0
	$(CURDIR)/hack/tools/crdoc \
		--resources $(CURDIR)/config/crd/bases \
		--output $(CURDIR)/docs/content/references/crd.md \
		--template $(CURDIR)/hack/api-template.tmpl

generate-deepcopy:
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		object:headerFile='hack/boilerplate.go.txt' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...'

generate-rbac:
	GOBIN='$(CURDIR)/hack/tools' ./hack/generate-rbac.sh \
		'./internal/...' 'config/rbac'

# Available versions: curl -s 'https://storage.googleapis.com/kubebuilder-tools/' | grep -o '<Key>[^<]*</Key>'
# - ENVTEST_K8S_VERSION=1.19.2
hack/tools/envtest: SHELL = bash
hack/tools/envtest:
	source '$(shell $(GO) list -f '{{ .Dir }}' -m 'sigs.k8s.io/controller-runtime')/hack/setup-envtest.sh' && fetch_envtest_tools $@

.PHONY: license licenses
license: licenses
licenses:
	./bin/license_aggregator.sh ./cmd/...
