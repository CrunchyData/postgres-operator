package backrest

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
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
)

type backrestJobTemplateFields struct {
	JobName                       string
	Name                          string
	ClusterName                   string
	Command                       string
	CommandOpts                   string
	PITRTarget                    string
	PodName                       string
	PGOImagePrefix                string
	PGOImageTag                   string
	SecurityContext               string
	PgbackrestStanza              string
	PgbackrestDBPath              string
	PgbackrestRepoPath            string
	PgbackrestRepoType            string
	BackrestLocalAndS3Storage     bool
	PgbackrestRestoreVolumes      string
	PgbackrestRestoreVolumeMounts string
}

// Backrest ...
func Backrest(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//create the Job to run the backrest command

	cmd := task.Spec.Parameters[config.LABEL_BACKREST_COMMAND]

	jobFields := backrestJobTemplateFields{
		JobName:                       task.Spec.Parameters[config.LABEL_JOB_NAME],
		ClusterName:                   task.Spec.Parameters[config.LABEL_PG_CLUSTER],
		PodName:                       task.Spec.Parameters[config.LABEL_POD_NAME],
		SecurityContext:               "",
		Command:                       cmd,
		CommandOpts:                   task.Spec.Parameters[config.LABEL_BACKREST_OPTS],
		PITRTarget:                    "",
		PGOImagePrefix:                operator.Pgo.Pgo.PGOImagePrefix,
		PGOImageTag:                   operator.Pgo.Pgo.PGOImageTag,
		PgbackrestStanza:              task.Spec.Parameters[config.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:              task.Spec.Parameters[config.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepoPath:            task.Spec.Parameters[config.LABEL_PGBACKREST_REPO_PATH],
		PgbackrestRestoreVolumes:      "",
		PgbackrestRestoreVolumeMounts: "",
		PgbackrestRepoType:            operator.GetRepoType(task.Spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE]),
		BackrestLocalAndS3Storage:     operator.IsLocalAndS3Storage(task.Spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE]),
	}

	var doc2 bytes.Buffer
	err := config.BackrestjobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.BackrestjobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	kubeapi.CreateJob(clientset, &newjob, namespace)

}
