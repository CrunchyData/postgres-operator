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
	"fmt"
	"os"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type restorejobTemplateFields struct {
	JobName             string
	TaskName            string
	ClusterName         string
	SecurityContext     string
	FromClusterPVCName  string
	PgRestoreHost       string
	PgRestoreDB         string
	PgRestoreUserSecret string
	PgPrimaryPort       string
	PGRestoreOpts       string
	PITRTarget          string
	CCPImagePrefix      string
	CCPImageTag         string
	PgPort              string
	NodeSelector        string
	Tolerations         string
	CustomLabels        string
}

// Restore ...
func Restore(namespace string, clientset kubeapi.Interface, task *crv1.Pgtask) {
	ctx := context.TODO()

	log.Infof(" PgDump Restore not implemented %s, %s", namespace, task.Name)

	clusterName := task.Spec.Parameters[config.LABEL_PGRESTORE_FROM_CLUSTER]

	fromPvcName := task.Spec.Parameters[config.LABEL_PGRESTORE_FROM_PVC]

	if !(len(fromPvcName) > 0) || !pvc.Exists(clientset, fromPvcName, namespace) {
		log.Errorf("pgrestore: could not find source pvc required for restore: %s", fromPvcName)
		return
	}

	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("pgrestore: could not find a pgcluster in Restore Workflow for %s", clusterName)
		return
	}

	// use the storage config from the primary PostgreSQL cluster
	storage := cluster.Spec.PrimaryStorage

	taskName := task.Name
	var nodeAffinity *v1.NodeAffinity

	if task.Spec.Parameters["NodeLabelKey"] != "" && task.Spec.Parameters["NodeLabelValue"] != "" {
		affinityType := crv1.NodeAffinityTypePreferred
		if task.Spec.Parameters[config.LABEL_NODE_AFFINITY_TYPE] == "required" {
			affinityType = crv1.NodeAffinityTypeRequired
		}

		nodeAffinity = util.GenerateNodeAffinity(affinityType,
			task.Spec.Parameters["NodeLabelKey"], []string{task.Spec.Parameters["NodeLabelValue"]})
	}

	jobFields := restorejobTemplateFields{
		JobName: fmt.Sprintf("pgrestore-%s-%s", task.Spec.Parameters[config.LABEL_PGRESTORE_FROM_CLUSTER],
			util.RandStringBytesRmndr(4)),
		TaskName:            taskName,
		ClusterName:         clusterName,
		SecurityContext:     operator.GetPodSecurityContext(storage.GetSupplementalGroups()),
		FromClusterPVCName:  fromPvcName,
		PgRestoreHost:       task.Spec.Parameters[config.LABEL_PGRESTORE_HOST],
		PgRestoreDB:         task.Spec.Parameters[config.LABEL_PGRESTORE_DB],
		PgRestoreUserSecret: task.Spec.Parameters[config.LABEL_PGRESTORE_USER],
		PgPrimaryPort:       operator.Pgo.Cluster.Port,
		PGRestoreOpts:       task.Spec.Parameters[config.LABEL_PGRESTORE_OPTS],
		PITRTarget:          task.Spec.Parameters[config.LABEL_PGRESTORE_PITR_TARGET],
		CCPImagePrefix:      util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImageTag: util.GetValueOrDefault(util.GetStandardImageTag(cluster.Spec.CCPImage, cluster.Spec.CCPImageTag),
			operator.Pgo.Cluster.CCPImageTag),
		NodeSelector: operator.GetNodeAffinity(nodeAffinity),
		Tolerations:  util.GetTolerations(cluster.Spec.Tolerations),
		CustomLabels: operator.GetLabelsFromMap(util.GetCustomLabels(cluster), false),
	}

	var doc2 bytes.Buffer
	err = config.PgRestoreJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		log.Error("restore workflow: error executing job template")
		return
	}

	if operator.CRUNCHY_DEBUG {
		_ = config.PgRestoreJobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("restore workflow: error unmarshalling json into Job " + err.Error())
		return
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA,
		&newjob.Spec.Template.Spec.Containers[0])

	j, err := clientset.BatchV1().Jobs(namespace).Create(ctx, &newjob, metav1.CreateOptions{})
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in creating restore job")
		return
	}
	log.Debugf("pgrestore job %s created", j.Name)
}
