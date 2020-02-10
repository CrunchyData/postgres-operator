package task

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type rmdatajobTemplateFields struct {
	JobName            string
	Name               string
	ClusterName        string
	ClusterPGHAScope   string
	ReplicaName        string
	PGOImagePrefix     string
	PGOImageTag        string
	SecurityContext    string
	ContainerResources string
	RemoveData         string
	RemoveBackup       string
	IsBackup           string
	IsReplica          string
}

// RemoveData ...
func RemoveData(namespace string, clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask) {

	//create marker (clustername, namespace)
	err := PatchpgtaskDeleteDataStatus(restclient, task, namespace)
	if err != nil {
		log.Error("could not set delete data started marker for task %s cluster %s", task.Spec.Name, task.Spec.Parameters[config.LABEL_PG_CLUSTER])
		return
	}

	//create the Job to remove the data
	//pvcName := task.Spec.Parameters[config.LABEL_PVC_NAME]
	clusterName := task.Spec.Parameters[config.LABEL_PG_CLUSTER]
	clusterPGHAScope := task.Spec.Parameters[config.LABEL_PGHA_SCOPE]
	replicaName := task.Spec.Parameters[config.LABEL_REPLICA_NAME]
	isReplica := task.Spec.Parameters[config.LABEL_IS_REPLICA]
	isBackup := task.Spec.Parameters[config.LABEL_IS_BACKUP]
	removeData := task.Spec.Parameters[config.LABEL_DELETE_DATA]
	removeBackup := task.Spec.Parameters[config.LABEL_DELETE_BACKUPS]

	cr := ""
	if operator.Pgo.DefaultRmdataResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultRmdataResources)
		if err != nil {
			log.Error(err)
			return
		}
		cr = operator.GetContainerResourcesJSON(&tmp)

	}

	jobName := clusterName + "-rmdata-" + util.RandStringBytesRmndr(4)

	jobFields := rmdatajobTemplateFields{
		JobName:            jobName,
		Name:               task.Spec.Name,
		ClusterName:        clusterName,
		ClusterPGHAScope:   clusterPGHAScope,
		ReplicaName:        replicaName,
		RemoveData:         removeData,
		RemoveBackup:       removeBackup,
		IsReplica:          isReplica,
		IsBackup:           isBackup,
		PGOImagePrefix:     operator.Pgo.Pgo.PGOImagePrefix,
		PGOImageTag:        operator.Pgo.Pgo.PGOImageTag,
		SecurityContext:    util.CreateSecContext(task.Spec.StorageSpec.Fsgroup, task.Spec.StorageSpec.SupplementalGroups),
		ContainerResources: cr,
	}
	log.Debugf("creating rmdata job %s for cluster %s ", jobName, task.Spec.Name)

	var doc2 bytes.Buffer
	err = config.RmdatajobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.RmdatajobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_RMDATA,
		&newjob.Spec.Template.Spec.Containers[0])

	var jobname string
	jobname, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		log.Errorf("got error when creating rmdata job %s", jobname)
		return
	}
	log.Debugf("successfully created rmdata job %s", jobname)

	publishDeleteCluster(task.Spec.Parameters[config.LABEL_PG_CLUSTER], task.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER],
		task.ObjectMeta.Labels[config.LABEL_PGOUSER], namespace)
}

func PatchpgtaskDeleteDataStatus(restclient *rest.RESTClient, oldCrd *crv1.Pgtask, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Spec.Parameters[config.LABEL_DELETE_DATA_STARTED] = time.Now().Format(time.RFC3339)

	//create the patch
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCrd)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}
	log.Debug(string(patchBytes))

	//apply patch
	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgtaskResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

func publishDeleteCluster(clusterName, identifier, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventDeleteClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeleteCluster,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}
