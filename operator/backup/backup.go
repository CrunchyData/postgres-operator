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
	"fmt"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"

	//	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/fields"
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

const JOB_PATH = "/pgconf/backup-job.json"

var JobTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(JOB_PATH)
	if err != nil {
		fmt.Println(err.Error())
		panic(err.Error())
	}
	JobTemplate = template.Must(template.New("backup job template").Parse(string(buf)))

}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.PgBackup)

	source := cache.NewListWatchFromClient(client, "pgbackups", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		job := obj.(*tpr.PgBackup)
		eventchan <- job
		addBackup(clientset, client, job)
	}
	createDeleteHandler := func(obj interface{}) {
		job := obj.(*tpr.PgBackup)
		eventchan <- job
		deleteBackup(clientset, client, job)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		job := obj.(*tpr.PgBackup)
		eventchan <- job
		//fmt.Println("updating PgBackup object")
		//fmt.Println("updated with Name=" + job.Spec.Name)
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
			fmt.Printf("%#v\n", event)
		}
	}

}

func addBackup(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgBackup) {
	fmt.Println("creating PgBackup object")
	fmt.Println("created with Name=" + job.Spec.Name)

	//create the job -
	jobFields := JobTemplateFields{
		Name:          job.Spec.Name,
		PVC_NAME:      job.Spec.PVC_NAME,
		CCP_IMAGE_TAG: job.Spec.CCP_IMAGE_TAG,
		BACKUP_HOST:   job.Spec.BACKUP_HOST,
		BACKUP_USER:   job.Spec.BACKUP_USER,
		BACKUP_PASS:   job.Spec.BACKUP_PASS,
		BACKUP_PORT:   job.Spec.BACKUP_PORT,
	}

	var doc2 bytes.Buffer
	err := JobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	jobDocString := doc2.String()
	fmt.Println(jobDocString)

	//newjob := v1beta1.Job{}
	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		fmt.Println("error unmarshalling json into Job ")
		fmt.Println(err.Error())
		return
	}

	//resultJob, err := clientset.ExtensionsV1beta1Client.Jobs(v1.NamespaceDefault).Create(&newjob)
	resultJob, err := clientset.Batch().Jobs(v1.NamespaceDefault).Create(&newjob)
	if err != nil {
		fmt.Println("error creating Job ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created Job " + resultJob.Name)

}

func deleteBackup(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgBackup) {
	fmt.Println("deleting PgBackup object")
	var jobName = "backup-" + job.Spec.Name
	fmt.Println("deleting Job with Name=" + jobName)

	//delete the job
	//err := clientset.ExtensionsV1beta1Client.Jobs(v1.NamespaceDefault).Delete(jobName,
	err := clientset.Batch().Jobs(v1.NamespaceDefault).Delete(jobName,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting Job " + jobName)
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted Job " + jobName)
}
