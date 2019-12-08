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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// instance info is a struct used to store the results of a query to find out
// information about an instance in a cluster
type instanceInfo struct {
	Name           string `json:"Member"`
	Type           string `json:"Role"`
	ReplicationLag int    `json:"Lag in MB"`
	State          string
	Timeline       int `json:"TL"`
}

// instanceInfoPrimary is the label used by Patroni to indicate that an instance
// is indeed a primary PostgreSQL instance
const instanceInfoPrimary = "Leader"

var (
	// instanceNamePattern is a regular expression usd to get the cluster instance
	instanceNamePattern = regexp.MustCompile("^([a-zA-Z0-9]+(-[a-zA-Z0-9]{4})?)-")
	// instanceInfoCommand is the command used to get information about the status
	// and other statistics about the instances in a PostgreSQL cluster, e.g.
	// replication lag
	instanceInfoCommand = []string{"patronictl", "list", "-f", "json"}
)

// ScaleCluster ...
func ScaleCluster(name, replicaCount, resourcesConfig, storageConfig, nodeLabel, ccpImageTag, serviceType, ns, pgouser string) msgs.ClusterScaleResponse {
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

	labels[config.LABEL_PGOUSER] = pgouser
	labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]

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

// ScaleQuery lists the replicas that are in the PostgreSQL cluster
// with information that is helpful in determining which one to fail over to,
// such as the lag behind the replica as well as the timeline
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

	//get replica pods using selector pg-cluster=clusterName-replica,role=replica
	selector := config.LABEL_PG_CLUSTER + "=" + name + "," + config.LABEL_PGHA_ROLE + "=replica"
	log.Debugf(`searching for pods with "%s"`, selector)

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)

	// If there is an error trying to get the pods, return here
	if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Begin to prepare returning the results
	response.Results = make([]msgs.ScaleQueryTargetSpec, 0)

	log.Debugf(`pods found "%d"`, len(pods.Items))

	// if no replicas are found, then return here
	if len(pods.Items) == 0 {
		return response
	}

	// First, we need to create a quick map of "instance name" => node name
	// We will iterate through the pod list once to extract the name we refere to
	// the specific instance as, as well as which node it is deployed on
	instanceNodeMap := createInstanceNodeMap(pods)

	// Now get the statistics about the current state of the replicas, which we
	// can delegate to Patroni vis-a-vis the information that it collects
	instanceInfoList, err := createInstanceInfoList(ns, pods)

	// if there is an error, record it here and return
	if err != nil {
		log.Error(err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// iterate through the results of the raw data, pick out any replicas,
	// and add them to the array
	for _, instance := range instanceInfoList {
		// if this is a primary, skip it
		if instance.Type == instanceInfoPrimary {
			continue
		}

		// create an result for the response
		result := msgs.ScaleQueryTargetSpec{
			Status:         instance.State,
			ReplicationLag: instance.ReplicationLag,
			Timeline:       instance.Timeline,
		}

		// get the instance name that is recognize by the Operator, which is the
		// first part of the name
		result.Name = instanceNamePattern.FindStringSubmatch(instance.Name)[1]

		// get the node that the replica is on based on the "replica name" for this
		// instance
		result.Node = instanceNodeMap[result.Name]

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

	//if this was the last replica then remove the replica service
	var replicaList *apps_v1.DeploymentList
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

	//create the rmdata task which does the cleanup

	deleteBackups := false
	isReplica := true
	isBackup := false
	taskName := replicaName + "-rmdata"
	err = apiserver.CreateRMDataTask(clusterName, replicaName, taskName, deleteBackups, deleteData, isReplica, isBackup, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	response.Results = append(response.Results, "deleted Pgreplica "+replicaName)
	return response
}

// makeInstanceNodeMap creates an mapping between the names of the PostgreSQL
// instances to the Nodes that they run on, based upon the output from a
// Kubernetes API query
func createInstanceNodeMap(pods *v1.PodList) map[string]string {
	instanceNodeMap := map[string]string{}

	// Iterate through each pod that is return and get the mapping between the
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

	return instanceNodeMap
}

// createInstanceInfo execs into a single pod that is returned in the collection
// and looks up the information that Patroni gives about each instance in the
// PostgreSQL cluster. This is returned to us in a JSON-parseable string,
// which, if valid, we can process and create a list of "instanceInfo` structs
func createInstanceInfoList(namespace string, pods *v1.PodList) ([]instanceInfo, error) {
	instanceInfoList := []instanceInfo{}

	// We can get the statistics about the current state of the managed instance
	// From executing and running a command in the first pod
	pod := pods.Items[0]

	commandStdOut, _, err := kubeapi.ExecToPodThroughAPI(
		apiserver.RESTConfig, apiserver.Clientset, instanceInfoCommand,
		pod.Spec.Containers[0].Name, pod.Name, namespace, nil)

	// if there is an error, return. We will log the error at a higher level
	if err != nil {
		return instanceInfoList, err
	}

	// parse the JSON and plast it into instanceInfoList
	json.Unmarshal([]byte(commandStdOut), &instanceInfoList)

	// return the list here
	return instanceInfoList, nil
}
