// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strconv"
	"text/template"
	//"time"
)

const AffinityInOperator = "In"
const AFFINITY_NOTINOperator = "NotIn"

type affinityTemplateFields struct {
	Node          string
	OperatorValue string
}

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

type collectTemplateFields struct {
	Name           string
	CCPImageTag    string
	CCPImagePrefix string
}

// Strategy1  ...
type Strategy1 struct{}

var affinityTemplate1 *template.Template
var containerResourcesTemplate1 *template.Template
var collectTemplate1 *template.Template
var deploymentTemplate1 *template.Template
var replicadeploymentTemplate1 *template.Template
var replicadeploymentTemplate1Shared *template.Template

//var ServiceTemplate1 *template.Template

func init() {

	//ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")
	replicadeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1.json")
	replicadeploymentTemplate1Shared = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1-shared.json")
	deploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-deployment-1.json")
	collectTemplate1 = util.LoadTemplate("/operator-conf/collect.json")
	affinityTemplate1 = util.LoadTemplate("/operator-conf/affinity.json")
	containerResourcesTemplate1 = util.LoadTemplate("/operator-conf/container-resources.json")
}

// AddCluster ...
func (r Strategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string, primaryPVCName string) error {
	var primaryDoc bytes.Buffer
	var err error
	var deploymentResult *v1beta1.Deployment

	log.Info("creating Pgcluster object using Strategy 1" + " in namespace " + namespace)
	log.Info("created with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//create the primary service
	serviceFields := ServiceTemplateFields{
		Name:        cl.Spec.Name,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error("error in creating primary service " + err.Error())
		return err
	}

	primaryLabels := getPrimaryLabels(cl.Spec.Name, cl.Spec.ClusterName, false, false, cl.Spec.UserLabels)

	log.Debug("CCPImagePrefix before create cluster " + operator.CCPImagePrefix)

	//create the primary deployment
	deploymentFields := DeploymentTemplateFields{
		Name:               cl.Spec.Name,
		ClusterName:        cl.Spec.Name,
		Port:               cl.Spec.Port,
		CCPImagePrefix:     operator.CCPImagePrefix,
		CCPImageTag:        cl.Spec.CCPImageTag,
		PVCName:            util.CreatePVCSnippet(cl.Spec.PrimaryStorage.StorageType, primaryPVCName),
		OperatorLabels:     util.GetLabelsFromMap(primaryLabels),
		BackupPVCName:      util.CreateBackupPVCSnippet(cl.Spec.BackupPVCName),
		BackupPath:         cl.Spec.BackupPath,
		DataPathOverride:   cl.Spec.Name,
		Database:           cl.Spec.Database,
		SecurityContext:    util.CreateSecContext(cl.Spec.PrimaryStorage.Fsgroup, cl.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:     cl.Spec.RootSecretName,
		PrimarySecretName:  cl.Spec.PrimarySecretName,
		UserSecretName:     cl.Spec.UserSecretName,
		NodeSelector:       GetAffinity(cl.Spec.NodeName, "In"),
		ContainerResources: GetContainerResources(&cl.Spec.ContainerResources),
		ConfVolume:         GetConfVolume(clientset, cl.Spec.CustomConfig, namespace),
		CollectAddon:       GetCollectAddon(&cl.Spec),
	}

	err = deploymentTemplate1.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
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

	if deploymentExists(clientset, namespace, cl.Spec.Name) == false {
		deploymentResult, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)
		if err != nil {
			log.Error("error creating primary Deployment " + err.Error())
			return err
		}
		log.Info("created primary Deployment " + deploymentResult.Name + " in namespace " + namespace)
	} else {
		log.Info("primary Deployment " + cl.Spec.Name + " in namespace " + namespace + " already existed so not creating it ")
	}

	err = util.PatchClusterCRD(client, primaryLabels, cl, namespace)
	if err != nil {
		log.Error("could not patch primary crv1 with labels")
		return err
	}

	newReplicas, err := strconv.Atoi(cl.Spec.Replicas)
	if err != nil {
		log.Error("could not convert Replicas config setting")
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

// DeleteCluster ...
func (r Strategy1) DeleteCluster(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {

	var err error
	log.Info("deleting Pgcluster object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//delete the primary and replica deployments and replica sets
	err = shutdownCluster(clientset, restclient, cl, namespace)
	if err != nil {
		log.Error("error deleting primary Deployment " + err.Error())
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
	listOptions.LabelSelector = "name=" + cl.Spec.Name + ReplicaSuffix
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

	//delete the primary service

	err = clientset.Core().Services(namespace).Delete(cl.Spec.Name,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting primary Service " + err.Error())
	}
	log.Info("deleted primary service " + cl.Spec.Name)

	//delete the replica service
	err = clientset.Core().Services(namespace).Delete(cl.Spec.Name+ReplicaSuffix,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Service " + err.Error())
	}
	log.Info("deleted replica service " + cl.Spec.Name + ReplicaSuffix + " in namespace " + namespace)

	//delete the pgpool deployment if necessary

	if cl.Spec.UserLabels["crunchy-pgpool"] == "true" {
		DeletePgpool(clientset, cl.Spec.Name, namespace)
	}

	return err

}

// shutdownCluster ...
func shutdownCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {
	var err error

	//var replicaName = cl.Spec.Name + ReplicaSuffix

	//get the deployments
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cl.Spec.Name}
	deployments, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return err
	}

	/**
	for _, d := range deployments.Items {
		log.Debug("draining deployment " + d.ObjectMeta.Name)
		err = util.DrainDeployment(clientset, d.ObjectMeta.Name, namespace)
		if err != nil {
			log.Error("error deleting replica Deployment " + err.Error())
		}
	}

	//sleep just a bit to give the drain time to work
	time.Sleep(9000 * time.Millisecond)
	*/

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

	for _, d := range deployments.Items {
		log.Debug("making sure deployment " + d.ObjectMeta.Name + " is deleted")
		err := util.WaitUntilDeploymentIsDeleted(clientset, d.ObjectMeta.Name, time.Second*39, namespace)
		if err != nil {
			log.Error("timeout waiting for deployment " + d.ObjectMeta.Name + " to delete " + err.Error())
		}
	}
	*/

	return err

}

// PrepareClone ...
func (r Strategy1) PrepareClone(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cloneName string, cl *crv1.Pgcluster, namespace string) error {
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

// deploymentExists ...
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

// UpdatePolicyLabels ...
func (r Strategy1) UpdatePolicyLabels(clientset *kubernetes.Clientset, clusterName string, namespace string, newLabels map[string]string) error {

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

// CreateReplica ...
func (r Strategy1) CreateReplica(serviceName string, clientset *kubernetes.Clientset, cl *crv1.Pgcluster, depName, pvcName, namespace string, cloneFlag bool) error {
	var replicaDoc bytes.Buffer
	var err error
	var replicaDeploymentResult *v1beta1.Deployment

	clusterName := cl.Spec.ClusterName

	if cloneFlag {
		clusterName = depName
	}

	replicaLabels := getPrimaryLabels(serviceName, clusterName, cloneFlag, true, cl.Spec.UserLabels)
	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:               depName,
		ClusterName:        clusterName,
		Port:               cl.Spec.Port,
		CCPImagePrefix:     operator.CCPImagePrefix,
		CCPImageTag:        cl.Spec.CCPImageTag,
		PVCName:            pvcName,
		PrimaryHost:        cl.Spec.PrimaryHost,
		Database:           cl.Spec.Database,
		Replicas:           "1",
		OperatorLabels:     util.GetLabelsFromMap(replicaLabels),
		SecurityContext:    util.CreateSecContext(cl.Spec.ReplicaStorage.Fsgroup, cl.Spec.ReplicaStorage.SupplementalGroups),
		RootSecretName:     cl.Spec.RootSecretName,
		PrimarySecretName:  cl.Spec.PrimarySecretName,
		ContainerResources: GetContainerResources(&cl.Spec.ContainerResources),
		UserSecretName:     cl.Spec.UserSecretName,
		NodeSelector:       GetAffinity(cl.Spec.NodeName, "NotIn"),
	}

	switch cl.Spec.ReplicaStorage.StorageType {
	case "", "emptydir":
		log.Debug("PrimaryStorage.StorageType is emptydir")
		err = replicadeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	case "existing", "create", "dynamic":
		log.Debug("using the shared replica template ")
		err = replicadeploymentTemplate1Shared.Execute(&replicaDoc, replicaDeploymentFields)
	}

	if err != nil {
		log.Error(err.Error())
		return err
	}
	replicaDeploymentDocString := replicaDoc.String()
	log.Debug(replicaDeploymentDocString)

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

// getPrimaryLabels ...
func getPrimaryLabels(Name string, ClusterName string, cloneFlag bool, replicaFlag bool, userLabels map[string]string) map[string]string {
	primaryLabels := make(map[string]string)
	if cloneFlag {
		primaryLabels["clone"] = "true"
	}
	if replicaFlag {
		primaryLabels["replica"] = "true"
	}

	primaryLabels["name"] = Name
	primaryLabels["pg-cluster"] = ClusterName

	for key, value := range userLabels {
		primaryLabels[key] = value
	}
	return primaryLabels
}

// GetAffinity ...
func GetAffinity(nodeName string, operator string) string {
	log.Debugf("GetAffinity with nodeName=[%s] and operator=[%s]\n", nodeName, operator)
	output := ""
	if nodeName == "" {
		return output
	}

	affinityTemplateFields := affinityTemplateFields{}
	affinityTemplateFields.Node = nodeName
	affinityTemplateFields.OperatorValue = operator

	var affinityDoc bytes.Buffer
	err := affinityTemplate1.Execute(&affinityDoc, affinityTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return output
	}

	affinityDocString := affinityDoc.String()
	log.Debug(affinityDocString)

	return affinityDocString
}

func GetCollectAddon(spec *crv1.PgclusterSpec) string {

	if spec.UserLabels["crunchy-collect"] == "true" {
		log.Debug("crunchy-collect was found as a label on cluster create")
		collectTemplateFields := collectTemplateFields{}
		collectTemplateFields.Name = spec.Name
		collectTemplateFields.CCPImageTag = spec.CCPImageTag
		collectTemplateFields.CCPImagePrefix = operator.CCPImagePrefix

		var collectDoc bytes.Buffer
		err := collectTemplate1.Execute(&collectDoc, collectTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}
		collectString := collectDoc.String()
		log.Debug(collectString)
		return collectString
	}
	return ""
}

// GetConfVolume ...
func GetConfVolume(clientset *kubernetes.Clientset, customConfig, namespace string) string {
	var err error

	//check for user provided configmap
	if customConfig != "" {
		_, err = clientset.CoreV1().ConfigMaps(namespace).Get(customConfig, meta_v1.GetOptions{})
		if kerrors.IsNotFound(err) {
			//you should NOT get this error because of apiserver validation of this value!
			log.Error(customConfig + " was not found, error, skipping user provided configMap")
		} else if err != nil {
			log.Error(err)
			log.Error(customConfig + " lookup error, skipping user provided configMap")
		}
		return "\"configMap\": { \"name\": \"" + customConfig + "\" }"

	}

	//check for global custom configmap "pgo-custom-pg-config"
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(util.GLOBAL_CUSTOM_CONFIGMAP, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		log.Debug(util.GLOBAL_CUSTOM_CONFIGMAP + " was not found, , skipping global configMap")
	} else if err != nil {
		log.Error(err)
		log.Error(util.GLOBAL_CUSTOM_CONFIGMAP + " lookup error, skipping global configMap")
	} else {
		return "\"configMap\": { \"name\": \"pgo-custom-pg-config\" }"
	}

	//the default situation
	return "\"emptyDir\": { \"medium\": \"Memory\" }"
}

// GetContainerResources ...
func GetContainerResources(resources *crv1.PgContainerResources) string {

	//test for the case where no container resources are specified
	if resources.RequestsMemory == "" || resources.RequestsCPU == "" ||
		resources.LimitsMemory == "" || resources.LimitsCPU == "" {
		return ""
	}
	fields := containerResourcesTemplateFields{}
	fields.RequestsMemory = resources.RequestsMemory
	fields.RequestsCPU = resources.RequestsCPU
	fields.LimitsMemory = resources.LimitsMemory
	fields.LimitsCPU = resources.LimitsCPU

	var doc bytes.Buffer
	err := containerResourcesTemplate1.Execute(&doc, fields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	docString := doc.String()
	log.Debug(docString)

	return docString
}
