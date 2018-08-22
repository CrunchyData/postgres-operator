RELTMPDIR=/tmp/release.$(CO_VERSION)
RELFILE=/tmp/postgres-operator.$(CO_VERSION).tar.gz

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


gendeps: 
	godep save \
	github.com/crunchydata/postgres-operator/apis/cr/v1 \
	github.com/crunchydata/postgres-operator/util \
	github.com/crunchydata/postgres-operator/operator \
	github.com/crunchydata/postgres-operator/operator/backup \
	github.com/crunchydata/postgres-operator/operator/cluster \
	github.com/crunchydata/postgres-operator/operator/pvc \
	github.com/crunchydata/postgres-operator/controller \
	github.com/crunchydata/postgres-operator/client \
	github.com/crunchydata/postgres-operator/pgo/cmd \
	github.com/crunchydata/postgres-operator/apiservermsgs \
	github.com/crunchydata/postgres-operator/apiserver \
	github.com/crunchydata/postgres-operator/apiserver/backupservice \
	github.com/crunchydata/postgres-operator/apiserver/cloneservice \
	github.com/crunchydata/postgres-operator/apiserver/clusterservice \
	github.com/crunchydata/postgres-operator/apiserver/labelservice \
	github.com/crunchydata/postgres-operator/apiserver/loadservice \
	github.com/crunchydata/postgres-operator/apiserver/policyservice \
	github.com/crunchydata/postgres-operator/apiserver/pvcservice \
	github.com/crunchydata/postgres-operator/apiserver/upgradeservice \
	github.com/crunchydata/postgres-operator/apiserver/userservice \
	github.com/crunchydata/postgres-operator/apiserver/util \
	github.com/crunchydata/postgres-operator/apiserver/versionservice 
installrbac:
	cd deploy && ./install-rbac.sh
setup:
	./bin/get-deps.sh
	cd examples/backrest-config && ./create.sh
setupnamespace:
	kubectl create -f ./examples/demo-namespace.json
	kubectl config set-context demo --cluster=kubernetes --namespace=demo --user=kubernetes-admin
	kubectl config use-context demo
bounce:
	kubectl get pod --selector=name=postgres-operator -o=jsonpath="{.items[0].metadata.name}" | xargs kubectl delete pod
deployoperator:
	cd deploy && ./deploy.sh
main:	check-go-vars
	go install postgres-operator.go
runmain:	check-go-vars
	postgres-operator --kubeconfig=/etc/kubernetes/admin.conf
runapiserver:	check-go-vars
	apiserver --kubeconfig=/etc/kubernetes/admin.conf
apiserver:	check-go-vars
	go install apiserver.go
pgo-backrest:	check-go-vars
	go install pgo-backrest/pgo-backrest.go
	mv $(GOBIN)/pgo-backrest ./bin/pgo-backrest/
pgo-backrest-image:	check-go-vars pgo-backrest
	docker build -t pgo-backrest -f $(CO_BASEOS)/Dockerfile.pgo-backrest.$(CO_BASEOS) .
	docker tag pgo-backrest $(CO_IMAGE_PREFIX)/pgo-backrest:$(CO_IMAGE_TAG)
pgo:	check-go-vars
	cd pgo && go install pgo.go
clean:	check-go-vars
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo
apiserverimage:	check-go-vars
	go install apiserver.go
	cp $(GOBIN)/apiserver bin/
	docker build -t pgo-apiserver -f $(CO_BASEOS)/Dockerfile.pgo-apiserver.$(CO_BASEOS) .
	docker tag pgo-apiserver $(CO_IMAGE_PREFIX)/pgo-apiserver:$(CO_IMAGE_TAG)
#	docker push $(CO_IMAGE_PREFIX)/pgo-apiserver:$(CO_IMAGE_TAG)
operator:	check-go-vars
	go install postgres-operator.go
operatorimage:	check-go-vars
	go install postgres-operator.go
	cp $(GOBIN)/postgres-operator bin/postgres-operator/
	docker build -t postgres-operator -f $(CO_BASEOS)/Dockerfile.postgres-operator.$(CO_BASEOS) .
	docker tag postgres-operator $(CO_IMAGE_PREFIX)/postgres-operator:$(CO_IMAGE_TAG)
#	docker push $(CO_IMAGE_PREFIX)/postgres-operator:$(CO_IMAGE_TAG)
deepsix:
	cd $(COROOT)/apis/cr/v1
	deepcopy-gen --go-header-file=$(COROOT)/apis/cr/v1/header.go.txt --input-dirs=.
lsimage:
	docker build -t pgo-lspvc -f $(CO_BASEOS)/Dockerfile.pgo-lspvc.$(CO_BASEOS) .
	docker tag pgo-lspvc $(CO_IMAGE_PREFIX)/pgo-lspvc:$(CO_IMAGE_TAG)
loadimage:
	docker build -t pgo-load -f $(CO_BASEOS)/Dockerfile.pgo-load.$(CO_BASEOS) .
	docker tag pgo-load $(CO_IMAGE_PREFIX)/pgo-load:$(CO_IMAGE_TAG)
rmdataimage:
	docker build -t pgo-rmdata -f $(CO_BASEOS)/Dockerfile.pgo-rmdata.$(CO_BASEOS) .
	docker tag pgo-rmdata $(CO_IMAGE_PREFIX)/pgo-rmdata:$(CO_IMAGE_TAG)
all:
	make operatorimage
	make apiserverimage
	make lsimage
	make loadimage
	make pgo
	make rmdataimage
	make pgo-backrest-image
push:
	docker push $(CO_IMAGE_PREFIX)/pgo-lspvc:$(CO_IMAGE_TAG)
	docker push $(CO_IMAGE_PREFIX)/pgo-rmdata:$(CO_IMAGE_TAG)
	docker push $(CO_IMAGE_PREFIX)/pgo-load:$(CO_IMAGE_TAG)
	docker push $(CO_IMAGE_PREFIX)/postgres-operator:$(CO_IMAGE_TAG)
	docker push $(CO_IMAGE_PREFIX)/pgo-apiserver:$(CO_IMAGE_TAG)
	docker push $(CO_IMAGE_PREFIX)/pgo-backrest:$(CO_IMAGE_TAG)
pull:
	docker pull $(CO_IMAGE_PREFIX)/pgo-lspvc:$(CO_IMAGE_TAG)
	docker pull $(CO_IMAGE_PREFIX)/pgo-rmdata:$(CO_IMAGE_TAG)
	docker pull $(CO_IMAGE_PREFIX)/pgo-load:$(CO_IMAGE_TAG)
	docker pull $(CO_IMAGE_PREFIX)/postgres-operator:$(CO_IMAGE_TAG)
	docker pull $(CO_IMAGE_PREFIX)/pgo-apiserver:$(CO_IMAGE_TAG)
release:	check-go-vars
	make macpgo
	make winpgo
	rm -rf $(RELTMPDIR) $(RELFILE)
	mkdir $(RELTMPDIR)
	cp -r $(COROOT)/examples $(RELTMPDIR)
	cp -r $(COROOT)/deploy $(RELTMPDIR)
	cp -r $(COROOT)/conf $(RELTMPDIR)
	cp $(GOBIN)/pgo $(RELTMPDIR)
	cp $(GOBIN)/pgo-mac $(RELTMPDIR)
	cp $(GOBIN)/pgo.exe $(RELTMPDIR)
	cp $(GOBIN)/expenv $(RELTMPDIR)
	cp $(GOBIN)/expenv-mac $(RELTMPDIR)
	cp $(GOBIN)/expenv.exe $(RELTMPDIR)
	cp $(COROOT)/examples/pgo-bash-completion $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .
default:
	all

