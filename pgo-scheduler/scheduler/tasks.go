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
	"github.com/crunchydata/postgres-operator/config"
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
	storageType   string
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
				config.LABEL_JOB_NAME:              p.taskName,
				config.LABEL_PG_CLUSTER:            p.clusterName,
				config.LABEL_POD_NAME:              p.podName,
				config.LABEL_CONTAINER_NAME:        p.containerName,
				config.LABEL_BACKREST_COMMAND:      crv1.PgtaskBackrestBackup,
				config.LABEL_BACKREST_OPTS:         fmt.Sprintf("--stanza=%s %s", p.stanza, p.backupOptions),
				config.LABEL_BACKREST_STORAGE_TYPE: p.storageType,
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
