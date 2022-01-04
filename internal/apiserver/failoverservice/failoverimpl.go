package failoverservice

/*
Copyright 2018 - 2022 Crunchy Data Solutions, Inc.
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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// CreateFailover is the API endpoint for triggering a manual failover of a
// cluster. It performs this function inline, i.e. it does not trigger any
// asynchronous methods.
//
// pgo failover mycluster
func CreateFailover(request *msgs.CreateFailoverRequest, ns, pgouser string) msgs.CreateFailoverResponse {
	log.Debugf("create failover called for %s", request.ClusterName)

	resp := msgs.CreateFailoverResponse{
		Results: "",
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	cluster, err := validateClusterName(request.ClusterName, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// check if the current cluster is not upgraded to the deployed
	// Operator version. If not, do not allow the command to complete
	if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = cluster.Name + msgs.UpgradeError
		return resp
	}

	if err := isValidFailoverTarget(request); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// perform the switchover or failover, depending on which flag is selected
	// if we are forcing the failover, we need to use "Failover", otherwise we
	// perform a controlled switchover
	if request.Force {
		err = operator.Failover(apiserver.Clientset, apiserver.RESTConfig, cluster, request.Target)
	} else {
		err = operator.Switchover(apiserver.Clientset, apiserver.RESTConfig, cluster, request.Target)
	}

	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = strings.ReplaceAll(err.Error(), "master", "primary")
		return resp
	}

	resp.Results = "failover success for cluster " + cluster.Name

	return resp
}

// QueryFailover provides the user with a list of replicas that can be failed
// over to
// pgo failover mycluster --query
func QueryFailover(name, ns string) msgs.QueryFailoverResponse {
	response := msgs.QueryFailoverResponse{
		Results: make([]msgs.FailoverTargetSpec, 0),
		Status:  msgs.Status{Code: msgs.Ok, Msg: ""},
	}

	cluster, err := validateClusterName(name, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("query failover called for %s", name)

	// indicate in the response whether or not a standby cluster
	response.Standby = cluster.Spec.Standby

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

	replicationStatusResponse, err := util.ReplicationStatus(replicationStatusRequest, false, false)
	// if an error is return, log the message, and return the response
	if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// if there are no results, return the response as is
	if len(replicationStatusResponse.Instances) == 0 {
		return response
	}

	// iterate through response results to create the API response
	for _, instance := range replicationStatusResponse.Instances {
		// create an result for the response
		result := msgs.FailoverTargetSpec{
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

func validateClusterName(clusterName, ns string) (*crv1.Pgcluster, error) {
	ctx := context.TODO()
	cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return cluster, errors.New("no cluster found named " + clusterName)
	}

	return cluster, err
}

// isValidFailoverTarget checks to see if the failover target specified in the request is valid,
// i.e. that it represents a valid replica deployment in the cluster specified.  This is
// done by first ensuring the deployment specified exists and is associated with the cluster
// specified, and then ensuring the PG pod created by the deployment is not the current primary.
// If the deployment is not found, or if the pod is the current primary, an error will be returned.
// Otherwise the deployment is returned.
func isValidFailoverTarget(request *msgs.CreateFailoverRequest) error {
	ctx := context.TODO()

	// if we're not forcing a failover and the target is blank, we can
	// return here
	// However, if we are forcing a failover and the target is blank, then we do
	// have an error
	if request.Target == "" {
		if !request.Force {
			return nil
		}

		return fmt.Errorf("target is required when forcing a failover.")
	}

	// Using the following label selector, ensure the deployment specified using deployName exists in the
	// cluster specified using clusterName:
	// pg-cluster=clusterName,deployment-name=deployName
	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, request.ClusterName),
			fields.OneTermEqualSelector(config.LABEL_DEPLOYMENT_NAME, request.Target),
		).String(),
	}
	deployments, err := apiserver.Clientset.AppsV1().Deployments(request.Namespace).List(ctx, options)

	if err != nil {
		log.Error(err)
		return err
	} else if len(deployments.Items) == 0 {
		return fmt.Errorf("no target found named %s", request.Target)
	} else if len(deployments.Items) > 1 {
		return fmt.Errorf("more than one target found named %s", request.Target)
	}

	// Using the following label selector, determine if the target specified is the current
	// primary for the cluster and return an error if it is:
	// pg-cluster=clusterName,deployment-name=deployName,role=primary
	options.FieldSelector = fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String()
	options.LabelSelector = fields.AndSelectors(
		fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, request.ClusterName),
		fields.OneTermEqualSelector(config.LABEL_DEPLOYMENT_NAME, request.Target),
		fields.OneTermEqualSelector(config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY),
	).String()

	pods, _ := apiserver.Clientset.CoreV1().Pods(request.Namespace).List(ctx, options)

	if len(pods.Items) > 0 {
		return fmt.Errorf("The primary database cannot be selected as a failover target")
	}

	return nil
}
