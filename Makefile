GOPATH ?= /home/odev/go
GOBIN ?= $(GOPATH)/bin

# Default values if not already set
PGOROOT ?= $(GOPATH)/src/github.com/crunchydata/postgres-operator
PGO_BASEOS ?= centos7
PGO_CMD ?= kubectl
PGO_IMAGE_PREFIX ?= crunchydata
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
PGO_OPERATOR_NAMESPACE ?= pgo
PGO_VERSION ?= 4.3.0
PGO_PG_VERSION ?= 12
PGO_PG_FULLVERSION ?= 12.2
PGO_BACKREST_VERSION ?= 2.24

RELTMPDIR=/tmp/release.$(PGO_VERSION)
RELFILE=/tmp/postgres-operator.$(PGO_VERSION).tar.gz

# Valid values: buildah (default), docker
IMGBUILDER ?= buildah
IMGCMDSTEM=sudo --preserve-env buildah bud --layers $(SQUASH)
DFSET=$(PGO_BASEOS)

# Allows simplification of IMGBUILDER switching
ifeq ("$(IMGBUILDER)","docker")
	IMGCMDSTEM=docker build
endif

# Allows consolidation of ubi/rhel Dockerfile sets
ifeq ("$(PGO_BASEOS)", "ubi7")
	DFSET=rhel7
endif

# To build a specific image, run 'make <name>-image' (e.g. 'make pgo-apiserver-image')
images = pgo-apiserver \
	pgo-backrest \
	pgo-backrest-repo \
	pgo-backrest-repo-sync \
	pgo-backrest-restore \
	pgo-event \
	pgo-load \
	pgo-rmdata \
	pgo-scheduler \
	pgo-sqlrunner \
	pgo-client \
	pgo-deployer \
	postgres-operator

.PHONY: all installrbac setup setupnamespaces cleannamespaces bounce \
	deployoperator runmain runapiserver cli-docs clean push pull \
	release default


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
build-pgo-apiserver:
	go install apiserver.go
	cp $(GOBIN)/apiserver bin/

build-pgo-backrest:
	go install pgo-backrest/pgo-backrest.go
	cp $(GOBIN)/pgo-backrest bin/pgo-backrest/

build-pgo-rmdata:
	go install pgo-rmdata/pgo-rmdata.go
	cp $(GOBIN)/pgo-rmdata bin/pgo-rmdata/

build-pgo-scheduler:
	go install pgo-scheduler/pgo-scheduler.go
	cp $(GOBIN)/pgo-scheduler bin/pgo-scheduler/

build-postgres-operator:
	go install postgres-operator.go
	cp $(GOBIN)/postgres-operator bin/postgres-operator/

build-pgo-client:
	go install pgo/pgo.go
	cp $(GOBIN)/pgo bin/pgo

build-pgo-%:
	$(info No binary build needed for $@)

linuxpgo: build-pgo-client

macpgo:
	cd pgo && env GOOS=darwin GOARCH=amd64 go build pgo.go && mv pgo $(GOBIN)/pgo-mac
	env GOOS=darwin GOARCH=amd64 go build github.com/blang/expenv && mv expenv $(GOBIN)/expenv-mac

winpgo:
	cd pgo && env GOOS=windows GOARCH=386 go build pgo.go && mv pgo.exe $(GOBIN)/pgo.exe
	env GOOS=windows GOARCH=386 go build github.com/blang/expenv && mv expenv.exe $(GOBIN)/expenv.exe


#======= Image builds =======
$(PGOROOT)/$(DFSET)/Dockerfile.%.$(DFSET):
	$(error No Dockerfile found for $* naming pattern: [$@])

%-img-build: pgo-base-$(IMGBUILDER) build-% $(PGOROOT)/$(DFSET)/Dockerfile.%.$(DFSET)
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/$(DFSET)/Dockerfile.$*.$(DFSET) \
		-t $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg BASEVER=$(PGO_VERSION) \
		--build-arg PREFIX=$(PGO_IMAGE_PREFIX) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg BACKREST_VERSION=$(PGO_BACKREST_VERSION) \
		$(PGOROOT)

%-img-buildah: %-img-build
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

%-img-docker: %-img-build ;

%-image: %-img-$(IMGBUILDER) ;

pgo-base: pgo-base-$(IMGBUILDER)

pgo-base-build: $(PGOROOT)/$(DFSET)/Dockerfile.pgo-base.$(DFSET)
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/$(DFSET)/Dockerfile.pgo-base.$(DFSET) \
		-t $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) \
		--build-arg BASEOS=$(PGO_BASEOS) \
		--build-arg RELVER=$(PGO_VERSION) \
		--build-arg PGVERSION=$(PGO_PG_VERSION) \
		--build-arg PG_FULL=$(PGO_PG_FULLVERSION) \
		$(PGOROOT)

pgo-base-buildah: pgo-base-build
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)

pgo-base-docker: pgo-base-build


#======== Utility =======
cli-docs:
	cd $(PGOROOT)/docs/content/operatorcli/cli && go run $(PGOROOT)/pgo/generatedocs.go

clean:
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo

push: $(images:%=push-%) ;

push-%:
	docker push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

pull: $(images:%=pull-%) ;

pull-%:
	docker pull $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

release:  linuxpgo macpgo winpgo
	rm -rf $(RELTMPDIR) $(RELFILE)
	mkdir $(RELTMPDIR)
	cp -r $(PGOROOT)/examples $(RELTMPDIR)
	cp -r $(PGOROOT)/deploy $(RELTMPDIR)
	cp -r $(PGOROOT)/conf $(RELTMPDIR)
	cp $(GOBIN)/pgo $(RELTMPDIR)
	cp $(GOBIN)/pgo-mac $(RELTMPDIR)
	cp $(GOBIN)/pgo.exe $(RELTMPDIR)
	cp $(GOBIN)/expenv $(RELTMPDIR)
	cp $(GOBIN)/expenv-mac $(RELTMPDIR)
	cp $(GOBIN)/expenv.exe $(RELTMPDIR)
	cp $(PGOROOT)/examples/pgo-bash-completion $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .

update-codegen:
	$(PGOROOT)/hack/update-codegen.sh

verify-codegen:
	$(PGOROOT)/hack/verify-codegen.sh
