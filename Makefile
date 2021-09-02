GOPATH ?= $(HOME)/odev/go
GOBIN ?= $(GOPATH)/bin

# Default values if not already set
ANSIBLE_VERSION ?= 2.9.*
PGOROOT ?= $(GOPATH)/src/github.com/crunchydata/postgres-operator
PGO_BASEOS ?= centos8
PGO_CMD ?= kubectl
PGO_IMAGE_PREFIX ?= crunchydata
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
PGO_OPERATOR_NAMESPACE ?= pgo
PGO_VERSION ?= 4.5.4
PGO_PG_VERSION ?= 12
PGO_PG_FULLVERSION ?= 12.8
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
GCFLAGS=
# Disable optimizations if creating a debug build
ifeq ("$(DEBUG_BUILD)", "true")
	GCFLAGS=all=-N -l
endif

# To build a specific image, run 'make <name>-image' (e.g. 'make pgo-apiserver-image')
images = pgo-apiserver \
	pgo-backrest \
	pgo-backrest-repo \
	pgo-backrest-repo-sync \
	pgo-backrest-restore \
	pgo-event \
	pgo-rmdata \
	pgo-scheduler \
	pgo-sqlrunner \
	pgo-client \
	pgo-deployer \
	crunchy-postgres-exporter \
	postgres-operator

.PHONY: all installrbac setup setupnamespaces cleannamespaces bounce \
	deployoperator cli-docs clean push pull \
	release default license


#======= Main functions =======
all: linuxpgo $(images:%=%-image)

installrbac:
	cd deploy && ./install-rbac.sh

setup:
	./bin/get-deps.sh

setupnamespaces:
	cd deploy && ./setupnamespaces.sh

cleannamespaces:
	cd deploy && ./cleannamespaces.sh

bounce:
	$(PGO_CMD) \
		--namespace=$(PGO_OPERATOR_NAMESPACE) \
		get pod \
		--selector=name=postgres-operator \
		-o=jsonpath="{.items[0].metadata.name}" \
	| xargs $(PGO_CMD) --namespace=$(PGO_OPERATOR_NAMESPACE) delete pod

deployoperator:
	cd deploy && ./deploy.sh


#======= Binary builds =======
build: build-postgres-operator build-pgo-apiserver build-pgo-client build-pgo-rmdata build-pgo-scheduler license

build-pgo-apiserver:
	go install -gcflags='$(GCFLAGS)' apiserver.go
	cp $(GOBIN)/apiserver bin/

build-pgo-backrest:
	go install -gcflags='$(GCFLAGS)' pgo-backrest/pgo-backrest.go
	cp $(GOBIN)/pgo-backrest bin/pgo-backrest/

build-pgo-rmdata:
	go install -gcflags='$(GCFLAGS)' pgo-rmdata/pgo-rmdata.go
	cp $(GOBIN)/pgo-rmdata bin/pgo-rmdata/

build-pgo-scheduler:
	go install -gcflags='$(GCFLAGS)' pgo-scheduler/pgo-scheduler.go
	cp $(GOBIN)/pgo-scheduler bin/pgo-scheduler/

build-postgres-operator:
	go install -gcflags='$(GCFLAGS)' postgres-operator.go
	cp $(GOBIN)/postgres-operator bin/postgres-operator/

build-pgo-client:
	go install -gcflags='$(GCFLAGS)' pgo/pgo.go
	cp $(GOBIN)/pgo bin/pgo

build-pgo-%:
	$(info No binary build needed for $@)

build-crunchy-postgres-exporter:
	$(info No binary build needed for $@)

linuxpgo: build-pgo-client

macpgo:
	cd pgo && env GOOS=darwin GOARCH=amd64 go build pgo.go && mv pgo $(GOBIN)/pgo-mac

winpgo:
	cd pgo && env GOOS=windows GOARCH=386 go build pgo.go && mv pgo.exe $(GOBIN)/pgo.exe


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

pgo-base-build: build $(PGOROOT)/build/pgo-base/Dockerfile
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
	cd $(PGOROOT)/docs/content/operatorcli/cli && go run $(PGOROOT)/pgo/generatedocs.go

clean:
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo

license:
	./bin/license_aggregator.sh

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
	cp $(GOBIN)/pgo $(RELTMPDIR)
	cp $(GOBIN)/pgo-mac $(RELTMPDIR)
	cp $(GOBIN)/pgo.exe $(RELTMPDIR)
	cp $(PGOROOT)/examples/pgo-bash-completion $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .

update-codegen:
	$(PGOROOT)/hack/update-codegen.sh

verify-codegen:
	$(PGOROOT)/hack/verify-codegen.sh
