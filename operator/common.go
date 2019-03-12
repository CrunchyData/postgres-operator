package operator

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	"bytes"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	//"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"os"
)

var CRUNCHY_DEBUG bool
var NAMESPACE string

var Pgo config.PgoConfig

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

func Initialize(clientset *kubernetes.Clientset, namespace string) {

	tmp := os.Getenv("CRUNCHY_DEBUG")
	if tmp == "true" {
		CRUNCHY_DEBUG = true
		log.Debug("CRUNCHY_DEBUG flag set to true")
	} else {
		CRUNCHY_DEBUG = false
		log.Info("CRUNCHY_DEBUG flag set to false")
	}

	NAMESPACE = os.Getenv("NAMESPACE")
	log.Debugf("setting NAMESPACE to %s", NAMESPACE)
	if NAMESPACE == "" {
		log.Error("NAMESPACE env var is set to empty string which pgo intprets as meaning you want it to watch 'all' namespaces.")
	}

	var err error

	/**
	Pgo.GetConf()

	log.Println("CCPImageTag=" + Pgo.Cluster.CCPImageTag)
	err = Pgo.Validate()
	if err != nil {
		log.Error(err)
		log.Error("pgo.yaml validation failed, can't continue")
		os.Exit(2)
	}
	*/

	err = Pgo.GetConfig(clientset, namespace)
	if err != nil {
		log.Error(err)
		log.Error("pgo-config files and templates did not load")
		os.Exit(2)
	}

	log.Printf("PrimaryStorage=%v\n", Pgo.Storage["storage1"])

	if Pgo.Cluster.CCPImagePrefix == "" {
		log.Debug("pgo.yaml CCPImagePrefix not set, using default")
		Pgo.Cluster.CCPImagePrefix = "crunchydata"
	} else {
		log.Debugf("pgo.yaml CCPImagePrefix set, using %s", Pgo.Cluster.CCPImagePrefix)
	}
	if Pgo.Pgo.COImagePrefix == "" {
		log.Debug("pgo.yaml COImagePrefix not set, using default")
		Pgo.Pgo.COImagePrefix = "crunchydata"
	} else {
		log.Debugf("COImagePrefix set, using %s", Pgo.Pgo.COImagePrefix)
	}

	if Pgo.Cluster.PgmonitorPassword == "" {
		log.Debug("pgo.yaml PgmonitorPassword not set, using default")
		Pgo.Cluster.PgmonitorPassword = "password"
	}

	if Pgo.Pgo.COImageTag == "" {
		log.Error("pgo.yaml COImageTag not set, required ")
		os.Exit(2)
	}
}

// GetContainerResources ...
func GetContainerResourcesJSON(resources *crv1.PgContainerResources) string {

	//test for the case where no container resources are specified
	if resources.RequestsMemory == "" || resources.RequestsCPU == "" ||
		resources.LimitsMemory == "" || resources.LimitsCPU == "" {
		return ""
	}
	fields := containerResourcesTemplateFields{}
	fields.RequestsMemory = resources.RequestsMemory
	fields.RequestsCPU = resources.RequestsCPU
	fields.LimitsMemory = resources.LimitsMemory
	fields.LimitsCPU = resources.LimitsCPU

	doc := bytes.Buffer{}
	err := config.ContainerResourcesTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	if log.GetLevel() == log.DebugLevel {
		config.ContainerResourcesTemplate.Execute(os.Stdout, fields)
	}

	return doc.String()
}
