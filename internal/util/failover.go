package util

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// InstanceReplicationInfo is the user friendly information for the current
// status of key replication metrics for a PostgreSQL instance
type InstanceReplicationInfo struct {
	Name           string
	Node           string
	ReplicationLag int
	Status         string
	Timeline       int
	PendingRestart bool
	PodName        string
	Role           string
}

type ReplicationStatusRequest struct {
	RESTConfig  *rest.Config
	Clientset   kubernetes.Interface
	Namespace   string
	ClusterName string
}

type ReplicationStatusResponse struct {
	Instances []InstanceReplicationInfo
}

// instanceReplicationInfoJSON is the information returned from the request to
// the Patroni REST endpoint for info on the replication status of all the
// replicas
type instanceReplicationInfoJSON struct {
	PodName        string `json:"Member"`
	Type           string `json:"Role"`
	ReplicationLag int    `json:"Lag in MB"`
	State          string
	Timeline       int    `json:"TL"`
	PendingRestart string `json:"Pending restart"`
}

// instanceInfo stores the name and node of a specific instance (primary or replica) within a
// PG cluster
type instanceInfo struct {
	name string
	node string
}

const (
	// instanceReplicationInfoTypePrimary is the label used by Patroni to indicate that an instance
	// is indeed a primary PostgreSQL instance
	instanceReplicationInfoTypePrimary = "Leader"
	// instanceReplicationInfoTypePrimaryStandby is the label used by Patroni to indicate that an
	// instance is indeed a primary PostgreSQL instance, specifically within a standby cluster
	instanceReplicationInfoTypePrimaryStandby = "Standby Leader"
	// InstanceRolePrimary indicates that an instance is a primary
	InstanceRolePrimary = "primary"
	// InstanceRoleReplica indicates that an instance is a replica
	InstanceRoleReplica = "replica"
	// instanceRoleUnknown indicates that an instance is of an unknown typ
	instanceRoleUnknown = "unknown"
	// instanceStatusUnavailable indicates an instance is unavailable
	instanceStatusUnavailable = "unavailable"
)

// instanceInfoCommand is the command used to get information about the status
// and other statistics about the instances in a PostgreSQL cluster, e.g.
// replication lag
var instanceInfoCommand = []string{"patronictl", "list", "-f", "json"}

// ReplicationStatus is responsible for retrieving and returning the replication
// information about the status of the replicas in a PostgreSQL cluster. It
// executes into a single replica pod and leverages the functionality of Patroni
// for getting the key metrics that are appropriate to help the user understand
// the current state of their replicas.
//
// Statistics include: the current node the replica is on, if it is up, the
// replication lag, etc.
//
// By default information is only returned for replicas within the cluster.  However,
// if primary information is also needed, the inlcudePrimary flag can set set to true
// and primary information will will also be included in the ReplicationStatusResponse.
//
// Also by default we do not include any "busted" Pods, e.g. a Pod that is not
// in a happy phase. That Pod may be lacking a "role" label. From there, we zero
// out the statistics and apply an error
func ReplicationStatus(request ReplicationStatusRequest, includePrimary, includeBusted bool) (ReplicationStatusResponse, error) {
	ctx := context.TODO()
	response := ReplicationStatusResponse{
		Instances: make([]InstanceReplicationInfo, 0),
	}

	// Build up the selector. First, create the base, which restricts to the
	// current cluster
	// pg-cluster=clusterName,pgo-pg-database
	selector := fmt.Sprintf("%s=%s,%s",
		config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_PG_DATABASE)

	// if we are not including the primary, determine if we are including busted
	// replicas or not
	if !includePrimary {
		if includeBusted {
			// include all Pods that identify as a database, but **not** a primary
			// pg-cluster=clusterName,pgo-pg-database,role!=config.LABEL_PGHA_ROLE_PRIMARY
			selector += fmt.Sprintf(",%s!=%s", config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)
		} else {
			// include all Pods that identify as a database and have a replica label
			// pg-cluster=clusterName,pgo-pg-database,role=replica
			selector += fmt.Sprintf(",%s=%s", config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_REPLICA)
		}
	}

	log.Debugf(`searching for pods with "%s"`, selector)
	pods, err := request.Clientset.CoreV1().Pods(request.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	// If there is an error trying to get the pods, return here. Allow the caller
	// to handle the error
	if err != nil {
		return response, err
	}

	// See how many replica instances were found. If none were found then return
	log.Debugf(`replica pods found "%d"`, len(pods.Items))

	if len(pods.Items) == 0 {
		return response, err
	}

	// We need to create a quick map of "pod name" => node name / instance name
	// We will iterate through the pod list once to extract the name we refer to
	// the specific instance as, as well as which node it is deployed on
	instanceInfoMap := createInstanceInfoMap(pods)

	// Now get the statistics about the current state of the replicas, which we
	// can delegate to Patroni vis-a-vis the information that it collects
	// We can get the statistics about the current state of the managed instance
	// From executing and running a command in the first active pod
	var pod *v1.Pod

	for i := range pods.Items {
		if pods.Items[i].Status.Phase == v1.PodRunning {
			pod = &pods.Items[i]
			break
		}
	}

	// if no active Pod can be found, we can only assume that all of the instances
	// are unavailable, and we should indicate as such
	if pod == nil {
		for _, p := range pods.Items {
			// set up the instance that will be returned
			instance := InstanceReplicationInfo{
				Name:           instanceInfoMap[p.Name].name,
				Node:           instanceInfoMap[p.Name].node,
				ReplicationLag: -1,
				Role:           instanceRoleUnknown,
				Status:         instanceStatusUnavailable,
				Timeline:       -1,
			}

			// append this newly created instance to the list that will be returned
			response.Instances = append(response.Instances, instance)
		}

		return response, nil
	}

	// Execute the command that will retrieve the replica information from Patroni
	commandStdOut, _, err := kubeapi.ExecToPodThroughAPI(
		request.RESTConfig, request.Clientset, instanceInfoCommand,
		pod.Spec.Containers[0].Name, pod.Name, request.Namespace, nil)
	// if there is an error, return. We will log the error at a higher level
	if err != nil {
		return response, err
	}

	// parse the JSON and plast it into instanceInfoList
	var rawInstances []instanceReplicationInfoJSON
	_ = json.Unmarshal([]byte(commandStdOut), &rawInstances)

	log.Debugf("patroni instance info: %v", rawInstances)

	// We need to iterate through this list to format the information for the
	// response
	for _, rawInstance := range rawInstances {
		var role string

		// skip the primary unless explicitly enabled
		if !includePrimary && (rawInstance.Type == instanceReplicationInfoTypePrimary ||
			rawInstance.Type == instanceReplicationInfoTypePrimaryStandby) {
			continue
		}

		// if this is a busted instance and we are not including it, skip
		if !includeBusted && rawInstance.State == "" {
			continue
		}

		// determine the role of the instnace
		switch rawInstance.Type {
		default:
			role = InstanceRoleReplica
		case instanceReplicationInfoTypePrimary, instanceReplicationInfoTypePrimaryStandby:
			role = InstanceRolePrimary
		}

		// set up the instance that will be returned
		instance := InstanceReplicationInfo{
			ReplicationLag: rawInstance.ReplicationLag,
			Status:         rawInstance.State,
			Timeline:       rawInstance.Timeline,
			Role:           role,
			Name:           instanceInfoMap[rawInstance.PodName].name,
			Node:           instanceInfoMap[rawInstance.PodName].node,
			PendingRestart: rawInstance.PendingRestart == "*",
			PodName:        rawInstance.PodName,
		}

		// update the instance info if the instance is busted
		if rawInstance.State == "" {
			instance.Status = instanceStatusUnavailable
			instance.ReplicationLag = -1
			instance.Timeline = -1
		}

		// append this newly created instance to the list that will be returned
		response.Instances = append(response.Instances, instance)
	}

	// pass along the response for the requestor to process
	return response, nil
}

// ToggleAutoFailover enables or disables autofailover for a cluster.  Disabling autofailover means "pausing"
// Patroni, which will result in Patroni stepping aside from managing the cluster.  This will effectively cause
// Patroni to stop responding to failures or other database activities, e.g. it will not attempt to start the
// database when stopped to perform maintenance
func ToggleAutoFailover(clientset kubernetes.Interface, enable bool, pghaScope, namespace string) error {
	ctx := context.TODO()

	// find the "config" configMap created by Patroni
	configMapName := pghaScope + "-config"
	log.Debugf("setting autofailover to %t for cluster with pgha scope %s", enable, pghaScope)

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return err
	}

	// return ErrMissingConfigAnnotation error if configMap is missing the "config" annotation.
	// This allows for graceful handling of scenarios where a failover toggle is attempted
	// (e.g. during cluster removal), but this annotation has not been created yet (e.g. due to
	// a failed cluster bootstrap)
	if _, ok := configMap.ObjectMeta.Annotations["config"]; !ok {
		return ErrMissingConfigAnnotation
	}

	configJSONStr := configMap.ObjectMeta.Annotations["config"]

	var configJSON map[string]interface{}
	_ = json.Unmarshal([]byte(configJSONStr), &configJSON)

	if !enable {
		// disable autofail condition
		_ = disableFailover(clientset, configMap, configJSON, namespace)
	} else {
		// enable autofail
		_ = enableFailover(clientset, configMap, configJSON, namespace)
	}

	return nil
}

// createInstanceInfoMap creates a mapping between the pod names for the PostgreSQL
// pods in a cluster to the a struct containing the associated instance name and the
// Nodes that it runs on, all based upon the output from a Kubernetes API query
func createInstanceInfoMap(pods *v1.PodList) map[string]instanceInfo {
	instanceInfoMap := make(map[string]instanceInfo)

	// Iterate through each pod that is returned and get the mapping between the
	// pod and the PostgreSQL instance name with node it is scheduled on
	for _, pod := range pods.Items {
		instanceInfoMap[pod.GetName()] = instanceInfo{
			name: pod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME],
			node: pod.Spec.NodeName,
		}
	}

	log.Debugf("instanceInfoMap: %v", instanceInfoMap)

	return instanceInfoMap
}

// If "pause" is present in the config and set to "true", then it needs to be removed to enable
// failover.  Otherwise, if "pause" isn't present in the config or if it has a value other than
// true, then assume autofail is enabled and do nothing (when Patroni see's an invalid value for
// "pause" it sets it to "true")
func enableFailover(clientset kubernetes.Interface, configMap *v1.ConfigMap, configJSON map[string]interface{},
	namespace string) error {
	ctx := context.TODO()
	if _, ok := configJSON["pause"]; ok && configJSON["pause"] == true {
		log.Debugf("updating pause key in configMap %s to enable autofailover", configMap.Name)
		//  disabled autofail by removing "pause" from the config
		delete(configJSON, "pause")
		configJSONFinalStr, err := json.Marshal(configJSON)
		if err != nil {
			return err
		}
		configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
		_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	} else {
		log.Debugf("autofailover already enabled according to the pause key (or lack thereof) in configMap %s",
			configMap.Name)
	}
	return nil
}

// If "pause" isn't present in the config then assume autofail is enabled and needs to be disabled
// by setting "pause" to true.  Or if it is present and set to something other than "true" (e.g.
// "false" or "null"), then it also needs to be disabled by setting "pause" to true.
func disableFailover(clientset kubernetes.Interface, configMap *v1.ConfigMap, configJSON map[string]interface{},
	namespace string) error {
	ctx := context.TODO()
	if _, ok := configJSON["pause"]; !ok || configJSON["pause"] != true {
		log.Debugf("updating pause key in configMap %s to disable autofailover", configMap.Name)
		// disable autofail by setting "pause" to true
		configJSON["pause"] = true
		configJSONFinalStr, err := json.Marshal(configJSON)
		if err != nil {
			return err
		}
		configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
		_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	} else {
		log.Debugf("autofailover already disabled according to the pause key in configMap %s",
			configMap.Name)
	}
	return nil
}
