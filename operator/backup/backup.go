package backup

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	"k8s.io/client-go/rest"
	"os"
	"time"
)

type jobTemplateFields struct {
	Name               string
	JobName            string
	PvcName            string
	CCPImagePrefix     string
	CCPImageTag        string
	SecurityContext    string
	BackupHost         string
	BackupUserSecret   string
	BackupPort         string
	BackupOpts         string
	ContainerResources string
}

// AddBackupBase creates a backup job and its pvc
func AddBackupBase(clientset *kubernetes.Clientset, client *rest.RESTClient, job *crv1.Pgbackup, namespace string) {
	var err error

	if job.Spec.BackupStatus == crv1.UpgradeCompletedStatus {
		log.Warn("pgbackup " + job.Spec.Name + " already completed, not recreating it")
		return
	}

	log.Info("creating Pgbackup object" + " in namespace " + namespace)
	log.Info("created with Name=" + job.Spec.Name + " in namespace " + namespace)

	//create the PVC if necessary
	var pvcName string
	if job.Spec.BackupPVC != "" {
		pvcName = job.Spec.BackupPVC
	} else {
		pvcName, err = pvc.CreatePVC(clientset, &job.Spec.StorageSpec, job.Spec.Name+"-backup", job.Spec.BackupHost, namespace)
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info("created backup PVC =" + pvcName + " in namespace " + namespace)
		}
	}

	//update the pvc name in the CRD
	err = util.Patch(client, "/spec/storagespec/name", pvcName, "pgbackups", job.Spec.Name, namespace)

	cr := ""
	if operator.Pgo.DefaultBackupResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultBackupResources)
		if err != nil {
			log.Error(err)
			return
		}
		cr = operator.GetContainerResourcesJSON(&tmp)

	}

	//generate a JobName
	jobName := "backup-" + job.Spec.Name + "-" + util.RandStringBytesRmndr(4)

	//create the job -
	jobFields := jobTemplateFields{
		JobName:            jobName,
		Name:               job.Spec.Name,
		PvcName:            util.CreatePVCSnippet(job.Spec.StorageSpec.StorageType, pvcName),
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        job.Spec.CCPImageTag,
		SecurityContext:    util.CreateSecContext(job.Spec.StorageSpec.Fsgroup, job.Spec.StorageSpec.SupplementalGroups),
		BackupHost:         job.Spec.BackupHost,
		BackupUserSecret:   job.Spec.BackupUserSecret,
		BackupPort:         job.Spec.BackupPort,
		BackupOpts:         job.Spec.BackupOpts,
		ContainerResources: cr,
	}

	var doc2 bytes.Buffer
	err = operator.JobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		operator.JobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	_, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		return
	}

	//update the backup CRD status to submitted
	err = util.Patch(client, "/spec/backupstatus", crv1.UpgradeSubmittedStatus, "pgbackups", job.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

}

// DeleteBackupBase deletes a backup job
func DeleteBackupBase(clientset *kubernetes.Clientset, client *rest.RESTClient, job *crv1.Pgbackup, namespace string) {
	var jobName = "backup-" + job.Spec.Name

	err := kubeapi.DeleteJob(clientset, jobName, namespace)
	if err != nil {
		log.Error("error deleting Job " + jobName + err.Error())
		return
	}

	//make sure job is actually reporting as deleted
	for i := 0; i < 5; i++ {
		_, found := kubeapi.GetJob(clientset, jobName, namespace)
		if !found {
			break
		}
		if err != nil {
			log.Error(err)
		}
		log.Debug("waiting for backup job to report being deleted")
		time.Sleep(time.Second * time.Duration(3))
	}
}
