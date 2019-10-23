RELTMPDIR=/tmp/release.$(PGO_VERSION)
RELFILE=/tmp/postgres-operator.$(PGO_VERSION).tar.gz

.PHONY: help check-go-vars installrbac setupnamespaces cleannamespaces bounce macpgo winpgo setup deployoperator \
main runmain runapiserver pgo-apiserver pgo-backrest pgo postgres-operator pgo-rmdata clean cli-docs \
pgo-backrest-image pgo-event-image pgo-backrest-restore-image pgo-backrest-repo-image pgo-apiserver-image \
postgres-operator-image pgo-lspvc-image pgo-load-image pgo-rmdata-image pgo-sqlrunner-image pgo-scheduler-image \
all push pull release deepsix default


.DEFAULT: help

# The help function searches the Makefile for targets that have a '##' after then prints the result
help: ## Prints the help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


#======= Safety checks =======
check-go-vars:
ifndef GOPATH
	$(error GOPATH is not set)
endif
ifndef GOBIN
	$(error GOBIN is not set)
endif

#======= Kubernetes =======
installrbac: ## Installs kubernetes rbac policies that are required for the operator pod to run
	cd deploy && ./install-rbac.sh

setupnamespaces: ## Creates the kubernetes namespace(s) as defined by $PGO_OPERATOR_NAMESPACE
	cd deploy && ./setupnamespaces.sh

cleannamespaces: ## Deletes the kubernetes namespace(s) as defined by $PGO_OPERATOR_NAMESPACE
	cd deploy && ./cleannamespaces.sh

bounce: ## Restarts the postgres-operator pods
	$(PGO_CMD) --namespace=$(PGO_OPERATOR_NAMESPACE) get pod --selector=name=postgres-operator -o=jsonpath="{.items[0].metadata.name}" | xargs $(PGO_CMD) --namespace=$(PGO_OPERATOR_NAMESPACE) delete pod


#======= Platform specific =======
macpgo:	check-go-vars ## Builds the pgo cli binary for the mac(darwin) platform
	cd pgo && env GOOS=darwin GOARCH=amd64 go build pgo.go && mv pgo $(GOBIN)/pgo-mac
	env GOOS=darwin GOARCH=amd64 go build github.com/blang/expen && mv expenv $(GOBIN)/expenv-mac

winpgo:	check-go-vars ## Builds the the pgo binary for the windows platform
	cd pgo && env GOOS=windows GOARCH=386 go build pgo.go && mv pgo.exe $(GOBIN)/pgo.exe
	env GOOS=windows GOARCH=386 go build github.com/blang/expenv && mv expenv.exe $(GOBIN)/expenv.exe


#====== golang =======
setup: ## Install the golang dependencies
	./bin/get-deps.sh

deployoperator: ## Deploys the operator into the kubernetes namespace(s) as defined by $PGO_OPERATOR_NAMESPACE
	cd deploy && ./deploy.sh

main:	check-go-vars
	go install postgres-operator.go

runmain:	check-go-vars
	postgres-operator --kubeconfig=/etc/kubernetes/admin.conf

runapiserver:	check-go-vars
	apiserver --kubeconfig=/etc/kubernetes/admin.conf

pgo-apiserver:	check-go-vars
	go install apiserver.go

pgo-backrest:	check-go-vars
	go install pgo-backrest/pgo-backrest.go
	mv $(GOBIN)/pgo-backrest ./bin/pgo-backrest/

pgo:	check-go-vars
	cd pgo && go install pgo.go

postgres-operator:	check-go-vars
	go install postgres-operator.go

pgo-rmdata:	check-go-vars
	go install pgo-rmdata/pgo-rmdata.go

clean:	check-go-vars
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo


#======= Documentation =======
cli-docs:	check-go-vars
	cd $(PGOROOT)/hugo/content/operatorcli/cli && go run $(PGOROOT)/pgo/generatedocs.go


#======= Images =======
pgo-backrest-image: check-go-vars pgo-backrest
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-backrest.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-backrest:$(PGO_IMAGE_TAG)

clean-pgo-backrest-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-backrest | xargs --no-run-if-empty sudo buildah rmi

pgo-event-image: check-go-vars
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-event.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-event:$(PGO_IMAGE_TAG)

clean-pgo-event-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-event | xargs --no-run-if-empty sudo buildah rmi

pgo-backrest-restore-image: check-go-vars
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-backrest-restore.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-backrest-restore:$(PGO_IMAGE_TAG)

clean-pgo-backrest-restore-image: check-go-vars ## Remove the local container image for backrest restores
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-backrest-restore | xargs --no-run-if-empty sudo buildah rmi

pgo-backrest-repo-image: check-go-vars 
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-backrest-repo.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-backrest-repo:$(PGO_IMAGE_TAG)

clean-pgo-backrest-repo-image: check-go-vars 
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-backrest-repo | xargs --no-run-if-empty sudo buildah rmi

pgo-apiserver-image: check-go-vars pgo-apiserver
	cp $(GOBIN)/apiserver bin/
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-apiserver.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)  $(PGO_IMAGE_PREFIX)/pgo-apiserver:$(PGO_IMAGE_TAG)

clean-pgo-apiserver-image: check-go-vars pgo-apiserver
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-apiserver | xargs --no-run-if-empty sudo buildah rmi

postgres-operator-image: check-go-vars postgres-operator
	cp $(GOBIN)/postgres-operator bin/postgres-operator/
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.postgres-operator.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/postgres-operator:$(PGO_IMAGE_TAG)

clean-postgres-operator-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/postgres-operator | xargs --no-run-if-empty sudo buildah rmi

pgo-lspvc-image: check-go-vars
	sudo --preserve-env buildah bud --layers $(SQUASH)  -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-lspvc.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-lspvc:$(PGO_IMAGE_TAG)

clean-pgo-lspvc-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-lspvc | xargs --no-run-if-empty sudo buildah rmi

pgo-load-image: check-go-vars
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-load.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-load:$(PGO_IMAGE_TAG)

clean-pgo-load-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-lspvc | xargs --no-run-if-empty sudo buildah rmi

pgo-rmdata-image: check-go-vars pgo-rmdata
	cp $(GOBIN)/pgo-rmdata bin/pgo-rmdata/
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-rmdata.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-rmdata:$(PGO_IMAGE_TAG)

clean-pgo-rmdata-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-rmdata | xargs --no-run-if-empty sudo buildah rmi

pgo-sqlrunner-image: check-go-vars
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-sqlrunner.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-sqlrunner:$(PGO_IMAGE_TAG)

clean-pgo-sqlrunner-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-sqlrunner | xargs --no-run-if-empty sudo buildah rmi

pgo-scheduler-image: check-go-vars pgo-scheduler
	go install pgo-scheduler/pgo-scheduler.go
	mv $(GOBIN)/pgo-scheduler ./bin/pgo-scheduler/
	sudo --preserve-env buildah bud --layers $(SQUASH) -f $(PGOROOT)/$(PGO_BASEOS)/Dockerfile.pgo-scheduler.$(PGO_BASEOS) -t $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG) $(PGOROOT)
	sudo --preserve-env buildah push $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG) docker-daemon:$(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)
	docker tag docker.io/$(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG) $(PGO_IMAGE_PREFIX)/pgo-scheduler:$(PGO_IMAGE_TAG)

clean-pgo-scheduler-image: check-go-vars
	sudo buildah images -q localhost/$(PGO_IMAGE_PREFIX)/pgo-scheduler | xargs --no-run-if-empty sudo buildah rmi

clean-images: clean-pgo-backrest-image clean-pgo-event-image clean-pgo-backrest-restore-image clean-pgo-backrest-repo-image ## Clean all images
clean-images: clean-pgo-apiserver-image clean-postgres-operator-image clean-pgo-lspvc-image clean-pgo-load-image clean-pgo-rmdata-image \
clean-pgo-sqlrunner-image clean-pgo-scheduler-image

all: postgres-operator-image pgo-apiserver-image ## Builds all images (does not build cli binaries)
all: pgo-event-image pgo-scheduler-image pgo-backrest-repo-image pgo-backrest-restore-image pgo pgo-lspvc-image pgo-load-image pgo-rmdata-image pgo-sqlrunner-image pgo-backrest-image

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

pull:  ## Pulls all the docker images needed to run the operator (does not include cli binaries)
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

release:	check-go-vars ## Creates package for release onto github
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


#======= Misc =======
deepsix:
	cd $(PGOROOT)/apis/cr/v1
	deepcopy-gen --go-header-file=$(PGOROOT)/apis/cr/v1/header.go.txt --input-dirs=.


default: help
