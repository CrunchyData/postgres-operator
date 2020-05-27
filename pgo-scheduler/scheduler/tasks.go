package scheduler

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pgBackRestTask struct {
	clusterName   string
	taskName      string
	podName       string
	containerName string
	backupOptions string
	stanza        string
	storageType   string
	imagePrefix   string
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
				config.LABEL_IMAGE_PREFIX:          p.imagePrefix,
			},
		},
	}
}
