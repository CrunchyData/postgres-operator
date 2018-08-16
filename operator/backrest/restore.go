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
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"time"
)

type backrestRestoreConfigMapTemplateFields struct {
	ToClusterName        string
	FromClusterName      string
	RestoreConfigMapName string
}

type backrestRestoreJobTemplateFields struct {
	RestoreName          string
	SecurityContext      string
	ToClusterName        string
	RestoreConfigMapName string
	FromClusterPVCName   string
	ToClusterPVCName     string
	BackrestRestoreOpts  string
	DeltaEnvVar          string
	PITRTargetEnvVar     string
	CCPImagePrefix       string
	CCPImageTag          string
}

// Restore ...
func Restore(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//use the storage config from pgo.yaml for Primary
	storage := operator.Pgo.Storage[operator.Pgo.PrimaryStorage]

	//create the "to-cluster" PVC to hold the new data
	pvcName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_CLUSTER]
	pgstoragespec := crv1.PgStorageSpec{}
	pgstoragespec.AccessMode = storage.AccessMode
	pgstoragespec.Size = storage.Size
	pgstoragespec.StorageType = storage.StorageType
	pgstoragespec.StorageClass = storage.StorageClass
	pgstoragespec.Fsgroup = storage.Fsgroup
	pgstoragespec.SupplementalGroups = storage.SupplementalGroups
	pgstoragespec.MatchLabels = storage.MatchLabels
	clusterName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_CLUSTER]

	_, found, err := kubeapi.GetPVC(clientset, pvcName, namespace)
	if !found {
		log.Debug("pvc " + pvcName + " not found, will create as part of restore")
		//delete the pvc if it already exists
		//kubeapi.DeletePVC(clientset, pvcName, namespace)

		//create the pvc
		err := pvc.Create(clientset, pvcName, clusterName, &pgstoragespec, namespace)
		if err != nil {
			log.Error(err.Error())
			return
		}
	} else {
		log.Debug("pvc " + pvcName + " found, will NOT recreate as part of restore")
	}

	//delete the configmap if it exists from a prior run
	kubeapi.DeleteConfigMap(clientset, task.Spec.Name, namespace)

	//create the backrest-restore configmap

	err = createRestoreJobConfigMap(clientset, task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_CLUSTER], task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER], task.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
		return
	}

	//delete the job if it exists from a prior run
	kubeapi.DeleteJob(clientset, task.Spec.Name, namespace)
	//add a small sleep, this is due to race condition in delete propagation
	time.Sleep(time.Second * 3)

	//create the Job to run the backrest restore container

	jobFields := backrestRestoreJobTemplateFields{
		RestoreName:          task.Spec.Name,
		SecurityContext:      util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		ToClusterName:        task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_CLUSTER],
		RestoreConfigMapName: task.Spec.Name,
		FromClusterPVCName:   task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER],
		ToClusterPVCName:     task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_CLUSTER],
		BackrestRestoreOpts:  task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_OPTS],

		CCPImagePrefix: operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:    operator.Pgo.Cluster.CCPImageTag,
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

// Create a restore job configmap
func createRestoreJobConfigMap(clientset *kubernetes.Clientset, toName, fromName, mapName string, namespace string) error {
	log.Debug("in createRestoreJobConfigMap")
	var doc2 bytes.Buffer
	var err error

	configmapFields := backrestRestoreConfigMapTemplateFields{
		ToClusterName:        toName,
		FromClusterName:      fromName,
		RestoreConfigMapName: mapName,
	}

	err = operator.BackrestRestoreConfigMapTemplate.Execute(&doc2, configmapFields)
	if operator.CRUNCHY_DEBUG {
		operator.BackrestRestoreConfigMapTemplate.Execute(os.Stdout, configmapFields)
	}

	newConfigMap := v1.ConfigMap{}
	err = json.Unmarshal(doc2.Bytes(), &newConfigMap)
	if err != nil {
		log.Error("error unmarshalling json into ConfigMap " + err.Error())
		return err
	}

	err = kubeapi.CreateConfigMap(clientset, &newConfigMap, namespace)
	if err != nil {
		return err
	}
	return err

}

/**
func getDeltaEnvVar(restoretype string) string {
	if restoretype == util.LABEL_BACKREST_RESTORE_DELTA {
		return "{ \"name\": \"DELTA\"" + "},"
	}
	return ""
}
func getPITREnvVar(restoretype, pitrtarget string) string {
	if restoretype == util.LABEL_BACKREST_RESTORE_PITR {
		tmp := "{"
		tmp = tmp + "\"name\":" + " \"PITR_TARGET\","
		tmp = tmp + "\"value\":" + " \"" + pitrtarget + "\""
		tmp = tmp + "},"
		return tmp
	}
	return ""
}
*/
