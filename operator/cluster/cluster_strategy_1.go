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

	ServiceTemplate1 = util.LoadTemplate("/pgconf/postgres-operator/cluster/1/cluster-service.json")
	ReplicaDeploymentTemplate1 = util.LoadTemplate("/pgconf/postgres-operator/cluster/1/cluster-replica-deployment.json")
	DeploymentTemplate1 = util.LoadTemplate("/pgconf/postgres-operator/cluster/1/cluster-deployment.json")
}

func (r ClusterStrategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgCluster, namespace string) error {
	var serviceDoc, replicaServiceDoc, masterDoc, replicaDoc bytes.Buffer
	var err error
	var replicaServiceResult, serviceResult *v1.Service
	var replicaDeploymentResult, deploymentResult *v1beta1.Deployment

	log.Info("creating PgCluster object using Strategy 1" + " in namespace " + namespace)
	log.Info("created with Name=" + db.Spec.Name  + " in namespace " + namespace)

	//create the master service
	serviceFields := ServiceTemplateFields{
		Name:        db.Spec.Name,
		ClusterName: db.Spec.Name,
		Port:        db.Spec.Port,
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
		Name:        db.Spec.Name + REPLICA_SUFFIX,
		ClusterName: db.Spec.Name,
		Port:        db.Spec.Port,
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
		Name:               db.Spec.Name,
		ClusterName:        db.Spec.Name,
		Port:               db.Spec.Port,
		CCP_IMAGE_TAG:      db.Spec.CCP_IMAGE_TAG,
		PVC_NAME:           db.Spec.PVC_NAME,
		PG_MASTER_USER:     db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD: db.Spec.PG_MASTER_PASSWORD,
		PG_USER:            db.Spec.PG_USER,
		PG_PASSWORD:        db.Spec.PG_PASSWORD,
		PG_DATABASE:        db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:   db.Spec.PG_ROOT_PASSWORD,
		SECURITY_CONTEXT:   util.CreateSecContext(db.Spec.FS_GROUP, db.Spec.SUPPLEMENTAL_GROUPS),
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
		Name:               db.Spec.Name + REPLICA_SUFFIX,
		ClusterName:        db.Spec.Name,
		Port:               db.Spec.Port,
		CCP_IMAGE_TAG:      db.Spec.CCP_IMAGE_TAG,
		PVC_NAME:           db.Spec.PVC_NAME,
		PG_MASTER_HOST:     db.Spec.PG_MASTER_HOST,
		PG_MASTER_USER:     db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD: db.Spec.PG_MASTER_PASSWORD,
		PG_USER:            db.Spec.PG_USER,
		PG_PASSWORD:        db.Spec.PG_PASSWORD,
		PG_DATABASE:        db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:   db.Spec.PG_ROOT_PASSWORD,
		REPLICAS:           db.Spec.REPLICAS,
		SECURITY_CONTEXT:   util.CreateSecContext(db.Spec.FS_GROUP, db.Spec.SUPPLEMENTAL_GROUPS),
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

func (r ClusterStrategy1) DeleteCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgCluster, namespace string) error {
	log.Info("deleting PgCluster object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + db.Spec.Name + " in namespace " + namespace)

	//delete the master service

	err := clientset.Services(namespace).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Service " + err.Error())
	}
	log.Info("deleted master service " + db.Spec.Name)

	//delete the replica service
	err = clientset.Services(namespace).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Service " + err.Error())
	}
	log.Info("deleted replica service " + db.Spec.Name + REPLICA_SUFFIX + " in namespace " + namespace)

	//delete the master deployment
	err = clientset.Deployments(namespace).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Deployment " + err.Error())
	}

	log.Info("deleted master Deployment " + db.Spec.Name + " in namespace " + namespace)
	//delete the master replicaset

	//find the replicaset pod name
	options := v1.ListOptions{}
	options.LabelSelector = "name=" + db.Spec.Name

	var reps *v1beta1.ReplicaSetList
	reps, err = clientset.ReplicaSets(namespace).List(options)
	if err != nil {
		log.Error("error getting master replicaset name" + err.Error())
	} else {
		if len(reps.Items) > 0 {
			err = clientset.ReplicaSets(namespace).Delete(reps.Items[0].Name,
				&v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting master replicaset " + err.Error())
			}

			log.Info("deleted master replicaset " + reps.Items[0].Name + " in namespace " + namespace)
		}
	}

	//delete the replica deployment
	err = clientset.Deployments(namespace).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Deployment " + err.Error())
	}
	log.Info("deleted replica Deployment " + db.Spec.Name + REPLICA_SUFFIX + " in namespace " + namespace)
	//delete the replica ReplicaSet
	options.LabelSelector = "name=" + db.Spec.Name + REPLICA_SUFFIX

	reps, err = clientset.ReplicaSets(namespace).List(options)
	if err != nil {
		log.Error("error getting replica replicaset name" + err.Error())
	} else {
		if len(reps.Items) > 0 {
			err = clientset.ReplicaSets(namespace).Delete(reps.Items[0].Name,
				&v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting replica replicaset " + err.Error())
			}
			log.Info("deleted replica replicaset " + reps.Items[0].Name + " in namespace " + namespace)
		}
	}

	//lastly, delete any remaining pods
	listOptions := v1.ListOptions{}
	listOptions.LabelSelector = "name=" + db.Spec.Name
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
	listOptions.LabelSelector = "name=" + db.Spec.Name + REPLICA_SUFFIX
	pods, err = clientset.Core().Pods(namespace).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name  + " in namespace " + namespace)
		err = clientset.Pods(namespace).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name + " in namespace " + namespace)

	}
	return err

}
