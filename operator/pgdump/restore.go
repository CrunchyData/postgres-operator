package pgdump

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
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
}

// Restore ...
func Restore(namespace string, clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask) {

	log.Infof(" PgDump Restore not implemented %s, %s", namespace, task.Name)

	clusterName := task.Spec.Parameters[config.LABEL_PGRESTORE_FROM_CLUSTER]

	fromPvcName := task.Spec.Parameters[config.LABEL_PGRESTORE_FROM_PVC]

	if !(len(fromPvcName) > 0) || !pvc.Exists(clientset, fromPvcName, namespace) {
		log.Errorf("pgrestore: could not find source pvc required for restore: %s", fromPvcName)
		return
	}

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("pgrestore: could not find a pgcluster in Restore Workflow for %s", clusterName)
		return
	}

	//use the storage config from pgo.yaml for Primary
	storage := operator.Pgo.Storage[operator.Pgo.PrimaryStorage]

	taskName := task.Name

	jobFields := restorejobTemplateFields{
		JobName:             "pgrestore-" + task.Spec.Parameters[config.LABEL_PGRESTORE_FROM_CLUSTER] + "-from-" + fromPvcName + "-" + util.RandStringBytesRmndr(4),
		TaskName:            taskName,
		ClusterName:         clusterName,
		SecurityContext:     util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		FromClusterPVCName:  fromPvcName,
		PgRestoreHost:       task.Spec.Parameters[config.LABEL_PGRESTORE_HOST],
		PgRestoreDB:         task.Spec.Parameters[config.LABEL_PGRESTORE_DB],
		PgRestoreUserSecret: task.Spec.Parameters[config.LABEL_PGRESTORE_USER],
		PgPrimaryPort:       operator.Pgo.Cluster.Port,
		PGRestoreOpts:       task.Spec.Parameters[config.LABEL_PGRESTORE_OPTS],
		PITRTarget:          task.Spec.Parameters[config.LABEL_PGRESTORE_PITR_TARGET],
		CCPImagePrefix:      operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:         operator.Pgo.Cluster.CCPImageTag,
		NodeSelector:        operator.GetAffinity(task.Spec.Parameters["NodeLabelKey"], task.Spec.Parameters["NodeLabelValue"], "In"),
	}

	var doc2 bytes.Buffer
	err = config.PgRestoreJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		log.Error("restore workflow: error executing job template")
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.PgRestoreJobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("restore workflow: error unmarshalling json into Job " + err.Error())
		return
	}

	var jobName string
	jobName, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in creating restore job")
		return
	}
	log.Debugf("pgrestore job %s created", jobName)

}
