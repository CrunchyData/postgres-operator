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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

type BaseBackupJob struct {
	backupType  string
	ccpImageTag string
	cluster     string
	container   string
	deployment  string
	hostname    string
	label       string
	namespace   string
	port        string
	pvc         string
	secret      string
}

func (s *ScheduleTemplate) NewBaseBackupSchedule() BaseBackupJob {
	return BaseBackupJob{
		namespace:   s.Namespace,
		deployment:  s.PGBackRest.Deployment,
		label:       s.PGBackRest.Label,
		container:   s.PGBackRest.Container,
		cluster:     s.Cluster,
		ccpImageTag: s.PGBaseBackup.ImageTag,
		hostname:    s.Cluster,
		secret:      s.PGBaseBackup.Secret,
		port:        s.PGBaseBackup.Port,
		pvc:         s.PGBaseBackup.BackupVolume,
	}
}

func (b BaseBackupJob) Run() {
	contextLogger := log.WithFields(log.Fields{
		"namespace":  b.namespace,
		"backupType": b.backupType,
		"cluster":    b.cluster})

	contextLogger.Info("Running pgBaseBackup schedule")

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

	taskName := fmt.Sprintf("%s", b.cluster)

	result := crv1.Pgbackup{}
	found, err = kubeapi.Getpgbackup(restClient, &result, taskName, b.namespace)

	if found {
		//update the status to re-submitted
		result.Spec.BackupStatus = crv1.PgBackupJobReSubmitted

		err = kubeapi.Updatepgbackup(restClient, &result, taskName, b.namespace)

		if err != nil {
			contextLogger.WithFields(log.Fields{
				"task":  taskName,
				"error": err,
			}).Error("error updating pgbackup")
			return
		}
	} else if err != nil && !kerrors.IsNotFound(err) {
		contextLogger.WithFields(log.Fields{
			"task":  taskName,
			"error": err,
		}).Error("error getting pgbackup")
		return

	}

	//if the pgbackup doesn't exist, create it
	if !found {
		basebackup := pgBaseBackupTask{
			clusterName: cluster.Name,
			taskName:    taskName,
			ccpImageTag: b.ccpImageTag,
			hostname:    cluster.Spec.PrimaryHost,
			port:        cluster.Spec.Port,
			status:      "initial",
			pvc:         b.pvc,
			secret:      cluster.Spec.PrimarySecretName,
		}

		task := basebackup.NewBaseBackupTask()
		err = kubeapi.Createpgbackup(restClient, task, b.namespace)
		if err != nil {
			contextLogger.WithFields(log.Fields{
				"task":  taskName,
				"error": err,
			}).Error("error creating backup task")
			return
		}
	}
}
