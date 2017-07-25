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
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/v1"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"text/template"
	"time"
)

type ClusterStrategy1 struct{}

var DeploymentTemplate1 *template.Template
var ReplicaDeploymentTemplate1 *template.Template
var ReplicaDeploymentTemplate1Shared *template.Template
var ServiceTemplate1 *template.Template

func init() {

	ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")
	ReplicaDeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1.json")
	ReplicaDeploymentTemplate1Shared = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1-shared.json")
	DeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-deployment-1.json")
}

func (r ClusterStrategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string, masterPvcName string) error {
	var serviceDoc, replicaServiceDoc, masterDoc, replicaDoc bytes.Buffer
	var err error
	var replicaServiceResult, serviceResult *v1.Service
	var replicaDeploymentResult, deploymentResult *v1beta1.Deployment

	log.Info("creating PgCluster object using Strategy 1" + " in namespace " + namespace)
	log.Info("created with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//create the master service
	serviceFields := ServiceTemplateFields{
		Name:        cl.Spec.Name,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
	}

	err = ServiceTemplate1.Execute(&serviceDoc, serviceFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	serviceDocString := serviceDoc.String()
	log.Info(serviceDocString)

	service := v1.Service{}
	err = json.Unmarshal(serviceDoc.Bytes(), &service)
	if err != nil {
		log.Error("error unmarshalling json into Service " + err.Error())
		return err
	}

	serviceResult, err = clientset.Services(namespace).Create(&service)
	if err != nil {
		log.Error("error creating Service " + err.Error())
		return err
	}
	log.Info("created master service " + serviceResult.Name + " in namespace " + namespace)

	//create the replica service
	replicaServiceFields := ServiceTemplateFields{
		Name:        cl.Spec.Name + REPLICA_SUFFIX,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
	}

	err = ServiceTemplate1.Execute(&replicaServiceDoc, replicaServiceFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	replicaServiceDocString := replicaServiceDoc.String()
	log.Info(replicaServiceDocString)

	replicaService := v1.Service{}
	err = json.Unmarshal(replicaServiceDoc.Bytes(), &replicaService)
	if err != nil {
		log.Error("error unmarshalling json into replica Service " + err.Error())
		return err
	}

	replicaServiceResult, err = clientset.Services(namespace).Create(&replicaService)
	if err != nil {
		log.Error("error creating replica Service " + err.Error())
		return err
	}
	log.Info("created replica service " + replicaServiceResult.Name + " in namespace " + namespace)

	//create the master deployment
	deploymentFields := DeploymentTemplateFields{
		Name:                 cl.Spec.Name,
		ClusterName:          cl.Spec.Name,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        cl.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             masterPvcName,
		OPERATOR_LABELS:      util.GetLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false),
		BACKUP_PVC_NAME:      util.CreateBackupPVCSnippet(cl.Spec.BACKUP_PVC_NAME),
		BACKUP_PATH:          cl.Spec.BACKUP_PATH,
		PGDATA_PATH_OVERRIDE: cl.Spec.Name,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
	}

	err = DeploymentTemplate1.Execute(&masterDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
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

	if deploymentExists(clientset, namespace, cl.Spec.Name) == false {
		deploymentResult, err = clientset.Deployments(namespace).Create(&deployment)
		if err != nil {
			log.Error("error creating master Deployment " + err.Error())
			return err
		}
		log.Info("created master Deployment " + deploymentResult.Name + " in namespace " + namespace)
	} else {
		log.Info("master Deployment " + cl.Spec.Name + " in namespace " + namespace + " already existed so not creating it ")
	}

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:                 cl.Spec.Name + REPLICA_SUFFIX,
		ClusterName:          cl.Spec.Name,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        cl.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             cl.Spec.PVC_NAME,
		PG_MASTER_HOST:       cl.Spec.PG_MASTER_HOST,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		REPLICAS:             cl.Spec.REPLICAS,
		OPERATOR_LABELS:      util.GetLabels(cl.Spec.Name, cl.Spec.ClusterName, false, true),
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
	}

	if cl.Spec.PVC_NAME == "" {
		//if no PVC_NAME then assume a non-shared volume type
		log.Debug("using the dynamic replica template ")
		err = ReplicaDeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	} else {
		log.Debug("using the shared replica template ")
		err = ReplicaDeploymentTemplate1Shared.Execute(&replicaDoc, replicaDeploymentFields)
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	replicaDeploymentDocString := replicaDoc.String()
	log.Info(replicaDeploymentDocString)

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling replica json into Deployment " + err.Error())
		return err
	}

	replicaDeploymentResult, err = clientset.Deployments(namespace).Create(&replicaDeployment)
	if err != nil {
		log.Error("error creating replica Deployment " + err.Error())
		return err
	}
	log.Info("created replica Deployment " + replicaDeploymentResult.Name)
	return err

}

func (r ClusterStrategy1) DeleteCluster(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, cl *tpr.PgCluster, namespace string) error {

	var err error
	log.Info("deleting PgCluster object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//delete the master and replica deployments and replica sets
	err = shutdownCluster(clientset, tprclient, cl, namespace)
	if err != nil {
		log.Error("error deleting master Deployment " + err.Error())
	}

	//delete any remaining pods that may be left lingering
	listOptions := v1.ListOptions{}
	listOptions.LabelSelector = "pg-cluster=" + cl.Spec.Name
	pods, err := clientset.Core().Pods(namespace).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name + " in namespace " + namespace)
		err = clientset.Pods(namespace).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name + " in namespace " + namespace)

	}
	listOptions.LabelSelector = "name=" + cl.Spec.Name + REPLICA_SUFFIX
	pods, err = clientset.Core().Pods(namespace).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name + " in namespace " + namespace)
		err = clientset.Pods(namespace).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name + " in namespace " + namespace)

	}

	//delete the master service

	err = clientset.Services(namespace).Delete(cl.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Service " + err.Error())
	}
	log.Info("deleted master service " + cl.Spec.Name)

	//delete the replica service
	err = clientset.Services(namespace).Delete(cl.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Service " + err.Error())
	}
	log.Info("deleted replica service " + cl.Spec.Name + REPLICA_SUFFIX + " in namespace " + namespace)

	return err

}

func shutdownCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string) error {
	var err error

	var replicaName = cl.Spec.Name + REPLICA_SUFFIX

	//drain the deployments
	err = util.DrainDeployment(clientset, replicaName, namespace)
	if err != nil {
		log.Error("error draining replica Deployment " + err.Error())
	}
	err = util.DrainDeployment(clientset, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error draining master Deployment " + err.Error())
	}

	//sleep just a bit to give the drain time to work
	time.Sleep(2000 * time.Millisecond)

	//TODO when client-go 3.0 is ready, use propagation_policy
	//in the delete options to also delete the replica sets

	//delete the replica deployment

	err = clientset.Deployments(namespace).Delete(replicaName, &v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Deployment " + err.Error())
	}

	//wait for the replica deployment to delete
	/**
	err = util.WaitUntilDeploymentIsDeleted(clientset, replicaName, 2 * time.Minute, namespace)
	if err != nil {
		log.Error("error waiting for replica Deployment deletion " + err.Error())
	}
	log.Info("deleted replica Deployment " + replicaName + " in namespace " + namespace)
	*/

	//delete the master deployment
	err = clientset.Deployments(namespace).Delete(cl.Spec.Name, &v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Deployment " + err.Error())
	}

	//wait for the master deployment to delete
	/**
	err = util.WaitUntilDeploymentIsDeleted(clientset, cl.Spec.Name, 2 * time.Minute, namespace)
	if err != nil {
		log.Error("error waiting for master Deployment deletion " + err.Error())
	}
	log.Info("deleted master Deployment " + cl.Spec.Name + " in namespace " + namespace)

	*/

	//TODO for k8s 1.6 and client-go 3.0 we can use propagation_policy
	// to have the replica sets removed as part of the deployment remove
	//delete replica sets if they exist
	options := v1.ListOptions{}
	options.LabelSelector = "pg-cluster=" + cl.Spec.Name

	var reps *v1beta1.ReplicaSetList
	reps, err = clientset.ReplicaSets(namespace).List(options)
	if err != nil {
		log.Error("error getting cluster replicaset name" + err.Error())
	} else {
		for _, r := range reps.Items {
			err = clientset.ReplicaSets(namespace).Delete(r.Name,
				&v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting cluster replicaset " + err.Error())
			}

			log.Info("deleted cluster replicaset " + r.Name + " in namespace " + namespace)
		}
	}

	return err

}

func (r ClusterStrategy1) PrepareClone(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, cloneName string, cl *tpr.PgCluster, namespace string) error {
	var replicaDoc bytes.Buffer
	var err error
	var replicaDeploymentResult *v1beta1.Deployment

	log.Info("creating clone deployment using Strategy 1 in namespace " + namespace)

	//create the clone replica deployment and set replicas to 1
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:                 cloneName,
		ClusterName:          cloneName,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        cl.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             cl.Spec.PVC_NAME,
		PG_MASTER_HOST:       cl.Spec.PG_MASTER_HOST,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		OPERATOR_LABELS:      util.GetLabels(cloneName, cloneName, true, false),
		REPLICAS:             "1",
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
	}

	if cl.Spec.PVC_NAME == "" {
		//if PVC_NAME is blank, assume a non-shared volume type
		log.Debug("using the dynamic replica template ")
		err = ReplicaDeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	} else {
		log.Debug("using the shared replica template ")
		err = ReplicaDeploymentTemplate1Shared.Execute(&replicaDoc, replicaDeploymentFields)
	}
	if err != nil {
		log.Error("error in clone rep dep tem exec " + err.Error())
		return err
	}

	replicaDeploymentDocString := replicaDoc.String()
	log.Info(replicaDeploymentDocString)

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling clone replica json into Deployment " + err.Error())
		return err
	}

	replicaDeploymentResult, err = clientset.Deployments(namespace).Create(&replicaDeployment)
	if err != nil {
		log.Error("error creating clone replica Deployment " + err.Error())
		return err
	}
	log.Info("created clone replica Deployment " + replicaDeploymentResult.Name)
	//get the original deployment
	d, err := clientset.Deployments(namespace).Get(cl.Spec.ClusterName)
	if err != nil {
		log.Error("getPolicyLabels deployment " + cl.Spec.ClusterName + " error " + err.Error())
		return err
	}

	//get the policy labels from it
	labels := d.ObjectMeta.Labels
	polyLabels := make(map[string]string)
	for key, value := range labels {
		if value == "pgpolicy" {
			polyLabels[key] = value
		}
	}

	//apply policy labels to new clone deployment
	err = r.UpdatePolicyLabels(clientset, cloneName, namespace, polyLabels)
	if err != nil {
		log.Error("getPolicyLabels error updating poly labels")
	}

	return err

}
func deploymentExists(clientset *kubernetes.Clientset, namespace, clusterName string) bool {

	_, err := clientset.Deployments(namespace).Get(clusterName)
	if kerrors.IsNotFound(err) {
		return false
	} else if err != nil {
		log.Error("deployment " + clusterName + " error " + err.Error())
		return false
	}

	return true
}

func (r ClusterStrategy1) UpdatePolicyLabels(clientset *kubernetes.Clientset, clusterName string, namespace string, newLabels map[string]string) error {

	var err error
	var deployment *v1beta1.Deployment

	//get the deployment
	deployment, err = clientset.Deployments(namespace).Get(clusterName)
	if err != nil {
		return err
	}
	log.Debug("got the deployment" + deployment.Name)

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	//update the deployment labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	createdPatch := err == nil
	if err != nil {
		return err
	}
	if createdPatch {
		log.Debug("created merge patch")
	}

	_, err = clientset.Deployments(namespace).Patch(clusterName, api.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}
