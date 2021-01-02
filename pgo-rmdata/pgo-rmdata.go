package main

/*
Copyright 2019 - 2021 Crunchy Data
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"flag"
	"os"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crunchylog "github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pgo-rmdata/rmdata"
	log "github.com/sirupsen/logrus"
)

var request rmdata.Request

func main() {
	request = rmdata.Request{
		RemoveData:       false,
		IsReplica:        false,
		IsBackup:         false,
		RemoveBackup:     false,
		ClusterName:      "",
		ClusterPGHAScope: "",
		ReplicaName:      "",
		Namespace:        "",
	}
	flag.BoolVar(&request.RemoveData, "remove-data", false, "")
	flag.BoolVar(&request.IsReplica, "is-replica", false, "")
	flag.BoolVar(&request.IsBackup, "is-backup", false, "")
	flag.BoolVar(&request.RemoveBackup, "remove-backup", false, "")
	flag.StringVar(&request.ClusterName, "pg-cluster", "", "")
	flag.StringVar(&request.ClusterPGHAScope, "pgha-scope", "", "")
	flag.StringVar(&request.ReplicaName, "replica-name", "", "")
	flag.StringVar(&request.Namespace, "namespace", "", "")
	flag.Parse()

	crunchylog.CrunchyLogger(crunchylog.SetParameters())
	if os.Getenv("CRUNCHY_DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	client, err := kubeapi.NewClient()
	if err != nil {
		log.Fatalln(err.Error())
	}

	request.Clientset = client

	log.Infoln("pgo-rmdata starts")
	log.Infof("request is %s", request.String())

	rmdata.Delete(request)

}
