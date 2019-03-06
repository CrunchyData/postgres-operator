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
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"os"
	"text/template"
)

var CRUNCHY_DEBUG bool
var NAMESPACE string

const pgoBackrestRepoTemplatePath = "/pgo-config/pgo-backrest-repo-template.json"
const pgoBackrestRepoServiceTemplatePath = "/pgo-config/pgo-backrest-repo-service-template.json"
const pgmonitorEnvVarsPath = "/pgo-config/pgmonitor-env-vars.json"
const pgbackrestEnvVarsPath = "/pgo-config/pgbackrest-env-vars.json"
const PgpoolTemplatePath = "/pgo-config/pgpool-template.json"
const PgpoolConfTemplatePath = "/pgo-config/pgpool.conf"
const PgpoolPasswdTemplatePath = "/pgo-config/pool_passwd"
const PgpoolHBATemplatePath = "/pgo-config/pool_hba.conf"
const PgbouncerTemplatePath = "/pgo-config/pgbouncer-template.json"
const PgbouncerConfTemplatePath = "/pgo-config/pgbouncer.ini"
const PgbouncerUsersTemplatePath = "/pgo-config/users.txt"
const PgbouncerHBATemplatePath = "/pgo-config/pgbouncer_hba.conf"
const ServiceTemplatePath = "/pgo-config/cluster-service.json"

const jobPath = "/pgo-config/backup-job.json"
const rmdatajobPath = "/pgo-config/rmdata-job.json"
const backrestjobPath = "/pgo-config/backrest-job.json"
const backrestRestorejobPath = "/pgo-config/backrest-restore-job.json"
const PVCPath = "/pgo-config/pvc.json"
const PVCMatchLabelsPath = "/pgo-config/pvc-matchlabels.json"
const PVCSCPath = "/pgo-config/pvc-storageclass.json"

const pgDumpBackupJobPath = "/pgo-config/pgdump-job.json"
const pgRestoreJobPath = "/pgo-config/pgrestore-job.json"

const DeploymentTemplatePath = "/pgo-config/cluster-deployment.json"
const CollectTemplatePath = "/pgo-config/collect.json"
const BadgerTemplatePath = "/pgo-config/pgbadger.json"
const AffinityTemplatePath = "/pgo-config/affinity.json"
const ContainerResourcesTemplatePath = "/pgo-config/container-resources.json"

var PgoBackrestRepoServiceTemplate *template.Template
var PgoBackrestRepoTemplate *template.Template
var PgmonitorEnvVarsTemplate *template.Template
var PgbackrestEnvVarsTemplate *template.Template
var JobTemplate *template.Template
var UpgradeJobTemplate *template.Template
var PgpoolTemplate *template.Template
var PgpoolConfTemplate *template.Template
var PgpoolPasswdTemplate *template.Template
var PgpoolHBATemplate *template.Template
var PgbouncerTemplate *template.Template
var PgbouncerConfTemplate *template.Template
var PgbouncerUsersTemplate *template.Template
var PgbouncerHBATemplate *template.Template
var ServiceTemplate *template.Template
var IngestjobTemplate *template.Template
var RmdatajobTemplate *template.Template
var BackrestjobTemplate *template.Template
var BackrestRestorejobTemplate *template.Template
var BackrestRestoreConfigMapTemplate *template.Template

var PgDumpBackupJobTemplate *template.Template
var PgRestoreJobTemplate *template.Template

var PVCTemplate *template.Template
var PVCMatchLabelsTemplate *template.Template
var PVCStorageClassTemplate *template.Template
var AffinityTemplate *template.Template
var ContainerResourcesTemplate *template.Template
var CollectTemplate *template.Template
var BadgerTemplate *template.Template
var DeploymentTemplate *template.Template
var ReplicadeploymentTemplate *template.Template
var ReplicadeploymentTemplateShared *template.Template

var Pgo config.PgoConfig

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

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
		log.Error("NAMESPACE env var is set to empty string which pgo intprets as meaning you want it to watch 'all' namespaces.")
	}

	PgoBackrestRepoTemplate = util.LoadTemplate(pgoBackrestRepoTemplatePath)
	PgoBackrestRepoServiceTemplate = util.LoadTemplate(pgoBackrestRepoServiceTemplatePath)
	PgmonitorEnvVarsTemplate = util.LoadTemplate(pgmonitorEnvVarsPath)
	PgbackrestEnvVarsTemplate = util.LoadTemplate(pgbackrestEnvVarsPath)
	JobTemplate = util.LoadTemplate(jobPath)
	PgpoolTemplate = util.LoadTemplate(PgpoolTemplatePath)
	PgpoolConfTemplate = util.LoadTemplate(PgpoolConfTemplatePath)
	PgpoolPasswdTemplate = util.LoadTemplate(PgpoolPasswdTemplatePath)
	PgpoolHBATemplate = util.LoadTemplate(PgpoolHBATemplatePath)
	PgbouncerTemplate = util.LoadTemplate(PgbouncerTemplatePath)
	PgbouncerConfTemplate = util.LoadTemplate(PgbouncerConfTemplatePath)
	PgbouncerUsersTemplate = util.LoadTemplate(PgbouncerUsersTemplatePath)
	PgbouncerHBATemplate = util.LoadTemplate(PgbouncerHBATemplatePath)
	ServiceTemplate = util.LoadTemplate(ServiceTemplatePath)

	RmdatajobTemplate = util.LoadTemplate(rmdatajobPath)
	BackrestjobTemplate = util.LoadTemplate(backrestjobPath)
	BackrestRestorejobTemplate = util.LoadTemplate(backrestRestorejobPath)
	PVCTemplate = util.LoadTemplate(PVCPath)
	PVCMatchLabelsTemplate = util.LoadTemplate(PVCMatchLabelsPath)
	PVCStorageClassTemplate = util.LoadTemplate(PVCSCPath)
	DeploymentTemplate = util.LoadTemplate(DeploymentTemplatePath)
	CollectTemplate = util.LoadTemplate(CollectTemplatePath)
	BadgerTemplate = util.LoadTemplate(BadgerTemplatePath)
	AffinityTemplate = util.LoadTemplate(AffinityTemplatePath)
	ContainerResourcesTemplate = util.LoadTemplate(ContainerResourcesTemplatePath)

	PgDumpBackupJobTemplate = util.LoadTemplate(pgDumpBackupJobPath)
	PgRestoreJobTemplate = util.LoadTemplate(pgRestoreJobPath)

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

	if Pgo.Cluster.PgmonitorPassword == "" {
		log.Debug("pgo.yaml PgmonitorPassword not set, using default")
		Pgo.Cluster.PgmonitorPassword = "password"
	}

	if Pgo.Pgo.COImageTag == "" {
		log.Error("pgo.yaml COImageTag not set, required ")
		panic("pgo.yaml COImageTag env var not set")
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
	err := ContainerResourcesTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	if log.GetLevel() == log.DebugLevel {
		ContainerResourcesTemplate.Execute(os.Stdout, fields)
	}

	return doc.String()
}
