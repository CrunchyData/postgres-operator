package backrest

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func StanzaCreate(namespace, clusterName string, clientset *kubernetes.Clientset, RESTClient *rest.RESTClient) {

	taskName := clusterName + "-" + crv1.PgtaskBackrestStanzaCreate

	//look up the backrest-repo pod name
	selector := util.LABEL_PG_CLUSTER + "=" + clusterName + "," + util.LABEL_PGO_BACKREST_REPO + "=true"
	pods, err := kubeapi.GetPods(clientset, selector, namespace)
	if len(pods.Items) != 1 {
		log.Errorf("pods len != 1 for cluster %s", clusterName)
		return
	}
	if err != nil {
		log.Error(err)
	}

	podName := pods.Items[0].Name

	//create the stanza-create task
	spec := crv1.PgtaskSpec{}
	spec.Name = taskName

	jobName := "backrest-" + crv1.PgtaskBackrestStanzaCreate + "-" + clusterName

	spec.TaskType = crv1.PgtaskBackrest
	spec.Parameters = make(map[string]string)
	spec.Parameters[util.LABEL_JOB_NAME] = jobName
	spec.Parameters[util.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[util.LABEL_POD_NAME] = podName
	spec.Parameters[util.LABEL_CONTAINER_NAME] = "pgo-backrest-repo"
	spec.Parameters[util.LABEL_BACKREST_COMMAND] = crv1.PgtaskBackrestStanzaCreate
	spec.Parameters[util.LABEL_BACKREST_OPTS] = ""

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[util.LABEL_PG_CLUSTER] = clusterName

	err = kubeapi.Createpgtask(RESTClient,
		newInstance,
		namespace)
	if err != nil {
		log.Error(err)
	}

}
