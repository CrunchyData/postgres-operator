package backrest

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
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
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
	CCPImagePrefix                string
	CCPImageTag                   string
	CustomLabels                  string
	SecurityContext               string
	PgbackrestStanza              string
	PgbackrestDBPath              string
	PgbackrestRepo1Path           string
	PgbackrestRepo1Type           crv1.BackrestStorageType
	BackrestLocalAndGCSStorage    bool
	BackrestLocalAndS3Storage     bool
	PgbackrestS3VerifyTLS         string
	PgbackrestRestoreVolumes      string
	PgbackrestRestoreVolumeMounts string
	Tolerations                   string
}

var (
	backrestPgHostRegex = regexp.MustCompile("--db-host|--pg1-host")
	backrestPgPathRegex = regexp.MustCompile("--db-path|--pg1-path")
)

// Backrest ...
func Backrest(namespace string, clientset kubeapi.Interface, task *crv1.Pgtask) {
	ctx := context.TODO()

	// get the cluster that is requesting the backup. if we cannot get the cluster
	// do not take the backup
	cluster, err := clientset.CrunchydataV1().Pgclusters(task.Namespace).Get(ctx,
		task.Spec.Parameters[config.LABEL_PG_CLUSTER], metav1.GetOptions{})

	if err != nil {
		log.Error(err)
		return
	}

	cmd := task.Spec.Parameters[config.LABEL_BACKREST_COMMAND]
	// determine the repo type. we need to make a special check for a standby
	// cluster (see below)
	repoType := operator.GetRepoType(cluster)

	// If this is a standby cluster and the stanza creation task, if posix storage
	// is specified then this ensures that the stanza is created on the local
	// repository only.
	//
	//The stanza for the S3/GCS repo will have already been created by the cluster
	// the standby is replicating from, and therefore does not need to be
	// attempted again.
	if cluster.Spec.Standby && cmd == crv1.PgtaskBackrestStanzaCreate {
		repoType = crv1.BackrestStorageTypePosix
	}

	// create the Job to run the backrest command
	jobFields := backrestJobTemplateFields{
		JobName:         task.Spec.Parameters[config.LABEL_JOB_NAME],
		ClusterName:     task.Spec.Parameters[config.LABEL_PG_CLUSTER],
		PodName:         task.Spec.Parameters[config.LABEL_POD_NAME],
		SecurityContext: `{"runAsNonRoot": true}`,
		Command:         cmd,
		CommandOpts:     task.Spec.Parameters[config.LABEL_BACKREST_OPTS],
		PITRTarget:      "",
		CCPImagePrefix:  util.GetValueOrDefault(task.Spec.Parameters[config.LABEL_IMAGE_PREFIX], operator.Pgo.Cluster.CCPImagePrefix),
		CCPImageTag: util.GetValueOrDefault(util.GetStandardImageTag(cluster.Spec.CCPImage, cluster.Spec.CCPImageTag),
			operator.Pgo.Cluster.CCPImageTag),
		CustomLabels:                  operator.GetLabelsFromMap(util.GetCustomLabels(cluster), false),
		PgbackrestStanza:              task.Spec.Parameters[config.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:              task.Spec.Parameters[config.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepo1Path:           task.Spec.Parameters[config.LABEL_PGBACKREST_REPO_PATH],
		PgbackrestRestoreVolumes:      "",
		PgbackrestRestoreVolumeMounts: "",
		PgbackrestRepo1Type:           repoType,
		BackrestLocalAndGCSStorage:    operator.IsLocalAndGCSStorage(cluster),
		BackrestLocalAndS3Storage:     operator.IsLocalAndS3Storage(cluster),
		PgbackrestS3VerifyTLS:         task.Spec.Parameters[config.LABEL_BACKREST_S3_VERIFY_TLS],
		Tolerations:                   util.GetTolerations(cluster.Spec.Tolerations),
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
		_ = config.BackrestjobTemplate.Execute(os.Stdout, jobFields)
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

	backupType := task.Spec.Parameters[config.LABEL_PGHA_BACKUP_TYPE]
	if backupType != "" {
		newjob.ObjectMeta.Labels[config.LABEL_PGHA_BACKUP_TYPE] = backupType
	}
	_, _ = clientset.BatchV1().Jobs(namespace).Create(ctx, &newjob, metav1.CreateOptions{})

	// publish backrest backup event
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
func CreateInitialBackup(clientset pgo.Interface, namespace, clusterName, podName string) (*crv1.Pgtask, error) {
	params := make(map[string]string)
	params[config.LABEL_PGHA_BACKUP_TYPE] = crv1.BackupTypeBootstrap
	return CreateBackup(clientset, namespace, clusterName, podName, params, "--type=full")
}

// CreatePostFailoverBackup creates a Pgtask in order to initiate the a pgBackRest backup following a failure
// event to ensure proper replica creation and/or reinitialization
func CreatePostFailoverBackup(clientset pgo.Interface, namespace, clusterName, podName string) (*crv1.Pgtask, error) {
	params := make(map[string]string)
	params[config.LABEL_PGHA_BACKUP_TYPE] = crv1.BackupTypeFailover
	return CreateBackup(clientset, namespace, clusterName, podName, params, "")
}

// CreateBackup creates a Pgtask in order to initiate a pgBackRest backup
func CreateBackup(clientset pgo.Interface, namespace, clusterName, podName string, params map[string]string,
	backupOpts string) (*crv1.Pgtask, error) {
	ctx := context.TODO()

	log.Debug("pgBackRest operator CreateBackup called")

	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var newInstance *crv1.Pgtask
	taskName := "backrest-backup-" + cluster.Name

	spec := crv1.PgtaskSpec{}
	spec.Name = taskName

	spec.TaskType = crv1.PgtaskBackrest
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_JOB_NAME] = "backrest-" + crv1.PgtaskBackrestBackup + "-" + cluster.Name
	spec.Parameters[config.LABEL_PG_CLUSTER] = cluster.Name
	spec.Parameters[config.LABEL_POD_NAME] = podName
	spec.Parameters[config.LABEL_CONTAINER_NAME] = "database"
	// pass along the appropriate image prefix for the backup task
	// this will be used by the associated backrest job
	spec.Parameters[config.LABEL_IMAGE_PREFIX] = util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix)
	spec.Parameters[config.LABEL_BACKREST_COMMAND] = crv1.PgtaskBackrestBackup
	spec.Parameters[config.LABEL_BACKREST_OPTS] = backupOpts
	// Get 'true' or 'false' for setting the pgBackRest S3 verify TLS value
	spec.Parameters[config.LABEL_BACKREST_S3_VERIFY_TLS] = operator.GetS3VerifyTLSSetting(cluster)

	for k, v := range params {
		spec.Parameters[k] = v
	}

	newInstance = &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = cluster.Name
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = cluster.ObjectMeta.Labels[config.LABEL_PGOUSER]

	_, err = clientset.CrunchydataV1().Pgtasks(cluster.Namespace).Create(ctx, newInstance, metav1.CreateOptions{})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return newInstance, nil
}

// CleanBackupResources is responsible for cleaning up Kubernetes resources from a previous
// pgBackRest backup.  Specifically, this function deletes the pgptask and job associate with a
// previous pgBackRest backup for the cluster.
func CleanBackupResources(clientset kubeapi.Interface, namespace, clusterName string) error {
	ctx := context.TODO()

	taskName := "backrest-backup-" + clusterName
	err := clientset.CrunchydataV1().Pgtasks(namespace).Delete(ctx, taskName, metav1.DeleteOptions{})
	if err != nil && !kubeapi.IsNotFound(err) {
		log.Error(err)
		return err
	}

	// remove previous backup job
	selector := config.LABEL_BACKREST_COMMAND + "=" + crv1.PgtaskBackrestBackup + "," +
		config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_BACKREST + "=true"
	deletePropagation := metav1.DeletePropagationForeground
	err = clientset.
		BatchV1().Jobs(namespace).
		DeleteCollection(ctx,
			metav1.DeleteOptions{PropagationPolicy: &deletePropagation},
			metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error(err)
	}

	if err := wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
		jobList, err := clientset.
			BatchV1().Jobs(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			log.Error(err)
			return false, err
		}

		return len(jobList.Items) == 0, nil
	}); err != nil {
		if errors.Is(err, wait.ErrWaitTimeout) {
			return fmt.Errorf("Timed out waiting for deletion of pgBackRest backup job for "+
				"cluster %s", clusterName)
		}

		return err
	}

	return nil
}

// getCommandOptsFromPod adds command line options from the primary pod to a backrest job.
// If not already specified in the command options provided in the pgtask, add the IP of the
// primary pod as the value for the "--db-host" parameter.  This will ensure direct
// communication between the repo pod and the primary via the primary's IP, instead of going
// through the primary pod's service (which could be unreliable). also if not already specified
// in the command options provided in the pgtask, then lookup the primary pod for the cluster
// and add the PGDATA dir of the pod as the value for the "--db-path" parameter
func getCommandOptsFromPod(clientset kubernetes.Interface, task *crv1.Pgtask,
	namespace string) (commandOpts string, err error) {
	ctx := context.TODO()

	// lookup the primary pod in order to determine the IP of the primary and the PGDATA directory for
	// the current primaty
	selector := fmt.Sprintf("%s=%s,%s in (%s,%s)", config.LABEL_PG_CLUSTER,
		task.Spec.Parameters[config.LABEL_PG_CLUSTER], config.LABEL_PGHA_ROLE,
		"promoted", config.LABEL_PGHA_ROLE_PRIMARY)

	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	// only consider pods that are running
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, options)

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
