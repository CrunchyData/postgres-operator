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
	"time"

	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/rest"
)

var JobTemplate1 *template.Template

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

const DB_UPGRADE_JOB_PATH = "/operator-conf/database-upgrade-job-1.json"

func init() {

	JobTemplate1 = util.LoadTemplate(DB_UPGRADE_JOB_PATH)

}

func (r DatabaseStrategy1) MinorUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error
	var doc2 bytes.Buffer

	log.Info("minor database upgrade for " + db.Spec.Name + " using Strategy 1 in namespace " + namespace)

	//stop pod
	err = clientset.Pods(namespace).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warn("delete of pod error - did not exist")
		} else {
			log.Error("error deleting Pod " + err.Error())
			return err
		}
	}
	log.Info("deleted pod " + db.Spec.Name + " in namespace " + namespace)

	err = util.WaitUntilPodIsDeleted(clientset, db.Spec.Name, time.Minute, namespace)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	//start pod

	podFields := PodTemplateFields{
		Name:                 db.Spec.Name,
		Port:                 db.Spec.Port,
		PVC_NAME:             db.Spec.PVC_NAME,
		CCP_IMAGE_TAG:        upgrade.Spec.CCP_IMAGE_TAG,
		PG_MASTER_USER:       db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD:   db.Spec.PG_MASTER_PASSWORD,
		PGDATA_PATH_OVERRIDE: db.Spec.Name,
		PG_USER:              db.Spec.PG_USER,
		PG_PASSWORD:          db.Spec.PG_PASSWORD,
		PG_DATABASE:          db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:     db.Spec.PG_ROOT_PASSWORD,
		BACKUP_PVC_NAME:      db.Spec.BACKUP_PVC_NAME,
		BACKUP_PATH:          db.Spec.BACKUP_PATH,
		SECURITY_CONTEXT:     util.CreateSecContext(db.Spec.FS_GROUP, db.Spec.SUPPLEMENTAL_GROUPS),
	}

	err = PodTemplate1.Execute(&doc2, podFields)
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

	resultPod, err := clientset.Pods(namespace).Create(&pod)
	if err != nil {
		log.Error("error creating Pod " + err.Error())
		return err
	}
	log.Info("created pod " + resultPod.Name + " in namespace " + namespace)

	return err

}

func (r DatabaseStrategy1) MajorUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error

	log.Info("major database upgrade using Strategy 1 in namespace " + namespace)
	//stop pod
	err = clientset.Pods(namespace).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warn("delete of pod error - did not exist")
		} else {
			log.Error("error deleting Pod " + err.Error())
			//return err
		}
	}
	log.Info("deleted pod " + db.Spec.Name + " in namespace " + namespace)
	//create the new PVC if necessary
	if upgrade.Spec.NEW_PVC_NAME != upgrade.Spec.OLD_PVC_NAME {
		if pvc.Exists(clientset, upgrade.Spec.NEW_PVC_NAME, namespace) {
			log.Info("pvc " + upgrade.Spec.NEW_PVC_NAME + " already exists, will not create")
		} else {
			log.Info("creating pvc " + upgrade.Spec.NEW_PVC_NAME)
			err = pvc.Create(clientset, upgrade.Spec.NEW_PVC_NAME, upgrade.Spec.PVC_ACCESS_MODE, upgrade.Spec.PVC_SIZE, namespace)
			if err != nil {
				log.Error(err.Error())
				return err
			}
			log.Info("created PVC =" + upgrade.Spec.NEW_PVC_NAME + " in namespace " + namespace)
		}

	}

	//create the upgrade job
	jobFields := JobTemplateFields{
		Name:              upgrade.Spec.Name,
		NEW_PVC_NAME:      upgrade.Spec.NEW_PVC_NAME,
		OLD_PVC_NAME:      upgrade.Spec.OLD_PVC_NAME,
		CCP_IMAGE_TAG:     upgrade.Spec.CCP_IMAGE_TAG,
		OLD_DATABASE_NAME: upgrade.Spec.OLD_DATABASE_NAME,
		NEW_DATABASE_NAME: upgrade.Spec.NEW_DATABASE_NAME,
		OLD_VERSION:       upgrade.Spec.OLD_VERSION,
		NEW_VERSION:       upgrade.Spec.NEW_VERSION,
	}

	var doc bytes.Buffer
	err = JobTemplate1.Execute(&doc, jobFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	jobDocString := doc.String()
	log.Debug(jobDocString)

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return err
	}

	resultJob, err := clientset.Batch().Jobs(namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return err
	}
	log.Info("created Job " + resultJob.Name)

	return err

}

func (r DatabaseStrategy1) MajorUpgradeFinalize(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, db *tpr.PgDatabase, upgrade *tpr.PgUpgrade, namespace string) error {

	var doc2 bytes.Buffer

	//start a database pod

	podFields := PodTemplateFields{
		Name:                 db.Spec.Name,
		Port:                 db.Spec.Port,
		PVC_NAME:             upgrade.Spec.NEW_PVC_NAME,
		CCP_IMAGE_TAG:        upgrade.Spec.CCP_IMAGE_TAG,
		PG_MASTER_USER:       db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD:   db.Spec.PG_MASTER_PASSWORD,
		PG_USER:              db.Spec.PG_USER,
		PG_PASSWORD:          db.Spec.PG_PASSWORD,
		PG_DATABASE:          db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:     db.Spec.PG_ROOT_PASSWORD,
		PGDATA_PATH_OVERRIDE: upgrade.Spec.NEW_DATABASE_NAME,
		BACKUP_PVC_NAME:      db.Spec.BACKUP_PVC_NAME,
		BACKUP_PATH:          db.Spec.BACKUP_PATH,
		SECURITY_CONTEXT:     util.CreateSecContext(db.Spec.FS_GROUP, db.Spec.SUPPLEMENTAL_GROUPS),
	}

	err := PodTemplate1.Execute(&doc2, podFields)
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

	resultPod, err := clientset.Pods(namespace).Create(&pod)
	if err != nil {
		log.Error("error creating Pod " + err.Error())
		return err
	}

	lo := v1.ListOptions{LabelSelector: "pg-database=" + upgrade.Spec.Name}
	err = util.WaitUntilPod(clientset, lo, v1.PodRunning, time.Minute, namespace)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("created pod " + resultPod.Name + " in namespace " + namespace)

	//update the upgrade TPR status to completed
	err = util.Patch(tprclient, "/spec/upgradestatus", tpr.UPGRADE_COMPLETED_STATUS, "pgupgrades", upgrade.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

	return err

}
