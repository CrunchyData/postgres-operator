
# Default values if not already set
ANSIBLE_VERSION ?= 2.9.*
PGOROOT ?= $(CURDIR)
PGO_BASEOS ?= centos7
PGO_IMAGE_PREFIX ?= crunchydata
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
PGO_VERSION ?= 4.5.0
PGO_PG_VERSION ?= 12
PGO_PG_FULLVERSION ?= 12.4
PGO_BACKREST_VERSION ?= 2.29
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

DOCKERBASEREGISTRY=registry.access.redhat.com/

# Allows simplification of IMGBUILDER switching
ifeq ("$(IMGBUILDER)","docker")
        IMGCMDSTEM=docker build
endif

# Allows consolidation of ubi/rhel/centos Dockerfile sets
ifeq ("$(PGO_BASEOS)", "rhel7")
        DFSET=rhel
endif

ifeq ("$(PGO_BASEOS)", "ubi7")
        DFSET=rhel
endif

ifeq ("$(PGO_BASEOS)", "ubi8")
        DFSET=rhel
        PACKAGER=dnf
endif

ifeq ("$(PGO_BASEOS)", "centos7")
        DFSET=centos
        DOCKERBASEREGISTRY=centos:
endif

ifeq ("$(PGO_BASEOS)", "centos8")
        DFSET=centos
        PACKAGER=dnf
        DOCKERBASEREGISTRY=centos:
endif

DEBUG_BUILD ?= false
GO_BUILD = $(GO_CMD) build
GO_CMD = $(GO_ENV) go

# Disable optimizations if creating a debug build
ifeq ("$(DEBUG_BUILD)", "true")
	GO_BUILD += -gcflags='all=-N -l'
endif

# To build a specific image, run 'make <name>-image' (e.g. 'make pgo-apiserver-image')
images = pgo-apiserver \
	pgo-backrest \
	pgo-backrest-repo \
	pgo-rmdata \
	pgo-sqlrunner \
	pgo-client \
	pgo-deployer \
	crunchy-postgres-exporter \
	postgres-operator

.PHONY: all installrbac setup setupnamespaces cleannamespaces \
	deployoperator cli-docs clean push pull release


#======= Main functions =======
all: linuxpgo $(images:%=%-image)

installrbac:
	PGOROOT='$(PGOROOT)' ./deploy/install-rbac.sh

setup:
	PGOROOT='$(PGOROOT)' ./bin/get-deps.sh
	./bin/check-deps.sh

setupnamespaces:
	PGOROOT='$(PGOROOT)' ./deploy/setupnamespaces.sh

cleannamespaces:
	PGOROOT='$(PGOROOT)' ./deploy/cleannamespaces.sh

deployoperator:
	PGOROOT='$(PGOROOT)' ./deploy/deploy.sh


#======= Binary builds =======
build-pgo-apiserver:
	$(GO_BUILD) -o bin/apiserver ./cmd/apiserver

build-pgo-backrest:
	$(GO_BUILD) -o bin/pgo-backrest/pgo-backrest ./cmd/pgo-backrest

build-pgo-rmdata:
	$(GO_BUILD) -o bin/pgo-rmdata/pgo-rmdata ./cmd/pgo-rmdata

build-postgres-operator:
	$(GO_BUILD) -o bin/postgres-operator ./cmd/postgres-operator

build-pgo-client:
	$(GO_BUILD) -o bin/pgo ./cmd/pgo

build-pgo-%:
	$(info No binary build needed for $@)

build-crunchy-postgres-exporter:
	$(info No binary build needed for $@)

linuxpgo: GO_ENV += GOOS=linux GOARCH=amd64
linuxpgo:
	$(GO_BUILD) -o bin/pgo ./cmd/pgo

macpgo: GO_ENV += GOOS=darwin GOARCH=amd64
macpgo:
	$(GO_BUILD) -o bin/pgo-mac ./cmd/pgo

winpgo: GO_ENV += GOOS=windows GOARCH=386
winpgo:
	$(GO_BUILD) -o bin/pgo.exe ./cmd/pgo


#======= Image builds =======
$(PGOROOT)/build/%/Dockerfile:
	$(error No Dockerfile found for $* naming pattern: [$@])

%-img-build: pgo-base-$(IMGBUILDER) build-% $(PGOROOT)/build/%/Dockerfile
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/build/$*/Dockerfile \
		-t $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg BASEVER=$(PGO_VERSION) \
		--build-arg PREFIX=$(PGO_IMAGE_PREFIX) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg BACKREST_VERSION=$(PGO_BACKREST_VERSION) \
		--build-arg ANSIBLE_VERSION=$(ANSIBLE_VERSION) \
		--build-arg DFSET=$(DFSET) \
		--build-arg PACKAGER=$(PACKAGER) \
		$(PGOROOT)

%-img-buildah: %-img-build ;
# only push to docker daemon if variable PGO_PUSH_TO_DOCKER_DAEMON is set to "true"
ifeq ("$(IMG_PUSH_TO_DOCKER_DAEMON)", "true")
	$(IMGCMDSUDO) buildah push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)
endif

%-img-docker: %-img-build ;

%-image: %-img-$(IMGBUILDER) ;

pgo-base: pgo-base-$(IMGBUILDER)

pgo-base-build: $(PGOROOT)/build/pgo-base/Dockerfile
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/build/pgo-base/Dockerfile \
		-t $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg RELVER=$(PGO_VERSION) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg PG_FULL=$(PGO_PG_FULLVERSION) \
		--build-arg DFSET=$(DFSET) \
		--build-arg PACKAGER=$(PACKAGER) \
		--build-arg DOCKERBASEREGISTRY=$(DOCKERBASEREGISTRY) \
		$(PGOROOT)

pgo-base-buildah: pgo-base-build ;
# only push to docker daemon if variable PGO_PUSH_TO_DOCKER_DAEMON is set to "true"
ifeq ("$(IMG_PUSH_TO_DOCKER_DAEMON)", "true")
	$(IMGCMDSUDO) buildah push $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)
endif

pgo-base-docker: pgo-base-build


#======== Utility =======
cli-docs:
	rm docs/content/pgo-client/reference/*.md
	cd docs/content/pgo-client/reference && go run ../../../../cmd/pgo/generatedocs.go
	sed -e '1,5 s|^title:.*|title: "pgo Client Reference"|' \
		docs/content/pgo-client/reference/pgo.md > \
		docs/content/pgo-client/reference/_index.md
	rm docs/content/pgo-client/reference/pgo.md

clean: clean-deprecated
	rm -f bin/apiserver
	rm -f bin/postgres-operator
	rm -f bin/pgo bin/pgo-mac bin/pgo.exe
	rm -f bin/pgo-backrest/pgo-backrest
	rm -f bin/pgo-rmdata/pgo-rmdata
	[ -z "$$(ls hack/tools)" ] || rm hack/tools/*

clean-deprecated:
	@# packages used to be downloaded into the vendor directory
	[ ! -d vendor ] || rm -r vendor
	@# executables used to be compiled into the $GOBIN directory
	[ ! -n '$(GOBIN)' ] || rm -f $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo
	[ ! -d bin/postgres-operator ] || rm -r bin/postgres-operator

push: $(images:%=push-%) ;

push-%:
	$(IMG_PUSHER_PULLER) push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

pull: $(images:%=pull-%) ;

pull-%:
	$(IMG_PUSHER_PULLER) pull $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

release:  linuxpgo macpgo winpgo
	rm -rf $(RELTMPDIR) $(RELFILE)
	mkdir $(RELTMPDIR)
	cp -r $(PGOROOT)/examples $(RELTMPDIR)
	cp -r $(PGOROOT)/deploy $(RELTMPDIR)
	cp -r $(PGOROOT)/conf $(RELTMPDIR)
	cp bin/pgo $(RELTMPDIR)
	cp bin/pgo-mac $(RELTMPDIR)
	cp bin/pgo.exe $(RELTMPDIR)
	cp $(PGOROOT)/examples/pgo-bash-completion $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .

generate: generate-crd generate-deepcopy
	GOBIN='$(CURDIR)/hack/tools' ./hack/update-codegen.sh

generate-crd:
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		crd:crdVersions='v1',preserveUnknownFields='false' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...' \
		output:dir='config/crd/bases' # config/crd/bases/{group}_{plural}.yaml

generate-deepcopy:
	GOBIN='$(CURDIR)/hack/tools' ./hack/controller-generator.sh \
		object:headerFile='hack/boilerplate.go.txt' \
		paths='./pkg/apis/postgres-operator.crunchydata.com/...'
