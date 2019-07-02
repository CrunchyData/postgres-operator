#!/bin/bash -e

# Copyright 2019 Crunchy Data Solutions, Inc.
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

# Precondition checks
if [ "$GOPATH" = "" ]; then
	echo "GOPATH not defined, exiting..." >&2
	exit
fi
if ! (echo $PATH | egrep -q "$GOPATH/bin") ; then
	echo '$GOPATH/bin not part of $PATH, exiting...' >&2
	exit
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

NSQ=nsq-1.1.0.linux-amd64.go1.10.3
wget https://s3.amazonaws.com/bitly-downloads/nsq/$NSQ.tar.gz
tar xvf $NSQ.tar.gz -C /tmp
cp /tmp/$NSQ/bin/* $PGOROOT/bin/pgo-event/
rm -rf /tmp/$NSQ*
rm $NSQ.tar.gz

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
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
fi


#echo "getting expenv go library..."
#go get github.com/blang/expenv
#
#echo "getting go dependencies for cli markdown generation"
#go get github.com/cpuguy83/go-md2man/md2man


