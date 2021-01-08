package task

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

	// if the clustername is not empty, get the pgcluster
	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
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
		PGOImagePrefix:   util.GetValueOrDefault(cluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:      operator.Pgo.Pgo.PGOImageTag,
		SecurityContext:  operator.GetPodSecurityContext(task.Spec.StorageSpec.GetSupplementalGroups()),
	}
	log.Debugf("creating rmdata job %s for cluster %s ", jobName, task.Spec.Name)

	var doc2 bytes.Buffer
	err = config.RmdatajobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		_ = config.RmdatajobTemplate.Execute(os.Stdout, jobFields)
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

	j, err := clientset.BatchV1().Jobs(namespace).Create(ctx, &newjob, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("got error when creating rmdata job %s", newjob.Name)
		return
	}
	log.Debugf("successfully created rmdata job %s", j.Name)
}
