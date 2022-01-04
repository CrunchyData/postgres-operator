package task

/*
 Copyright 2018 - 2022 Crunchy Data Solutions, Inc.
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
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type rmdatajobTemplateFields struct {
	JobName          string
	Name             string
	ClusterName      string
	ClusterPGHAScope string
	ReplicaName      string
	PGOImagePrefix   string
	PGOImageTag      string
	SecurityContext  string
	RemoveData       string
	RemoveBackup     string
	IsBackup         string
	IsReplica        string
	Tolerations      string
}

// RemoveData ...
func RemoveData(namespace string, clientset kubeapi.Interface, task *crv1.Pgtask) {
	ctx := context.TODO()

	// create marker (clustername, namespace)
	patch, err := kubeapi.NewJSONPatch().
		Add("spec", "parameters", config.LABEL_DELETE_DATA_STARTED)(time.Now().Format(time.RFC3339)).
		Bytes()
	if err == nil {
		log.Debugf("patching task %s: %s", task.Spec.Name, patch)
		_, err = clientset.CrunchydataV1().Pgtasks(namespace).
			Patch(ctx, task.Spec.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Errorf("could not set delete data started marker for task %s cluster %s", task.Spec.Name, task.Spec.Parameters[config.LABEL_PG_CLUSTER])
		return
	}

	// create the Job to remove the data
	// pvcName := task.Spec.Parameters[config.LABEL_PVC_NAME]
	clusterName := task.Spec.Parameters[config.LABEL_PG_CLUSTER]
	clusterPGHAScope := task.Spec.Parameters[config.LABEL_PGHA_SCOPE]
	replicaName := task.Spec.Parameters[config.LABEL_REPLICA_NAME]
	isReplica := task.Spec.Parameters[config.LABEL_IS_REPLICA]
	isBackup := task.Spec.Parameters[config.LABEL_IS_BACKUP]
	removeData := task.Spec.Parameters[config.LABEL_DELETE_DATA]
	removeBackup := task.Spec.Parameters[config.LABEL_DELETE_BACKUPS]

	// make sure the provided clustername is not empty
	if clusterName == "" {
		log.Error("unable to create pgdump job, clustername is empty.")
		return
	}

	jobName := clusterName + "-rmdata-" + util.RandStringBytesRmndr(4)

	jobFields := rmdatajobTemplateFields{
		JobName:          jobName,
		Name:             task.Spec.Name,
		ClusterName:      clusterName,
		ClusterPGHAScope: clusterPGHAScope,
		ReplicaName:      replicaName,
		RemoveData:       removeData,
		RemoveBackup:     removeBackup,
		IsReplica:        isReplica,
		IsBackup:         isBackup,
		PGOImagePrefix:   util.GetValueOrDefault(task.Spec.Parameters[config.LABEL_IMAGE_PREFIX], operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:      operator.Pgo.Pgo.PGOImageTag,
		SecurityContext:  operator.GetPodSecurityContext(task.Spec.StorageSpec.GetSupplementalGroups()),
		Tolerations:      task.Spec.Parameters[config.LABEL_RM_TOLERATIONS],
	}

	log.Debugf("creating rmdata job %s for cluster %s ", jobName, task.Spec.Name)

	if operator.CRUNCHY_DEBUG {
		_ = config.RmdatajobTemplate.Execute(os.Stdout, jobFields)
	}

	doc := bytes.Buffer{}
	if err := config.RmdatajobTemplate.Execute(&doc, jobFields); err != nil {
		log.Error(err)
		return
	}

	job := v1batch.Job{}
	if err := json.Unmarshal(doc.Bytes(), &job); err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_RMDATA,
		&job.Spec.Template.Spec.Containers[0])

	if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, &job, metav1.CreateOptions{}); err != nil {
		log.Error(err)
		return
	}

	log.Debugf("successfully created rmdata job %s", job.Name)

	publishDeleteCluster(task.Spec.Parameters[config.LABEL_PG_CLUSTER],
		task.ObjectMeta.Labels[config.LABEL_PGOUSER], namespace)
}

func publishDeleteCluster(clusterName, username, namespace string) {
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
