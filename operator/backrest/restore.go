package backrest

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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

type restorejobTemplateFields struct {
	JobName             string
	ClusterName         string
	ToClusterPVCName    string
	SecurityContext     string
	COImagePrefix       string
	COImageTag          string
	CommandOpts         string
	PITRTarget          string
	PgbackrestStanza    string
	PgbackrestDBPath    string
	PgbackrestRepo1Path string
	PgbackrestRepo1Host string
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

	jobFields := restorejobTemplateFields{
		JobName:             "backrest-restore-" + task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER] + "-to-" + pvcName,
		ClusterName:         task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER],
		SecurityContext:     util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		ToClusterPVCName:    pvcName,
		CommandOpts:         task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_OPTS],
		PITRTarget:          task.Spec.Parameters[util.LABEL_BACKREST_PITR_TARGET],
		COImagePrefix:       operator.Pgo.Pgo.COImagePrefix,
		COImageTag:          operator.Pgo.Pgo.COImageTag,
		PgbackrestStanza:    task.Spec.Parameters[util.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:    task.Spec.Parameters[util.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepo1Path: task.Spec.Parameters[util.LABEL_PGBACKREST_REPO_PATH],
		PgbackrestRepo1Host: task.Spec.Parameters[util.LABEL_PGBACKREST_REPO_HOST],
	}

	var doc2 bytes.Buffer
	err = operator.BackrestRestorejobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		operator.BackrestRestorejobTemplate.Execute(os.Stdout, jobFields)

	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	kubeapi.CreateJob(clientset, &newjob, namespace)

}
