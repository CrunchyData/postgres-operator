// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"text/template"
)

// JobTemplate1 ...
var JobTemplate1 *template.Template

// JobTemplateFields ...
type JobTemplateFields struct {
	Name            string
	OldPVCName      string
	NewPVCName      string
	CCPImagePrefix  string
	CCPImageTag     string
	OldDatabaseName string
	NewDatabaseName string
	OldVersion      string
	NewVersion      string
	SecurityContext string
}

// UpgradeJobPath ...
const UpgradeJobPath = "/operator-conf/cluster-upgrade-job-1.json"

func init() {

	JobTemplate1 = util.LoadTemplate(UpgradeJobPath)
}

// MinorUpgrade ..
func (r Strategy1) MinorUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, upgrade *crv1.Pgupgrade, namespace string) error {
	var err error
	var primaryDoc bytes.Buffer
	var deploymentResult *v1beta1.Deployment

	log.Info("minor cluster upgrade using Strategy 1 in namespace " + namespace)

	err = shutdownCluster(clientset, restclient, cl, namespace)
	if err != nil {
		log.Error("error in shutdownCluster " + err.Error())
	}

	//create the primary deployment

	primaryLabels := getPrimaryLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false, cl.Spec.UserLabels)

	deploymentFields := DeploymentTemplateFields{
		Name:              cl.Spec.Name,
		ClusterName:       cl.Spec.Name,
		Port:              cl.Spec.Port,
		CCPImagePrefix:    operator.CCPImagePrefix,
		CCPImageTag:       upgrade.Spec.CCPImageTag,
		PVCName:           util.CreatePVCSnippet(cl.Spec.PrimaryStorage.StorageType, cl.Spec.PrimaryStorage.Name),
		OperatorLabels:    util.GetLabelsFromMap(primaryLabels),
		BackupPVCName:     util.CreateBackupPVCSnippet(cl.Spec.BackupPVCName),
		BackupPath:        cl.Spec.BackupPath,
		DataPathOverride:  cl.Spec.Name,
		Database:          cl.Spec.Database,
		SecurityContext:   util.CreateSecContext(cl.Spec.PrimaryStorage.Fsgroup, cl.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:    cl.Spec.RootSecretName,
		PrimarySecretName: cl.Spec.PrimarySecretName,
		UserSecretName:    cl.Spec.UserSecretName,
		NodeSelector:      GetAffinity(cl.Spec.NodeName, "In"),
		ConfVolume:        GetConfVolume(clientset, cl.Spec.CustomConfig, namespace),
		CollectAddon:      GetCollectAddon(&cl.Spec),
	}

	err = deploymentTemplate1.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error("error in DeploymentTemplate Execute" + err.Error())
		return err
	}
	deploymentDocString := primaryDoc.String()
	log.Debug(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(primaryDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling primary json into Deployment " + err.Error())
		return err
	}

	deploymentResult, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating primary Deployment " + err.Error())
		return err
	}
	log.Info("created primary Deployment " + deploymentResult.Name + " in namespace " + namespace)

	//update the upgrade CRD status to completed
	err = util.Patch(restclient, "/spec/upgradestatus", crv1.UpgradeCompletedStatus, crv1.PgupgradeResourcePlural, upgrade.Spec.Name, namespace)
	if err != nil {
		log.Error("error in upgradestatus patch " + err.Error())
	}

	return err

}

// MajorUpgrade ...
func (r Strategy1) MajorUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, upgrade *crv1.Pgupgrade, namespace string) error {
	var err error

	log.Info("major cluster upgrade using Strategy 1 in namespace " + namespace)
	err = shutdownCluster(clientset, restclient, cl, namespace)
	if err != nil {
		log.Error("error in shutdownCluster " + err.Error())
	}

	//create the PVC if necessary
	pvcName, err := pvc.CreatePVC(clientset, cl.Spec.Name+"-upgrade", &cl.Spec.PrimaryStorage, namespace)
	log.Debug("created pvc for upgrade as [" + pvcName + "]")

	//upgrade the primary data
	jobFields := JobTemplateFields{
		Name:            upgrade.Spec.Name,
		NewPVCName:      pvcName,
		OldPVCName:      upgrade.Spec.OldPVCName,
		CCPImagePrefix:  operator.CCPImagePrefix,
		CCPImageTag:     upgrade.Spec.CCPImageTag,
		OldDatabaseName: upgrade.Spec.OldDatabaseName,
		NewDatabaseName: upgrade.Spec.NewDatabaseName,
		OldVersion:      upgrade.Spec.OldVersion,
		NewVersion:      upgrade.Spec.NewVersion,
		SecurityContext: util.CreateSecContext(cl.Spec.PrimaryStorage.Fsgroup, cl.Spec.PrimaryStorage.SupplementalGroups),
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

	//patch the upgrade crv1 with the new pvc name
	err = util.Patch(restclient, "/spec/newpvcname", pvcName, crv1.PgupgradeResourcePlural, upgrade.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	//the remainder of the major upgrade is done via the upgrade watcher

	return err

}

// MajorUpgradeFinalize ...
func (r Strategy1) MajorUpgradeFinalize(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, upgrade *crv1.Pgupgrade, namespace string) error {
	var err error
	var primaryDoc bytes.Buffer
	var deploymentResult *v1beta1.Deployment

	log.Info("major cluster upgrade finalize using Strategy 1 in namespace " + namespace)

	primaryLabels := getPrimaryLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false, cl.Spec.UserLabels)

	//start the primary deployment
	deploymentFields := DeploymentTemplateFields{
		Name:              cl.Spec.Name,
		ClusterName:       cl.Spec.Name,
		Port:              cl.Spec.Port,
		CCPImagePrefix:    operator.CCPImagePrefix,
		CCPImageTag:       upgrade.Spec.CCPImageTag,
		PVCName:           util.CreatePVCSnippet(cl.Spec.PrimaryStorage.StorageType, upgrade.Spec.NewPVCName),
		OperatorLabels:    util.GetLabelsFromMap(primaryLabels),
		BackupPVCName:     util.CreateBackupPVCSnippet(upgrade.Spec.BackupPVCName),
		DataPathOverride:  upgrade.Spec.NewDatabaseName,
		Database:          cl.Spec.Database,
		SecurityContext:   util.CreateSecContext(cl.Spec.PrimaryStorage.Fsgroup, cl.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:    cl.Spec.RootSecretName,
		PrimarySecretName: cl.Spec.PrimarySecretName,
		UserSecretName:    cl.Spec.UserSecretName,
		NodeSelector:      cl.Spec.NodeName,
		ConfVolume:        GetConfVolume(clientset, cl.Spec.CustomConfig, namespace),
		CollectAddon:      GetCollectAddon(&cl.Spec),
	}

	err = deploymentTemplate1.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error("error in dep template execute " + err.Error())
		return err
	}
	deploymentDocString := primaryDoc.String()
	log.Info(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(primaryDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling primary json into Deployment " + err.Error())
		return err
	}

	deploymentResult, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating primary Deployment " + err.Error())
		return err
	}
	log.Info("created primary Deployment " + deploymentResult.Name + " in namespace " + namespace)

	return err

}
