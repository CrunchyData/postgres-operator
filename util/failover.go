package util

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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
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
}

type ReplicationStatusRequest struct {
	RESTConfig  *rest.Config
	Clientset   *kubernetes.Clientset
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
	Timeline       int `json:"TL"`
}

const (
	// instanceReplicationInfoTypePrimary is the label used by Patroni to indicate that an instance
	// is indeed a primary PostgreSQL instance
	instanceReplicationInfoTypePrimary = "Leader"
	// pgPodNamePattern pattern is a pattern used by regexp to look up the
	// name of the pod
	pgPodNamePattern = "%s-[0-9a-z]{10}-[0-9a-z]{5}"
)

var (
	// instanceInfoCommand is the command used to get information about the status
	// and other statistics about the instances in a PostgreSQL cluster, e.g.
	// replication lag
	instanceInfoCommand = []string{"patronictl", "list", "-f", "json"}
)

// GetPod determines the best target to fail to
func GetPod(clientset *kubernetes.Clientset, deploymentName, namespace string) (*v1.Pod, error) {

	var err error

	var pod *v1.Pod
	var pods *v1.PodList

	selector := config.LABEL_DEPLOYMENT_NAME + "=" + deploymentName + "," + config.LABEL_PGHA_ROLE + "=replica"
	pods, err = kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		return pod, err
	}
	if len(pods.Items) != 1 {
		return pod, errors.New("could not determine which pod to failover to")
	}

	for _, v := range pods.Items {
		pod = &v
	}

	found := false

	//make sure the pod has a database container it it
	for _, c := range pod.Spec.Containers {
		if c.Name == "database" {
			found = true
		}
	}

	if !found {
		return pod, errors.New("could not find a database container in the pod")
	}

	return pod, err
}

// ReplicationStatus is responsible for retrieving and returning the replication
// information about the status of the replicas in a PostgreSQL cluster. It
// executes into a single replica pod and leverages the functionality of Patroni
// for getting the key metrics that are appropriate to help the user understand
// the current state of their replicas.
//
// Statistics include: the current node the replica is on, if it is up, the
// replication lag, etc.
func ReplicationStatus(request ReplicationStatusRequest) (ReplicationStatusResponse, error) {
	response := ReplicationStatusResponse{
		Instances: make([]InstanceReplicationInfo, 0),
	}

	// First, get replica pods using selector pg-cluster=clusterName-replica,role=replica
	selector := fmt.Sprintf("%s=%s,%s=replica",
		config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_PGHA_ROLE)

	log.Debugf(`searching for pods with "%s"`, selector)
	pods, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace)

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

	// We need to create a quick map of "instance name" => node name
	// We will iterate through the pod list once to extract the name we refer to
	// the specific instance as, as well as which node it is deployed on
	instanceNodeMap := createInstanceNodeMap(pods)

	// Now get the statistics about the current state of the replicas, which we
	// can delegate to Patroni vis-a-vis the information that it collects
	// We can get the statistics about the current state of the managed instance
	// From executing and running a command in the first pod
	pod := pods.Items[0]

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
	json.Unmarshal([]byte(commandStdOut), &rawInstances)

	log.Debugf("patroni instance info: %v", rawInstances)

	// We need to iterate through this list to format the information for the
	// response
	for _, rawInstance := range rawInstances {
		// if this is a primary, skip it
		if rawInstance.Type == instanceReplicationInfoTypePrimary {
			continue
		}

		// set up the instance that will be returned
		instance := InstanceReplicationInfo{
			ReplicationLag: rawInstance.ReplicationLag,
			Status:         rawInstance.State,
			Timeline:       rawInstance.Timeline,
		}

		// get the instance name that is recognized by the Operator, which is the
		// first part of the name and is kept on a deployment label. We have these
		// available in our instanceNodeMap, and because we skip over the primary,
		// this will not lead to false positive
		//
		// This is not the cleanest way of doing it, but it works
		for name, node := range instanceNodeMap {
			r, err := regexp.Compile(fmt.Sprintf(pgPodNamePattern, name))

			// if there is an error compiling the regular expression, add an error to
			// log log and keep iterating
			if err != nil {
				log.Error(err)
				continue
			}

			// see if there is a match in the names. If it , add the name and node for
			// this particular instance
			if r.Match([]byte(rawInstance.PodName)) {
				instance.Name = name
				instance.Node = node
				break
			}
		}

		// append this newly created instance to the list that will be returned
		response.Instances = append(response.Instances, instance)
	}

	// pass along the response for the requestor to process
	return response, nil
}

func GetPreferredNodes(clientset *kubernetes.Clientset, selector, namespace string) ([]string, error) {
	nodes := make([]string, 0)

	nodeList, err := kubeapi.GetNodes(clientset, selector, namespace)
	if err != nil {
		return nodes, err
	}

	log.Debugf("getPreferredNodes shows %d nodes", len(nodeList.Items))

	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}

	return nodes, err
}

// ToggleAutoFailover enables or disables autofailover for a cluster.  Disabling autofailover means "pausing"
// Patroni, which will result in Patroni stepping aside from managing the cluster.  This will effectively cause
// Patroni to stop responding to failures or other database activities, e.g. it will not attempt to start the
// database when stopped to perform maintenance
func ToggleAutoFailover(clientset *kubernetes.Clientset, enable bool, pghaScope, namespace string) error {

	// find the "config" configMap created by Patroni
	configMapName := pghaScope + "-config"
	log.Debugf("setting autofailover to %t for cluster with pgha scope %s", enable, pghaScope)

	configMap, found := kubeapi.GetConfigMap(clientset, configMapName, namespace)
	if !found {
		err := fmt.Errorf("Unable to find configMap %s when attempting disable autofailover", configMapName)
		log.Error(err)
		return err
	}

	configJSONStr := configMap.ObjectMeta.Annotations["config"]

	var configJSON map[string]interface{}
	json.Unmarshal([]byte(configJSONStr), &configJSON)

	if !enable {
		// disable autofail condition
		disableFailover(clientset, configMap, configJSON, namespace)
	} else {
		// enable autofail
		enableFailover(clientset, configMap, configJSON, namespace)
	}

	return nil
}

// createInstanceNodeMap creates a mapping between the names of the PostgreSQL
// instances to the Nodes that they run on, based upon the output from a
// Kubernetes API query
func createInstanceNodeMap(pods *v1.PodList) map[string]string {
	instanceNodeMap := map[string]string{}

	// Iterate through each pod that is returned and get the mapping between the
	// PostgreSQL instance name and the node it is scheduled on
	for _, pod := range pods.Items {
		// get the replica name from the metadata on the pod
		// for legacy purposes, we are using the "deployment name" label
		replicaName := pod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]
		// get the node name from the spec on the pod
		nodeName := pod.Spec.NodeName
		// add them to the map
		instanceNodeMap[replicaName] = nodeName
	}

	log.Debugf("instance/node map: %v", instanceNodeMap)

	return instanceNodeMap
}

// If "pause" is present in the config and set to "true", then it needs to be removed to enable
// failover.  Otherwise, if "pause" isn't present in the config or if it has a value other than
// true, then assume autofail is enabled and do nothing (when Patroni see's an invalid value for
// "pause" it sets it to "true")
func enableFailover(clientset *kubernetes.Clientset, configMap *v1.ConfigMap, configJSON map[string]interface{},
	namespace string) error {
	if _, ok := configJSON["pause"]; ok && configJSON["pause"] == true {
		log.Debugf("updating pause key in configMap %s to enable autofailover", configMap.Name)
		//  disabled autofail by removing "pause" from the config
		delete(configJSON, "pause")
		configJSONFinalStr, err := json.Marshal(configJSON)
		if err != nil {
			return err
		}
		configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
		err = kubeapi.UpdateConfigMap(clientset, configMap, namespace)
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
func disableFailover(clientset *kubernetes.Clientset, configMap *v1.ConfigMap, configJSON map[string]interface{},
	namespace string) error {
	if _, ok := configJSON["pause"]; !ok || configJSON["pause"] != true {
		log.Debugf("updating pause key in configMap %s to disable autofailover", configMap.Name)
		// disable autofail by setting "pause" to true
		configJSON["pause"] = true
		configJSONFinalStr, err := json.Marshal(configJSON)
		if err != nil {
			return err
		}
		configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
		err = kubeapi.UpdateConfigMap(clientset, configMap, namespace)
		if err != nil {
			return err
		}
	} else {
		log.Debugf("autofailover already disabled according to the pause key in configMap %s",
			configMap.Name)
	}
	return nil
}
