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
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"gopkg.in/yaml.v2"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// consolidate with cluster.affinityTemplateFields
const AffinityInOperator = "In"
const AFFINITY_NOTINOperator = "NotIn"

const DefaultArchiveTimeout = "60"

type affinityTemplateFields struct {
	NodeLabelKey   string
	NodeLabelValue string
	OperatorValue  string
}

// consolidate
type collectTemplateFields struct {
	Name           string
	JobName        string
	CCPImageTag    string
	CCPImagePrefix string
	PgPort         string
	ExporterPort   string
}

//consolidate
type badgerTemplateFields struct {
	CCPImageTag        string
	CCPImagePrefix     string
	BadgerTarget       string
	PGBadgerPort       string
	ContainerResources string
}

type PgbackrestEnvVarsTemplateFields struct {
	PgbackrestStanza            string
	PgbackrestDBPath            string
	PgbackrestRepo1Path         string
	PgbackrestRepo1Host         string
	PgbackrestRepo1Type         string
	PgbackrestLocalAndS3Storage bool
	PgbackrestPGPort            string
}

type PgbackrestS3EnvVarsTemplateFields struct {
	PgbackrestS3Bucket    string
	PgbackrestS3Endpoint  string
	PgbackrestS3Region    string
	PgbackrestS3Key       string
	PgbackrestS3KeySecret string
}

type PgmonitorEnvVarsTemplateFields struct {
	PgmonitorPassword string
}

// needs to be consolidated with cluster.DeploymentTemplateFields
// DeploymentTemplateFields ...
type DeploymentTemplateFields struct {
	Name                string
	ClusterName         string
	Port                string
	CCPImagePrefix      string
	CCPImageTag         string
	CCPImage            string
	Database            string
	DeploymentLabels    string
	PodLabels           string
	DataPathOverride    string
	ArchiveMode         string
	ArchivePVCName      string
	XLOGDir             string
	BackrestPVCName     string
	PVCName             string
	RootSecretName      string
	UserSecretName      string
	PrimarySecretName   string
	SecurityContext     string
	ContainerResources  string
	NodeSelector        string
	ConfVolume          string
	CollectAddon        string
	CollectVolume       string
	BadgerAddon         string
	PgbackrestEnvVars   string
	PgbackrestS3EnvVars string
	PgmonitorEnvVars    string
	ScopeLabel          string
	//next 2 are for the replica deployment only
	Replicas    string
	PrimaryHost string
	// PgBouncer deployment only
	PgbouncerPass string
	IsReplica     bool
}

type PostgresHaTemplateFields struct {
	LogStatement            string
	LogMinDurationStatement string
	ArchiveTimeout          string
}

//consolidate with cluster.GetPgbackrestEnvVars
func GetPgbackrestEnvVars(backrestEnabled, clusterName, depName, port, storageType string) string {
	if backrestEnabled == "true" {
		fields := PgbackrestEnvVarsTemplateFields{
			PgbackrestStanza:            "db",
			PgbackrestRepo1Host:         clusterName + "-backrest-shared-repo",
			PgbackrestRepo1Path:         "/backrestrepo/" + clusterName + "-backrest-shared-repo",
			PgbackrestDBPath:            "/pgdata/" + depName,
			PgbackrestPGPort:            port,
			PgbackrestRepo1Type:         GetRepoType(storageType),
			PgbackrestLocalAndS3Storage: IsLocalAndS3Storage(storageType),
		}

		var doc bytes.Buffer
		err := config.PgbackrestEnvVarsTemplate.Execute(&doc, fields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}
		return doc.String()
	}
	return ""

}

func GetBadgerAddon(clientset *kubernetes.Clientset, namespace string, cluster *crv1.Pgcluster, pgbadger_target string) string {

	spec := cluster.Spec

	if cluster.Labels[config.LABEL_BADGER] == "true" {
		log.Debug("crunchy_badger was found as a label on cluster create")
		badgerTemplateFields := badgerTemplateFields{}
		badgerTemplateFields.CCPImageTag = spec.CCPImageTag
		badgerTemplateFields.BadgerTarget = pgbadger_target
		badgerTemplateFields.PGBadgerPort = spec.PGBadgerPort
		badgerTemplateFields.CCPImagePrefix = Pgo.Cluster.CCPImagePrefix
		badgerTemplateFields.ContainerResources = ""

		if Pgo.DefaultBadgerResources != "" {
			tmp, err := Pgo.GetContainerResource(Pgo.DefaultBadgerResources)
			if err != nil {
				log.Error(err)
				return ""
			}
			badgerTemplateFields.ContainerResources = GetContainerResourcesJSON(&tmp)

		}

		var badgerDoc bytes.Buffer
		err := config.BadgerTemplate.Execute(&badgerDoc, badgerTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if CRUNCHY_DEBUG {
			config.BadgerTemplate.Execute(os.Stdout, badgerTemplateFields)
		}
		return badgerDoc.String()
	}
	return ""
}

func GetCollectAddon(clientset *kubernetes.Clientset, namespace string, spec *crv1.PgclusterSpec) string {

	if spec.UserLabels[config.LABEL_COLLECT] == "true" {
		log.Debug("crunchy_collect was found as a label on cluster create")

		log.Debug("creating collect secret for cluster %s", spec.Name)
		err := util.CreateSecret(clientset, spec.Name, spec.CollectSecretName, config.LABEL_COLLECT_PG_USER,
			Pgo.Cluster.PgmonitorPassword, namespace)

		collectTemplateFields := collectTemplateFields{}
		collectTemplateFields.Name = spec.Name
		collectTemplateFields.JobName = spec.Name
		collectTemplateFields.CCPImageTag = spec.CCPImageTag
		collectTemplateFields.ExporterPort = spec.ExporterPort
		collectTemplateFields.CCPImagePrefix = Pgo.Cluster.CCPImagePrefix
		collectTemplateFields.PgPort = spec.Port

		var collectDoc bytes.Buffer
		err = config.CollectTemplate.Execute(&collectDoc, collectTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if CRUNCHY_DEBUG {
			config.CollectTemplate.Execute(os.Stdout, collectTemplateFields)
		}
		return collectDoc.String()
	}
	return ""
}

//consolidate with cluster.GetConfVolume
func GetConfVolume(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string) string {
	var found bool
	var configMapStr string

	//check for global custom configmap "pgo-custom-pg-config"
	configMap, found := kubeapi.GetConfigMap(clientset, config.GLOBAL_CUSTOM_CONFIGMAP, PgoNamespace)
	if found {
		configMapStr = "\"pgo-custom-pg-config\""
	} else {
		log.Debug(config.GLOBAL_CUSTOM_CONFIGMAP + " was not found, skipping global configMap")

		//check for user provided configmap
		if cl.Spec.CustomConfig != "" {
			configMap, found = kubeapi.GetConfigMap(clientset, cl.Spec.CustomConfig, namespace)
			if !found {
				//you should NOT get this error because of apiserver validation of this value!
				log.Errorf("%s was not found, error, skipping user provided configMap", cl.Spec.CustomConfig)
			} else {
				log.Debugf("user provided configmap %s was used for this cluster", cl.Spec.CustomConfig)
				configMapStr = "\"" + cl.Spec.CustomConfig + "\""
			}
		}
	}

	if configMap == nil {
		log.Debugf("Custom postgres-ha configmap not found, creating default configmap")
		addDefaultPostgresHaConfigMap(clientset, cl, namespace)
	} else if configMap != nil {
		if _, exists := configMap.Data[config.PostgresHaTemplatePath]; !exists {
			log.Debugf("Custom postgres-ha configmap not found, creating default configmap")
			addDefaultPostgresHaConfigMap(clientset, cl, namespace)
		} else {
			log.Debugf("Custom postgres-ha configmap found, default configmap will not be created")
		}
	}

	return configMapStr
}

func addDefaultPostgresHaConfigMap(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, namespace string) error {

	var archiveTimeout string

	if _, exists := cluster.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]; !exists {
		archiveTimeout = cluster.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]
	} else {
		archiveTimeout = DefaultArchiveTimeout
	}

	labels := make(map[string]string)
	labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY

	data := make(map[string]string)

	postgresHaFields := PostgresHaTemplateFields{
		LogStatement:            Pgo.Cluster.LogStatement,
		LogMinDurationStatement: Pgo.Cluster.LogMinDurationStatement,
		ArchiveTimeout:          archiveTimeout,
	}

	var postgresHaConfig bytes.Buffer

	err := config.PostgresHaTemplate.Execute(&postgresHaConfig, postgresHaFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	data[config.PostgresHaTemplatePath] = postgresHaConfig.String()

	configmap := &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   "pgo-pgha-default-config",
			Labels: labels,
		},
		Data: data,
	}

	err = kubeapi.CreateConfigMap(clientset, configmap, namespace)
	if err != nil {
		return err
	}

	return nil
}

// sets the proper collect secret in the deployment spec if collect is enabled
func GetCollectVolume(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string) string {
	if cl.Spec.UserLabels[config.LABEL_COLLECT] == "true" {
		return "\"secret\": { \"secretName\": \"" + cl.Spec.CollectSecretName + "\" }"
	}

	return "\"emptyDir\": { \"secretName\": \"Memory\" }"
}

// needs to be consolidated with cluster.GetLabelsFromMap
// GetLabelsFromMap ...
func GetLabelsFromMap(labels map[string]string) string {
	var output string

	mapLen := len(labels)
	i := 1
	for key, value := range labels {
		if len(validation.IsQualifiedName(key)) == 0 && len(validation.IsValidLabelValue(value)) == 0 {
			output += fmt.Sprintf("\"%s\": \"%s\"", key, value)
			if i < mapLen {
				output += ","
			}
		}
		i++
	}
	return output
}

// GetPrimaryLabels ...
/**
func GetPrimaryLabels(serviceName string, ClusterName string, replicaFlag bool, userLabels map[string]string) map[string]string {
	primaryLabels := make(map[string]string)

	primaryLabels["name"] = serviceName
	primaryLabels[config.LABEL_PG_CLUSTER] = ClusterName

	for key, value := range userLabels {
		if key == config.LABEL_PGPOOL || key == config.LABEL_PGBOUNCER {
			//these dont apply to a primary or replica
		} else if key == config.LABEL_AUTOFAIL || key == config.LABEL_NODE_LABEL_KEY || key == config.LABEL_NODE_LABEL_VALUE ||
			key == config.LABEL_BACKREST_STORAGE_TYPE {
			//dont add these since they can break label expression checks
			//or autofail toggling
		} else {
			log.Debugf("JEFF label copying XXX key=%s value=%s", key, value)
			primaryLabels[key] = value
		}
	}

	return primaryLabels
}
*/

// GetAffinity ...
func GetAffinity(nodeLabelKey, nodeLabelValue string, affoperator string) string {
	log.Debugf("GetAffinity with nodeLabelKey=[%s] nodeLabelKey=[%s] and operator=[%s]\n", nodeLabelKey, nodeLabelValue, affoperator)
	output := ""
	if nodeLabelKey == "" {
		return output
	}

	affinityTemplateFields := affinityTemplateFields{}
	affinityTemplateFields.NodeLabelKey = nodeLabelKey
	affinityTemplateFields.NodeLabelValue = nodeLabelValue
	affinityTemplateFields.OperatorValue = affoperator

	var affinityDoc bytes.Buffer
	err := config.AffinityTemplate.Execute(&affinityDoc, affinityTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return output
	}

	if CRUNCHY_DEBUG {
		config.AffinityTemplate.Execute(os.Stdout, affinityTemplateFields)
	}

	return affinityDoc.String()
}

// GetReplicaAffinity ...
// use NotIn as an operator when a node-label is not specified on the
// replica, use the node labels from the primary in this case
// use In as an operator when a node-label is specified on the replica
// use the node labels from the replica in this case
func GetReplicaAffinity(clusterLabels, replicaLabels map[string]string) string {
	var operator, key, value string
	log.Debug("GetReplicaAffinity ")
	if replicaLabels[config.LABEL_NODE_LABEL_KEY] != "" {
		//use the replica labels
		operator = "In"
		key = replicaLabels[config.LABEL_NODE_LABEL_KEY]
		value = replicaLabels[config.LABEL_NODE_LABEL_VALUE]
	} else {
		//use the cluster labels
		operator = "NotIn"
		key = clusterLabels[config.LABEL_NODE_LABEL_KEY]
		value = clusterLabels[config.LABEL_NODE_LABEL_VALUE]
	}
	return GetAffinity(key, value, operator)
}

func GetPgmonitorEnvVars(metricsEnabled string) string {
	if metricsEnabled == "true" {
		fields := PgmonitorEnvVarsTemplateFields{
			PgmonitorPassword: Pgo.Cluster.PgmonitorPassword,
		}

		var doc bytes.Buffer
		err := config.PgmonitorEnvVarsTemplate.Execute(&doc, fields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}
		return doc.String()
	}
	return ""

}

func GetPgbackrestS3EnvVars(backrestLabel, backRestStorageTypeLabel string,
	clientset *kubernetes.Clientset, ns string) string {

	if backrestLabel == "true" && strings.Contains(backRestStorageTypeLabel, "s3") {

		s3EnvVars := PgbackrestS3EnvVarsTemplateFields{
			PgbackrestS3Bucket:   Pgo.Cluster.BackrestS3Bucket,
			PgbackrestS3Endpoint: Pgo.Cluster.BackrestS3Endpoint,
			PgbackrestS3Region:   Pgo.Cluster.BackrestS3Region,
		}

		secret, secretExists, err := kubeapi.GetSecret(clientset, "pgo-backrest-repo-config", PgoNamespace)
		if err != nil {
			log.Error(err.Error())
			return ""
		} else if !secretExists {
			log.Error("Secret 'pgo-backrest-repo-config' does not exist. Unable to set S3 env vars for pgBackRest")
			return ""
		}

		data := struct {
			Key       string `yaml:"aws-s3-key"`
			KeySecret string `yaml:"aws-s3-key-secret"`
		}{}

		err = yaml.Unmarshal(secret.Data["aws-s3-credentials.yaml"], &data)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		s3EnvVars.PgbackrestS3Key = data.Key
		s3EnvVars.PgbackrestS3KeySecret = data.KeySecret

		var b bytes.Buffer
		err = config.PgbackrestS3EnvVarsTemplate.Execute(&b, s3EnvVars)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		return b.String()
	}
	return ""
}
