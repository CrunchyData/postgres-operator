package scheduler

import (
	"fmt"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pgBaseBackupTask struct {
	clusterName string
	taskName    string
	ccpImageTag string
	hostname    string
	port        string
	status      string
	secret      string
	pvc         string
	opts        string
}

type pgBackRestTask struct {
	clusterName   string
	taskName      string
	podName       string
	containerName string
	backupOptions string
	stanza        string
}

func (p pgBackRestTask) NewBackRestTask() *crv1.Pgtask {
	return &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: p.taskName,
		},
		Spec: crv1.PgtaskSpec{
			Name:     p.taskName,
			TaskType: crv1.PgtaskBackrest,
			Parameters: map[string]string{
				util.LABEL_JOB_NAME:         p.taskName,
				util.LABEL_PG_CLUSTER:       p.clusterName,
				util.LABEL_POD_NAME:         p.podName,
				util.LABEL_CONTAINER_NAME:   p.containerName,
				util.LABEL_BACKREST_COMMAND: crv1.PgtaskBackrestBackup,
				util.LABEL_BACKREST_OPTS:    fmt.Sprintf("--stanza=%s %s", p.stanza, p.backupOptions),
			},
		},
	}
}

func (p pgBaseBackupTask) NewBaseBackupTask() *crv1.Pgbackup {
	return &crv1.Pgbackup{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: p.taskName,
		},
		Spec: crv1.PgbackupSpec{
			Name:             p.clusterName,
			CCPImageTag:      p.ccpImageTag,
			BackupHost:       p.hostname,
			BackupPort:       p.port,
			BackupUserSecret: p.secret,
			BackupStatus:     p.status,
			BackupOpts:       p.opts,
			BackupPVC:        p.pvc,
		},
	}
}
