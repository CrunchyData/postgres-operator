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
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

type restorejobTemplateFields struct {
	JobName             string
	ClusterName         string
	WorkflowID          string
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
		WorkflowID:          task.Spec.Parameters[crv1.PgtaskWorkflowID],
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

func UpdateRestoreWorkflow(restclient *rest.RESTClient, clientset *kubernetes.Clientset, clusterName, status, namespace, workflowID, restoreToName string) {
	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType
	log.Debugf("restore workflow: taskName is %s", taskName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow error: could not find a pgclustet in updateRestoreWorkflow for %s", clusterName)
		return
	}

	//turn off autofail if its on
	if cluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL] == "true" {
		cluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL] = "false"
		log.Debugf("restore workflow: turning off autofail on %s", clusterName)
		err = kubeapi.Updatepgcluster(restclient, &cluster, clusterName, namespace)
		if err != nil {
			log.Errorf("restore workflow error: could not turn off autofail on %s", clusterName)
			return
		}
	}

	//delete current primary deployment
	selector := util.LABEL_SERVICE_NAME + clusterName
	var depList *v1beta1.DeploymentList
	depList, err = kubeapi.GetDeployments(clientset, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get depList using %s", selector)
		return
	}
	if len(depList.Items) != 1 {
		log.Errorf("restore workflow error: depList not equal to 1 %s", selector)
		return
	}

	depToDelete := depList.Items[0]

	//delete the primary service as it will be recreated when
	//the new primary is created

	err = kubeapi.DeleteService(clientset, clusterName, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not delete primary service %s", clusterName)
		return
	}

	err = kubeapi.DeleteDeployment(clientset, depToDelete.Name, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not delete primary %s", depToDelete.Name)
		return
	}
	log.Debugf("restore workflow: deleted primary %s", depToDelete.Name)

	//create new deployment based on restored pvc
	//and include the workflowID in that new pgcluster as
	//a breakcrumb to keep the workflow going after the
	//new primary pod is Ready
	cluster.Spec.UserLabels[crv1.PgtaskWorkflowID] = workflowID

	log.Debugf("restore workflow: created restored primary was %s now %s", cluster.Spec.Name, restoreToName)
	cluster.Spec.Name = restoreToName
	err = kubeapi.Createpgcluster(restclient, &cluster, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not create primary %s", cluster.Name)
		return
	}

	//update workflow
	log.Debugf("restore workflow: update workflow %s", workflowID)
	selector = crv1.PgtaskWorkflowID + "=" + workflowID
	taskList := crv1.PgtaskList{}
	err = kubeapi.GetpgtasksBySelector(restclient, &taskList, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get workflow %s", workflowID)
		return
	}
	if len(taskList.Items) != 1 {
		log.Errorf("restore workflow error: workflow %s not found", workflowID)
		return
	}

	task := taskList.Items[0]
	task.Spec.Parameters[crv1.PgtaskWorkflowBackrestRestorePrimaryCreatedStatus] = time.Now().Format("2006-01-02.15.04.05")
	err = kubeapi.Updatepgtask(restclient, &task, task.Name, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not update workflow %s", workflowID)
		return
	}

	//update backrest repo with new data path

}
