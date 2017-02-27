
ifndef BUILDBASE
	export BUILDBASE=$(GOPATH)/src/github.com/crunchydata/crunchy-containers
endif

versiontest:
	if test -z "$$CCP_PGVERSION"; then echo "CCP_PGVERSION undefined"; exit 1;fi;
	if test -z "$$CCP_BASEOS"; then echo "CCP_BASEOS undefined"; exit 1;fi;
	if test -z "$$CCP_VERSION"; then echo "CCP_VERSION undefined"; exit 1;fi;
setup:
	$(BUILDBASE)/bin/install-deps.sh
gendeps:
	godep save \
	github.com/crunchydata/fishysmell/operator \
	github.com/crunchydata/fishysmell/operatorcontroller 

docbuild:
	cd docs && ./build-docs.sh
postgres:
	make versiontest
	cp `which kubectl` bin/postgres
	docker build -t crunchy-postgres -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.postgres.$(CCP_BASEOS) .
	docker tag crunchy-postgres crunchydata/crunchy-postgres:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
postgres-gis:
	make versiontest
	cp `which kubectl` bin/postgres
	docker build -t crunchy-postgres-gis -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.postgres-gis.$(CCP_BASEOS) .
	docker tag crunchy-postgres-gis crunchydata/crunchy-postgres-gis:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
watch:
	cp `which oc` bin/watch
	cp `which kubectl` bin/watch
	docker build -t crunchy-watch -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.watch.$(CCP_BASEOS) .
	docker tag crunchy-watch crunchydata/crunchy-watch:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
version:
	docker build -t crunchy-version -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.version.$(CCP_BASEOS) .
	docker tag crunchy-version crunchydata/crunchy-version:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
backrest:
	make versiontest
	docker build -t crunchy-backrest -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.backrest.$(CCP_BASEOS) .
	docker tag crunchy-backrest crunchydata/crunchy-backrest:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
backrestd:
	make versiontest
	docker build -t crunchy-backrestd -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.backrestd.$(CCP_BASEOS) .
	docker tag crunchy-backrestd crunchydata/crunchy-backrestd:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
pgbouncer:
	make versiontest
	cp `which oc` bin/pgbouncer
	cp `which kubectl` bin/pgbouncer
	cd bounce && go install bounce.go
	cp $(GOBIN)/bounce bin/pgbouncer/
	docker build -t crunchy-pgbouncer -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.pgbouncer.$(CCP_BASEOS) .
	docker tag crunchy-pgbouncer crunchydata/crunchy-pgbouncer:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
pgpool:
	make versiontest
	docker build -t crunchy-pgpool -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.pgpool.$(CCP_BASEOS) .
	docker tag crunchy-pgpool crunchydata/crunchy-pgpool:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
pgbadger:
	make versiontest
	cd badger && godep go install badgerserver.go
	cp $(GOBIN)/badgerserver bin/pgbadger
	docker build -t crunchy-pgbadger -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.pgbadger.$(CCP_BASEOS) .
	docker tag crunchy-pgbadger crunchydata/crunchy-pgbadger:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
operatorserver:
	make versiontest
	cd operator && go install crunchy-operator.go
	cp $(GOBIN)/crunchy-operator bin/crunchy-operator
	docker build -t crunchy-operator -f $(CCP_BASEOS)/Dockerfile.crunchy-operator.$(CCP_BASEOS) .
	docker tag crunchy-operator crunchydata/crunchy-operator:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
collectserver:
	make versiontest
	cd collect && godep go install collectserver.go
	cp $(GOBIN)/collectserver bin/collect
	docker build -t crunchy-collect -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.collect.$(CCP_BASEOS) .
	docker tag crunchy-collect crunchydata/crunchy-collect:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
backup:
	make versiontest
	docker build -t crunchy-backup -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.backup.$(CCP_BASEOS) .
	docker tag crunchy-backup crunchydata/crunchy-backup:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
pgadmin4: 
	make versiontest
	docker build -t crunchy-pgadmin4 -f $(CCP_BASEOS)/$(CCP_PGVERSION)/Dockerfile.pgadmin4.$(CCP_BASEOS) .
	docker tag crunchy-pgadmin4 crunchydata/crunchy-pgadmin4:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
prometheus: 
	make versiontest
	docker build -t crunchy-prometheus -f $(CCP_BASEOS)/Dockerfile.prometheus.$(CCP_BASEOS) .
	docker tag crunchy-prometheus crunchydata/crunchy-prometheus:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
promgateway: 
	make versiontest
	docker build -t crunchy-promgateway -f $(CCP_BASEOS)/Dockerfile.promgateway.$(CCP_BASEOS) .
	docker tag crunchy-promgateway crunchydata/crunchy-promgateway:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
grafana:
	make versiontest
	docker build -t crunchy-grafana -f $(CCP_BASEOS)/Dockerfile.grafana.$(CCP_BASEOS) .
	docker tag crunchy-grafana crunchydata/crunchy-grafana:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
vac:
	make versiontest
	cd vacuum && godep go install vacuum.go
	cp $(GOBIN)/vacuum bin/vacuum
	docker build -t crunchy-vacuum -f $(CCP_BASEOS)/Dockerfile.vacuum.$(CCP_BASEOS) .
	docker tag crunchy-vacuum crunchydata/crunchy-vacuum:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)
dbaserver:
	cp `which oc` bin/dba
	cp `which kubectl` bin/dba
	cd dba && godep go install dbaserver.go
	cp $(GOBIN)/dbaserver bin/dba
	docker build -t crunchy-dba -f $(CCP_BASEOS)/Dockerfile.dba.$(CCP_BASEOS) .
	docker tag crunchy-dba crunchydata/crunchy-dba:$(CCP_BASEOS)-$(CCP_PGVERSION)-$(CCP_VERSION)

all:
	make versiontest
	make postgres
	make postgres-gis
	make backup
	make watch
	make pgpool
	make pgbouncer
	make pgbadger
	make collectserver
	make grafana
	make promgateway
	make prometheus
	make vac
	make dbaserver
push:
	./bin/push-to-dockerhub.sh
default:
	all
test:
	./tests/docker/test-basic.sh; /usr/bin/test "$$?" -eq 0
	./tests/docker/test-vacuum.sh; /usr/bin/test "$$?" -eq 0
	./tests/docker/test-replica.sh; /usr/bin/test "$$?" -eq 0
	./tests/docker/test-pgpool.sh; /usr/bin/test "$$?" -eq 0
	./tests/docker/test-badger.sh; /usr/bin/test "$$?" -eq 0
	./tests/docker/test-backup.sh; /usr/bin/test "$$?" -eq 0
	./tests/docker/test-restore.sh; /usr/bin/test "$$?" -eq 0
	# ./tests/standalone/test-watch.sh; /usr/bin/test "$$?" -eq 0
	# docker stop master
testopenshift:
	./tests/openshift/test-master.sh; /usr/bin/test "$$?" -eq 0
	./tests/openshift/test-replica.sh; /usr/bin/test "$$?" -eq 0
	./tests/openshift/test-pgpool.sh; /usr/bin/test "$$?" -eq 0
	./tests/openshift/test-watch.sh; /usr/bin/test "$$?" -eq 0
	./tests/openshift/test-scope.sh; /usr/bin/test "$$?" -eq 0
	./tests/openshift/test-backup.sh; /usr/bin/test "$$?" -eq 0

