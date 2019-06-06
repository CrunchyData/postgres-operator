package operator

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"os"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

var CRUNCHY_DEBUG bool
var NAMESPACE string

var PgoNamespace string

var Pgo config.PgoConfig

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

func Initialize(clientset *kubernetes.Clientset) {

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

	PgoNamespace = os.Getenv("PGO_OPERATOR_NAMESPACE")
	if PgoNamespace == "" {
		log.Error("PGO_OPERATOR_NAMESPACE environment variable is not set and is required, this is the namespace that the Operator is to run within.")
		os.Exit(2)
	}

	var err error

	err = Pgo.GetConfig(clientset, PgoNamespace)
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
	if Pgo.Pgo.PGOImagePrefix == "" {
		log.Debug("pgo.yaml PGOImagePrefix not set, using default")
		Pgo.Pgo.PGOImagePrefix = "crunchydata"
	} else {
		log.Debugf("PGOImagePrefix set, using %s", Pgo.Pgo.PGOImagePrefix)
	}

	if Pgo.Cluster.PgmonitorPassword == "" {
		log.Debug("pgo.yaml PgmonitorPassword not set, using default")
		Pgo.Cluster.PgmonitorPassword = "password"
	}

	if Pgo.Pgo.PGOImageTag == "" {
		log.Error("pgo.yaml PGOImageTag not set, required ")
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

// GetRepoType returns the proper repo type to set in container based on the
// backrest storage type provided
func GetRepoType(backrestStorageType string) string {
	if backrestStorageType != "" && backrestStorageType == "s3" {
		return "s3"
	} else {
		return "posix"
	}
}

// IsLocalAndS3Storage a boolean indicating whether or not local and s3 storage should
// be enabled for pgBackRest based on the backrestStorageType string provided
func IsLocalAndS3Storage(backrestStorageType string) bool {
	if backrestStorageType != "" && strings.Contains(backrestStorageType, "s3") &&
		strings.Contains(backrestStorageType, "local") {
		return true
	}
	return false
}
