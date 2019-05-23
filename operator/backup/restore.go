package backup

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
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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

type restorejobTemplateFields struct {
	JobName          string
	ClusterName      string
	TaskName         string
	WorkflowID       string
	ToClusterPVCName string
	BackupPVCName    string
	SecurityContext  string
	CCPImagePrefix   string
	CCPImageTag      string
	NodeSelector     string
	BackupPath       string
	PgdataPath       string
}

// Restore uses a pgbasebackup pgtask to initiate the workflow to restore a cluster using a pg_basebackup backup.
// This includes deleting the existing cluster deployment, creating storage resources for the new (i.e. restored)
// cluster, and deploying the Kubernetes job that will perform the restore.
func Restore(restclient *rest.RESTClient, namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	clusterName := task.Spec.Parameters[config.LABEL_PGBASEBACKUP_RESTORE_FROM_CLUSTER]
	log.Debugf("restore workflow: started for cluster %s", clusterName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow error: could not find a pgcluster in Restore Workflow for %s", clusterName)
		return
	}

	//use the storage config from the pgcluster for the restored pvc
	storage := cluster.Spec.PrimaryStorage

	//create the "to-cluster" PVC to hold the new data
	pvcName, err := createRestorePVC(storage, clusterName, restclient, namespace, clientset, task)
	if err != nil {
		log.Error(err)
		return
	}

	//create the Job to run the pg_basebackup restore container
	workflowID := task.Spec.Parameters[crv1.PgtaskWorkflowID]
	jobFields := restorejobTemplateFields{
		JobName:          "pgbasebackup-restore-" + task.Spec.Parameters[config.LABEL_PGBASEBACKUP_RESTORE_FROM_CLUSTER] + "-" + util.RandStringBytesRmndr(4),
		TaskName:         task.Name,
		ClusterName:      task.Spec.Parameters[config.LABEL_PGBASEBACKUP_RESTORE_FROM_CLUSTER],
		SecurityContext:  util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		ToClusterPVCName: pvcName,
		BackupPVCName:    task.Spec.Parameters[config.LABEL_PGBASEBACKUP_RESTORE_FROM_PVC],
		WorkflowID:       workflowID,
		CCPImagePrefix:   operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:      operator.Pgo.Cluster.CCPImageTag,
		NodeSelector:     operator.GetAffinity(task.Spec.Parameters["NodeLabelKey"], task.Spec.Parameters["NodeLabelValue"], "In"),
		BackupPath:       task.Spec.Parameters[config.LABEL_PGBASEBACKUP_RESTORE_BACKUP_PATH],
		PgdataPath:       pvcName,
	}

	var doc2 bytes.Buffer
	err = config.PgBasebackupRestoreJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		log.Error("restore workflow: error executing job template")
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.PgBasebackupRestoreJobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("restore workflow: error unmarshalling json into Job " + err.Error())
		return
	}

	jobName, err := kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in creating restore job")
		return
	}
	log.Debugf("restore workflow: restore job %s created", jobName)

	err = updateWorkflow(restclient, workflowID, namespace, crv1.PgtaskWorkflowPgbasebackupRestoreJobCreatedStatus)
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in updating workflow status")
	}
}

func createRestorePVC(storage crv1.PgStorageSpec, clusterName string, restclient *rest.RESTClient, namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) (string, error) {
	var err error

	//create the "to-cluster" PVC to hold the new data
	pvcName := task.Spec.Parameters[config.LABEL_PGBASEBACKUP_RESTORE_TO_PVC]
	pgstoragespec := storage

	var found bool
	_, found, err = kubeapi.GetPVC(clientset, pvcName, namespace)
	if !found {
		log.Debugf("pvc %s not found, will create as part of restore", pvcName)
		//create the pvc
		err = pvc.Create(clientset, pvcName, clusterName, &pgstoragespec, namespace)
		if err != nil {
			log.Error(err.Error())
			return "", err
		}
	} else if err != nil {
		log.Error(err.Error())
		return "", err
	} else {
		log.Debugf("pvc %s found, will NOT recreate as part of restore", pvcName)
	}
	return pvcName, err

}

func updateWorkflow(restclient *rest.RESTClient, workflowID, namespace, status string) error {
	//update workflow
	log.Debugf("restore workflow: update workflow %s", workflowID)
	selector := crv1.PgtaskWorkflowID + "=" + workflowID
	taskList := crv1.PgtaskList{}
	err := kubeapi.GetpgtasksBySelector(restclient, &taskList, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get workflow %s", workflowID)
		return err
	}
	if len(taskList.Items) != 1 {
		log.Errorf("restore workflow error: workflow %s not found", workflowID)
		return errors.New("restore workflow error: workflow not found")
	}

	task := taskList.Items[0]
	task.Spec.Parameters[status] = time.Now().Format("2006-01-02.15.04.05")
	err = kubeapi.Updatepgtask(restclient, &task, task.Name, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not update workflow %s to status %s", workflowID, status)
		return err
	}
	return err
}
