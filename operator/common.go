package operator

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/util"
	"os"
	"text/template"
)

var CRUNCHY_DEBUG bool
var NAMESPACE string
var COImagePrefix string
var COImageTag string
var CCPImagePrefix string

const jobPath = "/operator-conf/backup-job.json"
const ingestPath = "/operator-conf/pgo-ingest-watch-job.json"
const rmdatajobPath = "/operator-conf/rmdata-job.json"
const PVCPath = "/operator-conf/pvc.json"
const PVCSCPath = "/operator-conf/pvc-storageclass.json"
const UpgradeJobPath = "/operator-conf/cluster-upgrade-job-1.json"

var JobTemplate *template.Template
var UpgradeJobTemplate1 *template.Template
var PgpoolTemplate *template.Template
var PgpoolConfTemplate *template.Template
var PgpoolPasswdTemplate *template.Template
var PgpoolHBATemplate *template.Template
var ServiceTemplate1 *template.Template
var IngestjobTemplate *template.Template
var RmdatajobTemplate *template.Template
var PVCTemplate *template.Template
var PVCStorageClassTemplate *template.Template
var AffinityTemplate1 *template.Template
var ContainerResourcesTemplate1 *template.Template
var CollectTemplate1 *template.Template
var DeploymentTemplate1 *template.Template
var ReplicadeploymentTemplate1 *template.Template
var ReplicadeploymentTemplate1Shared *template.Template

var Pgo config.PgoConfig

func Initialize() {
	CCPImagePrefix = os.Getenv("CCP_IMAGE_PREFIX")
	if CCPImagePrefix == "" {
		log.Debug("CCP_IMAGE_PREFIX not set, using default")
		CCPImagePrefix = "crunchydata"
	} else {
		log.Debug("CCP_IMAGE_PREFIX set, using " + CCPImagePrefix)
	}
	COImagePrefix = os.Getenv("CO_IMAGE_PREFIX")
	if COImagePrefix == "" {
		log.Debug("CO_IMAGE_PREFIX not set, using default")
		COImagePrefix = "crunchydata"
	} else {
		log.Debug("CO_IMAGE_PREFIX set, using " + COImagePrefix)
	}
	COImageTag = os.Getenv("CO_IMAGE_TAG")
	if COImageTag == "" {
		log.Error("CO_IMAGE_TAG not set, required ")
		panic("CO_IMAGE_TAG env var not set")
	}

	tmp := os.Getenv("CRUNCHY_DEBUG")
	if tmp == "true" {
		CRUNCHY_DEBUG = true
		log.Debug("CRUNCHY_DEBUG flag set to true")
	} else {
		CRUNCHY_DEBUG = false
		log.Info("CRUNCHY_DEBUG flag set to false")
	}

	NAMESPACE = os.Getenv("NAMESPACE")
	log.Debug("setting NAMESPACE to " + NAMESPACE)
	if NAMESPACE == "" {
		log.Error("NAMESPACE env var not set")
		panic("NAMESPACE env var not set")
	}

	JobTemplate = util.LoadTemplate(jobPath)
	PgpoolTemplate = util.LoadTemplate("/operator-conf/pgpool-template.json")
	PgpoolConfTemplate = util.LoadTemplate("/operator-conf/pgpool.conf")
	PgpoolPasswdTemplate = util.LoadTemplate("/operator-conf/pool_passwd")
	PgpoolHBATemplate = util.LoadTemplate("/operator-conf/pool_hba.conf")
	ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")
	IngestjobTemplate = util.LoadTemplate(ingestPath)
	RmdatajobTemplate = util.LoadTemplate(rmdatajobPath)
	PVCTemplate = util.LoadTemplate(PVCPath)
	PVCStorageClassTemplate = util.LoadTemplate(PVCSCPath)
	DeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-deployment-1.json")
	CollectTemplate1 = util.LoadTemplate("/operator-conf/collect.json")
	AffinityTemplate1 = util.LoadTemplate("/operator-conf/affinity.json")
	ContainerResourcesTemplate1 = util.LoadTemplate("/operator-conf/container-resources.json")
	UpgradeJobTemplate1 = util.LoadTemplate(UpgradeJobPath)

	Pgo.GetConf()
	log.Println("CCPImageTag=" + Pgo.Cluster.CCPImageTag)
	Pgo.Validate()
}
