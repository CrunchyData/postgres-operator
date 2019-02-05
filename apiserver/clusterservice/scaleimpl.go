package clusterservice

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

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
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

	//spec.UserLabels = make(map[string]string)
	spec.UserLabels = cluster.Spec.UserLabels

	if ccpImageTag != "" {
		spec.UserLabels[util.LABEL_CCP_IMAGE_TAG_KEY] = ccpImageTag
	}
	if serviceType != "" {
		if serviceType != config.DEFAULT_SERVICE_TYPE &&
			serviceType != config.NODEPORT_SERVICE_TYPE &&
			serviceType != config.LOAD_BALANCER_SERVICE_TYPE {
			response.Status.Code = msgs.Error
			response.Status.Msg = "error --service-type should be either ClusterIP, NodePort, or LoadBalancer "
			return response
		}
		spec.UserLabels[util.LABEL_SERVICE_TYPE] = serviceType
	}

	var parts []string

	if apiserver.Pgo.Cluster.ReplicaNodeLabel != "" {
		//should have been validated at apiserver startup
		parts = strings.Split(apiserver.Pgo.Cluster.ReplicaNodeLabel, "=")
		spec.UserLabels[util.LABEL_NODE_LABEL_KEY] = parts[0]
		spec.UserLabels[util.LABEL_NODE_LABEL_VALUE] = parts[1]
		log.Debug("using pgo.yaml ReplicaNodeLabel for replica creation")
	}

	//validate nodeLabel
	if nodeLabel != "" {
		parts = strings.Split(nodeLabel, "=")
		if len(parts) != 2 {
			response.Status.Code = msgs.Error
			response.Status.Msg = nodeLabel + " node label does not follow key=value format"
			return response
		}

		keyValid, valueValid, err := apiserver.IsValidNodeLabel(parts[0], parts[1])
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		if !keyValid {
			response.Status.Code = msgs.Error
			response.Status.Msg = nodeLabel + " key was not valid .. check node labels for correct values to specify"
			return response
		}
		if !valueValid {
			response.Status.Code = msgs.Error
			response.Status.Msg = nodeLabel + " node label value was not valid .. check node labels for correct values to specify"
			return response
		}
		spec.UserLabels[util.LABEL_NODE_LABEL_KEY] = parts[0]
		spec.UserLabels[util.LABEL_NODE_LABEL_VALUE] = parts[1]
		log.Debug("using user entered node label for replica creation")

	}

	labels := make(map[string]string)
	labels[util.LABEL_PG_CLUSTER] = cluster.Spec.Name

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
		labels[util.LABEL_NAME] = cluster.Spec.Name + "-" + uniqueName
		spec.Namespace = ns
		spec.Name = labels[util.LABEL_NAME]

		newInstance := &crv1.Pgreplica{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   labels[util.LABEL_NAME],
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

		response.Results = append(response.Results, "created Pgreplica "+labels[util.LABEL_NAME])
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

	//selector := util.LABEL_PRIMARY + "=false," + util.LABEL_PG_CLUSTER + "=" + name
	selector := util.LABEL_SERVICE_NAME + "=" + name + "-replica" + "," + util.LABEL_PG_CLUSTER + "=" + name

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
		target.ReadyStatus, target.Node = apiserver.GetPodStatus(dep.Name)
		//get the rep status
		receiveLocation, replayLocation, _ := util.GetRepStatus(apiserver.RESTClient, apiserver.Clientset, &dep, ns, apiserver.Pgo.Cluster.Port)

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
	var replicaList *v1beta1.DeploymentList
	//selector := util.LABEL_PG_CLUSTER + "=" + clusterName + "," + util.LABEL_PRIMARY + "=false"
	selector := util.LABEL_PG_CLUSTER + "=" + clusterName + "," + util.LABEL_SERVICE_NAME + "=" + clusterName + "-replica"
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

	err = kubeapi.Deletepgreplica(apiserver.RESTClient, replicaName, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//delete the replica deployment
	clusteroperator.ScaleDownBase(apiserver.Clientset, apiserver.RESTClient, &replica, ns)

	if deleteData {
		log.Debug("delete-data is true on replica scale down, createing rmdata task")
		selector := util.LABEL_REPLICA_NAME + "=" + replicaName
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

	log.Info("inside createDeleteDataTasksForReplica")

	selector := util.LABEL_REPLICA_NAME + "=" + replicaName
	log.Debugf("selector for delete is %s", selector)
	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("got %d cluster pods for %s\n", len(pods.Items), replicaName)

	for _, pod := range pods.Items {
		deploymentName := pod.ObjectMeta.Labels[util.LABEL_REPLICA_NAME]

		//get the volumes for this pod
		for _, v := range pod.Spec.Volumes {

			log.Debugf("volume name in delete logic is %s", v.Name)
			dataRoots := make([]string, 0)
			if v.Name == "pgdata" {
				dataRoots = append(dataRoots, deploymentName)
			} else if v.Name == "backrestrepo-volume" {
				dataRoots = append(dataRoots, deploymentName+"{-backups,-spool}")
			} else if v.Name == "backup" {
				dataRoots = append(dataRoots, deploymentName+"-backups")
			} else if v.Name == "pgwal-volume" {
				dataRoots = append(dataRoots, deploymentName+"-wal")
			}

			if v.VolumeSource.PersistentVolumeClaim != nil {
				log.Debugf("volume [%s] pvc [%s] dataroots [%v]\n", v.Name, v.VolumeSource.PersistentVolumeClaim.ClaimName, dataRoots)
				err = apiserver.CreateRMDataTask(storageSpec, replicaName, v.VolumeSource.PersistentVolumeClaim.ClaimName, dataRoots, ns)
				if err != nil {
					return err
				}
			}
		}
	}

	return err
}
