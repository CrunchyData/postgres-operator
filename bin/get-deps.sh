#!/bin/bash

# Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
#godep restore

go get github.com/inconshreveable/mousetrap
go get github.com/blang/expenv
go get github.com/tools/godep

go get k8s.io/client-go
cd $GOPATH/src/k8s.io/client-go
git fetch --all --tags --prune
git checkout kubernetes-1.8.5

cd $GOPATH/src/k8s.io/client-go
godep restore

go get github.com/Sirupsen/logrus;go get github.com/fatih/color;go get github.com/spf13/cobra;go get github.com/spf13/viper
go get github.com/lib/pq;go get github.com/fatih/color;go get github.com/Sirupsen/logrus;go get github.com/evanphx/json-patch;go get github.com/gorilla/websocket;go get github.com/gorilla/mux

cd $GOPATH/src/github.com/spf13/cobra
git checkout a3cd8ab85aeba3522b9b59242f3b86ddbc67f8bd

go get github.com/kubernetes/apiextensions-apiserver
for D in 'github.com/kubernetes' 'k8s.io'; do
  cd $GOPATH/src/${D}/apiextensions-apiserver
  git fetch --all --tags --prune
  git checkout kubernetes-1.8.5
  rm -rf  $GOPATH/src/${D}/apiextensions-apiserver/vendor
done

go get github.com/kubernetes/code-generator
cd $GOPATH/src/github.com/kubernetes/code-generator
git fetch --all --tags --prune
git checkout kubernetes-1.8.5
cd $GOPATH/src/github.com/kubernetes/code-generator/cmd/deepcopy-gen
go build main.go && mv main $GOPATH/bin/deepcopy-gen
