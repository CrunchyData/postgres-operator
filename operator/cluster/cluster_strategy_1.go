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

//var ServiceTemplate1 *template.Template

func init() {

	//ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")
	ReplicaDeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1.json")
	ReplicaDeploymentTemplate1Shared = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1-shared.json")
	DeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-deployment-1.json")
}

func (r ClusterStrategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string, masterPvcName string) error {
	var masterDoc bytes.Buffer
	var err error
	var deploymentResult *v1beta1.Deployment

	log.Info("creating PgCluster object using Strategy 1" + " in namespace " + namespace)
	log.Info("created with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//create the master service
	serviceFields := ServiceTemplateFields{
		Name:        cl.Spec.Name,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error("error in creating master service " + err.Error())
		return err
	}

	//create the master deployment
	deploymentFields := DeploymentTemplateFields{
		Name:                 cl.Spec.Name,
		ClusterName:          cl.Spec.Name,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        cl.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             util.CreatePVCSnippet(cl.Spec.MasterStorage.StorageType, masterPvcName),
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

	//var replicaName = cl.Spec.Name + REPLICA_SUFFIX

	//get the deployments
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + cl.Spec.Name}
	deployments, err := clientset.Deployments(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return err
	}

	for _, d := range deployments.Items {
		log.Debug("draining deployment " + d.ObjectMeta.Name)
		err = util.DrainDeployment(clientset, d.ObjectMeta.Name, namespace)
		if err != nil {
			log.Error("error deleting replica Deployment " + err.Error())
		}
	}

	//sleep just a bit to give the drain time to work
	time.Sleep(2000 * time.Millisecond)

	//TODO when client-go 3.0 is ready, use propagation_policy
	//in the delete options to also delete the replica sets

	//delete the deployments
	for _, d := range deployments.Items {
		log.Debug("deleting deployment " + d.ObjectMeta.Name)
		err = clientset.Deployments(namespace).Delete(d.ObjectMeta.Name, &v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting replica Deployment " + err.Error())
		}
	}

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

	for _, d := range deployments.Items {
		log.Debug("making sure deployment " + d.ObjectMeta.Name + " is deleted")
		err := util.WaitUntilDeploymentIsDeleted(clientset, d.ObjectMeta.Name, time.Second*3, namespace)
		if err != nil {
			log.Error("timeout waiting for deployment " + d.ObjectMeta.Name + " to delete")
		}
	}

	return err

}

func (r ClusterStrategy1) PrepareClone(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, cloneName string, cl *tpr.PgCluster, namespace string) error {
	var err error

	log.Info("creating clone deployment using Strategy 1 in namespace " + namespace)

	//create a PVC
	pvcName, err := createPVC(clientset, cloneName, &cl.Spec.ReplicaStorage, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	//create the clone replica service and deployment
	replicaServiceFields := ServiceTemplateFields{
		Name:        cloneName,
		ClusterName: cloneName,
		Port:        cl.Spec.Port,
	}

	err = CreateService(clientset, &replicaServiceFields, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	r.CreateReplica(cloneName, clientset, cl, cloneName, pvcName, namespace, true)

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

func (r ClusterStrategy1) CreateReplica(serviceName string, clientset *kubernetes.Clientset, cl *tpr.PgCluster, depName, pvcName, namespace string, cloneFlag bool) error {
	var replicaDoc bytes.Buffer
	var err error
	var replicaDeploymentResult *v1beta1.Deployment

	clusterName := cl.Spec.ClusterName

	if cloneFlag {
		clusterName = depName
	}

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:                 depName,
		ClusterName:          clusterName,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        cl.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             pvcName,
		PG_MASTER_HOST:       cl.Spec.PG_MASTER_HOST,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		REPLICAS:             "1",
		OPERATOR_LABELS:      util.GetLabels(serviceName, clusterName, cloneFlag, true),
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
	}

	switch cl.Spec.ReplicaStorage.StorageType {
	case "", "emptydir":
		log.Debug("MasterStorage.StorageType is emptydir")
		log.Debug("using the dynamic replica template ")
		err = ReplicaDeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	case "existing", "create":
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
