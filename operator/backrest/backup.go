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
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"																																																																																																																																																																						
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

	newjob.ObjectMeta.Labels[config.LABEL_PGOUSER] = task.ObjectMeta.Labels[config.LABEL_PGOUSER]
	newjob.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = task.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]

	if task.Spec.Parameters[config.LABEL_PGHA_BOOTSTRAP_BACKUP] == "true"  {
		newjob.ObjectMeta.Labels[config.LABEL_PGHA_BOOTSTRAP_BACKUP] = "true"
	}

	kubeapi.CreateJob(clientset, &newjob, namespace)

	//publish backrest backup event
	if cmd == "backup" {
		topics := make([]string, 1)
		topics[0] = events.EventTopicBackup

		f := events.EventCreateBackupFormat{
			EventHeader: events.EventHeader{
				Namespace: namespace,
				Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
				Topic:     topics,
				Timestamp: time.Now(),
				EventType: events.EventCreateBackup,
			},
			Clustername:       jobFields.ClusterName,
			BackupType:        "pgbackrest",
		}

		err := events.Publish(f)
		if err != nil {
			log.Error(err.Error())
		}
	}

}
 
// CreateInitialBackup creates a Pgtask in order to initiate the initial pgBackRest backup for a cluster
// as needed to support replica creation
func CreateInitialBackup(restclient *rest.RESTClient, namespace, clusterName, podName string) (*crv1.Pgtask, error) {
	var params map[string]string
	params = make(map[string]string)
	params[config.LABEL_PGHA_BOOTSTRAP_BACKUP] = "true"
	return CreateBackup(restclient, namespace, clusterName, podName, params, "--type=full")
}

// CreateBackup creates a Pgtask in order to initiate a pgBackRest backup
func CreateBackup(restclient *rest.RESTClient, namespace, clusterName, podName string, params map[string]string, 
	backupOpts string) (*crv1.Pgtask, error) {
	
	log.Debug("pgBackRest operator CreateBackup called")
	
	cluster := crv1.Pgcluster{}
	_, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if err != nil {
	    log.Error(err)
	    return nil, err
	}
	
	var newInstance *crv1.Pgtask
	taskName := "backrest-backup-" + cluster.Name

	spec := crv1.PgtaskSpec{}
	spec.Name = taskName
	spec.Namespace = namespace

	spec.TaskType = crv1.PgtaskBackrest
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_JOB_NAME] = "backrest-" + crv1.PgtaskBackrestBackup + "-" + cluster.Name
	spec.Parameters[config.LABEL_PG_CLUSTER] = cluster.Name
	spec.Parameters[config.LABEL_POD_NAME] = podName
	spec.Parameters[config.LABEL_CONTAINER_NAME] = "database"
	spec.Parameters[config.LABEL_BACKREST_COMMAND] = crv1.PgtaskBackrestBackup
	spec.Parameters[config.LABEL_BACKREST_OPTS] = backupOpts
	spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] = cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]

    for k, v := range params { 
		spec.Parameters[k] = v
	}

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = cluster.Name
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = cluster.ObjectMeta.Labels[config.LABEL_PGOUSER]
	
	err = kubeapi.Createpgtask(restclient, newInstance, cluster.Namespace)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return newInstance, nil
}
