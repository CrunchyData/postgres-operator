package config

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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const CustomConfigMapName = "pgo-config"
const DefaultConfigsPath = "/default-pgo-config/"
const CustomConfigsPath = "/pgo-config/"

var BenchmarkJobTemplate *template.Template

const benchmarkJobPath = "pgbench-job.json"

var PolicyJobTemplate *template.Template

const policyJobTemplatePath = "pgo.sqlrunner-template.json"

var PVCTemplate *template.Template

const pvcPath = "pvc.json"

var ContainerResourcesTemplate *template.Template

const containerResourcesTemplatePath = "container-resources.json"

var LoadTemplate *template.Template

const loadTemplatePath = "pgo.load-template.json"

var LspvcTemplate *template.Template

const lspvcTemplatePath = "pgo.lspvc-template.json"

var AffinityTemplate *template.Template

const affinityTemplatePath = "affinity.json"

var PgoBackrestRepoServiceTemplate *template.Template

const pgoBackrestRepoServiceTemplatePath = "pgo-backrest-repo-service-template.json"

var PgoBackrestRepoTemplate *template.Template

const pgoBackrestRepoTemplatePath = "pgo-backrest-repo-template.json"

var PgmonitorEnvVarsTemplate *template.Template

const pgmonitorEnvVarsPath = "pgmonitor-env-vars.json"

var PgbackrestEnvVarsTemplate *template.Template

const pgbackrestEnvVarsPath = "pgbackrest-env-vars.json"

var PgbackrestS3EnvVarsTemplate *template.Template

const pgbackrestS3EnvVarsPath = "pgbackrest-s3-env-vars.json"

var JobTemplate *template.Template

const jobPath = "backup-job.json"

var PgBasebackupRestoreJobTemplate *template.Template

const pgBasebackupRestoreJobTemplatePath = "pgbasebackup-restore-job.json"

var PgpoolTemplate *template.Template

const pgpoolTemplatePath = "pgpool-template.json"

var PgpoolConfTemplate *template.Template

const pgpoolConfTemplatePath = "pgpool.conf"

var PgpoolPasswdTemplate *template.Template

const pgpoolPasswdTemplatePath = "pool_passwd"

var PgpoolHBATemplate *template.Template

const pgpoolHBATemplatePath = "pool_hba.conf"

var PgbouncerTemplate *template.Template

const pgbouncerTemplatePath = "pgbouncer-template.json"

var PgbouncerConfTemplate *template.Template

const pgbouncerConfTemplatePath = "pgbouncer.ini"

var PgbouncerUsersTemplate *template.Template

const pgbouncerUsersTemplatePath = "users.txt"

var PgbouncerHBATemplate *template.Template

const pgbouncerHBATemplatePath = "pgbouncer_hba.conf"

var ServiceTemplate *template.Template

const serviceTemplatePath = "cluster-service.json"

var RmdatajobTemplate *template.Template

const rmdatajobPath = "rmdata-job.json"

var BackrestjobTemplate *template.Template

const backrestjobPath = "backrest-job.json"

var BackrestRestorejobTemplate *template.Template

const backrestRestorejobPath = "backrest-restore-job.json"

var PgDumpBackupJobTemplate *template.Template

const pgDumpBackupJobPath = "pgdump-job.json"

var PgRestoreJobTemplate *template.Template

const pgRestoreJobPath = "pgrestore-job.json"

var PVCMatchLabelsTemplate *template.Template

const pvcMatchLabelsPath = "pvc-matchlabels.json"

var PVCStorageClassTemplate *template.Template

const pvcSCPath = "pvc-storageclass.json"

var CollectTemplate *template.Template

const collectTemplatePath = "collect.json"

var BadgerTemplate *template.Template

const badgerTemplatePath = "pgbadger.json"

var DeploymentTemplate *template.Template

const deploymentTemplatePath = "cluster-deployment.json"

type ClusterStruct struct {
	CCPImagePrefix          string `yaml:"CCPImagePrefix"`
	CCPImageTag             string `yaml:"CCPImageTag"`
	PrimaryNodeLabel        string `yaml:"PrimaryNodeLabel"`
	ReplicaNodeLabel        string `yaml:"ReplicaNodeLabel"`
	Policies                string `yaml:"Policies"`
	LogStatement            string `yaml:"LogStatement"`
	LogMinDurationStatement string `yaml:"LogMinDurationStatement"`
	Metrics                 bool   `yaml:"Metrics"`
	Badger                  bool   `yaml:"Badger"`
	Port                    string `yaml:"Port"`
	User                    string `yaml:"User"`
	Database                string `yaml:"Database"`
	PasswordAgeDays         string `yaml:"PasswordAgeDays"`
	PasswordLength          string `yaml:"PasswordLength"`
	Strategy                string `yaml:"Strategy"`
	Replicas                string `yaml:"Replicas"`
	ServiceType             string `yaml:"ServiceType"`
	BackrestPort            int    `yaml:"BackrestPort"`
	Backrest                bool   `yaml:"Backrest"`
	BackrestS3Bucket        string `yaml:"BackrestS3Bucket"`
	BackrestS3Endpoint      string `yaml:"BackrestS3Endpoint"`
	BackrestS3Region        string `yaml:"BackrestS3Region"`
	Autofail                bool   `yaml:"Autofail"`
	AutofailReplaceReplica  bool   `yaml:"AutofailReplaceReplica"`
	PgmonitorPassword       string `yaml:"PgmonitorPassword"`
}

type StorageStruct struct {
	AccessMode         string `yaml:"AccessMode"`
	Size               string `yaml:"Size"`
	StorageType        string `yaml:"StorageType"`
	StorageClass       string `yaml:"StorageClass"`
	Fsgroup            string `yaml:"Fsgroup"`
	SupplementalGroups string `yaml:"SupplementalGroups"`
	MatchLabels        string `yaml:"MatchLabels"`
}

type ContainerResourcesStruct struct {
	RequestsMemory string `yaml:"RequestsMemory"`
	RequestsCPU    string `yaml:"RequestsCPU"`
	LimitsMemory   string `yaml:"LimitsMemory"`
	LimitsCPU      string `yaml:"LimitsCPU"`
}

type PgoStruct struct {
	PreferredFailoverNode     string `yaml:"PreferredFailoverNode"`
	AutofailSleepSeconds      string `yaml:"AutofailSleepSeconds"`
	AutofailSleepSecondsValue int
	Audit                     bool   `yaml:"Audit"`
	PGOImagePrefix            string `yaml:"PGOImagePrefix"`
	PGOImageTag               string `yaml:"PGOImageTag"`
}

type PgoConfig struct {
	BasicAuth                 string                              `yaml:"BasicAuth"`
	Cluster                   ClusterStruct                       `yaml:"Cluster"`
	Pgo                       PgoStruct                           `yaml:"Pgo"`
	ContainerResources        map[string]ContainerResourcesStruct `yaml:"ContainerResources"`
	PrimaryStorage            string                              `yaml:"PrimaryStorage"`
	BackupStorage             string                              `yaml:"BackupStorage"`
	ReplicaStorage            string                              `yaml:"ReplicaStorage"`
	BackrestStorage           string                              `yaml:"BackrestStorage"`
	Storage                   map[string]StorageStruct            `yaml:"Storage"`
	DefaultContainerResources string                              `yaml:"DefaultContainerResources"`
	DefaultLoadResources      string                              `yaml:"DefaultLoadResources"`
	DefaultLspvcResources     string                              `yaml:"DefaultLspvcResources"`
	DefaultRmdataResources    string                              `yaml:"DefaultRmdataResources"`
	DefaultBackupResources    string                              `yaml:"DefaultBackupResources"`
	DefaultBadgerResources    string                              `yaml:"DefaultBadgerResources"`
	DefaultPgpoolResources    string                              `yaml:"DefaultPgpoolResources"`
	DefaultPgbouncerResources string                              `yaml:"DefaultPgbouncerResources"`
}

const DEFAULT_AUTOFAIL_SLEEP_SECONDS = "30"
const DEFAULT_SERVICE_TYPE = "ClusterIP"
const LOAD_BALANCER_SERVICE_TYPE = "LoadBalancer"
const NODEPORT_SERVICE_TYPE = "NodePort"
const CONFIG_PATH = "pgo.yaml"

var log_statement_values = []string{"ddl", "none", "mod", "all"}

const DEFAULT_LOG_STATEMENT = "none"
const DEFAULT_LOG_MIN_DURATION_STATEMENT = "60000"
const DEFAULT_BACKREST_PORT = 2022
const DEFAULT_BACKREST_SSH_KEY_BITS = 2048

func (c *PgoConfig) Validate() error {
	var err error

	if c.Cluster.BackrestPort == 0 {
		c.Cluster.BackrestPort = DEFAULT_BACKREST_PORT
		log.Infof("setting BackrestPort to default %d", c.Cluster.BackrestPort)
	}

	if c.Cluster.LogStatement != "" {
		found := false
		for _, v := range log_statement_values {
			if v == c.Cluster.LogStatement {
				found = true
			}
		}
		if !found {
			return errors.New("Cluster.LogStatement does not container a valid value for log_statement")
		}
	} else {
		log.Info("using default log_statement value since it was not specified in pgo.yaml")
		c.Cluster.LogStatement = DEFAULT_LOG_STATEMENT
	}

	if c.Cluster.LogMinDurationStatement != "" {
		_, err = strconv.Atoi(c.Cluster.LogMinDurationStatement)
		if err != nil {
			return errors.New("Cluster.LogMinDurationStatement invalid int value found")
		}
	} else {
		log.Info("using default log_min_duration_statement value since it was not specified in pgo.yaml")
		c.Cluster.LogMinDurationStatement = DEFAULT_LOG_MIN_DURATION_STATEMENT
	}

	if c.Cluster.PrimaryNodeLabel != "" {
		parts := strings.Split(c.Cluster.PrimaryNodeLabel, "=")
		if len(parts) != 2 {
			return errors.New("Cluster.PrimaryNodeLabel does not follow key=value format")
		}
	}

	if c.Cluster.ReplicaNodeLabel != "" {
		parts := strings.Split(c.Cluster.ReplicaNodeLabel, "=")
		if len(parts) != 2 {
			return errors.New("Cluster.ReplicaNodeLabel does not follow key=value format")
		}
	}

	log.Infof("pgo.yaml Cluster.Backrest is %v", c.Cluster.Backrest)
	_, ok := c.Storage[c.PrimaryStorage]
	if !ok {
		return errors.New("PrimaryStorage setting required")
	}
	_, ok = c.Storage[c.BackupStorage]
	if !ok {
		return errors.New("BackupStorage setting required")
	}
	_, ok = c.Storage[c.BackrestStorage]
	if !ok {
		log.Warning("BackrestStorage setting not set, will use PrimaryStorage setting")
		c.Storage[c.BackrestStorage] = c.Storage[c.PrimaryStorage]
	}

	_, ok = c.Storage[c.ReplicaStorage]
	if !ok {
		return errors.New("ReplicaStorage setting required")
	}
	for k, _ := range c.Storage {
		_, err = c.GetStorageSpec(k)
		if err != nil {
			return err
		}
	}
	if c.Pgo.PGOImagePrefix == "" {
		return errors.New("Pgo.PGOImagePrefix is required")
	}
	if c.Pgo.PGOImageTag == "" {
		return errors.New("Pgo.PGOImageTag is required")
	}
	if c.Pgo.AutofailSleepSeconds == "" {
		log.Warn("Pgo.AutofailSleepSeconds not set, using default ")
		c.Pgo.AutofailSleepSeconds = DEFAULT_AUTOFAIL_SLEEP_SECONDS
	}
	c.Pgo.AutofailSleepSecondsValue, err = strconv.Atoi(c.Pgo.AutofailSleepSeconds)
	if err != nil {
		return errors.New("Pgo.AutofailSleepSeconds invalid int value found")
	}

	if c.DefaultContainerResources != "" {
		_, ok = c.ContainerResources[c.DefaultContainerResources]
		if !ok {
			return errors.New("DefaultContainerResources setting invalid")
		}
	}
	if c.DefaultLspvcResources != "" {
		_, ok = c.ContainerResources[c.DefaultLspvcResources]
		if !ok {
			return errors.New("DefaultLspvcResources setting invalid")
		}
	}
	if c.DefaultLoadResources != "" {
		_, ok = c.ContainerResources[c.DefaultLoadResources]
		if !ok {
			return errors.New("DefaultLoadResources setting invalid")
		}
	}
	if c.DefaultRmdataResources != "" {
		_, ok = c.ContainerResources[c.DefaultRmdataResources]
		if !ok {
			return errors.New("DefaultRmdataResources setting invalid")
		}
	}
	if c.DefaultBackupResources != "" {
		_, ok = c.ContainerResources[c.DefaultBackupResources]
		if !ok {
			return errors.New("DefaultBackupResources setting invalid")
		}
	}
	if c.DefaultBadgerResources != "" {
		_, ok = c.ContainerResources[c.DefaultBadgerResources]
		if !ok {
			return errors.New("DefaultBadgerResources setting invalid")
		}
	}
	if c.DefaultPgpoolResources != "" {
		_, ok = c.ContainerResources[c.DefaultPgpoolResources]
		if !ok {
			return errors.New("DefaultPgpoolResources setting invalid")
		}
	}
	if c.DefaultPgbouncerResources != "" {
		_, ok = c.ContainerResources[c.DefaultPgbouncerResources]
		if !ok {
			return errors.New("DefaultPgbouncerResources setting invalid")
		}
	}

	if c.Cluster.ServiceType == "" {
		log.Warn("Cluster.ServiceType not set, using default, ClusterIP ")
		c.Cluster.ServiceType = DEFAULT_SERVICE_TYPE
	} else {
		if c.Cluster.ServiceType != DEFAULT_SERVICE_TYPE &&
			c.Cluster.ServiceType != LOAD_BALANCER_SERVICE_TYPE &&
			c.Cluster.ServiceType != NODEPORT_SERVICE_TYPE {
			return errors.New("Cluster.ServiceType is required to be either ClusterIP, NodePort, or LoadBalancer")
		}
	}

	if c.Cluster.CCPImagePrefix == "" {
		return errors.New("Cluster.CCPImagePrefix is required")
	}

	if c.Cluster.CCPImageTag == "" {
		return errors.New("Cluster.CCPImageTag is required")
	}
	return err
}

func (c *PgoConfig) GetConf() *PgoConfig {

	yamlFile, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (c *PgoConfig) GetStorageSpec(name string) (crv1.PgStorageSpec, error) {
	var err error
	storage := crv1.PgStorageSpec{}

	s, ok := c.Storage[name]
	if !ok {
		err = errors.New("invalid Storage name " + name)
		log.Error(err)
		return storage, err
	}

	storage.StorageClass = s.StorageClass
	storage.AccessMode = s.AccessMode
	storage.Size = s.Size
	storage.StorageType = s.StorageType
	storage.Fsgroup = s.Fsgroup
	storage.MatchLabels = s.MatchLabels
	storage.SupplementalGroups = s.SupplementalGroups

	if s.Fsgroup != "" && s.SupplementalGroups != "" {
		err = errors.New("invalid Storage config " + name + " can not have both fsgroup and supplementalGroups specified in the same config, choose one.")
		log.Error(err)
		return storage, err
	}

	if storage.MatchLabels != "" {
		test := strings.Split(storage.MatchLabels, "=")
		if len(test) != 2 {
			err = errors.New("invalid Storage config " + name + " MatchLabels needs to be in key=value format.")
			log.Error(err)
			return storage, err
		}
	}

	return storage, err

}

func (c *PgoConfig) GetContainerResource(name string) (crv1.PgContainerResources, error) {
	var err error
	r := crv1.PgContainerResources{}

	s, ok := c.ContainerResources[name]
	if !ok {
		err = errors.New("invalid Container Resources name " + name)
		log.Error(err)
		return r, err
	}
	r.RequestsMemory = s.RequestsMemory
	r.RequestsCPU = s.RequestsCPU
	r.LimitsMemory = s.LimitsMemory
	r.LimitsCPU = s.LimitsCPU

	return r, err

}

func (c *PgoConfig) GetConfig(clientset *kubernetes.Clientset, namespace string) error {

	cMap, rootPath := getRootPath(clientset, namespace)

	var yamlFile []byte
	var err error

	//get the pgo.yaml config file
	if cMap != nil {
		str := cMap.Data[CONFIG_PATH]
		if str == "" {
			errMsg := fmt.Sprintf("could not get %s from ConfigMap", CONFIG_PATH)
			return errors.New(errMsg)
		}
		yamlFile = []byte(str)
	} else {
		yamlFile, err = ioutil.ReadFile(rootPath + CONFIG_PATH)
		if err != nil {
			log.Errorf("yamlFile.Get err   #%v ", err)
			return err
		}
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Errorf("Unmarshal: %v", err)
		return err
	}

	//validate the pgo.yaml config file
	err = c.Validate()
	if err != nil {
		log.Error(err)
		return err
	}

	//determine the default storage class if necessary
	if cMap == nil {
		err = c.SetDefaultStorageClass(clientset)
		if err != nil {
			return err
		}
	}

	c.CheckEnv()

	//load up all the templates
	BenchmarkJobTemplate, err = c.LoadTemplate(cMap, rootPath, benchmarkJobPath)
	if err != nil {
		return err
	}

	PVCTemplate, err = c.LoadTemplate(cMap, rootPath, pvcPath)
	if err != nil {
		return err
	}

	PolicyJobTemplate, err = c.LoadTemplate(cMap, rootPath, policyJobTemplatePath)
	if err != nil {
		return err
	}

	ContainerResourcesTemplate, err = c.LoadTemplate(cMap, rootPath, containerResourcesTemplatePath)
	if err != nil {
		return err
	}

	LoadTemplate, err = c.LoadTemplate(cMap, rootPath, loadTemplatePath)
	if err != nil {
		return err
	}

	LspvcTemplate, err = c.LoadTemplate(cMap, rootPath, lspvcTemplatePath)
	if err != nil {
		return err
	}

	JobTemplate, err = c.LoadTemplate(cMap, rootPath, jobPath)
	if err != nil {
		return err
	}

	PgoBackrestRepoServiceTemplate, err = c.LoadTemplate(cMap, rootPath, pgoBackrestRepoServiceTemplatePath)
	if err != nil {
		return err
	}

	PgoBackrestRepoTemplate, err = c.LoadTemplate(cMap, rootPath, pgoBackrestRepoTemplatePath)
	if err != nil {
		return err
	}

	PgmonitorEnvVarsTemplate, err = c.LoadTemplate(cMap, rootPath, pgmonitorEnvVarsPath)
	if err != nil {
		return err
	}

	PgbackrestEnvVarsTemplate, err = c.LoadTemplate(cMap, rootPath, pgbackrestEnvVarsPath)
	if err != nil {
		return err
	}

	PgbackrestS3EnvVarsTemplate, err = c.LoadTemplate(cMap, rootPath, pgbackrestS3EnvVarsPath)
	if err != nil {
		return err
	}

	JobTemplate, err = c.LoadTemplate(cMap, rootPath, jobPath)
	if err != nil {
		return err
	}

	PgBasebackupRestoreJobTemplate, err = c.LoadTemplate(cMap, rootPath, pgBasebackupRestoreJobTemplatePath)
	if err != nil {
		return err
	}

	PgpoolTemplate, err = c.LoadTemplate(cMap, rootPath, pgpoolTemplatePath)
	if err != nil {
		return err
	}

	PgpoolConfTemplate, err = c.LoadTemplate(cMap, rootPath, pgpoolConfTemplatePath)
	if err != nil {
		return err
	}

	PgpoolPasswdTemplate, err = c.LoadTemplate(cMap, rootPath, pgpoolPasswdTemplatePath)
	if err != nil {
		return err
	}

	PgpoolHBATemplate, err = c.LoadTemplate(cMap, rootPath, pgpoolHBATemplatePath)
	if err != nil {
		return err
	}

	PgbouncerTemplate, err = c.LoadTemplate(cMap, rootPath, pgbouncerTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerConfTemplate, err = c.LoadTemplate(cMap, rootPath, pgbouncerConfTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerUsersTemplate, err = c.LoadTemplate(cMap, rootPath, pgbouncerUsersTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerHBATemplate, err = c.LoadTemplate(cMap, rootPath, pgbouncerHBATemplatePath)
	if err != nil {
		return err
	}

	ServiceTemplate, err = c.LoadTemplate(cMap, rootPath, serviceTemplatePath)
	if err != nil {
		return err
	}

	RmdatajobTemplate, err = c.LoadTemplate(cMap, rootPath, rmdatajobPath)
	if err != nil {
		return err
	}

	BackrestjobTemplate, err = c.LoadTemplate(cMap, rootPath, backrestjobPath)
	if err != nil {
		return err
	}

	BackrestRestorejobTemplate, err = c.LoadTemplate(cMap, rootPath, backrestRestorejobPath)
	if err != nil {
		return err
	}

	PgDumpBackupJobTemplate, err = c.LoadTemplate(cMap, rootPath, pgDumpBackupJobPath)
	if err != nil {
		return err
	}

	PgRestoreJobTemplate, err = c.LoadTemplate(cMap, rootPath, pgRestoreJobPath)
	if err != nil {
		return err
	}

	PVCMatchLabelsTemplate, err = c.LoadTemplate(cMap, rootPath, pvcMatchLabelsPath)
	if err != nil {
		return err
	}

	PVCStorageClassTemplate, err = c.LoadTemplate(cMap, rootPath, pvcSCPath)
	if err != nil {
		return err
	}

	AffinityTemplate, err = c.LoadTemplate(cMap, rootPath, affinityTemplatePath)
	if err != nil {
		return err
	}

	CollectTemplate, err = c.LoadTemplate(cMap, rootPath, collectTemplatePath)
	if err != nil {
		return err
	}

	BadgerTemplate, err = c.LoadTemplate(cMap, rootPath, badgerTemplatePath)
	if err != nil {
		return err
	}

	DeploymentTemplate, err = c.LoadTemplate(cMap, rootPath, deploymentTemplatePath)
	return err
}

func getRootPath(clientset *kubernetes.Clientset, namespace string) (*v1.ConfigMap, string) {

	cMap, found := kubeapi.GetConfigMap(clientset, CustomConfigMapName, namespace)
	if found {
		log.Infof("Config: %s ConfigMap found, using config files from the configmap", CustomConfigMapName)
		return cMap, ""
	}
	log.Infof("Config: %s ConfigMap NOT found, using default baked-in config files from %s", CustomConfigMapName, DefaultConfigsPath)

	return nil, DefaultConfigsPath
}

// LoadTemplate will load a JSON template from a path
func (c *PgoConfig) LoadTemplate(cMap *v1.ConfigMap, rootPath, path string) (*template.Template, error) {
	var value string

	if cMap != nil {
		value = cMap.Data[path]
		if value == "" {
			errMsg := fmt.Sprintf("%s path not found in ConfigMap", path)
			return nil, errors.New(errMsg)
		}
	} else {
		fullPath := rootPath + path
		log.Debugf("loading path [%s]", fullPath)
		buf, err := ioutil.ReadFile(fullPath)
		if err != nil {
			log.Errorf("error: could not read %s", fullPath)
			log.Error(err)
			return nil, err
		}
		value = string(buf)
	}

	return template.Must(template.New(path).Parse(value)), nil

}

func (c *PgoConfig) SetDefaultStorageClass(clientset *kubernetes.Clientset) error {

	selector := LABEL_PGO_DEFAULT_SC + "=true"
	scList, err := kubeapi.GetStorageClasses(clientset, selector)
	if err != nil {
		return err
	}

	if len(scList.Items) == 0 {
		//no pgo default sc was found, so we will use 1st sc we find
		scList, err = kubeapi.GetAllStorageClasses(clientset)
		if err != nil {
			return err
		}
		if len(scList.Items) == 0 {
			return errors.New("no storage classes were found on this Kube system")
		}
		//configure with the 1st SC on the system
	} else {
		//configure with the default pgo sc
	}

	log.Infof("setting pgo-default-sc to %s", scList.Items[0].Name)

	//add the storage class into the config
	c.Storage[LABEL_PGO_DEFAULT_SC] = StorageStruct{
		AccessMode:         "ReadWriteOnce",
		Size:               "1G",
		StorageType:        "dynamic",
		StorageClass:       scList.Items[0].Name,
		Fsgroup:            "26",
		SupplementalGroups: "",
		MatchLabels:        "",
	}

	//set the default storage configs to this new one
	c.PrimaryStorage = LABEL_PGO_DEFAULT_SC
	c.BackupStorage = LABEL_PGO_DEFAULT_SC
	c.ReplicaStorage = LABEL_PGO_DEFAULT_SC
	c.BackrestStorage = LABEL_PGO_DEFAULT_SC

	return nil
}

// CheckEnv is mostly used for the OLM deployment use case
// when someone wants to deploy with OLM, use the baked-in
// configuration, but use a different set of images, by
// setting these env vars in the OLM CSV, users can override
// the baked in images
func (c *PgoConfig) CheckEnv() {
	pgoImageTag := os.Getenv("PGO_IMAGE_TAG")
	if pgoImageTag != "" {
		c.Pgo.PGOImageTag = pgoImageTag
		log.Infof("CheckEnv: using PGO_IMAGE_TAG env var: %s", pgoImageTag)
	}
	pgoImagePrefix := os.Getenv("PGO_IMAGE_PREFIX")
	if pgoImagePrefix != "" {
		c.Pgo.PGOImagePrefix = pgoImagePrefix
		log.Infof("CheckEnv: using PGO_IMAGE_PREFIX env var: %s", pgoImagePrefix)
	}
	ccpImageTag := os.Getenv("CCP_IMAGE_TAG")
	if ccpImageTag != "" {
		c.Cluster.CCPImageTag = ccpImageTag
		log.Infof("CheckEnv: using CCP_IMAGE_TAG env var: %s", ccpImageTag)
	}
	ccpImagePrefix := os.Getenv("CCP_IMAGE_PREFIX")
	if ccpImagePrefix != "" {
		c.Cluster.CCPImagePrefix = ccpImagePrefix
		log.Infof("CheckEnv: using CCP_IMAGE_PREFIX env var: %s", ccpImagePrefix)
	}
}
