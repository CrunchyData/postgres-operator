package pgdump

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type pgDumpJobTemplateFields struct {
	JobName          string
	TaskName         string
	Name             string // ??
	ClusterName      string
	Command          string // ??
	CommandOpts      string
	PvcName          string
	PodName          string // ??
	CCPImagePrefix   string
	CCPImageTag      string
	SecurityContext  string
	PgDumpHost       string
	PgDumpUserSecret string
	PgDumpDB         string
	PgDumpPort       string
	PgDumpOpts       string
	PgDumpFilename   string
	PgDumpAll        string
	PgDumpPVC        string
	Tolerations      string
	CustomLabels     string
}

// Dump ...
func Dump(namespace string, clientset kubeapi.Interface, task *crv1.Pgtask) {
	ctx := context.TODO()

	var err error

	// make sure the provided clustername is not empty
	clusterName := task.Spec.Parameters[config.LABEL_PG_CLUSTER]
	if clusterName == "" {
		log.Error("unable to create pgdump job, clustername is empty.")
		return
	}

	// get the pgcluster CRD for cases where a CCPImagePrefix is specified
	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return
	}

	// create the Job to run the pgdump command

	cmd := task.Spec.Parameters[config.LABEL_PGDUMP_COMMAND]

	pvcName := task.Spec.Parameters[config.LABEL_PVC_NAME]

	// create the PVC if name is empty or it doesn't exist
	if !(len(pvcName) > 0) || !pvc.Exists(clientset, pvcName, namespace) {

		// set pvcName if empty - should not be empty as apiserver code should have specified.
		if !(len(pvcName) > 0) {
			pvcName = task.Spec.Name + "-pvc"
		}

		pvcName, err = pvc.CreatePVC(clientset, &task.Spec.StorageSpec, pvcName,
			task.Spec.Parameters[config.LABEL_PGDUMP_HOST], namespace, util.GetCustomLabels(cluster))
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info("created backup PVC =" + pvcName + " in namespace " + namespace)
		}
	}

	// this task name should match
	taskName := task.Name
	jobName := taskName + "-" + util.RandStringBytesRmndr(4)

	jobFields := pgDumpJobTemplateFields{
		JobName:         jobName,
		TaskName:        taskName,
		ClusterName:     task.Spec.Parameters[config.LABEL_PG_CLUSTER],
		PodName:         task.Spec.Parameters[config.LABEL_POD_NAME],
		SecurityContext: operator.GetPodSecurityContext(task.Spec.StorageSpec.GetSupplementalGroups()),
		Command:         cmd, //??
		CommandOpts:     task.Spec.Parameters[config.LABEL_PGDUMP_OPTS],
		CCPImagePrefix:  util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImageTag: util.GetValueOrDefault(util.GetStandardImageTag(cluster.Spec.CCPImage, cluster.Spec.CCPImageTag),
			operator.Pgo.Cluster.CCPImageTag),
		PgDumpHost:       task.Spec.Parameters[config.LABEL_PGDUMP_HOST],
		PgDumpUserSecret: task.Spec.Parameters[config.LABEL_PGDUMP_USER],
		PgDumpDB:         task.Spec.Parameters[config.LABEL_PGDUMP_DB],
		PgDumpPort:       task.Spec.Parameters[config.LABEL_PGDUMP_PORT],
		PgDumpOpts:       task.Spec.Parameters[config.LABEL_PGDUMP_OPTS],
		PgDumpAll:        task.Spec.Parameters[config.LABEL_PGDUMP_ALL],
		PgDumpPVC:        pvcName,
		Tolerations:      util.GetTolerations(cluster.Spec.Tolerations),
		CustomLabels:     operator.GetLabelsFromMap(util.GetCustomLabels(cluster), false),
	}

	var doc2 bytes.Buffer
	err = config.PgDumpBackupJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		_ = config.PgDumpBackupJobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA,
		&newjob.Spec.Template.Spec.Containers[0])

	_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, &newjob, metav1.CreateOptions{})

	if err != nil {
		return
	}

	// update the pgdump task status to submitted - updates task, not the job.
	patch, err := kubeapi.NewJSONPatch().Add("spec", "status")(crv1.PgBackupJobSubmitted).Bytes()
	if err == nil {
		log.Debugf("patching task %s: %s", task.Spec.Name, patch)
		_, err = clientset.CrunchydataV1().Pgtasks(namespace).
			Patch(ctx, task.Spec.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error(err.Error())
	}
}
