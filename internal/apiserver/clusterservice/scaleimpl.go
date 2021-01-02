package clusterservice

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScaleCluster ...
func ScaleCluster(name, replicaCount, storageConfig, nodeLabel,
	ccpImageTag, serviceType, ns, pgouser string) msgs.ClusterScaleResponse {
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

	// check if the current cluster is not upgraded to the deployed
	// Operator version. If not, do not allow the command to complete
	if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
		response.Status.Code = msgs.Error
		response.Status.Msg = cluster.Name + msgs.UpgradeError
		return response
	}

	spec := crv1.PgreplicaSpec{}

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

	//set replica node lables to blank to start with, then check for overrides
	spec.UserLabels[config.LABEL_NODE_LABEL_KEY] = ""
	spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] = ""

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

	labels[config.LABEL_PGOUSER] = pgouser
	labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]

	for i := 0; i < rc; i++ {

		uniqueName := util.RandStringBytesRmndr(4)
		labels[config.LABEL_NAME] = cluster.Spec.Name + "-" + uniqueName
		spec.Namespace = ns
		spec.Name = labels[config.LABEL_NAME]

		newInstance := &crv1.Pgreplica{
			ObjectMeta: metav1.ObjectMeta{
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

// ScaleQuery lists the replicas that are in the PostgreSQL cluster
// with information that is helpful in determining which one to fail over to,
// such as the lag behind the replica as well as the timeline
func ScaleQuery(name, ns string) msgs.ScaleQueryResponse {
	var err error

	response := msgs.ScaleQueryResponse{
		Results: make([]msgs.ScaleQueryTargetSpec, 0),
		Status:  msgs.Status{Code: msgs.Ok, Msg: ""},
	}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(ns).
		Name(name).
		Do().Into(&cluster)

	// If no clusters are found, return a specific error message,
	// otherwise, pass forward the generic error message that Kubernetes sends
	if kerrors.IsNotFound(err) {
		errorMsg := fmt.Sprintf(`No cluster found for "%s"`, name)
		log.Error(errorMsg)
		response.Status.Code = msgs.Error
		response.Status.Msg = errorMsg
		return response
	} else if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Get information about the current status of all of the replicas. This is
	// handled by a helper function, that will return the information in a struct
	// with the key elements to help the user understand the current state of the
	// replicas in a cluster
	replicationStatusRequest := util.ReplicationStatusRequest{
		RESTConfig:  apiserver.RESTConfig,
		Clientset:   apiserver.Clientset,
		Namespace:   ns,
		ClusterName: name,
	}

	replicationStatusResponse, err := util.ReplicationStatus(replicationStatusRequest, false, true)

	// if an error is return, log the message, and return the response
	if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// indicate in the response whether or not a standby cluster
	response.Standby = cluster.Spec.Standby

	// if there are no results, return the response as is
	if len(replicationStatusResponse.Instances) == 0 {
		return response
	}

	// iterate through response results to create the API response
	for _, instance := range replicationStatusResponse.Instances {
		// create a result for the response
		result := msgs.ScaleQueryTargetSpec{
			Name:           instance.Name,
			Node:           instance.Node,
			Status:         instance.Status,
			ReplicationLag: instance.ReplicationLag,
			Timeline:       instance.Timeline,
			PendingRestart: instance.PendingRestart,
		}

		// append the result to the response list
		response.Results = append(response.Results, result)
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

	// dont proceed any further if the cluster is shutdown
	if cluster.Status.State == crv1.PgclusterStateShutdown {
		response.Status.Code = msgs.Error
		response.Status.Msg = "Nothing to scale, the cluster is currently " +
			"shutdown"
		return response
	}

	// selector in the format "pg-cluster=<cluster-name>,pgo-pg-database,role!=config.LABEL_PGHA_ROLE_PRIMARY"
	// which will grab all the replicas
	selector := fmt.Sprintf("%s=%s,%s,%s!=%s", config.LABEL_PG_CLUSTER, clusterName,
		config.LABEL_PG_DATABASE, config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)
	replicaList, err := apiserver.Clientset.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// check to see if the replica name provided matches the name of any of the
	// replicas found for the cluster
	var replicaNameFound bool
	for _, pod := range replicaList.Items {
		if pod.Labels[config.LABEL_DEPLOYMENT_NAME] == replicaName {
			replicaNameFound = true
			break
		}
	}
	// return an error if the replica name provided does not match the primary or any replicas
	if !replicaNameFound {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Unable to find replica with name %s",
			replicaName)
		return response
	}

	//create the rmdata task which does the cleanup

	clusterPGHAScope := cluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE]
	deleteBackups := false
	isReplica := true
	isBackup := false
	taskName := replicaName + "-rmdata"
	err = apiserver.CreateRMDataTask(clusterName, replicaName, taskName, deleteBackups, deleteData, isReplica, isBackup, ns, clusterPGHAScope)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	response.Results = append(response.Results, "deleted replica "+replicaName)
	return response
}
