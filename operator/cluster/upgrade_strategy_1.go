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

// Package cluster holds the cluster TPR logic and definitions
// A cluster is comprised of a master service, replica service,
// master deployment, and replica deployment
package cluster

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api/v1"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"text/template"
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

const DB_UPGRADE_JOB_PATH = "/operator-conf/cluster-upgrade-job-1.json"

func init() {

	JobTemplate1 = util.LoadTemplate(DB_UPGRADE_JOB_PATH)
}

func (r ClusterStrategy1) MinorUpgrade(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, cl *tpr.PgCluster, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error
	var masterDoc bytes.Buffer
	var deploymentResult *v1beta1.Deployment

	log.Info("minor cluster upgrade using Strategy 1 in namespace " + namespace)

	err = shutdownCluster(clientset, tprclient, cl, namespace)
	if err != nil {
		log.Error("error in shutdownCluster " + err.Error())
	}

	//create the master deployment

	deploymentFields := DeploymentTemplateFields{
		Name:                 cl.Spec.Name,
		ClusterName:          cl.Spec.Name,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        upgrade.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             util.CreatePVCSnippet(cl.Spec.MasterStorage.StorageType, cl.Spec.MasterStorage.PvcName),
		OPERATOR_LABELS:      util.GetLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false),
		BACKUP_PVC_NAME:      util.CreateBackupPVCSnippet(cl.Spec.BACKUP_PVC_NAME),
		BACKUP_PATH:          cl.Spec.BACKUP_PATH,
		PGDATA_PATH_OVERRIDE: cl.Spec.Name,
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
	}

	err = DeploymentTemplate1.Execute(&masterDoc, deploymentFields)
	if err != nil {
		log.Error("error in DeploymentTemplate Execute" + err.Error())
		return err
	}
	deploymentDocString := masterDoc.String()
	log.Info(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(masterDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling master json into Deployment " + err.Error())
		return err
	}

	deploymentResult, err = clientset.Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		return err
	}
	log.Info("created master Deployment " + deploymentResult.Name + " in namespace " + namespace)

	//update the upgrade TPR status to completed
	err = util.Patch(tprclient, "/spec/upgradestatus", tpr.UPGRADE_COMPLETED_STATUS, tpr.UPGRADE_RESOURCE, upgrade.Spec.Name, namespace)
	if err != nil {
		log.Error("error in upgradestatus patch " + err.Error())
	}

	return err

}

func (r ClusterStrategy1) MajorUpgrade(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, cl *tpr.PgCluster, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error

	log.Info("major cluster upgrade using Strategy 1 in namespace " + namespace)
	err = shutdownCluster(clientset, tprclient, cl, namespace)
	if err != nil {
		log.Error("error in shutdownCluster " + err.Error())
	}

	//create the PVC if necessary
	if upgrade.Spec.NEW_PVC_NAME != upgrade.Spec.OLD_PVC_NAME {
		if pvc.Exists(clientset, upgrade.Spec.NEW_PVC_NAME, namespace) {
			log.Info("pvc " + upgrade.Spec.NEW_PVC_NAME + " already exists, will not create")
		} else {
			log.Info("creating pvc " + upgrade.Spec.NEW_PVC_NAME)
			err = pvc.Create(clientset, upgrade.Spec.NEW_PVC_NAME, upgrade.Spec.StorageSpec.PvcAccessMode, upgrade.Spec.StorageSpec.PvcSize, upgrade.Spec.StorageSpec.StorageType, upgrade.Spec.StorageSpec.StorageClass, namespace)
			if err != nil {
				log.Error("error in pvc create " + err.Error())
				return err
			}
			log.Info("created PVC =" + upgrade.Spec.NEW_PVC_NAME + " in namespace " + namespace)
		}
	}

	//upgrade the master data
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
		log.Error("error in job template execute " + err.Error())
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

	//the remainder of the major upgrade is done via the upgrade watcher

	return err

}

func (r ClusterStrategy1) MajorUpgradeFinalize(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error
	var masterDoc bytes.Buffer
	var deploymentResult *v1beta1.Deployment

	log.Info("major cluster upgrade finalize using Strategy 1 in namespace " + namespace)

	//start the master deployment
	deploymentFields := DeploymentTemplateFields{
		Name:                 cl.Spec.Name,
		ClusterName:          cl.Spec.Name,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        upgrade.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             util.CreatePVCSnippet(cl.Spec.MasterStorage.StorageType, upgrade.Spec.NEW_PVC_NAME),
		OPERATOR_LABELS:      util.GetLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false),
		BACKUP_PVC_NAME:      util.CreateBackupPVCSnippet(upgrade.Spec.BACKUP_PVC_NAME),
		PGDATA_PATH_OVERRIDE: upgrade.Spec.NEW_DATABASE_NAME,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
	}

	err = DeploymentTemplate1.Execute(&masterDoc, deploymentFields)
	if err != nil {
		log.Error("error in dep template execute " + err.Error())
		return err
	}
	deploymentDocString := masterDoc.String()
	log.Info(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(masterDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling master json into Deployment " + err.Error())
		return err
	}

	deploymentResult, err = clientset.Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		return err
	}
	log.Info("created master Deployment " + deploymentResult.Name + " in namespace " + namespace)

	return err

}
