package clusterservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

// ScaleCluster ...
func ScaleCluster(name, replicaCount, resourcesConfig, storageConfig, nodeLabel, ccpImageTag, serviceType string) msgs.ClusterScaleResponse {
	var err error

	response := msgs.ClusterScaleResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(apiserver.Namespace).
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

	//get the storage-config
	if storageConfig != "" {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(storageConfig)
	} else {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.ReplicaStorage)
	}

	//spec.UserLabels = make(map[string]string)
	spec.UserLabels = cluster.Spec.UserLabels

	if ccpImageTag != "" {
		spec.UserLabels[util.LABEL_CCP_IMAGE_TAG_KEY] = ccpImageTag
	}
	if serviceType != "" {
		if serviceType != config.DEFAULT_SERVICE_TYPE &&
			serviceType != config.LOAD_BALANCER_SERVICE_TYPE {
			response.Status.Code = msgs.Error
			response.Status.Msg = "error --service-type should be either ClusterIP or LoadBalancer "
			return response
		}
		spec.UserLabels[util.LABEL_SERVICE_TYPE] = serviceType
	}

	var parts []string
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
		spec.Name = labels[util.LABEL_NAME]

		//copy cluster info over to replica to avoid a CRD read later
		//spec.Strategy = cluster.Spec.Strategy
		//		spec.Port = cluster.Spec.Port
		//spec.CCPImageTag = cluster.Spec.CCPImageTag
		//spec.PrimaryHost = cluster.Spec.PrimaryHost
		//spec.Database = cluster.Spec.Database
		//spec.RootSecretName = cluster.Spec.RootSecretName
		//spec.PrimarySecretName = cluster.Spec.PrimarySecretName
		//spec.UserSecretName = cluster.Spec.UserSecretName

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
			Namespace(apiserver.Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error(" in creating Pgreplica instance" + err.Error())
		}

		response.Results = append(response.Results, "created Pgreplica "+labels[util.LABEL_NAME])
	}

	/**
	futureReplicas := currentReplicas + replicaCount
	log.Debug("scaling %s to %d from %d\n", name, futureReplicas, currentReplicas)
	err = util.Patch(apiserver.RESTClient, "/spec/replicas", futureReplicas, crv1.PgclusterResourcePlural, name, apiserver.Namespace)
	if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}
	*/

	return response
}

/**
type ScaleQueryTargetSpec struct {
	Name        string
	ReadyStatus string
	Node        string
	RepStatus   string
}

type ScaleQueryResponse struct {
	Results []string
	Targets []ScaleQueryTargetSpec
	Status
}
*/

// ScaleQuery ...
func ScaleQuery(name string) msgs.ScaleQueryResponse {
	var err error

	response := msgs.ScaleQueryResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(apiserver.Namespace).
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
	//deployments with --selector=primary=false,pg-cluster=ClusterName

	selector := util.LABEL_PRIMARY + "=false," + util.LABEL_PG_CLUSTER + "=" + name

	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
	if kerrors.IsNotFound(err) {
		log.Debug("no replicas found ")
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
		log.Debug("found " + dep.Name)
		target := msgs.ScaleQueryTargetSpec{}
		target.Name = dep.Name
		//get the pod status
		target.ReadyStatus, target.Node = apiserver.GetPodStatus(dep.Name)
		//get the rep status
		response.Targets = append(response.Targets, target)
	}

	return response
}
