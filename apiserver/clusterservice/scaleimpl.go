package clusterservice

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strconv"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	//clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScaleCluster ...
func ScaleCluster(name, replicaCount, resourcesConfig, storageConfig, nodeLabel, ccpImageTag, serviceType, ns string) msgs.ClusterScaleResponse {
	var err error

	response := msgs.ClusterScaleResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		response.Status.Code = msgs.Error
		response.Status.Msg = "all is not allowed for the scale command"
		return response
	}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(ns).
		Name(name).
		Do().Into(&cluster)

	if kerrors.IsNotFound(err) {
		log.Error("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if err != nil {
		log.Error("error getting cluster" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	spec := crv1.PgreplicaSpec{}

	//get the resource-config
	if resourcesConfig != "" {
		spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(resourcesConfig)
	} else {
		defaultContainerResource := apiserver.Pgo.DefaultContainerResources
		if defaultContainerResource != "" {
			spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(defaultContainerResource)
		}
	}

	//refer to the cluster's replica storage setting by default
	spec.ReplicaStorage = cluster.Spec.ReplicaStorage

	//allow for user override
	if storageConfig != "" {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(storageConfig)
	}

	spec.UserLabels = cluster.Spec.UserLabels

	if ccpImageTag != "" {
		spec.UserLabels[config.LABEL_CCP_IMAGE_TAG_KEY] = ccpImageTag
	}
	if serviceType != "" {
		if serviceType != config.DEFAULT_SERVICE_TYPE &&
			serviceType != config.NODEPORT_SERVICE_TYPE &&
			serviceType != config.LOAD_BALANCER_SERVICE_TYPE {
			response.Status.Code = msgs.Error
			response.Status.Msg = "error --service-type should be either ClusterIP, NodePort, or LoadBalancer "
			return response
		}
		spec.UserLabels[config.LABEL_SERVICE_TYPE] = serviceType
	}

	var parts []string

	//set replica node lables to blank to start with, then check for overrides
	spec.UserLabels[config.LABEL_NODE_LABEL_KEY] = ""
	spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] = ""

	if apiserver.Pgo.Cluster.ReplicaNodeLabel != "" {
		//should have been validated at apiserver startup
		parts = strings.Split(apiserver.Pgo.Cluster.ReplicaNodeLabel, "=")
		spec.UserLabels[config.LABEL_NODE_LABEL_KEY] = parts[0]
		spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] = parts[1]
		log.Debug("using pgo.yaml ReplicaNodeLabel for replica creation")
	}

	// validate & parse nodeLabel if exists
	if nodeLabel != "" {

		if err = apiserver.ValidateNodeLabel(nodeLabel); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		parts := strings.Split(nodeLabel, "=")
		spec.UserLabels[config.LABEL_NODE_LABEL_KEY] = parts[0]
		spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] = parts[1]

		log.Debug("using user entered node label for replica creation")
	}

	labels := make(map[string]string)
	labels[config.LABEL_PG_CLUSTER] = cluster.Spec.Name

	spec.ClusterName = cluster.Spec.Name

	var rc int
	rc, err = strconv.Atoi(replicaCount)
	if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	for i := 0; i < rc; i++ {

		uniqueName := util.RandStringBytesRmndr(4)
		labels[config.LABEL_NAME] = cluster.Spec.Name + "-" + uniqueName
		spec.Namespace = ns
		spec.Name = labels[config.LABEL_NAME]

		newInstance := &crv1.Pgreplica{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   labels[config.LABEL_NAME],
				Labels: labels,
			},
			Spec: spec,
			Status: crv1.PgreplicaStatus{
				State:   crv1.PgreplicaStateCreated,
				Message: "Created, not processed yet",
			},
		}

		result := crv1.Pgreplica{}

		err = apiserver.RESTClient.Post().
			Resource(crv1.PgreplicaResourcePlural).
			Namespace(ns).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error(" in creating Pgreplica instance" + err.Error())
		}

		response.Results = append(response.Results, "created Pgreplica "+labels[config.LABEL_NAME])
	}

	return response
}

// ScaleQuery ...
func ScaleQuery(name, ns string) msgs.ScaleQueryResponse {
	var err error

	response := msgs.ScaleQueryResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(ns).
		Name(name).
		Do().Into(&cluster)

	if kerrors.IsNotFound(err) {
		log.Error("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if err != nil {
		log.Error("error getting cluster" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//get replicas for this cluster
	//deployments with --selector=service-name=ClusterName-replica,pg-cluster=ClusterName

	selector := config.LABEL_SERVICE_NAME + "=" + name + "-replica" + "," + config.LABEL_PG_CLUSTER + "=" + name

	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if kerrors.IsNotFound(err) {
		log.Debug("no replicas found")
		response.Status.Msg = "no replicas found for " + name
		return response
	} else if err != nil {
		log.Error("error getting deployments " + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	response.Results = make([]string, 0)
	response.Targets = make([]msgs.ScaleQueryTargetSpec, 0)

	log.Debugf("deps len %d\n", len(deployments.Items))

	for _, dep := range deployments.Items {
		log.Debugf("found %s", dep.Name)
		target := msgs.ScaleQueryTargetSpec{}
		target.Name = dep.Name
		//get the pod status
		target.ReadyStatus, target.Node = apiserver.GetPodStatus(dep.Name, ns)
		//get the rep status
		receiveLocation, replayLocation, _, err := util.GetRepStatus(apiserver.RESTClient, apiserver.Clientset, &dep, ns, apiserver.Pgo.Cluster.Port)
		if err != nil {
			log.Error("error getting rep status " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		target.RepStatus = fmt.Sprintf("receive %d replay %d", receiveLocation, replayLocation)
		response.Targets = append(response.Targets, target)
	}

	return response
}

// ScaleDown ...
func ScaleDown(deleteData bool, clusterName, replicaName, ns string) msgs.ScaleDownResponse {
	var err error

	response := msgs.ScaleDownResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(ns).
		Name(clusterName).
		Do().Into(&cluster)

	if kerrors.IsNotFound(err) {
		log.Error("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if err != nil {
		log.Error("error getting cluster" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//if this was the last replica then remove the replica service
	var replicaList *v1.DeploymentList
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_SERVICE_NAME + "=" + clusterName + "-replica"
	replicaList, err = kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}
	if len(replicaList.Items) == 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = "no replicas found for this cluster"
		return response
	}

	//validate the replica name that was passed
	replica := crv1.Pgreplica{}
	found, err := kubeapi.Getpgreplica(apiserver.RESTClient, &replica, replicaName, ns)
	if !found || err != nil {
		log.Error(err)
		response.Status.Code = msgs.Error
		if !found {
			response.Status.Msg = replicaName + " replica not found"
		} else {
			response.Status.Msg = err.Error()
		}
		return response
	}

	if len(replicaList.Items) == 1 {
		log.Debug("removing replica service when scaling down to 0 replicas")
		err = kubeapi.DeleteService(apiserver.Clientset, clusterName+"-replica", ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	//delete the pgreplica CRD which will case the replica to be
	//deleted
	err = kubeapi.Deletepgreplica(apiserver.RESTClient, replicaName, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//delete the replica deployment
	err = kubeapi.DeleteDeployment(apiserver.Clientset, replicaName, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//clusteroperator.ScaleDownBase(apiserver.Clientset, apiserver.RESTClient, &replica, ns)

	if deleteData {
		log.Debug("delete-data is true on replica scale down, createing rmdata task")
		selector := config.LABEL_REPLICA_NAME + "=" + replicaName
		pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		} else {
			if len(pods.Items) == 0 {
				response.Status.Code = msgs.Error
				response.Status.Msg = "pod not found for scale down replica and delete-data"
				return response
			}

			err = createDeleteDataTasksForReplica(replicaName, replica.Spec.ReplicaStorage, ns)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

		}
	}
	response.Results = append(response.Results, "deleted Pgreplica "+replicaName)
	return response
}

// removes data and or backup volumes for all pods in a cluster replica
func createDeleteDataTasksForReplica(replicaName string, storageSpec crv1.PgStorageSpec, ns string) error {

	var err error

	log.Debugf("inside createDeleteDataTasksForReplica %s", replicaName)

	dataRoots := make([]string, 0)
	dataRoots = append(dataRoots, replicaName)

	claimName := replicaName

	err = apiserver.CreateRMDataTask(storageSpec, replicaName, claimName, dataRoots, replicaName+"-rmdata-pgdata", ns)
	if err != nil {
		return err
	}

	return err
}
