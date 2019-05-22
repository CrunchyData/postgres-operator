#!/bin/bash

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

echo "Getting project dependencies..."

if [ $(command -v apt-get) ]; then
	echo "using apt-get as package manager..."
	PM="apt-get"
elif [ $(command -v yum) ]; then
	echo "using yum as package manager..."
	PM="yum"
fi

#sudo yum -y install mercurial golang
which go
if [ $? -eq 1 ]; then
	echo "installing golang..."
	cd /tmp
	wget https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
	ls *.rpm
	sudo rpm -ivh epel*.rpm
	sudo yum -y install golang
fi

which buildah
if [ $? -eq 1 ]; then
	echo "installing buildah"
	sudo subscription-manager repos --enable=rhel-7-server-extras-rpms
	sudo yum -y install buildah
fi

which dep
if [ $? -eq 1 ]; then
	echo "installing dep"
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
fi


echo "getting expenv go library..."
go get github.com/blang/expenv

echo "getting go dependencies for cli markdown generation"
go get github.com/cpuguy83/go-md2man/md2man
go get github.com/spf13/cobra


# uncomment only if you want to develop on the project
#echo "getting all libraries for project..."
#dep ensure

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

#echo "pre-pulling container suite images used by the operator..."
#$DIR/pre-pull-crunchy-containers.sh
