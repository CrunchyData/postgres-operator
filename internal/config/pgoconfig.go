package config

/*
Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	CustomConfigMapName     = "pgo-config"
	defaultConfigPath       = "/default-pgo-config/"
	openShiftAPIGroupSuffix = ".openshift.io"
)

var PgoDefaultServiceAccountTemplate *template.Template

const PGODefaultServiceAccountPath = "pgo-default-sa.json"

var PgoTargetRoleBindingTemplate *template.Template

const PGOTargetRoleBindingPath = "pgo-target-role-binding.json"

var PgoBackrestServiceAccountTemplate *template.Template

const PGOBackrestServiceAccountPath = "pgo-backrest-sa.json"

var PgoTargetServiceAccountTemplate *template.Template

const PGOTargetServiceAccountPath = "pgo-target-sa.json"

var PgoBackrestRoleTemplate *template.Template

const PGOBackrestRolePath = "pgo-backrest-role.json"

var PgoBackrestRoleBindingTemplate *template.Template

const PGOBackrestRoleBindingPath = "pgo-backrest-role-binding.json"

var PgoTargetRoleTemplate *template.Template

const PGOTargetRolePath = "pgo-target-role.json"

var PgoPgServiceAccountTemplate *template.Template

const PGOPgServiceAccountPath = "pgo-pg-sa.json"

var PgoPgRoleTemplate *template.Template

const PGOPgRolePath = "pgo-pg-role.json"

var PgoPgRoleBindingTemplate *template.Template

const PGOPgRoleBindingPath = "pgo-pg-role-binding.json"

var PolicyJobTemplate *template.Template

const policyJobTemplatePath = "pgo.sqlrunner-template.json"

var PVCTemplate *template.Template

const pvcPath = "pvc.json"

var ContainerResourcesTemplate *template.Template

const containerResourcesTemplatePath = "container-resources.json"

var PodAntiAffinityTemplate *template.Template

const podAntiAffinityTemplatePath = "pod-anti-affinity.json"

var PgoBackrestRepoServiceTemplate *template.Template

const pgoBackrestRepoServiceTemplatePath = "pgo-backrest-repo-service-template.json"

var PgoBackrestRepoTemplate *template.Template

const pgoBackrestRepoTemplatePath = "pgo-backrest-repo-template.json"

var PgmonitorEnvVarsTemplate *template.Template

const pgmonitorEnvVarsPath = "pgmonitor-env-vars.json"

var PgbackrestEnvVarsTemplate *template.Template

const pgbackrestEnvVarsPath = "pgbackrest-env-vars.json"

var PgbackrestGCSEnvVarsTemplate *template.Template

const pgbackrestGCSEnvVarsPath = "pgbackrest-gcs-env-vars.json"

var PgbackrestS3EnvVarsTemplate *template.Template

const pgbackrestS3EnvVarsPath = "pgbackrest-s3-env-vars.json"

var PgAdminTemplate *template.Template

const pgAdminTemplatePath = "pgadmin-template.json"

var PgAdminServiceTemplate *template.Template

const pgAdminServiceTemplatePath = "pgadmin-service-template.json"

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

var PgDumpBackupJobTemplate *template.Template

const pgDumpBackupJobPath = "pgdump-job.json"

var PgRestoreJobTemplate *template.Template

const pgRestoreJobPath = "pgrestore-job.json"

var PVCMatchLabelsTemplate *template.Template

const pvcMatchLabelsPath = "pvc-matchlabels.json"

var PVCStorageClassTemplate *template.Template

const pvcSCPath = "pvc-storageclass.json"

var ExporterTemplate *template.Template

const exporterTemplatePath = "exporter.json"

var BadgerTemplate *template.Template

const badgerTemplatePath = "pgbadger.json"

var DeploymentTemplate *template.Template

const deploymentTemplatePath = "cluster-deployment.json"

var BootstrapTemplate *template.Template

const bootstrapTemplatePath = "cluster-bootstrap-job.json"

type ClusterStruct struct {
	CCPImagePrefix                 string
	CCPImageTag                    string
	Policies                       string
	Metrics                        bool
	Badger                         bool
	Port                           string
	PGBadgerPort                   string
	ExporterPort                   string
	User                           string
	Database                       string
	PasswordAgeDays                string
	PasswordLength                 string
	Replicas                       string
	ServiceType                    v1.ServiceType
	BackrestPort                   int
	BackrestGCSBucket              string
	BackrestGCSEndpoint            string
	BackrestGCSKeyType             string
	BackrestS3Bucket               string
	BackrestS3Endpoint             string
	BackrestS3Region               string
	BackrestS3URIStyle             string
	BackrestS3VerifyTLS            string
	DisableAutofail                bool
	DisableReplicaStartFailReinit  bool
	PodAntiAffinity                string
	PodAntiAffinityPgBackRest      string
	PodAntiAffinityPgBouncer       string
	SyncReplication                bool
	DefaultInstanceResourceMemory  resource.Quantity `json:"DefaultInstanceMemory"`
	DefaultBackrestResourceMemory  resource.Quantity `json:"DefaultBackrestMemory"`
	DefaultPgBouncerResourceMemory resource.Quantity `json:"DefaultPgBouncerMemory"`
	DefaultExporterResourceMemory  resource.Quantity `json:"DefaultExporterMemory"`
	DisableFSGroup                 *bool
}

type StorageStruct struct {
	AccessMode         string
	Size               string
	StorageType        string
	StorageClass       string
	SupplementalGroups string
	MatchLabels        string
}

// PgoStruct defines various configuration settings for the PostgreSQL Operator
type PgoStruct struct {
	Audit                          bool
	ConfigMapWorkerCount           *int
	ControllerGroupRefreshInterval *int
	DisableReconcileRBAC           bool
	NamespaceRefreshInterval       *int
	NamespaceWorkerCount           *int
	PGClusterWorkerCount           *int
	PGOImagePrefix                 string
	PGOImageTag                    string
	PGReplicaWorkerCount           *int
	PGTaskWorkerCount              *int
}

type PgoConfig struct {
	BasicAuth       string
	Cluster         ClusterStruct
	Pgo             PgoStruct
	PrimaryStorage  string
	WALStorage      string
	BackupStorage   string
	ReplicaStorage  string
	BackrestStorage string
	PGAdminStorage  string
	Storage         map[string]StorageStruct
	OpenShift       bool
}

const (
	DefaultServiceType = v1.ServiceTypeClusterIP
	CONFIG_PATH        = "pgo.yaml"
)

const (
	DEFAULT_BACKREST_PORT = 2022
	DEFAULT_PGADMIN_PORT  = "5050"
	DEFAULT_PGBADGER_PORT = "10000"
	DEFAULT_EXPORTER_PORT = "9187"
	DEFAULT_POSTGRES_PORT = "5432"
	DEFAULT_PATRONI_PORT  = "8009"
)

func (c *PgoConfig) Validate() error {
	var err error
	errPrefix := "Error in pgoconfig: check pgo.yaml: "

	if c.Cluster.BackrestPort == 0 {
		c.Cluster.BackrestPort = DEFAULT_BACKREST_PORT
		log.Infof("setting BackrestPort to default %d", c.Cluster.BackrestPort)
	}
	if c.Cluster.PGBadgerPort == "" {
		c.Cluster.PGBadgerPort = DEFAULT_PGBADGER_PORT
		log.Infof("setting PGBadgerPort to default %s", c.Cluster.PGBadgerPort)
	} else {
		if _, err := strconv.Atoi(c.Cluster.PGBadgerPort); err != nil {
			return errors.New(errPrefix + "Invalid PGBadgerPort: " + err.Error())
		}
	}
	if c.Cluster.ExporterPort == "" {
		c.Cluster.ExporterPort = DEFAULT_EXPORTER_PORT
		log.Infof("setting ExporterPort to default %s", c.Cluster.ExporterPort)
	} else {
		if _, err := strconv.Atoi(c.Cluster.ExporterPort); err != nil {
			return errors.New(errPrefix + "Invalid ExporterPort: " + err.Error())
		}
	}
	if c.Cluster.Port == "" {
		c.Cluster.Port = DEFAULT_POSTGRES_PORT
		log.Infof("setting Postgres Port to default %s", c.Cluster.Port)
	} else {
		if _, err := strconv.Atoi(c.Cluster.Port); err != nil {
			return errors.New(errPrefix + "Invalid Port: " + err.Error())
		}
	}

	{
		storageNotDefined := func(setting, value string) error {
			return fmt.Errorf("%s%s setting is invalid: %q is not defined", errPrefix, setting, value)
		}
		if _, ok := c.Storage[c.PrimaryStorage]; !ok {
			return storageNotDefined("PrimaryStorage", c.PrimaryStorage)
		}
		if _, ok := c.Storage[c.BackrestStorage]; !ok {
			log.Warning("BackrestStorage setting not set, will use PrimaryStorage setting")
			c.Storage[c.BackrestStorage] = c.Storage[c.PrimaryStorage]
		}
		if _, ok := c.Storage[c.BackupStorage]; !ok {
			return storageNotDefined("BackupStorage", c.BackupStorage)
		}
		if _, ok := c.Storage[c.ReplicaStorage]; !ok {
			return storageNotDefined("ReplicaStorage", c.ReplicaStorage)
		}
		if _, ok := c.Storage[c.PGAdminStorage]; !ok {
			log.Warning("PGAdminStorage setting not set, will use PrimaryStorage setting")
			c.Storage[c.PGAdminStorage] = c.Storage[c.PrimaryStorage]
		}
		if _, ok := c.Storage[c.WALStorage]; c.WALStorage != "" && !ok {
			return storageNotDefined("WALStorage", c.WALStorage)
		}
		for k := range c.Storage {
			_, err = c.GetStorageSpec(k)
			if err != nil {
				return err
			}
		}
	}

	if c.Pgo.PGOImagePrefix == "" {
		return errors.New(errPrefix + "Pgo.PGOImagePrefix is required")
	}
	if c.Pgo.PGOImageTag == "" {
		return errors.New(errPrefix + "Pgo.PGOImageTag is required")
	}

	// if ServiceType is set, ensure it is valid
	switch c.Cluster.ServiceType {
	default:
		return fmt.Errorf("Cluster.ServiceType is an invalid ServiceType: %q", c.Cluster.ServiceType)
	case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
		v1.ServiceTypeLoadBalancer, v1.ServiceTypeExternalName, "": // no-op
	}

	if c.Cluster.CCPImagePrefix == "" {
		return errors.New(errPrefix + "Cluster.CCPImagePrefix is required")
	}

	if c.Cluster.CCPImageTag == "" {
		return errors.New(errPrefix + "Cluster.CCPImageTag is required")
	}

	if c.Cluster.User == "" {
		return errors.New(errPrefix + "Cluster.User is required")
	} else {
		// validates that username can be used as the kubernetes secret name
		// Must consist of lower case alphanumeric characters,
		// '-' or '.', and must start and end with an alphanumeric character
		errs := validation.IsDNS1123Subdomain(c.Cluster.User)
		if len(errs) > 0 {
			var msg string
			for i := range errs {
				msg = msg + errs[i]
			}
			return errors.New(errPrefix + msg)
		}

		// validate any of the resources and if they are unavailable, set defaults
		if c.Cluster.DefaultInstanceResourceMemory.IsZero() {
			c.Cluster.DefaultInstanceResourceMemory = DefaultInstanceResourceMemory
		}

		log.Infof("default instance memory set to [%s]", c.Cluster.DefaultInstanceResourceMemory.String())

		if c.Cluster.DefaultBackrestResourceMemory.IsZero() {
			c.Cluster.DefaultBackrestResourceMemory = DefaultBackrestResourceMemory
		}

		log.Infof("default pgbackrest repository memory set to [%s]", c.Cluster.DefaultBackrestResourceMemory.String())

		if c.Cluster.DefaultPgBouncerResourceMemory.IsZero() {
			c.Cluster.DefaultPgBouncerResourceMemory = DefaultPgBouncerResourceMemory
		}

		log.Infof("default pgbouncer memory set to [%s]", c.Cluster.DefaultPgBouncerResourceMemory.String())
	}

	// if provided, ensure that the type of pod anti-affinity values are valid
	podAntiAffinityType := crv1.PodAntiAffinityType(c.Cluster.PodAntiAffinity)
	if err := podAntiAffinityType.Validate(); err != nil {
		return errors.New(errPrefix + "Invalid value provided for Cluster.PodAntiAffinityType")
	}

	podAntiAffinityType = crv1.PodAntiAffinityType(c.Cluster.PodAntiAffinityPgBackRest)
	if err := podAntiAffinityType.Validate(); err != nil {
		return errors.New(errPrefix + "Invalid value provided for Cluster.PodAntiAffinityPgBackRest")
	}

	podAntiAffinityType = crv1.PodAntiAffinityType(c.Cluster.PodAntiAffinityPgBouncer)
	if err := podAntiAffinityType.Validate(); err != nil {
		return errors.New(errPrefix + "Invalid value provided for Cluster.PodAntiAffinityPgBouncer")
	}

	return err
}

// GetPodAntiAffinitySpec accepts possible user-defined values for what the
// pod anti-affinity spec should be, which include rules for:
// - PostgreSQL instances
// - pgBackRest
// - pgBouncer
func (c *PgoConfig) GetPodAntiAffinitySpec(cluster, pgBackRest, pgBouncer crv1.PodAntiAffinityType) (crv1.PodAntiAffinitySpec, error) {
	spec := crv1.PodAntiAffinitySpec{}

	// first, set the values for the PostgreSQL cluster, which is the "default"
	// value. Otherwise, set the default to that in the configuration
	if cluster != "" {
		spec.Default = cluster
	} else {
		spec.Default = crv1.PodAntiAffinityType(c.Cluster.PodAntiAffinity)
	}

	// perform a validation check against the default type
	if err := spec.Default.Validate(); err != nil {
		log.Error(err)
		return spec, err
	}

	// now that the default is set, determine if the user or the configuration
	// overrode the settings for pgBackRest and pgBouncer. The heuristic is as
	// such:
	//
	// 1. If the user provides a value, use that value
	// 2. If there is a value provided in the configuration, use that value
	// 3. If there is a value in the cluster default, use that value, which also
	//    encompasses using the default value in the config at this point in the
	//    execution.
	//
	// First, do pgBackRest:
	switch {
	case pgBackRest != "":
		spec.PgBackRest = pgBackRest
	case c.Cluster.PodAntiAffinityPgBackRest != "":
		spec.PgBackRest = crv1.PodAntiAffinityType(c.Cluster.PodAntiAffinityPgBackRest)
	case spec.Default != "":
		spec.PgBackRest = spec.Default
	}

	// perform a validation check against the pgBackRest type
	if err := spec.PgBackRest.Validate(); err != nil {
		log.Error(err)
		return spec, err
	}

	// Now, pgBouncer:
	switch {
	case pgBouncer != "":
		spec.PgBouncer = pgBouncer
	case c.Cluster.PodAntiAffinityPgBackRest != "":
		spec.PgBouncer = crv1.PodAntiAffinityType(c.Cluster.PodAntiAffinityPgBouncer)
	case spec.Default != "":
		spec.PgBouncer = spec.Default
	}

	// perform a validation check against the pgBackRest type
	if err := spec.PgBouncer.Validate(); err != nil {
		log.Error(err)
		return spec, err
	}

	return spec, nil
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
	storage.MatchLabels = s.MatchLabels
	storage.SupplementalGroups = s.SupplementalGroups

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

func (c *PgoConfig) GetConfig(clientset kubernetes.Interface, namespace string) error {
	cMap, err := initialize(clientset, namespace)
	if err != nil {
		log.Errorf("could not get ConfigMap: %s", err.Error())
		return err
	}

	// get the pgo.yaml config file
	str := cMap.Data[CONFIG_PATH]
	if str == "" {
		return fmt.Errorf("could not get %s from ConfigMap", CONFIG_PATH)
	}

	yamlFile := []byte(str)

	if err := yaml.Unmarshal(yamlFile, c); err != nil {
		log.Errorf("Unmarshal: %v", err)
		return err
	}

	// determine if this cluster is inside openshift
	c.OpenShift = isOpenShift(clientset)

	// validate the pgo.yaml config file
	if err := c.Validate(); err != nil {
		log.Error(err)
		return err
	}

	c.CheckEnv()

	// load up all the templates
	PgoDefaultServiceAccountTemplate, err = c.LoadTemplate(cMap, PGODefaultServiceAccountPath)
	if err != nil {
		return err
	}
	PgoBackrestServiceAccountTemplate, err = c.LoadTemplate(cMap, PGOBackrestServiceAccountPath)
	if err != nil {
		return err
	}
	PgoTargetServiceAccountTemplate, err = c.LoadTemplate(cMap, PGOTargetServiceAccountPath)
	if err != nil {
		return err
	}
	PgoTargetRoleBindingTemplate, err = c.LoadTemplate(cMap, PGOTargetRoleBindingPath)
	if err != nil {
		return err
	}
	PgoBackrestRoleTemplate, err = c.LoadTemplate(cMap, PGOBackrestRolePath)
	if err != nil {
		return err
	}
	PgoBackrestRoleBindingTemplate, err = c.LoadTemplate(cMap, PGOBackrestRoleBindingPath)
	if err != nil {
		return err
	}
	PgoTargetRoleTemplate, err = c.LoadTemplate(cMap, PGOTargetRolePath)
	if err != nil {
		return err
	}
	PgoPgServiceAccountTemplate, err = c.LoadTemplate(cMap, PGOPgServiceAccountPath)
	if err != nil {
		return err
	}
	PgoPgRoleTemplate, err = c.LoadTemplate(cMap, PGOPgRolePath)
	if err != nil {
		return err
	}
	PgoPgRoleBindingTemplate, err = c.LoadTemplate(cMap, PGOPgRoleBindingPath)
	if err != nil {
		return err
	}

	PVCTemplate, err = c.LoadTemplate(cMap, pvcPath)
	if err != nil {
		return err
	}

	PolicyJobTemplate, err = c.LoadTemplate(cMap, policyJobTemplatePath)
	if err != nil {
		return err
	}

	ContainerResourcesTemplate, err = c.LoadTemplate(cMap, containerResourcesTemplatePath)
	if err != nil {
		return err
	}

	PgoBackrestRepoServiceTemplate, err = c.LoadTemplate(cMap, pgoBackrestRepoServiceTemplatePath)
	if err != nil {
		return err
	}

	PgoBackrestRepoTemplate, err = c.LoadTemplate(cMap, pgoBackrestRepoTemplatePath)
	if err != nil {
		return err
	}

	PgmonitorEnvVarsTemplate, err = c.LoadTemplate(cMap, pgmonitorEnvVarsPath)
	if err != nil {
		return err
	}

	PgbackrestEnvVarsTemplate, err = c.LoadTemplate(cMap, pgbackrestEnvVarsPath)
	if err != nil {
		return err
	}

	PgbackrestGCSEnvVarsTemplate, err = c.LoadTemplate(cMap, pgbackrestGCSEnvVarsPath)
	if err != nil {
		return err
	}

	PgbackrestS3EnvVarsTemplate, err = c.LoadTemplate(cMap, pgbackrestS3EnvVarsPath)
	if err != nil {
		return err
	}

	PgAdminTemplate, err = c.LoadTemplate(cMap, pgAdminTemplatePath)
	if err != nil {
		return err
	}

	PgAdminServiceTemplate, err = c.LoadTemplate(cMap, pgAdminServiceTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerTemplate, err = c.LoadTemplate(cMap, pgbouncerTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerConfTemplate, err = c.LoadTemplate(cMap, pgbouncerConfTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerUsersTemplate, err = c.LoadTemplate(cMap, pgbouncerUsersTemplatePath)
	if err != nil {
		return err
	}

	PgbouncerHBATemplate, err = c.LoadTemplate(cMap, pgbouncerHBATemplatePath)
	if err != nil {
		return err
	}

	ServiceTemplate, err = c.LoadTemplate(cMap, serviceTemplatePath)
	if err != nil {
		return err
	}

	RmdatajobTemplate, err = c.LoadTemplate(cMap, rmdatajobPath)
	if err != nil {
		return err
	}

	BackrestjobTemplate, err = c.LoadTemplate(cMap, backrestjobPath)
	if err != nil {
		return err
	}

	PgDumpBackupJobTemplate, err = c.LoadTemplate(cMap, pgDumpBackupJobPath)
	if err != nil {
		return err
	}

	PgRestoreJobTemplate, err = c.LoadTemplate(cMap, pgRestoreJobPath)
	if err != nil {
		return err
	}

	PVCMatchLabelsTemplate, err = c.LoadTemplate(cMap, pvcMatchLabelsPath)
	if err != nil {
		return err
	}

	PVCStorageClassTemplate, err = c.LoadTemplate(cMap, pvcSCPath)
	if err != nil {
		return err
	}

	PodAntiAffinityTemplate, err = c.LoadTemplate(cMap, podAntiAffinityTemplatePath)
	if err != nil {
		return err
	}

	ExporterTemplate, err = c.LoadTemplate(cMap, exporterTemplatePath)
	if err != nil {
		return err
	}

	BadgerTemplate, err = c.LoadTemplate(cMap, badgerTemplatePath)
	if err != nil {
		return err
	}

	DeploymentTemplate, err = c.LoadTemplate(cMap, deploymentTemplatePath)
	if err != nil {
		return err
	}

	BootstrapTemplate, err = c.LoadTemplate(cMap, bootstrapTemplatePath)
	if err != nil {
		return err
	}

	return nil
}

// getOperatorConfigMap returns the config map that contains all of the
// configuration for the Operator
func getOperatorConfigMap(clientset kubernetes.Interface, namespace string) (*v1.ConfigMap, error) {
	ctx := context.TODO()

	return clientset.CoreV1().ConfigMaps(namespace).Get(ctx, CustomConfigMapName, metav1.GetOptions{})
}

// initialize attempts to get the configuration ConfigMap based on a name.
// If the ConfigMap does not exist, a ConfigMap is created from the default
// configuration path
func initialize(clientset kubernetes.Interface, namespace string) (*v1.ConfigMap, error) {
	ctx := context.TODO()

	// if the ConfigMap exists, exit
	if cm, err := getOperatorConfigMap(clientset, namespace); err == nil {
		log.Infof("Config: %q ConfigMap found, using config files from the configmap", CustomConfigMapName)
		return cm, nil
	}

	// otherwise, create a ConfigMap
	log.Infof("Config: %q ConfigMap NOT found, creating ConfigMap from files from %q", CustomConfigMapName, defaultConfigPath)

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: CustomConfigMapName,
			Labels: map[string]string{
				LABEL_VENDOR: LABEL_CRUNCHY,
			},
		},
		Data: map[string]string{},
	}

	// get all of the file names that are in the default configuration directory
	if err := filepath.Walk(defaultConfigPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip if a directory
		if info.IsDir() {
			return nil
		}

		// get all of the contents of a default configuration and load it into
		// a ConfigMap
		if contents, err := ioutil.ReadFile(path); err != nil {
			return err
		} else {
			cm.Data[info.Name()] = string(contents)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// create the ConfigMap. If the error is that the ConfigMap was already
	// created, then grab the new ConfigMap
	if _, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		if kerrors.IsAlreadyExists(err) {
			return getOperatorConfigMap(clientset, namespace)
		}

		return nil, err
	}

	return cm, nil
}

// LoadTemplate will load a JSON template from a path
func (c *PgoConfig) LoadTemplate(cMap *v1.ConfigMap, path string) (*template.Template, error) {
	var value string
	var err error

	// Determine if there exists a configmap entry for the template file.
	if cMap != nil {
		// Get the data that is stored in the configmap
		value = cMap.Data[path]
	}

	// if the configmap does not exist, or there is no data in the configmap for
	// this particular configuration template, attempt to load the template from
	// the default configuration
	if cMap == nil || value == "" {
		value, err = c.DefaultTemplate(path)

		if err != nil {
			return nil, err
		}
	}

	// if we have a value for the templated file, return
	return template.Must(template.New(path).Parse(value)), nil
}

// DefaultTemplate attempts to load a default configuration template file
func (c *PgoConfig) DefaultTemplate(path string) (string, error) {
	// set the lookup value for the file path based on the default configuration
	// path and the template file requested to be loaded
	fullPath := defaultConfigPath + path

	log.Debugf("No entry in cmap loading default path [%s]", fullPath)

	// read in the file from the default path
	buf, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Errorf("error: could not read %s", fullPath)
		log.Error(err)
		return "", err
	}

	// extract the value of the default configuration file and return
	value := string(buf)

	return value, nil
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

// HasDisableFSGroup returns either the value of DisableFSGroup if it is
// explicitly set; otherwise it will determine the value from the environment
func (c *PgoConfig) DisableFSGroup() bool {
	if c.Cluster.DisableFSGroup != nil {
		log.Debugf("setting disable fsgroup to %t", *c.Cluster.DisableFSGroup)
		return *c.Cluster.DisableFSGroup
	}

	// if this is OpenShift, disable the FSGroup
	log.Debugf("setting disable fsgroup to %t", c.OpenShift)
	return c.OpenShift
}

// isOpenShift returns true if we've detected that we're in an OpenShift cluster
func isOpenShift(clientset kubernetes.Interface) bool {
	groups, _, err := clientset.Discovery().ServerGroupsAndResources()

	if err != nil {
		log.Errorf("could not get server api groups: %s", err.Error())
		return false
	}

	// ff we detect that any API group name ends with "openshift.io", we'll return
	// that this is an OpenShift environment
	for _, g := range groups {
		if strings.HasSuffix(g.Name, openShiftAPIGroupSuffix) {
			return true
		}
	}

	return false
}
