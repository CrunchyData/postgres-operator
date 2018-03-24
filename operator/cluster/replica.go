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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	//"github.com/crunchydata/postgres-operator/operator/pvc"
	//"github.com/crunchydata/postgres-operator/util"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"strconv"
)

// ScaleReplicasBase ...
/**
func ScaleReplicasBase(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string) {

	serviceName := cl.Spec.Name + "-replica"

	//create the service if it doesn't exist
	serviceFields := ServiceTemplateFields{
		Name:        serviceName,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
	}

	err := CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	//get the strategy to use
	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default cluster strategy")
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster upgrade" + cl.Spec.Strategy)
		return
	}

	log.Debug("scale up called ")

	//generate a unique name suffix
	uniqueName := util.RandStringBytesRmndr(4)
	depName := cl.Spec.Name + "-replica-" + uniqueName

	//create a PVC
	pvcName, err := pvc.CreatePVC(clientset, depName, &cl.Spec.ReplicaStorage, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	strategy.CreateReplica(serviceName, clientset, cl, depName, pvcName, namespace, false)
}
*/

// DeletePgreplicas
func DeletePgreplicas(restclient *rest.RESTClient, clusterName, namespace string) {

	replicaList := crv1.PgreplicaList{}

	myselector, err := labels.Parse("pg-cluster=" + clusterName)

	//get a list of pgreplicas for this cluster
	err = restclient.Get().
		Resource(crv1.PgreplicaResourcePlural).
		Namespace(namespace).
		Param("labelSelector", myselector.String()).
		Do().Into(&replicaList)
	if err != nil {
		log.Error("error getting list of pgreplicas" + err.Error())
		return
	}

	log.Debug("pgreplicas found len is %d\n", len(replicaList.Items))

	for _, r := range replicaList.Items {
		err = restclient.Delete().
			Resource(crv1.PgreplicaResourcePlural).
			Namespace(namespace).
			Name(r.Spec.Name).
			Do().
			Error()
		if err != nil {
			log.Error(err)
		}

	}

}
