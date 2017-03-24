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
	"text/template"

	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type DatabaseStrategy1 struct{}

var PodTemplate1 *template.Template
var RestorePodTemplate1 *template.Template
var ServiceTemplate1 *template.Template

func init() {

	ServiceTemplate1 = util.LoadTemplate("/pgconf/postgres-operator/database/1/database-service.json")
	PodTemplate1 = util.LoadTemplate("/pgconf/postgres-operator/database/1/database-pod.json")
	RestorePodTemplate1 = util.LoadTemplate("/pgconf/postgres-operator/database/1/restore-database-pod.json")

}

// database consists of a Service and a Pod
func (r DatabaseStrategy1) AddDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase) error {
	var err error
	log.Info("creating PgDatabase object using Strategy 1")

	serviceFields := ServiceTemplateFields{
		Name: db.Spec.Name,
		Port: db.Spec.Port,
	}

	var doc bytes.Buffer
	err = ServiceTemplate1.Execute(&doc, serviceFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	serviceDocString := doc.String()
	log.Info(serviceDocString)

	service := v1.Service{}
	err = json.Unmarshal(doc.Bytes(), &service)
	if err != nil {
		log.Error("error unmarshalling json into Service " + err.Error())
		return err
	}

	svc, err := clientset.Services(v1.NamespaceDefault).Create(&service)
	if err != nil {
		log.Error("error creating Service " + err.Error())
		return err
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
		err = RestorePodTemplate1.Execute(&doc2, podFields)
		log.Infof("doing a restore!!! with pvc %s and path %s\n", db.Spec.BACKUP_PVC_NAME, db.Spec.BACKUP_PATH)
	} else {
		err = PodTemplate1.Execute(&doc2, podFields)
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	podDocString := doc2.String()
	log.Info(podDocString)

	pod := v1.Pod{}
	err = json.Unmarshal(doc2.Bytes(), &pod)
	if err != nil {
		log.Error("error unmarshalling json into Pod " + err.Error())
		return err
	}

	resultPod, err := clientset.Pods(v1.NamespaceDefault).Create(&pod)
	if err != nil {
		log.Error("error creating Pod " + err.Error())
		return err
	}
	log.Info("created pod " + resultPod.Name)
	return err

}

func (r DatabaseStrategy1) DeleteDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase) error {
	log.Debug("deleting PgDatabase object with Strategy 1")
	log.Debug("deleting with Name=" + db.Spec.Name)

	var err error

	err = clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Service " + err.Error())
		return err
	}
	log.Info("deleted service " + db.Spec.Name)

	err = clientset.Pods(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Pod " + err.Error())
		return err
	}
	log.Info("deleted pod " + db.Spec.Name)

	err = pvc.Delete(clientset, db.Spec.Name+"-pvc")
	if err != nil {
		log.Error("error deleting pvc " + err.Error())
		return err
	}
	return err

}
