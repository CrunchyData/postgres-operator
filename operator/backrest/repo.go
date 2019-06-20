package backrest

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
	"encoding/json"
	"os"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type RepoDeploymentTemplateFields struct {
	SecurityContext       string
	PGOImagePrefix        string
	PGOImageTag           string
	ContainerResources    string
	BackrestRepoClaimName string
	SshdSecretsName       string
	PGbackrestDBHost      string
	PgbackrestRepoPath    string
	PgbackrestDBPath      string
	PgbackrestPGPort      string
	SshdPort              int
	PgbackrestStanza      string
	PgbackrestRepoType    string
	PgbackrestS3EnvVars   string
	Name                  string
	ClusterName           string
}

type RepoServiceTemplateFields struct {
	Name        string
	ClusterName string
	Port        string
}

func CreateRepoDeployment(clientset *kubernetes.Clientset, namespace string, cluster *crv1.Pgcluster) error {

	var b bytes.Buffer

	repoName := cluster.Name + "-pgbr-repo"
	serviceName := cluster.Name + "-backrest-shared-repo"

	//create backrest repo service
	serviceFields := RepoServiceTemplateFields{
		Name:        serviceName,
		ClusterName: cluster.Name,
		Port:        "2022",
	}

	err := createService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	//create backrest repo PVC with same name as repoName

	_, found, err := kubeapi.GetPVC(clientset, repoName, namespace)
	if found {
		log.Debugf("pvc [%s] already present, will not recreate", repoName)
	} else {
		_, err = pvc.CreatePVC(clientset, &cluster.Spec.BackrestStorage, repoName, cluster.Name, namespace)
		if err != nil {
			return err
		}
		log.Debugf("created backrest-shared-repo pvc [%s]", repoName)
	}

	//create backrest repo deployment
	log.Debug("hi from backup create repo deploy")
	fields := RepoDeploymentTemplateFields{
		PGOImagePrefix:        operator.Pgo.Pgo.PGOImagePrefix,
		PGOImageTag:           operator.Pgo.Pgo.PGOImageTag,
		ContainerResources:    "",
		BackrestRepoClaimName: repoName,
		SshdSecretsName:       "pgo-backrest-repo-config",
		PGbackrestDBHost:      cluster.Name,
		PgbackrestRepoPath:    "/backrestrepo/" + serviceName,
		PgbackrestDBPath:      "/pgdata/" + cluster.Name,
		PgbackrestPGPort:      cluster.Spec.Port,
		SshdPort:              operator.Pgo.Cluster.BackrestPort,
		PgbackrestStanza:      "db",
		PgbackrestRepoType:    operator.GetRepoType(cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(cluster.Labels[config.LABEL_BACKREST],
			cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], clientset, namespace),
		Name:            serviceName,
		ClusterName:     cluster.Name,
		SecurityContext: util.CreateSecContext(cluster.Spec.PrimaryStorage.Fsgroup, cluster.Spec.PrimaryStorage.SupplementalGroups),
	}
	log.Debugf(fields.Name)

	err = config.PgoBackrestRepoTemplate.Execute(&b, fields)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.PgoBackrestRepoTemplate.Execute(os.Stdout, fields)
	}

	deployment := v1.Deployment{}
	err = json.Unmarshal(b.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling backrest repo json into Deployment " + err.Error())
		return err
	}

	err = kubeapi.CreateDeploymentV1(clientset, &deployment, namespace)

	return err

}

func createService(clientset *kubernetes.Clientset, fields *RepoServiceTemplateFields, namespace string) error {
	var err error

	var b bytes.Buffer

	_, found, err := kubeapi.GetService(clientset, fields.Name, namespace)
	if !found || err != nil {

		err = config.PgoBackrestRepoServiceTemplate.Execute(&b, fields)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if operator.CRUNCHY_DEBUG {
			config.PgoBackrestRepoServiceTemplate.Execute(os.Stdout, fields)
		}

		s := corev1.Service{}
		err = json.Unmarshal(b.Bytes(), &s)
		if err != nil {
			log.Error("error unmarshalling repo service json into repo Service " + err.Error())
			return err
		}

		_, err = kubeapi.CreateService(clientset, &s, namespace)
	}

	return err
}
