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
	"os"
)

type pgDumpJobTemplateFields struct {
	JobName            string
	TaskName           string
	Name               string // ??
	ClusterName        string
	Command            string // ??
	CommandOpts        string
	PvcName            string
	PodName            string // ??
	CCPImagePrefix     string
	CCPImageTag        string
	SecurityContext    string
	PgDumpHost         string
	PgDumpUserSecret   string
	PgDumpDB           string
	PgDumpPort         string
	PgDumpOpts         string
	PgDumpFilename     string
	PgDumpAll          string
	PgDumpPVC          string
	ContainerResources string
}

// Dump ...
func Dump(namespace string, clientset *kubernetes.Clientset, client *rest.RESTClient, task *crv1.Pgtask) {

	var err error
	//create the Job to run the pgdump command

	cmd := task.Spec.Parameters[config.LABEL_PGDUMP_COMMAND]

	pvcName := task.Spec.Parameters[config.LABEL_PVC_NAME]

	// create the PVC if name is empty or it doesn't exist
	if !(len(pvcName) > 0) || !pvc.Exists(clientset, pvcName, namespace) {

		// set pvcName if empty - should not be empty as apiserver code should have specified.
		if !(len(pvcName) > 0) {
			pvcName = task.Spec.Name + "-pvc"
		}

		pvcName, err = pvc.CreatePVC(clientset, &task.Spec.StorageSpec, pvcName,
			task.Spec.Parameters[config.LABEL_PGDUMP_HOST], namespace)
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info("created backup PVC =" + pvcName + " in namespace " + namespace)
		}
	}

	cr := ""
	if operator.Pgo.DefaultBackupResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultBackupResources)
		if err != nil {
			log.Error(err)
			return
		}
		cr = operator.GetContainerResourcesJSON(&tmp)

	}

	// this task name should match
	taskName := task.Name
	jobName := taskName + "-" + util.RandStringBytesRmndr(4)

	jobFields := pgDumpJobTemplateFields{
		JobName:            jobName,
		TaskName:           taskName,
		ClusterName:        task.Spec.Parameters[config.LABEL_PG_CLUSTER],
		PodName:            task.Spec.Parameters[config.LABEL_POD_NAME],
		SecurityContext:    util.CreateSecContext(task.Spec.StorageSpec.Fsgroup, task.Spec.StorageSpec.SupplementalGroups),
		Command:            cmd, //??
		CommandOpts:        task.Spec.Parameters[config.LABEL_PGDUMP_OPTS],
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        operator.Pgo.Cluster.CCPImageTag,
		PgDumpHost:         task.Spec.Parameters[config.LABEL_PGDUMP_HOST],
		PgDumpUserSecret:   task.Spec.Parameters[config.LABEL_PGDUMP_USER],
		PgDumpDB:           task.Spec.Parameters[config.LABEL_PGDUMP_DB],
		PgDumpPort:         task.Spec.Parameters[config.LABEL_PGDUMP_PORT],
		PgDumpOpts:         task.Spec.Parameters[config.LABEL_PGDUMP_OPTS],
		PgDumpAll:          task.Spec.Parameters[config.LABEL_PGDUMP_ALL],
		PgDumpPVC:          pvcName,
		ContainerResources: cr,
	}

	var doc2 bytes.Buffer
	err = config.PgDumpBackupJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.PgDumpBackupJobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	_, err = kubeapi.CreateJob(clientset, &newjob, namespace)

	if err != nil {
		return
	}

	//update the pgdump task status to submitted - updates task, not the job.
	err = util.Patch(client, "/spec/status", crv1.PgBackupJobSubmitted, "pgtasks", task.Spec.Name, namespace)

	if err != nil {
		log.Error(err.Error())
	}

}
