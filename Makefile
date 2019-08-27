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
macpgo:	check-go-vars
	cd pgo && env GOOS=darwin GOARCH=amd64 go build pgo.go && mv pgo $(GOBIN)/pgo-mac
	env GOOS=darwin GOARCH=amd64 go build github.com/blang/expenv && mv expenv $(GOBIN)/expenv-mac
winpgo:	check-go-vars
	cd pgo && env GOOS=windows GOARCH=386 go build pgo.go && mv pgo.exe $(GOBIN)/pgo.exe
	env GOOS=windows GOARCH=386 go build github.com/blang/expenv && mv expenv.exe $(GOBIN)/expenv.exe

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
main:	check-go-vars
	go install postgres-operator.go
runmain:	check-go-vars
	postgres-operator --kubeconfig=/etc/kubernetes/admin.conf
runapiserver:	check-go-vars
	apiserver --kubeconfig=/etc/kubernetes/admin.conf

#======= Image builds =======
pgo-apiserver:	check-go-vars
	go install apiserver.go
pgo-backrest:	check-go-vars
	go install pgo-backrest/pgo-backrest.go
	mv $(GOBIN)/pgo-backrest ./bin/pgo-backrest/
pgo-backrest-image: check-go-vars pgo-backrest
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-backrest.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)

pgo-event-image: check-go-vars
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-event.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)

pgo-backrest-restore-image:	check-go-vars 
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-backrest-restore.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)
pgo-backrest-repo-image:	check-go-vars 
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-backrest-repo.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)
cli-docs:	check-go-vars
	cd $(PGOROOT)/hugo/content/operatorcli/cli && go run $(PGOROOT)/pgo/generatedocs.go
pgo:	check-go-vars
	cd pgo && go install pgo.go
clean:	check-go-vars
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo
pgo-apiserver-image:	check-go-vars pgo-apiserver
	cp $(GOBIN)/apiserver bin/
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-apiserver.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)

postgres-operator:	check-go-vars
	go install postgres-operator.go
postgres-operator-image:	check-go-vars postgres-operator
	cp $(GOBIN)/postgres-operator bin/postgres-operator/
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.postgres-operator.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG)
deepsix:
	cd $(PGOROOT)/apis/cr/v1
	deepcopy-gen --go-header-file=$(PGOROOT)/apis/cr/v1/header.go.txt --input-dirs=.
pgo-lspvc-image:
	sudo --preserve-env buildah bud $(SQUASH)  -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-lspvc.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)
pgo-load-image:
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-load.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)
pgo-rmdata:	check-go-vars
	go install pgo-rmdata/pgo-rmdata.go
pgo-rmdata-image: 	check-go-vars pgo-rmdata
	cp $(GOBIN)/pgo-rmdata bin/pgo-rmdata/
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-rmdata.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)
pgo-sqlrunner-image:
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-sqlrunner.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)
pgo-scheduler-image: check-go-vars pgo-scheduler
	go install pgo-scheduler/pgo-scheduler.go
	mv $(GOBIN)/pgo-scheduler ./bin/pgo-scheduler/
	sudo --preserve-env buildah bud $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-scheduler.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)

all:
	make postgres-operator-image
	make pgo-apiserver-image
	make pgo-event-image
	make pgo-scheduler-image
	make pgo-backrest-repo-image
	make pgo-backrest-restore-image
	make pgo
	make pgo-lspvc-image
	make pgo-load-image
	make pgo-rmdata-image
	make pgo-sqlrunner-image
	make pgo-backrest-image
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
	docker push $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)
	docker pull $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)
release:	check-go-vars
	make macpgo
	make winpgo
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

