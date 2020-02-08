package backrest

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
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

var backrestPgHostRegex = regexp.MustCompile("--db-host|--pg1-host")
var backrestPgPathRegex = regexp.MustCompile("--db-path|--pg1-path")

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

	podCommandOpts, err := getCommandOptsFromPod(clientset, task, namespace)
	if err != nil {
		log.Error(err.Error())
		return
	}
	jobFields.CommandOpts = jobFields.CommandOpts + " " + podCommandOpts

	var doc2 bytes.Buffer
	if err := config.BackrestjobTemplate.Execute(&doc2, jobFields); err != nil {
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

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_BACKREST,
		&newjob.Spec.Template.Spec.Containers[0])

	newjob.ObjectMeta.Labels[config.LABEL_PGOUSER] = task.ObjectMeta.Labels[config.LABEL_PGOUSER]
	newjob.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = task.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]

	backupType := task.Spec.Parameters[config.LABEL_PGHA_BACKUP_TYPE]
	if backupType != "" {
		newjob.ObjectMeta.Labels[config.LABEL_PGHA_BACKUP_TYPE] = backupType
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
			Clustername: jobFields.ClusterName,
			BackupType:  "pgbackrest",
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
	params[config.LABEL_PGHA_BACKUP_TYPE] = crv1.BackupTypeBootstrap
	return CreateBackup(restclient, namespace, clusterName, podName, params, "--type=full")
}

// CreatePostFailoverBackup creates a Pgtask in order to initiate the a pgBackRest backup following a failure
// event to ensure proper replica creation and/or reinitialization
func CreatePostFailoverBackup(restclient *rest.RESTClient, namespace, clusterName, podName string) (*crv1.Pgtask, error) {
	var params map[string]string
	params = make(map[string]string)
	params[config.LABEL_PGHA_BACKUP_TYPE] = crv1.BackupTypeFailover
	return CreateBackup(restclient, namespace, clusterName, podName, params, "")
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

// CleanBackupResources is responsible for cleaning up Kubernetes resources from a previous
// pgBackRest backup.  Specifically, this function deletes the pgptask and job associate with a
// previous pgBackRest backup for the cluster.
func CleanBackupResources(restclient *rest.RESTClient, clientset *kubernetes.Clientset, namespace,
	clusterName string) error {

	taskName := "backrest-backup-" + clusterName
	// lookup the pgBackRest backup pgtask for the cluster to determine if it exsits
	found, err := kubeapi.Getpgtask(restclient, &crv1.Pgtask{}, taskName, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("pgtask %s found was %t when cleaning backup resources prior to creating a "+
		"new backrest backup pgtask for cluster %s", taskName, found, clusterName)
	// if the pgBackRest backup pgtask was found, then delete it so that a new pgBackRest backup
	// pgtask can be created in order to initiate a new backup
	if found {
		log.Debugf("deleting pgtask %s for cluster %s", taskName, clusterName)
		// delete the existing pgBackRest backup pgtask
		if err = kubeapi.Deletepgtask(restclient, taskName, namespace); err != nil {
			return err
		}
	}

	//remove previous backup job
	selector := config.LABEL_BACKREST_COMMAND + "=" + crv1.PgtaskBackrestBackup + "," +
		config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_BACKREST + "=true"
	err = kubeapi.DeleteJobs(clientset, selector, namespace)
	if err != nil {
		log.Error(err)
	}

	timeout := time.After(30 * time.Second)
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timed out waiting for deletion of pgBackRest backup job for "+
				"cluster %s", clusterName)
		case <-tick:
			jobList, err := kubeapi.GetJobs(clientset, selector, namespace)
			if err != nil {
				log.Error(err)
				return err
			}
			if len(jobList.Items) == 0 {
				return nil
			}
		}
	}
}

// getCommandOptsFromPod adds command line options from the primary pod to a backrest job.
// If not already specified in the command options provided in the pgtask, add the IP of the
// primary pod as the value for the "--db-host" parameter.  This will ensure direct
// communication between the repo pod and the primary via the primary's IP, instead of going
// through the primary pod's service (which could be unreliable). also if not already specified
// in the command options provided in the pgtask, then lookup the primary pod for the cluster
// and add the PGDATA dir of the pod as the value for the "--db-path" parameter
func getCommandOptsFromPod(clientset *kubernetes.Clientset, task *crv1.Pgtask,
	namespace string) (commandOpts string, err error) {

	// lookup the primary pod in order to determine the IP of the primary and the PGDATA directory for
	// the current primaty
	selector := fmt.Sprintf("%s=%s,%s in (%s,%s)", config.LABEL_PG_CLUSTER,
		task.Spec.Parameters[config.LABEL_PG_CLUSTER], config.LABEL_PGHA_ROLE,
		"promoted", "master")
	pods, err := kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		return
	} else if len(pods.Items) > 1 {
		err = fmt.Errorf("More than one primary found when creating backrest job %s",
			task.Spec.Parameters[config.LABEL_JOB_NAME])
		return
	} else if len(pods.Items) == 0 {
		err = fmt.Errorf("Unable to find primary when creating backrest job %s",
			task.Spec.Parameters[config.LABEL_JOB_NAME])
		return
	}
	pod := pods.Items[0]

	var cmdOpts []string

	if !backrestPgHostRegex.MatchString(task.Spec.Parameters[config.LABEL_BACKREST_OPTS]) {
		cmdOpts = append(cmdOpts, fmt.Sprintf("--db-host=%s", pod.Status.PodIP))
	}
	if !backrestPgPathRegex.MatchString(task.Spec.Parameters[config.LABEL_BACKREST_OPTS]) {
		var podDbPath string
		for _, envVar := range pod.Spec.Containers[0].Env {
			if envVar.Name == "PGBACKREST_DB_PATH" {
				podDbPath = envVar.Value
				break
			}
		}
		if podDbPath != "" {
			cmdOpts = append(cmdOpts, fmt.Sprintf("--db-path=%s", podDbPath))
		} else {
			log.Errorf("Unable to find PGBACKREST_DB_PATH on primary pod %s for backrest job %s",
				pod.Name, task.Spec.Parameters[config.LABEL_JOB_NAME])
			return
		}
	}
	// join options using a space
	commandOpts = strings.Join(cmdOpts, " ")
	return
}
