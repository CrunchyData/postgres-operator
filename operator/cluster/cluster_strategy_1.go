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
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/operator/pvc"
	"github.com/crunchydata/kraken/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jsonpatch "github.com/evanphx/json-patch"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"strconv"

	//"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	//"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"text/template"
	"time"
)

const AFFINITY_IN_OPERATOR = "In"
const AFFINITY_NOTIN_OPERATOR = "NotIn"

type AffinityTemplateFields struct {
	NODE     string
	OPERATOR string
}

type ClusterStrategy1 struct{}

var AffinityTemplate1 *template.Template
var DeploymentTemplate1 *template.Template
var ReplicaDeploymentTemplate1 *template.Template
var ReplicaDeploymentTemplate1Shared *template.Template

//var ServiceTemplate1 *template.Template

func init() {

	//ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")
	ReplicaDeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1.json")
	ReplicaDeploymentTemplate1Shared = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1-shared.json")
	DeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-deployment-1.json")
	AffinityTemplate1 = util.LoadTemplate("/operator-conf/affinity.json")
}

func (r ClusterStrategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string, masterPvcName string) error {
	var masterDoc bytes.Buffer
	var err error
	var deploymentResult *v1beta1.Deployment

	log.Info("creating Pgcluster object using Strategy 1" + " in namespace " + namespace)
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

	masterLabels := getMasterLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false, cl.Spec.UserLabels)

	//create the master deployment
	deploymentFields := DeploymentTemplateFields{
		Name:                 cl.Spec.Name,
		ClusterName:          cl.Spec.Name,
		Port:                 cl.Spec.Port,
		CCP_IMAGE_TAG:        cl.Spec.CCP_IMAGE_TAG,
		PVC_NAME:             util.CreatePVCSnippet(cl.Spec.MasterStorage.StorageType, masterPvcName),
		OPERATOR_LABELS:      util.GetLabelsFromMap(masterLabels),
		BACKUP_PVC_NAME:      util.CreateBackupPVCSnippet(cl.Spec.BACKUP_PVC_NAME),
		BACKUP_PATH:          cl.Spec.BACKUP_PATH,
		PGDATA_PATH_OVERRIDE: cl.Spec.Name,
		PG_DATABASE:          cl.Spec.PG_DATABASE,
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.MasterStorage.FSGROUP, cl.Spec.MasterStorage.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
		NODE_SELECTOR:        GetAffinity(cl.Spec.NodeName, "In"),
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
		deploymentResult, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)
		if err != nil {
			log.Error("error creating master Deployment " + err.Error())
			return err
		}
		log.Info("created master Deployment " + deploymentResult.Name + " in namespace " + namespace)
	} else {
		log.Info("master Deployment " + cl.Spec.Name + " in namespace " + namespace + " already existed so not creating it ")
	}

	err = util.PatchClusterTPR(client, masterLabels, cl, namespace)
	if err != nil {
		log.Error("could not patch master crv1 with labels")
		return err
	}

	newReplicas, err := strconv.Atoi(cl.Spec.REPLICAS)
	if err != nil {
		log.Error("could not convert REPLICAS config setting")
	} else {
		if newReplicas > 0 {
			//create the replica service
			serviceName := cl.Spec.Name + "-replica"
			repserviceFields := ServiceTemplateFields{
				Name:        serviceName,
				ClusterName: cl.Spec.Name,
				Port:        cl.Spec.Port,
			}

			err = CreateService(clientset, &repserviceFields, namespace)
			if err != nil {
				log.Error("error in creating replica service " + err.Error())
				return err
			}

			ScaleReplicasBase(serviceName, clientset, cl, newReplicas, namespace)
		}
	}

	return err

}

func (r ClusterStrategy1) DeleteCluster(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {

	var err error
	log.Info("deleting Pgcluster object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//delete the master and replica deployments and replica sets
	err = shutdownCluster(clientset, restclient, cl, namespace)
	if err != nil {
		log.Error("error deleting master Deployment " + err.Error())
	}

	//delete any remaining pods that may be left lingering
	listOptions := meta_v1.ListOptions{}
	listOptions.LabelSelector = "pg-cluster=" + cl.Spec.Name
	pods, err := clientset.CoreV1().Pods(namespace).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name + " in namespace " + namespace)
		err = clientset.Core().Pods(namespace).Delete(pod.Name,
			&meta_v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name + " in namespace " + namespace)

	}
	listOptions.LabelSelector = "name=" + cl.Spec.Name + REPLICA_SUFFIX
	pods, err = clientset.CoreV1().Pods(namespace).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name + " in namespace " + namespace)
		err = clientset.Core().Pods(namespace).Delete(pod.Name,
			&meta_v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name + " in namespace " + namespace)

	}

	//delete the master service

	err = clientset.Core().Services(namespace).Delete(cl.Spec.Name,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Service " + err.Error())
	}
	log.Info("deleted master service " + cl.Spec.Name)

	//delete the replica service
	err = clientset.Core().Services(namespace).Delete(cl.Spec.Name+REPLICA_SUFFIX,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Service " + err.Error())
	}
	log.Info("deleted replica service " + cl.Spec.Name + REPLICA_SUFFIX + " in namespace " + namespace)

	return err

}

func shutdownCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {
	var err error

	//var replicaName = cl.Spec.Name + REPLICA_SUFFIX

	//get the deployments
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cl.Spec.Name}
	deployments, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(lo)
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
	time.Sleep(9000 * time.Millisecond)

	//TODO when client-go 3.0 is ready, use propagation_policy
	//in the delete options to also delete the replica sets

	//delete the deployments
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	for _, d := range deployments.Items {
		log.Debug("deleting deployment " + d.ObjectMeta.Name)
		err = clientset.ExtensionsV1beta1().Deployments(namespace).Delete(d.ObjectMeta.Name, &delOptions)
		if err != nil {
			log.Error("error deleting replica Deployment " + err.Error())
		}
	}

	/**
	//TODO for k8s 1.6 and client-go 3.0 we can use propagation_policy
	// to have the replica sets removed as part of the deployment remove
	//delete replica sets if they exist
	options := meta_v1.ListOptions{}
	options.LabelSelector = "pg-cluster=" + cl.Spec.Name

	var reps *v1beta1.ReplicaSetList
	reps, err = clientset.ReplicaSets(namespace).List(options)
	if err != nil {
		log.Error("error getting cluster replicaset name" + err.Error())
	} else {
		for _, r := range reps.Items {
			err = clientset.ReplicaSets(namespace).Delete(r.Name,
				&meta_v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting cluster replicaset " + err.Error())
			}

			log.Info("deleted cluster replicaset " + r.Name + " in namespace " + namespace)
		}
	}
	*/

	for _, d := range deployments.Items {
		log.Debug("making sure deployment " + d.ObjectMeta.Name + " is deleted")
		err := util.WaitUntilDeploymentIsDeleted(clientset, d.ObjectMeta.Name, time.Second*39, namespace)
		if err != nil {
			log.Error("timeout waiting for deployment " + d.ObjectMeta.Name + " to delete " + err.Error())
		}
	}

	return err

}

func (r ClusterStrategy1) PrepareClone(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cloneName string, cl *crv1.Pgcluster, namespace string) error {
	var err error

	log.Info("creating clone deployment using Strategy 1 in namespace " + namespace)

	//create a PVC
	pvcName, err := pvc.CreatePVC(clientset, cloneName, &cl.Spec.ReplicaStorage, namespace)
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
	d, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(cl.Spec.ClusterName, meta_v1.GetOptions{})
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

	_, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(clusterName, meta_v1.GetOptions{})
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
	deployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(clusterName, meta_v1.GetOptions{})
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

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}

func (r ClusterStrategy1) CreateReplica(serviceName string, clientset *kubernetes.Clientset, cl *crv1.Pgcluster, depName, pvcName, namespace string, cloneFlag bool) error {
	var replicaDoc bytes.Buffer
	var err error
	var replicaDeploymentResult *v1beta1.Deployment

	clusterName := cl.Spec.ClusterName

	if cloneFlag {
		clusterName = depName
	}

	replicaLabels := getMasterLabels(serviceName, clusterName, cloneFlag, true, cl.Spec.UserLabels)
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
		OPERATOR_LABELS:      util.GetLabelsFromMap(replicaLabels),
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.ReplicaStorage.FSGROUP, cl.Spec.ReplicaStorage.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
		NODE_SELECTOR:        GetAffinity(cl.Spec.NodeName, "NotIn"),
	}

	switch cl.Spec.ReplicaStorage.StorageType {
	case "", "emptydir":
		log.Debug("MasterStorage.StorageType is emptydir")
		err = ReplicaDeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	case "existing", "create", "dynamic":
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

	replicaDeploymentResult, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&replicaDeployment)
	if err != nil {
		log.Error("error creating replica Deployment " + err.Error())
		return err
	}

	log.Info("created replica Deployment " + replicaDeploymentResult.Name)
	return err
}

func getMasterLabels(Name string, ClusterName string, cloneFlag bool, replicaFlag bool, userLabels map[string]string) map[string]string {
	masterLabels := make(map[string]string)
	if cloneFlag {
		masterLabels["clone"] = "true"
	}
	if replicaFlag {
		masterLabels["replica"] = "true"
	}

	masterLabels["name"] = Name
	masterLabels["pg-cluster"] = ClusterName

	for key, value := range userLabels {
		masterLabels[key] = value
	}
	return masterLabels
}

func GetAffinity(nodeName string, operator string) string {
	output := ""
	if nodeName == "" {
		return output
	}

	affinityTemplateFields := AffinityTemplateFields{}
	affinityTemplateFields.NODE = nodeName
	affinityTemplateFields.OPERATOR = operator

	var affinityDoc bytes.Buffer
	err := AffinityTemplate1.Execute(&affinityDoc, affinityTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return output
	}

	affinityDocString := affinityDoc.String()
	log.Info(affinityDocString)

	return affinityDocString
}
