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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"os"
)

// consolidate with cluster.affinityTemplateFields
const AffinityInOperator = "In"
const AFFINITY_NOTINOperator = "NotIn"

type affinityTemplateFields struct {
	NodeLabelKey   string
	NodeLabelValue string
	OperatorValue  string
}

// consolidate
type collectTemplateFields struct {
	Name            string
	JobName         string
	PrimaryPassword string
	CCPImageTag     string
	CCPImagePrefix  string
}

//consolidate
type badgerTemplateFields struct {
	CCPImageTag        string
	CCPImagePrefix     string
	BadgerTarget       string
	ContainerResources string
}

type PgbackrestEnvVarsTemplateFields struct {
	PgbackrestStanza    string
	PgbackrestDBPath    string
	PgbackrestRepo1Path string
	PgbackrestRepo1Host string
	PgbackrestPGPort    string
}

type PgmonitorEnvVarsTemplateFields struct {
	PgmonitorPassword string
}

// needs to be consolidated with cluster.DeploymentTemplateFields
// DeploymentTemplateFields ...
type DeploymentTemplateFields struct {
	Name                    string
	ClusterName             string
	Port                    string
	PgMode                  string
	LogStatement            string
	LogMinDurationStatement string
	CCPImagePrefix          string
	CCPImageTag             string
	CCPImage                string
	Database                string
	DeploymentLabels        string
	PodLabels               string
	DataPathOverride        string
	ArchiveMode             string
	ArchivePVCName          string
	ArchiveTimeout          string
	XLOGDir                 string
	BackrestPVCName         string
	PVCName                 string
	BackupPVCName           string
	BackupPath              string
	RootSecretName          string
	UserSecretName          string
	PrimarySecretName       string
	SecurityContext         string
	ContainerResources      string
	NodeSelector            string
	ConfVolume              string
	CollectAddon            string
	BadgerAddon             string
	PgbackrestEnvVars       string
	PgmonitorEnvVars        string
	//next 2 are for the replica deployment only
	Replicas    string
	PrimaryHost string
}

//consolidate with cluster.GetPgbackrestEnvVars
func GetPgbackrestEnvVars(backrestEnabled, clusterName, depName, port string) string {
	if backrestEnabled == "true" {
		fields := PgbackrestEnvVarsTemplateFields{
			PgbackrestStanza:    "db",
			PgbackrestRepo1Host: clusterName + "-backrest-shared-repo",
			PgbackrestRepo1Path: "/backrestrepo/" + clusterName + "-backrest-shared-repo",
			PgbackrestDBPath:    "/pgdata/" + depName,
			PgbackrestPGPort:    port,
		}

		var doc bytes.Buffer
		err := PgbackrestEnvVarsTemplate.Execute(&doc, fields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}
		return doc.String()
	}
	return ""

}

func GetBadgerAddon(clientset *kubernetes.Clientset, namespace string, spec *crv1.PgclusterSpec) string {

	if spec.UserLabels[util.LABEL_BADGER] == "true" {
		log.Debug("crunchy_badger was found as a label on cluster create")
		badgerTemplateFields := badgerTemplateFields{}
		badgerTemplateFields.CCPImageTag = spec.CCPImageTag
		badgerTemplateFields.BadgerTarget = spec.Name
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
		err := BadgerTemplate1.Execute(&badgerDoc, badgerTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if CRUNCHY_DEBUG {
			BadgerTemplate1.Execute(os.Stdout, badgerTemplateFields)
		}
		return badgerDoc.String()
	}
	return ""
}

func GetCollectAddon(clientset *kubernetes.Clientset, namespace string, spec *crv1.PgclusterSpec) string {

	if spec.UserLabels[util.LABEL_COLLECT] == "true" {
		log.Debug("crunchy_collect was found as a label on cluster create")
		_, PrimaryPassword, err3 := util.GetPasswordFromSecret(clientset, namespace, spec.PrimarySecretName)
		if err3 != nil {
			log.Error(err3)
		}

		collectTemplateFields := collectTemplateFields{}
		collectTemplateFields.Name = spec.Name
		collectTemplateFields.JobName = spec.Name
		collectTemplateFields.PrimaryPassword = PrimaryPassword
		collectTemplateFields.CCPImageTag = spec.CCPImageTag
		collectTemplateFields.CCPImagePrefix = Pgo.Cluster.CCPImagePrefix

		var collectDoc bytes.Buffer
		err := CollectTemplate1.Execute(&collectDoc, collectTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if CRUNCHY_DEBUG {
			CollectTemplate1.Execute(os.Stdout, collectTemplateFields)
		}
		return collectDoc.String()
	}
	return ""
}

//consolidate with cluster.GetConfVolume
func GetConfVolume(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string) string {
	var found bool

	//check for user provided configmap
	if cl.Spec.CustomConfig != "" {
		_, found = kubeapi.GetConfigMap(clientset, cl.Spec.CustomConfig, namespace)
		if !found {
			//you should NOT get this error because of apiserver validation of this value!
			log.Errorf("%s was not found, error, skipping user provided configMap", cl.Spec.CustomConfig)
		} else {
			log.Debugf("user provided configmap %s was used for this cluster", cl.Spec.CustomConfig)
			return "\"configMap\": { \"name\": \"" + cl.Spec.CustomConfig + "\" }"
		}

	}

	//check for global custom configmap "pgo-custom-pg-config"
	_, found = kubeapi.GetConfigMap(clientset, util.GLOBAL_CUSTOM_CONFIGMAP, namespace)
	if !found {
		log.Debug(util.GLOBAL_CUSTOM_CONFIGMAP + " was not found, , skipping global configMap")
	} else {
		return "\"configMap\": { \"name\": \"pgo-custom-pg-config\" }"
	}

	//the default situation
	return "\"emptyDir\": { \"medium\": \"Memory\" }"
}

// needs to be consolidated with cluster.GetLabelsFromMap
// GetLabelsFromMap ...
func GetLabelsFromMap(labels map[string]string) string {
	var output string

	mapLen := len(labels)
	i := 1
	for key, value := range labels {
		if i < mapLen {
			output += fmt.Sprintf("\"" + key + "\": \"" + value + "\",")
		} else {
			output += fmt.Sprintf("\"" + key + "\": \"" + value + "\"")
		}
		i++
	}
	return output
}

// GetPrimaryLabels ...
func GetPrimaryLabels(Name string, ClusterName string, replicaFlag bool, userLabels map[string]string) map[string]string {
	primaryLabels := make(map[string]string)
	primaryLabels[util.LABEL_PRIMARY] = "true"
	if replicaFlag {
		primaryLabels[util.LABEL_PRIMARY] = "false"
	}

	primaryLabels["name"] = Name
	primaryLabels[util.LABEL_PG_CLUSTER] = ClusterName

	for key, value := range userLabels {
		if key == util.LABEL_AUTOFAIL || key == util.LABEL_NODE_LABEL_KEY || key == util.LABEL_NODE_LABEL_VALUE {
			//dont add these since they can break label expression checks
			//or autofail toggling
		} else {
			primaryLabels[key] = value
		}
	}
	return primaryLabels
}

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
	err := AffinityTemplate1.Execute(&affinityDoc, affinityTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return output
	}

	if CRUNCHY_DEBUG {
		AffinityTemplate1.Execute(os.Stdout, affinityTemplateFields)
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
	if replicaLabels[util.LABEL_NODE_LABEL_KEY] != "" {
		//use the replica labels
		operator = "In"
		key = replicaLabels[util.LABEL_NODE_LABEL_KEY]
		value = replicaLabels[util.LABEL_NODE_LABEL_VALUE]
	} else {
		//use the cluster labels
		operator = "NotIn"
		key = clusterLabels[util.LABEL_NODE_LABEL_KEY]
		value = clusterLabels[util.LABEL_NODE_LABEL_VALUE]
	}
	return GetAffinity(key, value, operator)
}

func GetPgmonitorEnvVars(metricsEnabled string) string {
	if metricsEnabled == "true" {
		fields := PgmonitorEnvVarsTemplateFields{
			PgmonitorPassword: Pgo.Cluster.PgmonitorPassword,
		}

		var doc bytes.Buffer
		err := PgmonitorEnvVarsTemplate.Execute(&doc, fields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}
		return doc.String()
	}
	return ""

}
