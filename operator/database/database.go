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

package database

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

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ServiceTemplateFields struct {
	Name string
	Port string
}

type PodTemplateFields struct {
	Name               string
	Port               string
	PVC_NAME           string
	CCP_IMAGE_TAG      string
	PG_MASTER_USER     string
	PG_MASTER_PASSWORD string
	PG_USER            string
	PG_PASSWORD        string
	PG_DATABASE        string
	PG_ROOT_PASSWORD   string
	BACKUP_PVC_NAME    string
	BACKUP_PATH        string
	SECURITY_CONTEXT   string
}

const SERVICE_PATH = "/pgconf/database-service.json"
const POD_PATH = "/pgconf/database-pod.json"
const RESTORE_POD_PATH = "/pgconf/restore-database-pod.json"

var PodTemplate *template.Template
var RestorePodTemplate *template.Template
var ServiceTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(POD_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	PodTemplate = template.Must(template.New("pod template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(RESTORE_POD_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	RestorePodTemplate = template.Must(template.New("restore pod template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(SERVICE_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}

	ServiceTemplate = template.Must(template.New("service template").Parse(string(buf)))
}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.PgDatabase)

	source := cache.NewListWatchFromClient(client, "pgdatabases", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		db := obj.(*tpr.PgDatabase)
		eventchan <- db
		addDatabase(clientset, client, db)
	}
	createDeleteHandler := func(obj interface{}) {
		db := obj.(*tpr.PgDatabase)
		eventchan <- db
		deleteDatabase(clientset, client, db)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		db := obj.(*tpr.PgDatabase)
		eventchan <- db
		//log.Info("updating PgDatabase object")
		//log.Info("updated with Name=" + db.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgDatabase{},
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
			if event == nil {
				//log.Infof("%#v\n", event)
			}
		}
	}

}

// database consists of a Service and a Pod
func addDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase) {
	var err error
	log.Info("creating PgDatabase object")

	if db.Spec.PVC_NAME == "" {
		db.Spec.PVC_NAME = db.Spec.Name + "-pvc"
		log.Debug("PVC_NAME=%s PVC_SIZE=%s PVC_ACCESS_MODE=%s\n",
			db.Spec.PVC_NAME, db.Spec.PVC_ACCESS_MODE, db.Spec.PVC_SIZE)
		err = pvc.Create(clientset, db.Spec.PVC_NAME, db.Spec.PVC_ACCESS_MODE, db.Spec.PVC_SIZE)
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Info("created PVC =" + db.Spec.PVC_NAME)
	}

	//create the service - TODO get these fields from
	//the TPR instance
	serviceFields := ServiceTemplateFields{
		Name: db.Spec.Name,
		Port: "5432",
	}

	var doc bytes.Buffer
	err = ServiceTemplate.Execute(&doc, serviceFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	serviceDocString := doc.String()
	log.Info(serviceDocString)

	service := v1.Service{}
	err = json.Unmarshal(doc.Bytes(), &service)
	if err != nil {
		log.Error("error unmarshalling json into Service " + err.Error())
		return
	}

	//var result api.Service

	svc, err := clientset.Services(v1.NamespaceDefault).Create(&service)
	if err != nil {
		log.Error("error creating Service " + err.Error())
		return
	}
	log.Info("created service " + svc.Name)

	podFields := PodTemplateFields{
		Name:               db.Spec.Name,
		Port:               db.Spec.Port,
		PVC_NAME:           db.Spec.PVC_NAME,
		CCP_IMAGE_TAG:      db.Spec.CCP_IMAGE_TAG,
		PG_MASTER_USER:     db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD: db.Spec.PG_MASTER_PASSWORD,
		PG_USER:            db.Spec.PG_USER,
		PG_PASSWORD:        db.Spec.PG_PASSWORD,
		PG_DATABASE:        db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:   db.Spec.PG_ROOT_PASSWORD,
		BACKUP_PVC_NAME:    db.Spec.BACKUP_PVC_NAME,
		BACKUP_PATH:        db.Spec.BACKUP_PATH,
		SECURITY_CONTEXT:   util.CreateSecContext(db.Spec.FS_GROUP, db.Spec.SUPPLEMENTAL_GROUPS),
	}

	var doc2 bytes.Buffer
	//the client should make sure that BOTH
	//the backup pvc and backup path are specified if at all
	if db.Spec.BACKUP_PVC_NAME != "" {
		err = RestorePodTemplate.Execute(&doc2, podFields)
		log.Infof("doing a restore!!! with pvc %s and path %s\n", db.Spec.BACKUP_PVC_NAME, db.Spec.BACKUP_PATH)
	} else {
		err = PodTemplate.Execute(&doc2, podFields)
	}
	if err != nil {
		log.Error(err.Error())
		return
	}
	podDocString := doc2.String()
	log.Info(podDocString)

	pod := v1.Pod{}
	err = json.Unmarshal(doc2.Bytes(), &pod)
	if err != nil {
		log.Error("error unmarshalling json into Pod " + err.Error())
		return
	}

	resultPod, err := clientset.Pods(v1.NamespaceDefault).Create(&pod)
	if err != nil {
		log.Error("error creating Pod " + err.Error())
		return
	}
	log.Info("created pod " + resultPod.Name)

}

func deleteDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase) {
	log.Info("deleting PgDatabase object")
	log.Info("deleting with Name=" + db.Spec.Name)

	//delete the service
	err := clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Service " + err.Error())
		return
	}
	log.Info("deleted service " + db.Spec.Name)

	//delete the pod
	err = clientset.Pods(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Pod " + err.Error())
		return
	}
	log.Info("deleted pod " + db.Spec.Name)

	err = pvc.Delete(clientset, db.Spec.Name+"-pvc")
	if err != nil {
		log.Error("error deleting pvc " + err.Error())
		return
	}

}
