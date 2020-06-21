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

if which expenv; then
	echo "  Found expenv"
else
	echo "=== Installing expenv ==="
	# TODO: expenv uses Go modules, could retrieve specific version
	go get github.com/blang/expenv
fi
