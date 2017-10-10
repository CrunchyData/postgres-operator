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
etlclient:      check-go-vars
	        go build -buildmode=plugin -o client/etlclient.so client/etlclient.go
#======= Main functions =======
#	cd pgo && env GOOS=darwin GOARCH=amd64 go build pgo.go 
deployoperator:
	cd deploy && ./deploy.sh
main:	check-go-vars
	go install postgres-operator.go
runmain:	check-go-vars
	main --kubeconfig=/etc/kubernetes/admin.conf
runapiserver:	check-go-vars
	apiserver --kubeconfig=/etc/kubernetes/admin.conf
apiserver:	check-go-vars
	go install apiserver.go
rpgo:	check-go-vars
	cd rpgo && go install rpgo.go
pgo:	check-go-vars
	cd pgo && go install pgo.go
runpgo:	check-go-vars
	pgo --kubeconfig=/etc/kubernetes/admin.conf
clean:	check-go-vars
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/main $(GOBIN)/pgo
	godep restore
operatorimage:	check-go-vars
	go install postgres-operator.go
	cp $(GOBIN)/postgres-operator bin/postgres-operator/
	docker build -t postgres-operator -f $(CO_BASEOS)/Dockerfile.postgres-operator.$(CO_BASEOS) .
	docker tag postgres-operator crunchydata/postgres-operator:$(CO_BASEOS)-$(CO_VERSION)
lsimage:
	docker build -t lspvc -f $(CO_BASEOS)/Dockerfile.lspvc.$(CO_BASEOS) .
	docker tag lspvc crunchydata/lspvc:$(CO_BASEOS)-$(CO_VERSION)
csvloadimage:
	docker build -t csvload -f $(CO_BASEOS)/Dockerfile.csvload.$(CO_BASEOS) .
	docker tag csvload crunchydata/csvload:$(CO_BASEOS)-$(CO_VERSION)
all:
	make operatorimage
	make lsimage
	make csvloadimage
	make pgo
push:
	docker push crunchydata/lspvc:$(CO_IMAGE_TAG)
	docker push crunchydata/csvload:$(CO_IMAGE_TAG)
	docker push crunchydata/postgres-operator:$(CO_IMAGE_TAG)
release:	check-go-vars
	rm -rf $(RELTMPDIR) $(RELFILE)
	mkdir $(RELTMPDIR)
	cp $(GOBIN)/pgo $(RELTMPDIR)
	cp $(COROOT)/examples/*pgo.yaml* $(RELTMPDIR)
	cp $(COROOT)/examples/*pgo.lspvc-template.json $(RELTMPDIR)
	cp $(COROOT)/examples/*pgo.csvload-template.json $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .
default:
	all

