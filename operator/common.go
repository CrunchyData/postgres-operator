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

const PgpoolTemplatePath = "/pgo-config/pgpool-template.json"
const PgpoolConfTemplatePath = "/pgo-config/pgpool.conf"
const PgpoolPasswdTemplatePath = "/pgo-config/pool_passwd"
const PgpoolHBATemplatePath = "/pgo-config/pool_hba.conf"
const PgbouncerTemplatePath = "/pgo-config/pgbouncer-template.json"
const PgbouncerConfTemplatePath = "/pgo-config/pgbouncer.ini"
const PgbouncerUsersTemplatePath = "/pgo-config/users.txt"
const PgbouncerHBATemplatePath = "/pgo-config/pgbouncer_hba.conf"
const ServiceTemplate1Path = "/pgo-config/cluster-service-1.json"

const jobPath = "/pgo-config/backup-job.json"
const ingestPath = "/pgo-config/pgo-ingest-watch-job.json"
const rmdatajobPath = "/pgo-config/rmdata-job.json"
const backrestjobPath = "/pgo-config/backrest-job.json"
const backrestRestorejobPath = "/pgo-config/backrest-restore-job.json"
const backrestRestoreConfigMapPath = "/pgo-config/backrest-restore-configmap.json"
const backrestRestoreVolumesPath = "/pgo-config/backrest-restore-volumes.json"
const backrestRestoreVolumeMountsPath = "/pgo-config/backrest-restore-volume-mounts.json"
const PVCPath = "/pgo-config/pvc.json"
const PVCMatchLabelsPath = "/pgo-config/pvc-matchlabels.json"
const PVCSCPath = "/pgo-config/pvc-storageclass.json"
const UpgradeJobPath = "/pgo-config/cluster-upgrade-job-1.json"

const DeploymentTemplate1Path = "/pgo-config/cluster-deployment-1.json"
const CollectTemplate1Path = "/pgo-config/collect.json"
const BadgerTemplate1Path = "/pgo-config/pgbadger.json"
const AffinityTemplate1Path = "/pgo-config/affinity.json"
const ContainerResourcesTemplate1Path = "/pgo-config/container-resources.json"

var JobTemplate *template.Template
var UpgradeJobTemplate1 *template.Template
var PgpoolTemplate *template.Template
var PgpoolConfTemplate *template.Template
var PgpoolPasswdTemplate *template.Template
var PgpoolHBATemplate *template.Template
var PgbouncerTemplate *template.Template
var PgbouncerConfTemplate *template.Template
var PgbouncerUsersTemplate *template.Template
var PgbouncerHBATemplate *template.Template
var ServiceTemplate1 *template.Template
var IngestjobTemplate *template.Template
var RmdatajobTemplate *template.Template
var BackrestjobTemplate *template.Template
var BackrestRestoreVolumesTemplate *template.Template
var BackrestRestoreVolumeMountsTemplate *template.Template
var BackrestRestorejobTemplate *template.Template
var BackrestRestoreConfigMapTemplate *template.Template
var PVCTemplate *template.Template
var PVCMatchLabelsTemplate *template.Template
var PVCStorageClassTemplate *template.Template
var AffinityTemplate1 *template.Template
var ContainerResourcesTemplate1 *template.Template
var CollectTemplate1 *template.Template
var BadgerTemplate1 *template.Template
var DeploymentTemplate1 *template.Template
var ReplicadeploymentTemplate1 *template.Template
var ReplicadeploymentTemplate1Shared *template.Template

var Pgo config.PgoConfig

func Initialize() {

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
		log.Error("NAMESPACE env var not set")
		panic("NAMESPACE env var not set")
	}

	JobTemplate = util.LoadTemplate(jobPath)
	PgpoolTemplate = util.LoadTemplate(PgpoolTemplatePath)
	PgpoolConfTemplate = util.LoadTemplate(PgpoolConfTemplatePath)
	PgpoolPasswdTemplate = util.LoadTemplate(PgpoolPasswdTemplatePath)
	PgpoolHBATemplate = util.LoadTemplate(PgpoolHBATemplatePath)
	PgbouncerTemplate = util.LoadTemplate(PgbouncerTemplatePath)
	PgbouncerConfTemplate = util.LoadTemplate(PgbouncerConfTemplatePath)
	PgbouncerUsersTemplate = util.LoadTemplate(PgbouncerUsersTemplatePath)
	PgbouncerHBATemplate = util.LoadTemplate(PgbouncerHBATemplatePath)
	ServiceTemplate1 = util.LoadTemplate(ServiceTemplate1Path)

	IngestjobTemplate = util.LoadTemplate(ingestPath)
	RmdatajobTemplate = util.LoadTemplate(rmdatajobPath)
	BackrestjobTemplate = util.LoadTemplate(backrestjobPath)
	BackrestRestoreVolumesTemplate = util.LoadTemplate(backrestRestoreVolumesPath)
	BackrestRestoreVolumeMountsTemplate = util.LoadTemplate(backrestRestoreVolumeMountsPath)
	BackrestRestorejobTemplate = util.LoadTemplate(backrestRestorejobPath)
	BackrestRestoreConfigMapTemplate = util.LoadTemplate(backrestRestoreConfigMapPath)
	PVCTemplate = util.LoadTemplate(PVCPath)
	PVCMatchLabelsTemplate = util.LoadTemplate(PVCMatchLabelsPath)
	PVCStorageClassTemplate = util.LoadTemplate(PVCSCPath)
	DeploymentTemplate1 = util.LoadTemplate(DeploymentTemplate1Path)
	CollectTemplate1 = util.LoadTemplate(CollectTemplate1Path)
	BadgerTemplate1 = util.LoadTemplate(BadgerTemplate1Path)
	AffinityTemplate1 = util.LoadTemplate(AffinityTemplate1Path)
	ContainerResourcesTemplate1 = util.LoadTemplate(ContainerResourcesTemplate1Path)
	UpgradeJobTemplate1 = util.LoadTemplate(UpgradeJobPath)

	Pgo.GetConf()
	log.Println("CCPImageTag=" + Pgo.Cluster.CCPImageTag)
	err := Pgo.Validate()
	if err != nil {
		log.Error(err)
		log.Error("pgo.yaml validation failed, can't continue")
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
	if Pgo.Pgo.COImageTag == "" {
		log.Error("pgo.yaml COImageTag not set, required ")
		panic("pgo.yaml COImageTag env var not set")
	}
}
