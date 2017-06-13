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
	"time"

	log "github.com/Sirupsen/logrus"
	"text/template"

	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type ClusterStrategy1 struct{}

var DeploymentTemplate1 *template.Template
var ReplicaDeploymentTemplate1 *template.Template
var ServiceTemplate1 *template.Template

func init() {

	ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")
	ReplicaDeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-replica-deployment-1.json")
	DeploymentTemplate1 = util.LoadTemplate("/operator-conf/cluster-deployment-1.json")
}

func (r ClusterStrategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string) error {
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
		PVC_NAME:             cl.Spec.PVC_NAME,
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

	deploymentResult, err = clientset.Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		return err
	}
	log.Info("created master Deployment " + deploymentResult.Name + " in namespace " + namespace)

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
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
	}

	err = ReplicaDeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
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

	//delete the replica deployment
	err = clientset.Deployments(namespace).Delete(replicaName, &v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Deployment " + err.Error())
	}

	//wait for the replica deployment to delete
	err = util.WaitUntilDeploymentIsDeleted(clientset, replicaName, time.Minute, namespace)
	if err != nil {
		log.Error("error waiting for replica Deployment deletion " + err.Error())
	}
	log.Info("deleted replica Deployment " + replicaName + " in namespace " + namespace)

	//delete the master deployment
	err = clientset.Deployments(namespace).Delete(cl.Spec.Name, &v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Deployment " + err.Error())
	}

	//wait for the master deployment to delete
	err = util.WaitUntilDeploymentIsDeleted(clientset, cl.Spec.Name, time.Minute, namespace)
	if err != nil {
		log.Error("error waiting for master Deployment deletion " + err.Error())
	}
	log.Info("deleted master Deployment " + cl.Spec.Name + " in namespace " + namespace)

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

func (r ClusterStrategy1) CloneCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string) error {
	var serviceDoc, replicaServiceDoc, masterDoc, replicaDoc bytes.Buffer
	var err error
	var replicaServiceResult, serviceResult *v1.Service
	var replicaDeploymentResult, deploymentResult *v1beta1.Deployment

	log.Info("creating clone object using Strategy 1" + " in namespace " + namespace)
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
		PVC_NAME:             cl.Spec.PVC_NAME,
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

	deploymentResult, err = clientset.Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		return err
	}
	log.Info("created master Deployment " + deploymentResult.Name + " in namespace " + namespace)

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
		SECURITY_CONTEXT:     util.CreateSecContext(cl.Spec.FS_GROUP, cl.Spec.SUPPLEMENTAL_GROUPS),
		PGROOT_SECRET_NAME:   cl.Spec.PGROOT_SECRET_NAME,
		PGMASTER_SECRET_NAME: cl.Spec.PGMASTER_SECRET_NAME,
		PGUSER_SECRET_NAME:   cl.Spec.PGUSER_SECRET_NAME,
	}

	err = ReplicaDeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
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
