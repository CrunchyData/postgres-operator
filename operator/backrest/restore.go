package backrest

/*
 Copyright 2018-2019 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"time"
)

type backrestRestoreVolumesFields struct {
	ToClusterPVCName   string
	FromClusterPVCName string
}

type backrestRestoreVolumeMountsFields struct {
}

// Restore ...
func Restore(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//use the storage config from pgo.yaml for Primary
	storage := operator.Pgo.Storage[operator.Pgo.PrimaryStorage]

	//create the "to-cluster" PVC to hold the new data
	pvcName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC]
	pgstoragespec := crv1.PgStorageSpec{}
	pgstoragespec.AccessMode = storage.AccessMode
	pgstoragespec.Size = storage.Size
	pgstoragespec.StorageType = storage.StorageType
	pgstoragespec.StorageClass = storage.StorageClass
	pgstoragespec.Fsgroup = storage.Fsgroup
	pgstoragespec.SupplementalGroups = storage.SupplementalGroups
	pgstoragespec.MatchLabels = storage.MatchLabels
	clusterName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC]

	_, found, err := kubeapi.GetPVC(clientset, pvcName, namespace)
	if !found {
		log.Debugf("pvc %s not found, will create as part of restore", pvcName)
		//delete the pvc if it already exists
		//kubeapi.DeletePVC(clientset, pvcName, namespace)

		//create the pvc
		err := pvc.Create(clientset, pvcName, clusterName, &pgstoragespec, namespace)
		if err != nil {
			log.Error(err.Error())
			return
		}
	} else {
		log.Debugf("pvc %s found, will NOT recreate as part of restore", pvcName)
	}

	//delete the job if it exists from a prior run
	kubeapi.DeleteJob(clientset, task.Spec.Name, namespace)
	//add a small sleep, this is due to race condition in delete propagation
	time.Sleep(time.Second * 3)

	//create the Job to run the backrest restore container

	jobFields := backrestJobTemplateFields{
		JobName:                       "backrest-restore-" + task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER] + "-to-" + pvcName,
		ClusterName:                   task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER],
		PodName:                       "na",
		Command:                       crv1.PgtaskBackrestRestore,
		SecurityContext:               util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		CommandOpts:                   task.Spec.Parameters[util.LABEL_BACKREST_OPTS],
		COImagePrefix:                 operator.Pgo.Pgo.COImagePrefix,
		COImageTag:                    operator.Pgo.Pgo.COImageTag,
		PgbackrestStanza:              task.Spec.Parameters[util.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:              task.Spec.Parameters[util.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepoPath:            task.Spec.Parameters[util.LABEL_PGBACKREST_REPO_PATH],
		PgbackrestRestoreVolumes:      getRestoreVolumes(task),
		PgbackrestRestoreVolumeMounts: getRestoreVolumeMounts(),
	}

	var doc2 bytes.Buffer
	err = operator.BackrestjobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		operator.BackrestjobTemplate.Execute(os.Stdout, jobFields)

	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	kubeapi.CreateJob(clientset, &newjob, namespace)

}

func getRestoreVolumes(task *crv1.Pgtask) string {
	var doc2 bytes.Buffer

	fields := backrestRestoreVolumesFields{
		FromClusterPVCName: task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER] + "-backrestrepo",
		ToClusterPVCName:   task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC],
	}

	err := operator.BackrestRestoreVolumesTemplate.Execute(&doc2, fields)
	if operator.CRUNCHY_DEBUG {
		operator.BackrestRestoreVolumesTemplate.Execute(os.Stdout, fields)
	}
	if err != nil {
		log.Error(err)
		return ""
	}

	return doc2.String()
}

func getRestoreVolumeMounts() string {
	var doc2 bytes.Buffer

	fields := backrestRestoreVolumeMountsFields{}

	err := operator.BackrestRestoreVolumeMountsTemplate.Execute(&doc2, fields)
	if err != nil {
		log.Error(err)
		return ""
	}
	if operator.CRUNCHY_DEBUG {
		operator.BackrestRestoreVolumeMountsTemplate.Execute(os.Stdout, fields)
	}

	return doc2.String()
}
