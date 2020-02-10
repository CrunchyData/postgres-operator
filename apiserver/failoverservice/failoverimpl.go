package failoverservice

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//  CreateFailover ...
// pgo failover mycluster
// pgo failover all
// pgo failover --selector=name=mycluster
func CreateFailover(request *msgs.CreateFailoverRequest, ns, pgouser string) msgs.CreateFailoverResponse {
	var err error
	resp := msgs.CreateFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	_, err = validateClusterName(request.ClusterName, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if request.Target != "" {
		_, err = isValidFailoverTarget(request.Target, request.ClusterName, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	log.Debugf("create failover called for %s", request.ClusterName)

	// Create a pgtask
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = request.ClusterName + "-" + config.LABEL_FAILOVER

	// previous failovers will leave a pgtask so remove it first
	kubeapi.Deletepgtask(apiserver.RESTClient, spec.Name, ns)

	spec.TaskType = crv1.PgtaskFailover
	spec.Parameters = make(map[string]string)
	spec.Parameters[request.ClusterName] = request.ClusterName

	labels := make(map[string]string)
	labels["target"] = request.Target
	labels[config.LABEL_PG_CLUSTER] = request.ClusterName
	labels[config.LABEL_PGOUSER] = pgouser

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	err = kubeapi.Createpgtask(apiserver.RESTClient,
		newInstance, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, "created Pgtask (failover) for cluster "+request.ClusterName)

	return resp
}

// QueryFailover provides the user with a list of replicas that can be failed
// over to
// pgo failover mycluster --query
func QueryFailover(name, ns string) msgs.QueryFailoverResponse {
	var err error
	response := msgs.QueryFailoverResponse{
		Results: make([]msgs.FailoverTargetSpec, 0),
		Status:  msgs.Status{Code: msgs.Ok, Msg: ""},
	}

	_, err = validateClusterName(name, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("query failover called for %s", name)

	var nodes []string

	if apiserver.Pgo.Pgo.PreferredFailoverNode != "" {
		log.Debug("PreferredFailoverNode is set to %s", apiserver.Pgo.Pgo.PreferredFailoverNode)
		nodes, err = util.GetPreferredNodes(apiserver.Clientset, apiserver.Pgo.Pgo.PreferredFailoverNode, ns)
		log.Debug(nodes)
		if err != nil {
			log.Error("error getting preferred nodes " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	} else {
		log.Debug("PreferredFailoverNode is not set")
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

	replicationStatusResponse, err := util.ReplicationStatus(replicationStatusRequest)

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
			PreferredNode:  preferredNode(nodes, instance.Node),
			Status:         instance.Status,
			ReplicationLag: instance.ReplicationLag,
			Timeline:       instance.Timeline,
		}

		// append the result to the response list
		response.Results = append(response.Results, result)
	}

	return response
}

func validateClusterName(clusterName, ns string) (*crv1.Pgcluster, error) {
	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient,
		&cluster, clusterName, ns)
	if !found {
		return &cluster, errors.New("no cluster found named " + clusterName)
	}

	return &cluster, err
}

// isValidFailoverTarget checks to see if the failover target specified in the request is valid,
// i.e. that it represents a valid replica deployment in the cluster specified.  This is
// done by first ensuring the deployment specified exists and is associated with the cluster
// specified, and then ensuring the PG pod created by the deployment is not the current primary.
// If the deployment is not found, or if the pod is the current primary, an error will be returned.
// Otherwise the deployment is returned.
func isValidFailoverTarget(deployName, clusterName, ns string) (*v1.Deployment, error) {

	// Using the following label selector, ensure the deployment specified using deployName exists in the
	// cluster specified using clusterName:
	// pg-cluster=clusterName,deployment-name=deployName
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_DEPLOYMENT_NAME + "=" + deployName
	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		log.Error(err)
		return nil, err
	} else if len(deployments.Items) == 0 {
		return nil, errors.New("no target found named " + deployName)
	} else if len(deployments.Items) > 1 {
		return nil, errors.New("more than one target found named " + deployName)
	}

	// Using the following label selector, determine if the target specified is the current
	// primary for the cluster and return an error if it is:
	// pg-cluster=clusterName,deployment-name=deployName,role=master
	selector = config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_DEPLOYMENT_NAME + "=" + deployName +
		"," + config.LABEL_PGHA_ROLE + "=master"
	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if len(pods.Items) > 0 {
		return nil, errors.New("The primary database cannot be selected as a failover target")
	}

	return &deployments.Items[0], nil

}

func preferredNode(nodes []string, targetNode string) bool {
	for _, n := range nodes {
		if n == targetNode {
			return true
		}
	}
	return false
}
