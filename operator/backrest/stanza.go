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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func StanzaCreate(namespace, clusterName string, clientset *kubernetes.Clientset, RESTClient *rest.RESTClient) {

	taskName := clusterName + "-" + crv1.PgtaskBackrestStanzaCreate

	//look up the backrest-repo pod name
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_PGO_BACKREST_REPO + "=true"
	pods, err := kubeapi.GetPods(clientset, selector, namespace)
	if len(pods.Items) != 1 {
		log.Errorf("pods len != 1 for cluster %s", clusterName)
		return
	}
	if err != nil {
		log.Error(err)
	}

	podName := pods.Items[0].Name

	// get the cluster to determine the proper storage type
	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(RESTClient, &cluster,
		clusterName, namespace)
	if err != nil {
		return
	}

	//create the stanza-create task
	spec := crv1.PgtaskSpec{}
	spec.Name = taskName

	jobName := clusterName + "-" + crv1.PgtaskBackrestStanzaCreate

	spec.TaskType = crv1.PgtaskBackrest
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_JOB_NAME] = jobName
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[config.LABEL_POD_NAME] = podName
	spec.Parameters[config.LABEL_CONTAINER_NAME] = "pgo-backrest-repo"
	spec.Parameters[config.LABEL_BACKREST_COMMAND] = crv1.PgtaskBackrestStanzaCreate
	spec.Parameters[config.LABEL_BACKREST_OPTS] = ""
	spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] = cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName

	err = kubeapi.Createpgtask(RESTClient,
		newInstance,
		namespace)
	if err != nil {
		log.Error(err)
	}

}
