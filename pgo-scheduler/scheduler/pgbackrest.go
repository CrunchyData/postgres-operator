package scheduler

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
	"fmt"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

type BackRestBackupJob struct {
	backupType  string
	stanza      string
	namespace   string
	deployment  string
	label       string
	container   string
	cluster     string
	storageType string
}

func (s *ScheduleTemplate) NewBackRestSchedule() BackRestBackupJob {
	return BackRestBackupJob{
		backupType:  s.PGBackRest.Type,
		stanza:      "db",
		namespace:   s.Namespace,
		deployment:  s.PGBackRest.Deployment,
		label:       s.PGBackRest.Label,
		container:   s.PGBackRest.Container,
		cluster:     s.Cluster,
		storageType: s.PGBackRest.StorageType,
	}
}

func (b BackRestBackupJob) Run() {
	contextLogger := log.WithFields(log.Fields{
		"namespace":   b.namespace,
		"deployment":  b.deployment,
		"label":       b.label,
		"container":   b.container,
		"backupType":  b.backupType,
		"cluster":     b.cluster,
		"storageType": b.storageType})

	contextLogger.Info("Running pgBackRest backup")

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(restClient, &cluster, b.cluster, b.namespace)

	if !found {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("pgCluster not found")
		return
	} else if err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("error retrieving pgCluster")
		return
	}

	taskName := fmt.Sprintf("%s-backrest-%s-backup-schedule", b.cluster, b.backupType)

	result := crv1.Pgtask{}
	found, err = kubeapi.Getpgtask(restClient, &result, taskName, b.namespace)

	if found {
		err := kubeapi.Deletepgtask(restClient, taskName, b.namespace)
		if err != nil {
			contextLogger.WithFields(log.Fields{
				"task":  taskName,
				"error": err,
			}).Error("error deleting pgTask")
			return
		}

		job, _ := kubeapi.GetJob(kubeClient, taskName, b.namespace)

		err = kubeapi.DeleteJob(kubeClient, taskName, b.namespace)
		if err != nil {
			contextLogger.WithFields(log.Fields{
				"task":  taskName,
				"error": err,
			}).Error("error deleting backup job")
			return
		}

		timeout := time.Second * 60
		err = kubeapi.IsJobDeleted(kubeClient, b.namespace, job, timeout)
		if err != nil {
			contextLogger.WithFields(log.Fields{
				"task":  taskName,
				"error": err,
			}).Error("error waiting for job to delete")
			return
		}
	} else if err != nil && !kerrors.IsNotFound(err) {
		contextLogger.WithFields(log.Fields{
			"task":  taskName,
			"error": err,
		}).Error("error getting pgTask")
		return
	}

	selector := fmt.Sprintf("%s=%s,pgo-backrest-repo=true", config.LABEL_PG_CLUSTER, b.cluster)
	pods, err := kubeapi.GetPods(kubeClient, selector, b.namespace)
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"selector": selector,
			"error":    err,
		}).Error("error getting pods from selector")
		return
	}

	if len(pods.Items) != 1 {
		contextLogger.WithFields(log.Fields{
			"selector":  selector,
			"error":     err,
			"podsFound": len(pods.Items),
		}).Error("pods returned does not equal 1, it should")
		return
	}

	backrest := pgBackRestTask{
		clusterName:   cluster.Name,
		taskName:      taskName,
		podName:       pods.Items[0].Name,
		containerName: "database",
		backupOptions: fmt.Sprintf("--type=%s", b.backupType),
		stanza:        b.stanza,
		storageType:   b.storageType,
	}

	err = kubeapi.Createpgtask(restClient, backrest.NewBackRestTask(), b.namespace)
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("could not create new pgtask")
		return
	}
}
