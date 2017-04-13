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

package upgrade

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/crunchydata/postgres-operator/operator/pvc"
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
	Name              string
	OLD_PVC_NAME      string
	NEW_PVC_NAME      string
	CCP_IMAGE_TAG     string
	OLD_DATABASE_NAME string
	NEW_DATABASE_NAME string
	OLD_VERSION       string
	NEW_VERSION       string
}

const JOB_PATH = "/pgconf/postgres-operator/upgrade-job.json"

var JobTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(JOB_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	JobTemplate = template.Must(template.New("upgrade job template").Parse(string(buf)))

}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgUpgrade)

	source := cache.NewListWatchFromClient(client, "pgupgrades", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		job := obj.(*tpr.PgUpgrade)
		eventchan <- job
		addUpgrade(clientset, client, job, namespace)
	}
	createDeleteHandler := func(obj interface{}) {
		job := obj.(*tpr.PgUpgrade)
		eventchan <- job
		deleteUpgrade(clientset, client, job, namespace)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		job := obj.(*tpr.PgUpgrade)
		eventchan <- job
		//log.Info("updating PgUpgrade object")
		//log.Info("updated with Name=" + job.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgUpgrade{},
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

func addUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgUpgrade, namespace string) {
	log.Info("addUpgrade called " + " in namespace " + namespace)
	log.Info("Name=" + job.Spec.Name + " in namespace " + namespace)
	if true {
		log.Info(" resource type is " + job.Spec.RESOURCE_TYPE)
		return
	}
	if job.Spec.RESOURCE_TYPE == "database" {
		addUpgradeDatabase(clientset, client, job, namespace)
	} else if job.Spec.RESOURCE_TYPE == "cluster" {
		addUpgradeCluster(clientset, client, job, namespace)
	} else {
		log.Error("error in addUpgrade. unknown RESOURCE_TYPE " + job.Spec.RESOURCE_TYPE)
		return
	}

}

func addUpgradeDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgUpgrade, namespace string) {
	var err error
	//stop old database

	//create the new database PVC if necessary
	if job.Spec.NEW_PVC_NAME == "" {
		job.Spec.NEW_PVC_NAME = job.Spec.Name + "-upgrade-pvc"
		err = pvc.Create(clientset, job.Spec.NEW_PVC_NAME, job.Spec.PVC_ACCESS_MODE, job.Spec.PVC_SIZE, namespace)
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Info("created upgrade PVC =" + job.Spec.NEW_PVC_NAME + " in namespace " + namespace)

	}

	//if major upgrade, create the upgrade job
	jobFields := JobTemplateFields{
		Name:              job.Spec.Name,
		NEW_PVC_NAME:      job.Spec.NEW_PVC_NAME,
		OLD_PVC_NAME:      job.Spec.OLD_PVC_NAME,
		CCP_IMAGE_TAG:     job.Spec.CCP_IMAGE_TAG,
		OLD_DATABASE_NAME: job.Spec.OLD_DATABASE_NAME,
		NEW_DATABASE_NAME: job.Spec.NEW_DATABASE_NAME,
		OLD_VERSION:       job.Spec.OLD_VERSION,
		NEW_VERSION:       job.Spec.NEW_VERSION,
	}

	var doc2 bytes.Buffer
	err = JobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	//newjob := v1beta1.Job{}
	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	//resultJob, err := clientset.ExtensionsV1beta1Client.Jobs(v1.NamespaceDefault).Create(&newjob)
	resultJob, err := clientset.Batch().Jobs(namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return
	}
	log.Info("created Job " + resultJob.Name)

	//create watch of job

	//if success, start new database pod

}

func addUpgradeCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgUpgrade, namespace string) {
	var err error
	//stop old database

	//create the new database PVC if necessary
	if job.Spec.NEW_PVC_NAME == "" {
		job.Spec.NEW_PVC_NAME = job.Spec.Name + "-upgrade-pvc"
		err = pvc.Create(clientset, job.Spec.NEW_PVC_NAME, job.Spec.PVC_ACCESS_MODE, job.Spec.PVC_SIZE, namespace)
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Info("created upgrade PVC =" + job.Spec.NEW_PVC_NAME + " in namespace " + namespace)

	}

	//if major upgrade, create the upgrade job
	jobFields := JobTemplateFields{
		Name:              job.Spec.Name,
		NEW_PVC_NAME:      job.Spec.NEW_PVC_NAME,
		OLD_PVC_NAME:      job.Spec.OLD_PVC_NAME,
		CCP_IMAGE_TAG:     job.Spec.CCP_IMAGE_TAG,
		OLD_DATABASE_NAME: job.Spec.OLD_DATABASE_NAME,
		NEW_DATABASE_NAME: job.Spec.NEW_DATABASE_NAME,
		OLD_VERSION:       job.Spec.OLD_VERSION,
		NEW_VERSION:       job.Spec.NEW_VERSION,
	}

	var doc2 bytes.Buffer
	err = JobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	//newjob := v1beta1.Job{}
	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	//resultJob, err := clientset.ExtensionsV1beta1Client.Jobs(v1.NamespaceDefault).Create(&newjob)
	resultJob, err := clientset.Batch().Jobs(namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return
	}
	log.Info("created Job " + resultJob.Name)

	//create watch of job

	//if success, start new database pod

}

func deleteUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, job *tpr.PgUpgrade, namespace string) {
	var jobName = "upgrade-" + job.Spec.Name
	log.Debug("deleting Job with Name=" + jobName + " in namespace " + namespace)

	//delete the job
	//err := clientset.ExtensionsV1beta1Client.Jobs(v1.NamespaceDefault).Delete(jobName,
	err := clientset.Batch().Jobs(namespace).Delete(jobName,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Job " + jobName + err.Error())
		return
	}
	log.Debug("deleted Job " + jobName)
}
