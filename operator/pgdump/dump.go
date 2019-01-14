package pgdump

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
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

type pgDumpJobTemplateFields struct {
	JobName         string
	Name            string // ??
	ClusterName     string
	Command         string // ??
	CommandOpts     string
	PvcName         string
	PodName         string // ??
	CPPImagePrefix  string
	CPPImageTag     string
	SecurityContext string
	PgDumpHost      string
	PgDumpUser      string
	PgDumpPort      string
	PgDumpOpts      string
	PgDumpFilename  string
	PgDumpAll       string
}

// Dump ...
func Dump(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//create the Job to run the pgdump command

	cmd := task.Spec.Parameters[util.LABEL_PGDUMP_COMMAND]

	jobFields := pgDumpJobTemplateFields{
		JobName:         task.Spec.Parameters[util.LABEL_PGDUMP_COMMAND] + "-" + task.Spec.Parameters[util.LABEL_PG_CLUSTER],
		ClusterName:     task.Spec.Parameters[util.LABEL_PG_CLUSTER],
		PodName:         task.Spec.Parameters[util.LABEL_POD_NAME],
		SecurityContext: util.CreateSecContext(task.Spec.StorageSpec.Fsgroup, task.Spec.StorageSpec.SupplementalGroups),
		Command:         cmd, //??
		CommandOpts:     task.Spec.Parameters[util.LABEL_PGDUMP_OPTS],
		CPPImagePrefix:  operator.Pgo.Cluster.CCPImagePrefix,
		CPPImageTag:     operator.Pgo.Cluster.CCPImageTag,
		PgDumpHost:      task.Spec.Parameters[util.LABEL_PGDUMP_HOST],
		PgDumpUser:      task.Spec.Parameters[util.LABEL_PGDUMP_USER],
		PgDumpPort:      task.Spec.Parameters[util.LABEL_PGDUMP_PORT],
		PgDumpOpts:      task.Spec.Parameters[util.LABEL_PGDUMP_OPTS],
		PgDumpAll:       task.Spec.Parameters[util.LABEL_PGDUMP_ALL],
	}

	var doc2 bytes.Buffer
	err := operator.PgDumpBackupJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		operator.PgDumpBackupJobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	_, err = kubeapi.CreateJob(clientset, &newjob, namespace)

	if err != nil {
		log.Error(err.Error())
	}

}
