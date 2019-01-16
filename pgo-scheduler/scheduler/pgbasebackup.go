package scheduler

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
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

	taskName := fmt.Sprintf("backup-%s", b.cluster)

	result := crv1.Pgbackup{}
	found, err = kubeapi.Getpgbackup(restClient, &result, taskName, b.namespace)

	if found {
		err := kubeapi.Deletepgbackup(restClient, taskName, b.namespace)
		if err != nil {
			contextLogger.WithFields(log.Fields{
				"task":  taskName,
				"error": err,
			}).Error("error deleting pgBackup")
			return
		}
	} else if err != nil && !kerrors.IsNotFound(err) {
		contextLogger.WithFields(log.Fields{
			"task":  taskName,
			"error": err,
		}).Error("error getting pgbackup")
		return

	}

	job, found := kubeapi.GetJob(kubeClient, taskName, b.namespace)
	if found {
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
	}

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
