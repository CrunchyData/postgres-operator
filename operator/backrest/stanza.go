package backrest

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
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
		return
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
	// pass along the appropriate image prefix for the backup task
	// this will be used by the associated backrest job
	spec.Parameters[config.LABEL_IMAGE_PREFIX] = util.GetValueOrDefault(cluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix)
	spec.Parameters[config.LABEL_BACKREST_COMMAND] = crv1.PgtaskBackrestStanzaCreate

	// Handle stanza creation for a standby cluster, which requires some additional consideration.
	// This includes setting the pgBackRest storage type and command options as needed to support
	// stanza creation for a standby cluster.  If not a standby cluster then simply set the
	// storage type and options as usual.
	if cluster.Spec.Standby {
		// Since this is a standby cluster, if local storage is specified then ensure stanza
		// creation is for the local repo only.  The stanza for the S3 repo will have already been
		// created by the cluster the standby is replicating from, and therefore does not need to
		// be attempted again.
		if strings.Contains(cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], "local") {
			spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] = "local"
		}
		// Since the primary will not be directly accessible to the standby cluster, create the
		// stanza in offline mode
		spec.Parameters[config.LABEL_BACKREST_OPTS] = "--no-online"
	} else {
		spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] =
			cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]
		spec.Parameters[config.LABEL_BACKREST_OPTS] = ""
	}

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
