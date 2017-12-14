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
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/rest"
	"text/template"
)

type jobTemplateFields struct {
	Name            string
	PvcName         string
	CCPImagePrefix  string
	CCPImageTag     string
	SecurityContext string
	BackupHost      string
	BackupUser      string
	BackupPass      string
	BackupPort      string
}

const jobPath = "/operator-conf/backup-job.json"

var jobTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(jobPath)
	if err != nil {
		log.Error("error in backup.go init " + err.Error())
		panic(err.Error())
	}
	jobTemplate = template.Must(template.New("backup job template").Parse(string(buf)))

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
	pvcName, err = pvc.CreatePVC(clientset, job.Spec.Name+"-backup", &job.Spec.StorageSpec, namespace)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("created backup PVC =" + pvcName + " in namespace " + namespace)
	}

	//update the pvc name in the CRD
	err = util.Patch(client, "/spec/storagespec/name", pvcName, "pgbackups", job.Spec.Name, namespace)

	//create the job -
	jobFields := jobTemplateFields{
		Name:            job.Spec.Name,
		PvcName:         util.CreatePVCSnippet(job.Spec.StorageSpec.StorageType, pvcName),
		CCPImagePrefix:  operator.CCPImagePrefix,
		CCPImageTag:     job.Spec.CCPImageTag,
		SecurityContext: util.CreateSecContext(job.Spec.StorageSpec.Fsgroup, job.Spec.StorageSpec.SupplementalGroups),
		BackupHost:      job.Spec.BackupHost,
		BackupUser:      job.Spec.BackupUser,
		BackupPass:      job.Spec.BackupPass,
		BackupPort:      job.Spec.BackupPort,
	}

	var doc2 bytes.Buffer
	err = jobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	resultJob, err := clientset.Batch().Jobs(namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return
	}
	log.Info("created Job " + resultJob.Name)

	//update the backup CRD status to submitted
	err = util.Patch(client, "/spec/backupstatus", crv1.UpgradeSubmittedStatus, "pgbackups", job.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

}

// DeleteBackupBase deletes a backup job
func DeleteBackupBase(clientset *kubernetes.Clientset, client *rest.RESTClient, job *crv1.Pgbackup, namespace string) {
	var jobName = "backup-" + job.Spec.Name
	log.Debug("deleting Job with Name=" + jobName + " in namespace " + namespace)

	//delete the job
	err := clientset.Batch().Jobs(namespace).Delete(jobName,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Job " + jobName + err.Error())
		return
	}
	log.Debug("deleted Job " + jobName)
}
