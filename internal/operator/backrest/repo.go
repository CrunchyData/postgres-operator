package backrest

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"os"

	crv1 "github.com/crunchydata/postgres-operator/internal/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type RepoDeploymentTemplateFields struct {
	SecurityContext           string
	PGOImagePrefix            string
	PGOImageTag               string
	ContainerResources        string
	BackrestRepoClaimName     string
	SshdSecretsName           string
	PGbackrestDBHost          string
	PgbackrestRepoPath        string
	PgbackrestDBPath          string
	PgbackrestPGPort          string
	SshdPort                  int
	PgbackrestStanza          string
	PgbackrestRepoType        string
	PgbackrestS3EnvVars       string
	Name                      string
	ClusterName               string
	PodAntiAffinity           string
	PodAntiAffinityLabelName  string
	PodAntiAffinityLabelValue string
	Replicas                  int
}

type RepoServiceTemplateFields struct {
	Name        string
	ClusterName string
	Port        string
}

func CreateRepoDeployment(clientset *kubernetes.Clientset, namespace string, cluster *crv1.Pgcluster, createPVC bool,
	replicas int) error {

	var b bytes.Buffer

	repoName := fmt.Sprintf(util.BackrestRepoPVCName, cluster.Name)
	serviceName := fmt.Sprintf(util.BackrestRepoServiceName, cluster.Name)

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

	// if createPVC is set to true, attempt to create the PVC
	if createPVC {
		// create backrest repo PVC with same name as repoName
		existing, err := kubeapi.GetPVCIfExists(clientset, repoName, namespace)
		if err != nil {
			return err
		}
		if existing != nil {
			log.Debugf("pvc [%s] already present, will not recreate", repoName)
		} else {
			_, err = pvc.CreatePVC(clientset, &cluster.Spec.BackrestStorage, repoName, cluster.Name, namespace)
			if err != nil {
				return err
			}
			log.Debugf("created backrest-shared-repo pvc [%s]", repoName)
		}
	}

	//create backrest repo deployment
	fields := RepoDeploymentTemplateFields{
		PGOImagePrefix:        util.GetValueOrDefault(cluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:           operator.Pgo.Pgo.PGOImageTag,
		ContainerResources:    operator.GetResourcesJSON(cluster.Spec.BackrestResources, cluster.Spec.EnableBackrestMemoryLimit),
		BackrestRepoClaimName: repoName,
		SshdSecretsName:       "pgo-backrest-repo-config",
		PGbackrestDBHost:      cluster.Name,
		PgbackrestRepoPath:    util.GetPGBackRestRepoPath(*cluster),
		PgbackrestDBPath:      "/pgdata/" + cluster.Name,
		PgbackrestPGPort:      cluster.Spec.Port,
		SshdPort:              operator.Pgo.Cluster.BackrestPort,
		PgbackrestStanza:      "db",
		PgbackrestRepoType:    operator.GetRepoType(cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:   operator.GetPgbackrestS3EnvVars(*cluster, clientset, namespace),
		Name:                  serviceName,
		ClusterName:           cluster.Name,
		SecurityContext:       util.GetPodSecurityContext(cluster.Spec.BackrestStorage.GetSupplementalGroups()),
		Replicas:              replicas,
		PodAntiAffinity: operator.GetPodAntiAffinity(cluster,
			crv1.PodAntiAffinityDeploymentPgBackRest, cluster.Spec.PodAntiAffinity.PgBackRest),
		PodAntiAffinityLabelName: config.LABEL_POD_ANTI_AFFINITY,
		PodAntiAffinityLabelValue: string(operator.GetPodAntiAffinityType(cluster,
			crv1.PodAntiAffinityDeploymentPgBackRest, cluster.Spec.PodAntiAffinity.PgBackRest)),
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

	deployment := appsv1.Deployment{}
	err = json.Unmarshal(b.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling backrest repo json into Deployment " + err.Error())
		return err
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_BACKREST_REPO,
		&deployment.Spec.Template.Spec.Containers[0])

	err = kubeapi.CreateDeployment(clientset, &deployment, namespace)

	return err

}

// UpdateResources updates the pgBackRest repository Deployment to reflect any
// resource updates
func UpdateResources(clientset *kubernetes.Clientset, restConfig *rest.Config, cluster *crv1.Pgcluster) error {
	// get a list of all of the instance deployments for the cluster
	deployment, err := operator.GetBackrestDeployment(clientset, cluster)

	if err != nil {
		return err
	}

	// iterate through each PostgreSQL instance deployment and update the
	// resource values for the pgBackRest repository container. This is the first
	// container
	requestResourceList := v1.ResourceList{}
	limitResourceList := v1.ResourceList{}

	// if there is a request or limit resource list available already, use that
	// one
	// NOTE: this works as the "database" container is always first
	if deployment.Spec.Template.Spec.Containers[0].Resources.Requests != nil {
		requestResourceList = deployment.Spec.Template.Spec.Containers[0].Resources.Requests
	}

	if deployment.Spec.Template.Spec.Containers[0].Resources.Limits != nil {
		limitResourceList = deployment.Spec.Template.Spec.Containers[0].Resources.Limits
	}

	// handle the CPU update. For the CPU updates, we set both set/unset the
	// request and the limit
	if resource, ok := cluster.Spec.BackrestResources[v1.ResourceCPU]; ok {
		requestResourceList[v1.ResourceCPU] = resource
		limitResourceList[v1.ResourceCPU] = resource
	} else {
		delete(requestResourceList, v1.ResourceCPU)
		delete(limitResourceList, v1.ResourceCPU)
	}

	// handle the memory update. For memory, due to behavior of the OOM killer,
	// we only set the **request*
	if resource, ok := cluster.Spec.BackrestResources[v1.ResourceMemory]; ok {
		requestResourceList[v1.ResourceMemory] = resource
		limitResourceList[v1.ResourceMemory] = resource
	} else {
		delete(requestResourceList, v1.ResourceMemory)
		delete(limitResourceList, v1.ResourceMemory)
	}

	// we do need to separately determine whether or not to include the memory
	// limit based on the user's preference we make this check regardless if the
	// limit was cleared
	if !cluster.Spec.EnableBackrestMemoryLimit {
		delete(limitResourceList, v1.ResourceMemory)
	}

	// update the requests / limits resourcelist
	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = requestResourceList
	deployment.Spec.Template.Spec.Containers[0].Resources.Limits = limitResourceList

	// update the deployment with the new values
	if err := kubeapi.UpdateDeployment(clientset, deployment); err != nil {
		return err
	}

	return nil
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

		s := v1.Service{}
		err = json.Unmarshal(b.Bytes(), &s)
		if err != nil {
			log.Error("error unmarshalling repo service json into repo Service " + err.Error())
			return err
		}

		_, err = kubeapi.CreateService(clientset, &s, namespace)
	}

	return err
}
