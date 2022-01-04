package restartservice

/*
Copyright 2020 - 2022 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Restart restarts either all PostgreSQL databases within a PostgreSQL cluster (i.e. the primary
// and all replicas) or if targets are specified, just those targets.
// pgo restart mycluster
// pgo restart mycluster --target=mycluster-abcd
func Restart(request *msgs.RestartRequest, pgouser string) msgs.RestartResponse {

	log.Debugf("restart called for %s", request.ClusterName)

	resp := msgs.RestartResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	clusterName := request.ClusterName
	namespace := request.Namespace

	cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).Get(clusterName,
		metav1.GetOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// check if the current cluster is not upgraded to the deployed
	// Operator version. If not, do not allow the command to complete
	if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("%s %s", cluster.Name, msgs.UpgradeError)
		return resp
	}

	var restartResults []patroni.RestartResult
	// restart either the whole cluster, or just any targets specified
	patroniClient := patroni.NewPatroniClient(apiserver.RESTConfig, apiserver.Clientset,
		cluster.GetName(), namespace)
	if len(request.Targets) > 0 {
		restartResults, err = patroniClient.RestartInstances(request.Targets...)
	} else {
		restartResults, err = patroniClient.RestartCluster()
	}
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	restartDetails := msgs.RestartDetail{ClusterName: clusterName}
	for _, restartResult := range restartResults {

		instanceDetail := msgs.InstanceDetail{InstanceName: restartResult.Instance}
		if restartResult.Error != nil {
			instanceDetail.Error = true
			instanceDetail.ErrorMessage = restartResult.Error.Error()
		}

		restartDetails.Instances = append(restartDetails.Instances, instanceDetail)
	}

	resp.Result = restartDetails

	return resp
}

// QueryRestart queries a cluster for instances available to use as as targets for a PostgreSQL restart.
// pgo restart mycluster --query
func QueryRestart(clusterName, namespace string) msgs.QueryRestartResponse {

	log.Debugf("query restart called for %s", clusterName)

	resp := msgs.QueryRestartResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).Get(clusterName,
		metav1.GetOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// Get information about the current status of all of all cluster instances. This is
	// handled by a helper function, that will return the information in a struct with the
	// key elements to help the user understand the current state of the instances in a cluster
	replicationStatusRequest := util.ReplicationStatusRequest{
		RESTConfig:  apiserver.RESTConfig,
		Clientset:   apiserver.Clientset,
		Namespace:   namespace,
		ClusterName: clusterName,
	}

	// get a list of all the Pods...note that we can included "busted" pods as
	// by including the primary, we're getting all of the database pods anyway.
	replicationStatusResponse, err := util.ReplicationStatus(replicationStatusRequest, true, true)
	if err != nil {
		log.Error(err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// if there are no results, return the response as is
	if len(replicationStatusResponse.Instances) == 0 {
		return resp
	}

	// iterate through response results to create the API response
	for _, instance := range replicationStatusResponse.Instances {
		// create an result for the response
		resp.Results = append(resp.Results, msgs.RestartTargetSpec{
			Name:           instance.Name,
			Node:           instance.Node,
			Status:         instance.Status,
			ReplicationLag: instance.ReplicationLag,
			Timeline:       instance.Timeline,
			PendingRestart: instance.PendingRestart,
			Role:           instance.Role,
		})
	}

	resp.Standby = cluster.Spec.Standby

	return resp
}
