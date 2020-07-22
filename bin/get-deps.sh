#!/bin/bash -e

# Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo "Ensuring project dependencies..."
BINDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
EVTDIR="$BINDIR/pgo-event"
POSTGRES_EXPORTER_VERSION=0.8.0
PGMONITOR_COMMIT='v4.2'

# Precondition checks
if [ "$GOPATH" = "" ]; then
	# Alternatively, take dep approach of go env GOPATH later in the process
	echo "GOPATH not defined, exiting..." >&2
	exit 1
fi
if ! (echo $PATH | egrep -q "$GOPATH/bin") ; then
	echo '$GOPATH/bin not part of $PATH, exiting...' >&2
	exit 2
fi


# Idempotent installations
if (yum repolist | egrep -q '^epel/') ; then
	echo "Confirmed EPEL repo exists..."
else
	echo "=== Installing EPEL ==="
	# Prefer distro-managed epel-release if it exists (e.g. CentOS)
	if (yum -q list epel-release 2>/dev/null); then
		sudo yum -y install epel-release
	else
		sudo yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
	fi
fi

if which go; then
	echo -n "  Found: " && go version
else
	echo "=== Installing golang ==="
	sudo yum -y install golang
fi

if ! [ -f $EVTDIR/nsqd -a -f $EVTDIR/nsqadmin ]; then
	echo "=== Installing NSQ binaries ==="
	NSQ=nsq-1.1.0.linux-amd64.go1.10.3
	curl -S https://s3.amazonaws.com/bitly-downloads/nsq/$NSQ.tar.gz | \
		tar xz --strip=2 -C $EVTDIR/ '*/bin/*'
fi

if which docker; then
	# Suppress errors for this call, as docker returns non-zero when it can't talk to the daemon
	set +e
	echo -n "  Found: " && docker version --format '{{.Client.Version}}' 2>/dev/null
	set -e
else
	echo "=== Installing docker ==="
	if [ -f /etc/centos-release ]; then
		sudo yum -y install docker
	else
		sudo yum -y install docker --enablerepo=rhel-7-server-extras-rpms
	fi
fi

if which buildah; then
	echo -n "  Found: " && buildah --version
else
	echo "=== Installing buildah ==="
	if [ -f /etc/centos-release ]; then
		sudo yum -y install buildah
	else
		sudo yum -y install buildah --enablerepo=rhel-7-server-extras-rpms
	fi
fi

if which dep; then
	echo -n "  Found: " && (dep version | egrep '^ version')
else
	echo "=== Installing dep ==="
	curl -S https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
fi

# Download Postgres Exporter, only required to build the Crunchy Postgres Exporter container
wget -O $PGOROOT/postgres_exporter.tar.gz https://github.com/wrouesnel/postgres_exporter/releases/download/v${POSTGRES_EXPORTER_VERSION?}/postgres_exporter_v${POSTGRES_EXPORTER_VERSION?}_linux-amd64.tar.gz

# pgMonitor Setup
if [[ -d ${PGOROOT?}/tools/pgmonitor ]]
then
    rm -rf ${PGOROOT?}/tools/pgmonitor
fi

git clone https://github.com/CrunchyData/pgmonitor.git ${PGOROOT?}/tools/pgmonitor
cd ${PGOROOT?}/tools/pgmonitor
git checkout ${PGMONITOR_COMMIT?}

# create pgMonitor documentation data directory, if it doesn't exist
if [[ ! -d ${PGOROOT?}/docs/data/pgmonitor ]]
then
    mkdir -p ${PGOROOT?}/docs/data/pgmonitor
fi

# pgMonitor Script Documentation Setup
rm -rf ${PGOROOT?}/docs/data/pgmonitor/*

# link pgMonitor metrics files to Hugo data directory
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_common.yml $PGOROOT/docs/data/pgmonitor/queries_common.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_per_db.yml $PGOROOT/docs/data/pgmonitor/queries_per_db.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_backrest.yml $PGOROOT/docs/data/pgmonitor/queries_backrest.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_pg12.yml $PGOROOT/docs/data/pgmonitor/queries_pg12.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_pg11.yml $PGOROOT/docs/data/pgmonitor/queries_pg11.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_pg10.yml $PGOROOT/docs/data/pgmonitor/queries_pg10.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_pg96.yml $PGOROOT/docs/data/pgmonitor/queries_pg96.yml
ln -s $PGOROOT/tools/pgmonitor/exporter/postgres/queries_pg95.yml $PGOROOT/docs/data/pgmonitor/queries_pg95.yml
