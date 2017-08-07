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

package backup

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type JobTemplateFields struct {
	Name          string
	PVC_NAME      string
	CCP_IMAGE_TAG string
	BACKUP_HOST   string
	BACKUP_USER   string
	BACKUP_PASS   string
	BACKUP_PORT   string
}

const JOB_PATH = "/operator-conf/backup-job.json"

var JobTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(JOB_PATH)
	if err != nil {
		log.Error("error in backup.go init " + err.Error())
		panic(err.Error())
	}
	JobTemplate = template.Must(template.New("backup job template").Parse(string(buf)))

}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgBackup)

	source := cache.NewListWatchFromClient(client, "pgbackups", namespace, fields.Everything())

	createAddHandler := func(obj interface{}) {
		job := obj.(*tpr.PgBackup)
		eventchan <- job
		addBackup(clientset, client, job, namespace)
	}
	createDeleteHandler := func(obj interface{}) {
		job := obj.(*tpr.PgBackup)
		eventchan <- job
		deleteBackup(clientset, client, job, namespace)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		job := obj.(*tpr.PgBackup)
		eventchan <- job
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgBackup{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createAddHandler,
			UpdateFunc: updateHandler,
			DeleteFunc: createDeleteHandler,
		})

	go controller.Run(stopchan)

	for {
		select {
		case event := <-eventchan:
			//log.Infof("%#v\n", event)
			if event == nil {
				log.Info("event was null")
			}
		}
	}

}

func addBackup(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgBackup, namespace string) {
	var err error

	if job.Spec.BACKUP_STATUS == tpr.UPGRADE_COMPLETED_STATUS {
		log.Warn("pgbackup " + job.Spec.Name + " already completed, not recreating it")
		return
	}

	log.Info("creating PgBackup object" + " in namespace " + namespace)
	log.Info("created with Name=" + job.Spec.Name + " in namespace " + namespace)

	//create the PVC if necessary
	if job.Spec.StorageSpec.PvcName == "" {
		job.Spec.StorageSpec.PvcName = job.Spec.Name + "-backup-pvc"
		err = pvc.Create(clientset, job.Spec.StorageSpec.PvcName, job.Spec.StorageSpec.PvcAccessMode, job.Spec.StorageSpec.PvcSize, job.Spec.StorageSpec.StorageType, job.Spec.StorageSpec.StorageClass, namespace)
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info("created backup PVC =" + job.Spec.StorageSpec.PvcName + " in namespace " + namespace)
		}

	}

	//update the pvc name in the TPR
	err = util.Patch(client, "/spec/pvcname", job.Spec.StorageSpec.PvcName, "pgbackups", job.Spec.Name, namespace)

	//create the job -
	jobFields := JobTemplateFields{
		Name:          job.Spec.Name,
		PVC_NAME:      job.Spec.StorageSpec.PvcName,
		CCP_IMAGE_TAG: job.Spec.CCP_IMAGE_TAG,
		BACKUP_HOST:   job.Spec.BACKUP_HOST,
		BACKUP_USER:   job.Spec.BACKUP_USER,
		BACKUP_PASS:   job.Spec.BACKUP_PASS,
		BACKUP_PORT:   job.Spec.BACKUP_PORT,
	}

	var doc2 bytes.Buffer
	err = JobTemplate.Execute(&doc2, jobFields)
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

	//update the backup TPR status to submitted
	err = util.Patch(client, "/spec/backupstatus", tpr.UPGRADE_SUBMITTED_STATUS, "pgbackups", job.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

}

func deleteBackup(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgBackup, namespace string) {
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
