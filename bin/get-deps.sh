#!/bin/bash 

# Copyright 2017 Crunchy Data Solutions, Inc.
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

echo "getting project dependencies...."
godep restore

checkout be41f5093e2b05c7a0befe35b04b715eb325ab43 of apiextensions-apiserver

rm -rf $GOPATH/src/k8s.io/apiextensions-apiserver/vendor
rm -rf $GOPATH/src/k8s.io/apiextensions-apiserver/examples

checkout v4.0.0 of client-go

checkout release-1.7 of apimachinery

go get github.com/lib/pq
go get github.com/fatih/color
go get github.com/Sirupsen/logrus
go get github.com/evanphx/json-patch
go get github.com/gorilla/websocket
go get github.com/gorilla/mux
go get github.com/spf13/cobra
go get github.com/spf13/viper

cd $GOPATH/src/github.com/spf13/cobra
git checkout a3cd8ab85aeba3522b9b59242f3b86ddbc67f8bd

