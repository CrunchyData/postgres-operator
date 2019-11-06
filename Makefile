RELTMPDIR=/tmp/release.$(PGO_VERSION)
RELFILE=/tmp/postgres-operator.$(PGO_VERSION).tar.gz

#======= Safety checks =======
check-go-vars:
ifndef GOPATH
	$(error GOPATH is not set)
endif
ifndef GOBIN
	$(error GOBIN is not set)
endif

#======= Main functions =======
installrbac:
	cd deploy && ./install-rbac.sh

setup:
	./bin/get-deps.sh

setupnamespaces:
	cd deploy && ./setupnamespaces.sh

cleannamespaces:
	cd deploy && ./cleannamespaces.sh

bounce:
	$(PGO_CMD) --namespace=$(PGO_OPERATOR_NAMESPACE) get pod --selector=name=postgres-operator -o=jsonpath="{.items[0].metadata.name}" | xargs $(PGO_CMD) --namespace=$(PGO_OPERATOR_NAMESPACE) delete pod

deployoperator:
	cd deploy && ./deploy.sh

main: check-go-vars
	go install postgres-operator.go

runmain: check-go-vars
	postgres-operator --kubeconfig=/etc/kubernetes/admin.conf

runapiserver: check-go-vars
	apiserver --kubeconfig=/etc/kubernetes/admin.conf

#======= Binary builds =======
build-pgo-apiserver: check-go-vars
	go install apiserver.go
	cp $(GOBIN)/apiserver bin/

build-pgo-backrest: check-go-vars
	go install pgo-backrest/pgo-backrest.go
	mv $(GOBIN)/pgo-backrest ./bin/pgo-backrest/

build-pgo-rmdata:	check-go-vars
	go install pgo-rmdata/pgo-rmdata.go
	cp $(GOBIN)/pgo-rmdata bin/pgo-rmdata/

build-pgo-scheduler: check-go-vars
	go install pgo-scheduler/pgo-scheduler.go
	mv $(GOBIN)/pgo-scheduler ./bin/pgo-scheduler/

build-postgres-operator: check-go-vars
	go install postgres-operator.go
	cp $(GOBIN)/postgres-operator bin/postgres-operator/

build-pgo-%:
	$(info No binary build needed for $@)

linuxpgo: check-go-vars
	cd pgo && go install pgo.go

macpgo:	check-go-vars
	cd pgo && env GOOS=darwin GOARCH=amd64 go build pgo.go && mv pgo $(GOBIN)/pgo-mac
	env GOOS=darwin GOARCH=amd64 go build github.com/blang/expenv && mv expenv $(GOBIN)/expenv-mac

winpgo:	check-go-vars
	cd pgo && env GOOS=windows GOARCH=386 go build pgo.go && mv pgo.exe $(GOBIN)/pgo.exe
	env GOOS=windows GOARCH=386 go build github.com/blang/expenv && mv expenv.exe $(GOBIN)/expenv.exe


#======= Image builds =======
%-image: check-go-vars build-%
	sudo --preserve-env buildah bud --layers $(SQUASH) \
		-f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.$*.$(PGO_BASEOS) \
		-t $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) \
		--build-arg PREFIX=$(PGO_IMAGE_PREFIX) \
		--build-arg BASEVER=$(PGO_VERSION) \
		$(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

pgo-base: check-go-vars
	sudo --preserve-env buildah bud --layers $(SQUASH) \
		-f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-base.$(PGO_BASEOS) \
		-t $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) \
		--build-arg RELVER=$(PGO_VERSION) \
		$(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG)

cli-docs:	check-go-vars
	cd $(PGOROOT)/hugo/content/operatorcli/cli && go run $(PGOROOT)/pgo/generatedocs.go

clean:	check-go-vars
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo

deepsix:
	cd $(PGOROOT)/apis/cr/v1
	deepcopy-gen --go-header-file=$(PGOROOT)/apis/cr/v1/header.go.txt --input-dirs=.

all: linuxpgo \
	pgo-base \
	postgres-operator-image \
	pgo-apiserver-image \
	pgo-event-image \
	pgo-scheduler-image \
	pgo-backrest-repo-image \
	pgo-backrest-restore-image \
	pgo-lspvc-image \
	pgo-load-image \
	pgo-rmdata-image \
	pgo-sqlrunner-image \
	pgo-backrest-image

push:
	docker push $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)
	docker push $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)

pull:
	docker pull $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)

release: check-go-vars macpgo winpgo
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

default:
	all

