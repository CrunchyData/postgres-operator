
# Environment definitions
RELTMPDIR=/tmp/release.$(PGO_VERSION)
RELFILE=/tmp/postgres-operator.$(PGO_VERSION).tar.gz
GOPATH ?= /home/odev/go
GOBIN ?= $(GOPATH)/bin

# Valid values: buildah (default), docker
IMGBUILDER ?= buildah
IMGCMDSTEM=sudo --preserve-env buildah bud --layers $(SQUASH)

# Allows simplification of IMGBUILDER switching
ifeq ("$(IMGBUILDER)","docker")
	IMGCMDSTEM=docker build
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
	postgres-operator

.PHONY: all installrbac setup setupnamespaces cleannamespaces bounce \
	deployoperator runmain runapiserver cli-docs clean deepsix push pull \
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

runmain: build-postgres-operator
	postgres-operator --kubeconfig=/etc/kubernetes/admin.conf

runapiserver: build-pgo-apiserver
	apiserver --kubeconfig=/etc/kubernetes/admin.conf


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
$(PGOROOT)/$(PGO_BASEOS)/Dockerfile.%.$(PGO_BASEOS):
	$(error No Dockerfile found for $* naming pattern: [$@])

%-img-build: pgo-base-$(IMGBUILDER) build-% $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.%.$(PGO_BASEOS)
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.$*.$(PGO_BASEOS) \
		-t $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) \
		--build-arg PREFIX=$(PGO_IMAGE_PREFIX) \
		--build-arg BASEVER=$(PGO_VERSION) \
		$(PGOROOT)

%-img-buildah: %-img-build
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

%-img-docker: %-img-build ;

%-image: %-img-$(IMGBUILDER) ;

pgo-base: pgo-base-$(IMGBUILDER)

pgo-base-build: $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-base.$(PGO_BASEOS)
	$(IMGCMDSTEM) \
		-f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-base.$(PGO_BASEOS) \
		-t $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) \
		--build-arg RELVER=$(PGO_VERSION) \
		$(PGOROOT)

pgo-base-buildah: pgo-base-build
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)

pgo-base-docker: pgo-base-build


#======== Utility =======
cli-docs:
	cd $(PGOROOT)/hugo/content/operatorcli/cli && go run $(PGOROOT)/pgo/generatedocs.go

clean:
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo

deepsix:
	cd $(PGOROOT)/apis/cr/v1
	deepcopy-gen --go-header-file=$(PGOROOT)/apis/cr/v1/header.go.txt --input-dirs=.

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
