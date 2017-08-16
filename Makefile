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
deploy:
	cd examples/operator && ./deploy.sh
pgo:	check-go-vars
	cd client && go build -o $(GOBIN)/pgo pgo.go
clean:	check-go-vars
	rm -rf $(GOPATH)/pkg/* $(GOBIN)/*
	go get github.com/tools/godep
	godep restore
operatorimage:	check-go-vars
	cd operator && go install postgres-operator.go
	cp $(GOBIN)/postgres-operator bin/postgres-operator
	docker build -t postgres-operator -f $(CO_BASEOS)/Dockerfile.postgres-operator.$(CO_BASEOS) .
	docker tag postgres-operator crunchydata/postgres-operator:$(CO_BASEOS)-$(CO_VERSION)
lsimage:
	docker build -t lspvc -f $(CO_BASEOS)/Dockerfile.lspvc.$(CO_BASEOS) .
	docker tag lspvc crunchydata/lspvc:$(CO_BASEOS)-$(CO_VERSION)
all:
	make operatorimage
	make lsimage
	make pgo
push:
	docker push crunchydata/lspvc:$(CO_IMAGE_TAG)
	docker push crunchydata/postgres-operator:$(CO_IMAGE_TAG)
release:	check-go-vars
	rm -rf $(RELTMPDIR) $(RELFILE)
	mkdir $(RELTMPDIR)
	cp $(GOBIN)/pgo $(RELTMPDIR)
	cp $(COROOT)/examples/*pgo.yaml* $(RELTMPDIR)
	cp $(COROOT)/examples/*pgo.lspvc-template.json $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .
default:
	all

