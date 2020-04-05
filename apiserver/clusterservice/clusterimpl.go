package clusterservice

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"

	"io/ioutil"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	log "github.com/sirupsen/logrus"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// DeleteCluster ...
func DeleteCluster(name, selector string, deleteData, deleteBackups bool, ns, pgouser string) msgs.DeleteClusterResponse {
	var err error

	response := msgs.DeleteClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	if name != "all" {
		if selector == "" {
			selector = "name=" + name
		}
	}

	log.Debugf("delete-data is [%t]", deleteData)
	log.Debugf("delete-backups is [%t]", deleteBackups)

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if len(clusterList.Items) == 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = "no clusters found"
		return response
	}

	for _, cluster := range clusterList.Items {

		log.Debugf("deleting cluster %s", cluster.Spec.Name)
		taskName := cluster.Spec.Name + "-rmdata"
		log.Debugf("creating taskName %s", taskName)
		isBackup := false
		isReplica := false
		replicaName := ""
		clusterPGHAScope := cluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE]

		// first delete any existing rmdata pgtask with the same name
		if err := kubeapi.Deletepgtask(apiserver.RESTClient, taskName, ns); err != nil &&
			!kerrors.IsNotFound(err) {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		err := apiserver.CreateRMDataTask(cluster.Spec.Name, replicaName, taskName, deleteBackups, deleteData, isReplica, isBackup, ns, clusterPGHAScope)
		if err != nil {
			log.Debugf("error on creating rmdata task %s", err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		response.Results = append(response.Results, "deleted pgcluster "+cluster.Spec.Name)

	}

	return response

}

// ShowCluster ...
func ShowCluster(name, selector, ccpimagetag, ns string, allflag bool) msgs.ShowClusterResponse {
	var err error

	response := msgs.ShowClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]msgs.ShowClusterDetail, 0)

	if selector == "" && allflag {
		log.Debugf("allflags set to true")
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	log.Debugf("selector on showCluster is %s", selector)

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("clusters found len is %d", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c
		detail.Deployments, err = getDeployments(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Pods, err = GetPods(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Services, err = getServices(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Replicas, err = getReplicas(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		// capture whether or not the cluster is currently a standby cluster
		detail.Standby = c.Spec.Standby

		if ccpimagetag == "" {
			response.Results = append(response.Results, detail)
		} else if ccpimagetag == c.Spec.CCPImageTag {
			response.Results = append(response.Results, detail)
		}
	}

	return response

}

func getDeployments(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterDeployment, error) {
	output := make([]msgs.ShowClusterDeployment, 0)

	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}

	for _, dep := range deployments.Items {
		d := msgs.ShowClusterDeployment{}
		d.Name = dep.Name
		d.PolicyLabels = make([]string, 0)

		for k, v := range cluster.ObjectMeta.Labels {
			if v == "pgpolicy" {
				d.PolicyLabels = append(d.PolicyLabels, k)
			}
		}
		output = append(output, d)

	}

	return output, err
}

func GetPods(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)

	//get pods, but exclude backup pods and backrest repo
	selector := config.LABEL_BACKREST_JOB + "!=true," + config.LABEL_BACKREST_RESTORE + "!=true," + config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
	log.Debugf("selector for GetPods is %s", selector)

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)
		d.PVCName = apiserver.GetPVCName(&p)

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}

	return output, err

}

func getServices(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterService, error) {

	output := make([]msgs.ShowClusterService, 0)
	selector := config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	services, err := kubeapi.GetServices(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}

	log.Debugf("got %d services for %s", len(services.Items), cluster.Spec.Name)
	for _, p := range services.Items {
		d := msgs.ShowClusterService{}
		d.Name = p.Name
		if strings.Contains(p.Name, "-backrest-repo") {
			d.BackrestRepo = true
			d.ClusterName = cluster.Name
		} else if strings.Contains(p.Name, "-pgbouncer") {
			d.Pgbouncer = true
			d.ClusterName = cluster.Name
		}
		d.ClusterIP = p.Spec.ClusterIP
		if len(p.Spec.ExternalIPs) > 0 {
			d.ExternalIP = p.Spec.ExternalIPs[0]
		}
		if len(p.Status.LoadBalancer.Ingress) > 0 {
			d.ExternalIP = p.Status.LoadBalancer.Ingress[0].IP
		}

		output = append(output, d)

	}

	return output, err
}

// TestCluster performs a variety of readiness checks against one or more
// clusters within a namespace. It leverages the following two Kubernetes
// constructs in order to determine the availability of PostgreSQL clusters:
//	- Pod readiness checks. The Pod readiness checks leverage "pg_isready" to
//	determine if the PostgreSQL cluster is able to accept connecions
//	- Endpoint checks. The check sees if the services in front of the the
//	PostgreSQL instances are able to route connections from the "outside" into
//	the instances
func TestCluster(name, selector, ns, pgouser string, allFlag bool) msgs.ClusterTestResponse {
	var err error

	log.Debugf("TestCluster(%s,%s,%s,%s,%s): Called",
		name, selector, ns, pgouser, allFlag)

	response := msgs.ClusterTestResponse{}
	response.Results = make([]msgs.ClusterTestResult, 0)
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	log.Debugf("selector is: %s", selector)

	// if the select is empty, determine if its because the flag for
	// "all clusters" in a namespace is set
	//
	// otherwise, a name cluster name must be passed in, and said name should
	// be used
	if selector == "" {
		if allFlag {
			log.Debug("selector is : all clusters in %s", ns)
		} else {
			selector = "name=" + name
			log.Debugf("selector is: %s", selector)
		}
	}

	// Find a list of a clusters that match the given selector
	clusterList := crv1.PgclusterList{}
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector, ns)

	// If the response errors, return here, as we won't be able to return any
	// useful information in the test
	if err != nil {
		log.Errorf("Cluster lookup failed: %s", err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("Total clusters found: %d", len(clusterList.Items))

	// Iterate through each cluster and perform the various tests against them
	for _, c := range clusterList.Items {
		// Set up the object that will be appended to the response that
		// indicates the availability of the endpoints / instances for this
		// cluster
		result := msgs.ClusterTestResult{
			ClusterName: c.Name,
			Endpoints:   make([]msgs.ClusterTestDetail, 0),
			Instances:   make([]msgs.ClusterTestDetail, 0),
		}

		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c

		// Get the PostgreSQL instances!
		log.Debugf("Looking up instance pods for cluster: %s", c.Name)
		pods, err := GetPrimaryAndReplicaPods(&c, ns)

		// if there is an error with returning the primary/replica instances,
		// then error and continue
		if err != nil {
			log.Errorf("Instance pod lookup failed: %s", err.Error())
			instance := msgs.ClusterTestDetail{
				Available:    false,
				InstanceType: msgs.ClusterTestInstanceTypePrimary,
			}
			result.Instances = append(result.Instances, instance)
			response.Results = append(response.Results, result)
			continue
		}

		log.Debugf("pods found %d", len(pods))

		// if there are no pods found, then the cluster is not ready at all, and
		// we can make an early on checking the availability of this cluster
		if len(pods) == 0 {
			log.Infof("Cluster has no instances available: %s", c.Name)
			instance := msgs.ClusterTestDetail{
				Available:    false,
				InstanceType: msgs.ClusterTestInstanceTypePrimary,
			}
			result.Instances = append(result.Instances, instance)
			response.Results = append(response.Results, result)
			continue
		}

		// Check each instance (i.e. pod) to see if its readiness check passes.
		//
		// (We are assuming that the readiness check is performing the
		// equivalent to a "pg_isready" which denotes that a PostgreSQL instance
		// is connectable. If you have any doubts about this, check the
		// readiness check code)
		//
		// Also denotes the type of PostgreSQL instance this is. All of the pods
		// returned are either primaries or replicas
		for _, pod := range pods {
			// set up the object with the instance status
			instance := msgs.ClusterTestDetail{
				Available: pod.Ready,
				Message:   pod.Name,
			}
			switch pod.Type {
			default:
				instance.InstanceType = msgs.ClusterTestInstanceTypeUnknown
			case msgs.PodTypePrimary:
				instance.InstanceType = msgs.ClusterTestInstanceTypePrimary
			case msgs.PodTypeReplica:
				instance.InstanceType = msgs.ClusterTestInstanceTypeReplica
			}
			log.Debugf("Instance found with attributes: (%s, %s, %s)",
				instance.InstanceType, instance.Message, instance.Available)
			// Add the report on the pods to this set
			result.Instances = append(result.Instances, instance)
		}

		// Time to check the endpoints. We will check the available endpoints
		// vis-a-vis the services
		detail.Services, err = getServices(&c, ns)

		// if the services are unavailable, report an error and continue
		// iterating
		if err != nil {
			log.Errorf("Service lookup failed: %s", err.Error())
			endpoint := msgs.ClusterTestDetail{
				Available:    false,
				InstanceType: msgs.ClusterTestInstanceTypePrimary,
			}
			result.Endpoints = append(result.Endpoints, endpoint)
			response.Results = append(response.Results, result)
			continue
		}

		// Iterate through the services and determine if they are reachable via
		// their endpionts
		for _, service := range detail.Services {
			//  prepare the endpoint request
			endpointRequest := &kubeapi.GetEndpointRequest{
				Clientset: apiserver.Clientset, // current clientset
				Name:      service.Name,        // name of the service, used to find the endpoint
				Namespace: ns,                  // namespace the service / endpoint resides in
			}
			// prepare the end result, add the endpoint connection information
			endpoint := msgs.ClusterTestDetail{
				Message: fmt.Sprintf("%s:%s", service.ClusterIP, c.Spec.Port),
			}

			// determine the type of endpoint that is being checked based on
			// the information available in the service
			switch {
			default:
				endpoint.InstanceType = msgs.ClusterTestInstanceTypePrimary
			case strings.Contains(service.Name, msgs.PodTypeReplica):
				endpoint.InstanceType = msgs.ClusterTestInstanceTypeReplica
			case service.Pgbouncer:
				endpoint.InstanceType = msgs.ClusterTestInstanceTypePGBouncer
			case service.BackrestRepo:
				endpoint.InstanceType = msgs.ClusterTestInstanceTypeBackups
			}

			// make a call to the Kubernetes API to see if the endpoint exists
			// if there is an error, indicate that this endpoint is inaccessible
			// otherwise inspect the endpoint response to see if the Pods that
			// comprise the Service are in the "NotReadyAddresses"
			endpoint.Available = true
			if endpointResponse, err := kubeapi.GetEndpoint(endpointRequest); err != nil {
				endpoint.Available = false
			} else {
				for _, subset := range endpointResponse.Endpoint.Subsets {
					// if any of the addresses are not ready in the endpoint,
					// or there are no address ready, then the endpoint is not
					// ready
					if len(subset.NotReadyAddresses) > 0 && len(subset.Addresses) == 0 {
						endpoint.Available = false
					}
				}
			}

			log.Debugf("Endpoint found with attributes: (%s, %s, %s)",
				endpoint.InstanceType, endpoint.Message, endpoint.Available)

			// append the endpoint to the list
			result.Endpoints = append(result.Endpoints, endpoint)

		}

		// concaentate to the results and continue
		response.Results = append(response.Results, result)
	}

	return response
}

// CreateCluster ...
// pgo create cluster mycluster
func CreateCluster(request *msgs.CreateClusterRequest, ns, pgouser string) msgs.CreateClusterResponse {
	var id string
	resp := msgs.CreateClusterResponse{
		Result: msgs.CreateClusterDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	clusterName := request.Name

	if clusterName == "all" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid cluster name 'all' is not allowed as a cluster name"
		return resp
	}

	if request.ReplicaCount < 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid replica-count , should be greater than or equal to 0"
		return resp
	}

	errs := validation.IsDNS1035Label(clusterName)
	if len(errs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid cluster name format " + errs[0]
		return resp
	}

	log.Debugf("create cluster called for %s", clusterName)
	result := crv1.Pgcluster{}

	// error if it already exists
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &result, clusterName, ns)
	if err == nil {
		log.Debugf("pgcluster %s was found so we will not create it", clusterName)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "pgcluster " + clusterName + " was found so we will not create it"
		return resp
	} else if !found {
		log.Debugf("pgcluster %s not found so we will create it", clusterName)
	} else {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "error getting pgcluster " + clusterName + err.Error()
		return resp
	}

	userLabelsMap := make(map[string]string)
	if request.UserLabels != "" {
		labels := strings.Split(request.UserLabels, ",")
		for _, v := range labels {
			p := strings.Split(v, "=")
			if len(p) < 2 {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "invalid labels format"
				return resp
			}
			userLabelsMap[p[0]] = p[1]
		}
	}

	// if any of the the PVCSizes are set to a customized value, ensure that they
	// are recognizable by Kubernetes
	// first, the primary/replica PVC size
	if request.PVCSize != "" {
		if err := apiserver.ValidateQuantity(request.PVCSize); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessagePVCSize, request.PVCSize, err.Error())
			return resp
		}
	}

	// next, the pgBackRest repo PVC size
	if request.BackrestPVCSize != "" {
		if err := apiserver.ValidateQuantity(request.BackrestPVCSize); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessagePVCSize, request.BackrestPVCSize, err.Error())
			return resp
		}
	}

	// evaluate if the CPU / Memory have been set to custom values
	if request.CPURequest != "" {
		if err := apiserver.ValidateQuantity(request.CPURequest); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessageCPURequest, request.CPURequest, err.Error())
			return resp
		}
	}

	if request.MemoryRequest != "" {
		if err := apiserver.ValidateQuantity(request.MemoryRequest); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessageMemoryRequest, request.MemoryRequest, err.Error())
			return resp
		}
	}

	// validate the storage type for each specified tablespace actually exists.
	// if a PVCSize is passed in, also validate that it follows the Kubernetes
	// format
	if err := validateTablespaces(request.Tablespaces); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// validate the TLS parameters for enabling TLS in a PostgreSQL cluster
	if err := validateClusterTLS(request); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if request.CustomConfig != "" {
		found, err := validateCustomConfig(request.CustomConfig, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.CustomConfig + " configmap was not found "
			return resp
		}
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		//add a label for the custom config
		userLabelsMap[config.LABEL_CUSTOM_CONFIG] = request.CustomConfig
	}

	//set the metrics flag with the global setting first
	userLabelsMap[config.LABEL_COLLECT] = strconv.FormatBool(apiserver.MetricsFlag)
	if err != nil {
		log.Error(err)
	}

	//if metrics is chosen on the pgo command, stick it into the user labels
	if request.MetricsFlag {
		userLabelsMap[config.LABEL_COLLECT] = "true"
	}
	if request.ServiceType != "" {
		if request.ServiceType != config.DEFAULT_SERVICE_TYPE && request.ServiceType != config.LOAD_BALANCER_SERVICE_TYPE && request.ServiceType != config.NODEPORT_SERVICE_TYPE {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "error ServiceType should be either ClusterIP or LoadBalancer "

			return resp
		}
		userLabelsMap[config.LABEL_SERVICE_TYPE] = request.ServiceType
	}

	// if the request is for a standby cluster then validate it to ensure all parameters have
	// been properly specifed as required to create a standby cluster
	if request.Standby {
		if err := validateStandbyCluster(request); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	// ensure the backrest storage type specified for the cluster is valid, and that the
	// configruation required to use that storage type (e.g. a bucket, endpoint and region
	// when using aws s3 storage) has been provided
	err = validateBackrestStorageTypeOnCreate(request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if request.BackrestStorageType != "" {
		log.Debug("using backrest storage type provided by user")
		userLabelsMap[config.LABEL_BACKREST_STORAGE_TYPE] = request.BackrestStorageType
	}

	// if a value for BackrestStorageConfig is provided, validate it here
	if request.BackrestStorageConfig != "" && !apiserver.IsValidStorageName(request.BackrestStorageConfig) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("\"%s\" storage config was not found", request.BackrestStorageConfig)
		return resp
	}

	log.Debug("userLabelsMap")
	log.Debugf("%v", userLabelsMap)

	if existsGlobalConfig(ns) {
		userLabelsMap[config.LABEL_CUSTOM_CONFIG] = config.GLOBAL_CUSTOM_CONFIGMAP
	}

	if request.StorageConfig != "" {
		if apiserver.IsValidStorageName(request.StorageConfig) == false {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.StorageConfig + " Storage config was not found "
			return resp
		}
	}

	if apiserver.Pgo.Cluster.PrimaryNodeLabel != "" {
		//already should be validate at apiserver startup
		parts := strings.Split(apiserver.Pgo.Cluster.PrimaryNodeLabel, "=")
		userLabelsMap[config.LABEL_NODE_LABEL_KEY] = parts[0]
		userLabelsMap[config.LABEL_NODE_LABEL_VALUE] = parts[1]
		log.Debug("primary node labels used from pgo.yaml")
	}

	// validate & parse nodeLabel if exists
	if request.NodeLabel != "" {
		if err = apiserver.ValidateNodeLabel(request.NodeLabel); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		parts := strings.Split(request.NodeLabel, "=")
		userLabelsMap[config.LABEL_NODE_LABEL_KEY] = parts[0]
		userLabelsMap[config.LABEL_NODE_LABEL_VALUE] = parts[1]

		log.Debug("primary node labels used from user entered flag")
	}

	if request.ReplicaStorageConfig != "" {
		if apiserver.IsValidStorageName(request.ReplicaStorageConfig) == false {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.ReplicaStorageConfig + " Storage config was not found "
			return resp
		}
	}

	if request.ContainerResources != "" {
		if apiserver.IsValidContainerResource(request.ContainerResources) == false {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.ContainerResources + " ContainerResource config was not found "
			return resp
		}
	}

	// if a value is provided in the request for PodAntiAffinity, then ensure is valid.  If
	// it is, then set the user label for pod anti-affinity to the request value
	// (which in turn becomes a *Label* which is important for anti-affinity).
	// Otherwise, return the validation error.
	if request.PodAntiAffinity != "" {
		podAntiAffinityType := crv1.PodAntiAffinityType(request.PodAntiAffinity)
		if err := podAntiAffinityType.Validate(); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		userLabelsMap[config.LABEL_POD_ANTI_AFFINITY] = request.PodAntiAffinity
	} else {
		userLabelsMap[config.LABEL_POD_ANTI_AFFINITY] = ""
	}

	// check to see if there are any pod anti-affinity overrides, specifically for
	// pgBackRest and pgBouncer. If there are, ensure that they are valid values
	if request.PodAntiAffinityPgBackRest != "" {
		podAntiAffinityType := crv1.PodAntiAffinityType(request.PodAntiAffinityPgBackRest)

		if err := podAntiAffinityType.Validate(); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	if request.PodAntiAffinityPgBouncer != "" {
		podAntiAffinityType := crv1.PodAntiAffinityType(request.PodAntiAffinityPgBouncer)

		if err := podAntiAffinityType.Validate(); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	// if synchronous replication has been enabled, then add to user labels
	if request.SyncReplication != nil {
		userLabelsMap[config.LABEL_SYNC_REPLICATION] =
			string(strconv.FormatBool(*request.SyncReplication))
	}

	// Create an instance of our CRD
	newInstance := getClusterParams(request, clusterName, userLabelsMap, ns)
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser

	if request.SecretFrom != "" {
		err = validateSecretFrom(request.SecretFrom, newInstance.Spec.User, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.SecretFrom + " secret was not found "
			return resp
		}
	}

	validateConfigPolicies(clusterName, request.Policies, ns)

	// create the user secrets
	// first, the superuser
	if secretName, password, err := createUserSecret(request, newInstance, crv1.RootSecretSuffix,
		crv1.PGUserSuperuser, request.PasswordSuperuser); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else {
		newInstance.Spec.RootSecretName = secretName

		// if the user requests to show system accounts, append it to the list
		if request.ShowSystemAccounts {
			user := msgs.CreateClusterDetailUser{
				Username: crv1.PGUserSuperuser,
				Password: password,
			}

			resp.Result.Users = append(resp.Result.Users, user)
		}
	}

	// next, the replication user
	if secretName, password, err := createUserSecret(request, newInstance, crv1.PrimarySecretSuffix,
		crv1.PGUserReplication, request.PasswordReplication); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else {
		newInstance.Spec.PrimarySecretName = secretName

		// if the user requests to show system accounts, append it to the list
		if request.ShowSystemAccounts {
			user := msgs.CreateClusterDetailUser{
				Username: crv1.PGUserReplication,
				Password: password,
			}

			resp.Result.Users = append(resp.Result.Users, user)
		}
	}

	// finally, the user from the request and/or default user
	userSecretSuffix := fmt.Sprintf("-%s%s", newInstance.Spec.User, crv1.UserSecretSuffix)
	if secretName, password, err := createUserSecret(request, newInstance, userSecretSuffix, newInstance.Spec.User,
		request.Password); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else {
		newInstance.Spec.UserSecretName = secretName

		user := msgs.CreateClusterDetailUser{
			Username: newInstance.Spec.User,
			Password: password,
		}

		resp.Result.Users = append(resp.Result.Users, user)
	}

	// there's a secret for the monitoring user too
	newInstance.Spec.CollectSecretName = clusterName + crv1.CollectSecretSuffix

	// Create Backrest secret for S3/SSH Keys:
	// We make this regardless if backrest is enabled or not because
	// the deployment template always tries to mount /sshd volume
	secretName := fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET)
	_, _, err = kubeapi.GetSecret(apiserver.Clientset, secretName, request.Namespace)
	if kerrors.IsNotFound(err) {
		// determine if a custom CA secret should be used
		backrestS3CACert := []byte{}

		if request.BackrestS3CASecretName != "" {
			backrestSecret, _, err := kubeapi.GetSecret(apiserver.Clientset, request.BackrestS3CASecretName, request.Namespace)

			if err != nil {
				log.Error(err)
				resp.Status.Code = msgs.Error
				resp.Status.Msg = fmt.Sprintf("Error finding pgBackRest S3 CA secret \"%s\": %s",
					request.BackrestS3CASecretName, err.Error())
				return resp
			}

			// attempt to retrieves the custom CA, assuming it has the name
			// "aws-s3-ca.crt"
			backrestS3CACert = backrestSecret.Data[util.BackRestRepoSecretKeyAWSS3KeyAWSS3CACert]
		}

		err := util.CreateBackrestRepoSecrets(apiserver.Clientset,
			util.BackrestRepoConfig{
				BackrestS3CA:        backrestS3CACert,
				BackrestS3Key:       request.BackrestS3Key,
				BackrestS3KeySecret: request.BackrestS3KeySecret,
				ClusterName:         clusterName,
				ClusterNamespace:    request.Namespace,
				OperatorNamespace:   apiserver.PgoNamespace,
			})

		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf("could not create backrest repo secret: %s", err)
			return resp
		}
	} else if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("could not query if backrest repo secret exits: %s", err)
		return resp
	}

	//create a workflow for this new cluster
	id, err = createWorkflowTask(clusterName, ns, pgouser)
	if err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// assign the workflow information to rhe result, as well as the use labels
	// for the CRD
	resp.Result.WorkflowID = id
	newInstance.Spec.UserLabels[config.LABEL_WORKFLOW_ID] = id
	resp.Result.Database = newInstance.Spec.Database

	//create CRD for new cluster
	err = kubeapi.Createpgcluster(apiserver.RESTClient,
		newInstance, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// assign the cluster information to the result
	resp.Result.Name = newInstance.Spec.Name

	// and return!
	return resp
}

func validateConfigPolicies(clusterName, PoliciesFlag, ns string) error {
	var err error
	var configPolicies string

	if PoliciesFlag == "" {
		log.Debugf("%s is Pgo.Cluster.Policies", apiserver.Pgo.Cluster.Policies)
		configPolicies = apiserver.Pgo.Cluster.Policies
	} else {
		configPolicies = PoliciesFlag
	}

	if configPolicies == "" {
		log.Debug("no policies are specified in either pgo.yaml or from user")
		return err
	}

	policies := strings.Split(configPolicies, ",")

	for _, v := range policies {
		result := crv1.Pgpolicy{}

		// error if it already exists
		found, err := kubeapi.Getpgpolicy(apiserver.RESTClient,
			&result, v, ns)
		if !found {
			log.Error("policy " + v + " specified in configuration was not found")
			return err
		}

		if err != nil {
			log.Error("error getting pgpolicy " + v + err.Error())
			return err
		}
		//create a pgtask to add the policy after the db is ready
	}

	spec := crv1.PgtaskSpec{}
	spec.StorageSpec = crv1.PgStorageSpec{}
	spec.TaskType = crv1.PgtaskAddPolicies
	spec.Status = "requested"
	spec.Parameters = make(map[string]string)
	for _, v := range policies {
		spec.Parameters[v] = v
	}
	spec.Name = clusterName + "-policies"
	spec.Namespace = ns
	labels := make(map[string]string)
	labels[config.LABEL_PG_CLUSTER] = clusterName

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)

	return err
}

func getClusterParams(request *msgs.CreateClusterRequest, name string, userLabelsMap map[string]string, ns string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{
		Resources: v1.ResourceList{},
	}

	if userLabelsMap[config.LABEL_CUSTOM_CONFIG] != "" {
		spec.CustomConfig = userLabelsMap[config.LABEL_CUSTOM_CONFIG]
	}

	// if the request has overriding CPURequest and/or MemoryRequest parameters,
	// these will take precedence over the defaults
	if request.CPURequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.CPURequest)
		spec.Resources[v1.ResourceCPU] = quantity
	}

	if request.MemoryRequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.MemoryRequest)
		spec.Resources[v1.ResourceMemory] = quantity
	}

	spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.PrimaryStorage)
	if request.StorageConfig != "" {
		spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
	}

	// set the pd anti-affinity values
	if podAntiAffinity, err := apiserver.Pgo.GetPodAntiAffinitySpec(
		crv1.PodAntiAffinityType(request.PodAntiAffinity),
		crv1.PodAntiAffinityType(request.PodAntiAffinityPgBackRest),
		crv1.PodAntiAffinityType(request.PodAntiAffinityPgBouncer),
	); err != nil {
		log.Warn("Could not set pod anti-affinity rules:", err.Error())
		spec.PodAntiAffinity = crv1.PodAntiAffinitySpec{}
	} else {
		spec.PodAntiAffinity = podAntiAffinity
	}

	// if the PVCSize is overwritten, update the primary storage spec with this
	// value
	if request.PVCSize != "" {
		log.Debugf("PVC Size is overwritten to be [%s]", request.PVCSize)
		spec.PrimaryStorage.Size = request.PVCSize
	}

	// extract the parameters for the TablespaceMounts and put them in the format
	// that is required by the pgcluster CRD
	spec.TablespaceMounts = map[string]crv1.PgStorageSpec{}

	for _, tablespace := range request.Tablespaces {
		storageSpec, _ := apiserver.Pgo.GetStorageSpec(tablespace.StorageConfig)

		// if a PVCSize is specified, override the value of the Size parameter in
		// storage spec
		if tablespace.PVCSize != "" {
			storageSpec.Size = tablespace.PVCSize
		}

		spec.TablespaceMounts[tablespace.Name] = storageSpec
	}

	spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.ReplicaStorage)
	if request.ReplicaStorageConfig != "" {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(request.ReplicaStorageConfig)
	}

	// if the PVCSize is overwritten, update the replica storage spec with this
	// value
	if request.PVCSize != "" {
		log.Debugf("PVC Size is overwritten to be [%s]", request.PVCSize)
		spec.ReplicaStorage.Size = request.PVCSize
	}

	spec.BackrestStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.BackrestStorage)

	// if the user passed in a value to override the pgBackRest storage
	// configuration, apply it here. Note that (and this follows the legacy code)
	// given we've validated this storage configruation exists, this call should
	// be ok
	if request.BackrestStorageConfig != "" {
		spec.BackrestStorage, _ = apiserver.Pgo.GetStorageSpec(request.BackrestStorageConfig)
	}

	// if the BackrestPVCSize is overwritten, update the backrest storage spec
	// with this value
	if request.BackrestPVCSize != "" {
		log.Debugf("pgBackRest PVC Size is overwritten to be [%s]",
			request.BackrestPVCSize)
		spec.BackrestStorage.Size = request.BackrestPVCSize
	}

	spec.CCPImageTag = apiserver.Pgo.Cluster.CCPImageTag
	if request.CCPImageTag != "" {
		spec.CCPImageTag = request.CCPImageTag
		log.Debugf("using CCPImageTag from command line %s", request.CCPImageTag)
	}

	if request.CCPImage != "" {
		spec.CCPImage = request.CCPImage
		log.Debugf("user is overriding CCPImage from command line %s", request.CCPImage)
	} else {
		spec.CCPImage = "crunchy-postgres-ha"
	}
	spec.Namespace = ns
	spec.Name = name
	spec.ClusterName = name
	spec.Port = apiserver.Pgo.Cluster.Port
	spec.PGBadgerPort = apiserver.Pgo.Cluster.PGBadgerPort
	spec.ExporterPort = apiserver.Pgo.Cluster.ExporterPort
	spec.SecretFrom = ""
	spec.PrimaryHost = name
	if request.Policies == "" {
		spec.Policies = apiserver.Pgo.Cluster.Policies
		log.Debugf("Pgo.Cluster.Policies %s", apiserver.Pgo.Cluster.Policies)
	} else {
		spec.Policies = request.Policies
	}

	spec.Replicas = "0"
	str := apiserver.Pgo.Cluster.Replicas
	log.Debugf("[%s] is Pgo.Cluster.Replicas", str)
	if str != "" {
		spec.Replicas = str
	}
	log.Debugf("replica count is %d", request.ReplicaCount)
	if request.ReplicaCount > 0 {
		spec.Replicas = strconv.Itoa(request.ReplicaCount)
		log.Debugf("replicas is  %s", spec.Replicas)
	}
	spec.UserLabels = userLabelsMap
	spec.UserLabels[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION

	//override any values from config file
	str = apiserver.Pgo.Cluster.Port
	log.Debugf("%s", apiserver.Pgo.Cluster.Port)
	if str != "" {
		spec.Port = str
	}

	// set the user. First, attempt to default to the user that is in the pgo.yaml
	// configuration file. If the user has entered a username in the request,
	// then use that one
	spec.User = apiserver.Pgo.Cluster.User

	if request.Username != "" {
		spec.User = request.Username
	}

	log.Debugf("username set to [%s]", spec.User)

	// set the name of the database. The hierarchy is as such:
	// 1. Use the name that the user provides in the request
	// 2. Use the name that is in the pgo.yaml file
	// 3. Use the name of the cluster
	switch {
	case request.Database != "":
		spec.Database = request.Database
	case apiserver.Pgo.Cluster.Database != "":
		spec.Database = apiserver.Pgo.Cluster.Database
	default:
		spec.Database = spec.Name
	}

	log.Debugf("database set to [%s]", spec.Database)

	// set up TLS
	spec.TLSOnly = request.TLSOnly
	spec.TLS.CASecret = request.CASecret
	spec.TLS.TLSSecret = request.TLSSecret

	//pass along command line flags for a restore
	if request.SecretFrom != "" {
		spec.SecretFrom = request.SecretFrom
	}

	spec.CustomConfig = request.CustomConfig
	spec.SyncReplication = request.SyncReplication

	// set pgBackRest S3 settings in the spec if included in the request
	if request.BackrestS3Bucket != "" {
		spec.BackrestS3Bucket = request.BackrestS3Bucket
	}
	if request.BackrestS3Endpoint != "" {
		spec.BackrestS3Endpoint = request.BackrestS3Endpoint
	}
	if request.BackrestS3Region != "" {
		spec.BackrestS3Region = request.BackrestS3Region
	}

	labels := make(map[string]string)
	labels[config.LABEL_NAME] = name
	if !request.AutofailFlag || apiserver.Pgo.Cluster.DisableAutofail {
		labels[config.LABEL_AUTOFAIL] = "false"
	} else {
		labels[config.LABEL_AUTOFAIL] = "true"
	}

	// set whether or not the cluster will be a standby cluster
	spec.Standby = request.Standby
	// set the pgBackRest repository path
	spec.BackrestRepoPath = request.BackrestRepoPath

	//pgbadger - set with global flag first then check for a user flag
	labels[config.LABEL_BADGER] = strconv.FormatBool(apiserver.BadgerFlag)
	if request.BadgerFlag {
		labels[config.LABEL_BADGER] = "true"
	}

	// pgBackRest is always set to true. This is here due to a time where
	// pgBackRest was not the only way
	labels[config.LABEL_BACKREST] = "true"

	// if the pgBouncer flag is set to true, add a label to indicate that this
	// cluster shoul dhave a pgbouncer
	if request.PgbouncerFlag {
		labels[config.LABEL_PGBOUNCER] = "true"
	}

	newInstance := &crv1.Pgcluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: spec,
		Status: crv1.PgclusterStatus{
			State:   crv1.PgclusterStateCreated,
			Message: "Created, not processed yet",
		},
	}
	return newInstance
}

func validateSecretFrom(secretname, user, ns string) error {
	var err error
	selector := config.LABEL_PG_CLUSTER + "=" + secretname
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, ns)
	if err != nil {
		return err
	}

	log.Debugf("secrets for %s", secretname)
	pgprimaryFound := false
	pgrootFound := false
	pguserFound := false

	for _, s := range secrets.Items {
		if s.ObjectMeta.Name == secretname+crv1.PrimarySecretSuffix {
			pgprimaryFound = true
		} else if s.ObjectMeta.Name == secretname+crv1.RootSecretSuffix {
			pgrootFound = true
		} else if s.ObjectMeta.Name == secretname+"-"+user+crv1.UserSecretSuffix {
			pguserFound = true
		}
	}
	if !pgprimaryFound {
		return errors.New(secretname + crv1.PrimarySecretSuffix + " not found")
	}
	if !pgrootFound {
		return errors.New(secretname + crv1.RootSecretSuffix + " not found")
	}
	if !pguserFound {
		return errors.New(secretname + "-" + user + crv1.UserSecretSuffix + " not found")
	}

	return err
}

func getReadyStatus(pod *v1.Pod) (string, bool) {
	equal := false
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	if readyCount == containerCount {
		equal = true
	}
	return fmt.Sprintf("%d/%d", readyCount, containerCount), equal

}

func createDeleteDataTasks(clusterName string, storageSpec crv1.PgStorageSpec, deleteBackups bool, ns string) error {

	var err error

	log.Debugf("creatingDeleteDataTasks deployments for pg-cluster=%s\n", clusterName)

	return err
}

func createWorkflowTask(clusterName, ns, pgouser string) (string, error) {

	//create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = clusterName + "-" + crv1.PgtaskWorkflowCreateClusterType
	spec.TaskType = crv1.PgtaskWorkflow

	spec.Parameters = make(map[string]string)
	spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format(time.RFC3339)
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
		return "", err
	}
	spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	err = kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err
}

func getType(pod *v1.Pod, clusterName string) string {

	//log.Debugf("%v\n", pod.ObjectMeta.Labels)
	if pod.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] != "" {
		return msgs.PodTypePgbackrest
	} else if pod.ObjectMeta.Labels[config.LABEL_PGBOUNCER] != "" {
		return msgs.PodTypePgbouncer
	} else if pod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "master" {
		return msgs.PodTypePrimary
	} else if pod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "replica" {
		return msgs.PodTypeReplica
	}
	return msgs.PodTypeUnknown

}

func validateCustomConfig(configmapname, ns string) (bool, error) {
	var err error
	_, found := kubeapi.GetConfigMap(apiserver.Clientset, configmapname, ns)
	if !found {
		return found, err
	}

	return found, err
}

func existsGlobalConfig(ns string) bool {
	_, found := kubeapi.GetConfigMap(apiserver.Clientset, config.GLOBAL_CUSTOM_CONFIGMAP, ns)
	return found
}

func getReplicas(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterReplica, error) {

	output := make([]msgs.ShowClusterReplica, 0)
	replicaList := crv1.PgreplicaList{}

	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	err := kubeapi.GetpgreplicasBySelector(apiserver.RESTClient,
		&replicaList, selector, ns)
	if err != nil {
		return output, err
	}

	if len(replicaList.Items) == 0 {
		log.Debug("no replicas found")
		return output, err
	}

	for _, replica := range replicaList.Items {
		d := msgs.ShowClusterReplica{}
		d.Name = replica.Spec.Name
		output = append(output, d)
	}

	return output, err
}

// createUserSecret is modeled off of the legacy "createSecrets" method to
// create a user secret for a specified username and password. It determines how
// to assign the credentials to the user based on whether or not they selected
// one of the following in precedence order, with the first in order having
// higher precedence:
//
// 1. The password is supplied directly by the user
// 2. The password is loaded from a pre-existing secret and copied into a new
//    secret.
// 3. The password is generated based on the length provided by the user
// 4. The password is generated based on the value available in the Operator
//    configuration
// 5. The password is generated by the global Operator default value for
//    password length
//
// returns the secertname, password as well as any errors
func createUserSecret(request *msgs.CreateClusterRequest, cluster *crv1.Pgcluster, secretNameSuffix, username, password string) (string, string, error) {
	// the secretName is just the combination cluster name and the secretNameSuffix
	secretName := fmt.Sprintf("%s%s", cluster.Spec.Name, secretNameSuffix)

	// if the secret already exists, we can perform an early exit
	// if there is an error, we'll ignore it
	if secret, found, _ := kubeapi.GetSecret(apiserver.Clientset, secretName, cluster.Spec.Namespace); found {
		log.Infof("secret exists: [%s] - skipping", secretName)

		return secretName, string(secret.Data["password"][:]), nil
	}

	// alright, go through the hierarchy and determine if we need to set the
	// password.
	switch {
	// if the user password is already set, then we can move on to the next step
	case password != "":
		break
		// if the "SecretFrom" parameter is set, then load the password from a prexisting password
	case request.SecretFrom != "":
		// set up the name of the secret that we are loading the secret from
		secretFromSecretName := fmt.Sprintf("%s%s", request.SecretFrom, secretNameSuffix)

		// now attempt to load said secret
		oldPassword, err := util.GetPasswordFromSecret(apiserver.Clientset, cluster.Spec.Namespace, secretFromSecretName)

		// if there is an error, abandon here, otherwise set the oldPassword as the
		// current password
		if err != nil {
			return "", "", err
		}

		password = oldPassword
	// if the user set the password length in the request, honor that here
	// otherwise use either the configured or hard coded default
	default:
		passwordLength := request.PasswordLength

		if request.PasswordLength <= 0 {
			passwordLength = util.GeneratedPasswordLength(apiserver.Pgo.Cluster.PasswordLength)
		}

		password = util.GeneratePassword(passwordLength)
	}

	// great, now we can create the secret! if we can't, return an error
	if err := util.CreateSecret(apiserver.Clientset, cluster.Spec.Name, secretName,
		username, password, cluster.Spec.Namespace); err != nil {
		return "", "", err
	}

	// otherwise, return the secret name, password
	return secretName, password, nil
}

// UpdateCluster ...
func UpdateCluster(request *msgs.UpdateClusterRequest) msgs.UpdateClusterResponse {
	var err error

	response := msgs.UpdateClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("autofail is [%t]\n", request.Autofail)

	switch {
	case request.Startup && request.Shutdown:
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("A startup and a shutdown were requested. " +
			"Please specify one or the other.")
		return response
	}

	if request.Startup && request.Shutdown {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Both a startup and a shutdown was requested. " +
			"Please specify one or the other")
		return response
	}

	// evaluate if the CPU / Memory have been set to custom values
	if request.CPURequest != "" {
		if err := apiserver.ValidateQuantity(request.CPURequest); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf(apiserver.ErrMessageCPURequest, request.CPURequest, err.Error())
			return response
		}
	}

	if request.MemoryRequest != "" {
		if err := apiserver.ValidateQuantity(request.MemoryRequest); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf(apiserver.ErrMessageMemoryRequest, request.MemoryRequest, err.Error())
			return response
		}
	}

	// validate the storage type for each specified tablespace actually exists.
	// if a PVCSize is passed in, also validate that it follows the Kubernetes
	// format
	if err := validateTablespaces(request.Tablespaces); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	if request.AllFlag {
		err = kubeapi.Getpgclusters(apiserver.RESTClient, &clusterList, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	} else if request.Selector != "" {
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, request.Selector, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	} else {
		for _, v := range request.Clustername {
			cl := crv1.Pgcluster{}

			_, err = kubeapi.Getpgcluster(apiserver.RESTClient, &cl, v, request.Namespace)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
			clusterList.Items = append(clusterList.Items, cl)
		}
	}

	if len(clusterList.Items) == 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = "no clusters found"
		return response
	}

	for _, cluster := range clusterList.Items {

		//set autofail=true or false on each pgcluster CRD
		// Make the change based on the value of Autofail vis-a-vis UpdateClusterAutofailStatus
		switch request.Autofail {
		case msgs.UpdateClusterAutofailEnable:
			cluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] = "true"
		case msgs.UpdateClusterAutofailDisable:
			cluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] = "false"
		}

		// enable or disable standby mode based on UpdateClusterStandbyStatus provided in
		// the request
		switch request.Standby {
		case msgs.UpdateClusterStandbyEnable:
			if cluster.Status.State == crv1.PgclusterStateShutdown {
				cluster.Spec.Standby = true
			} else {
				response.Status.Code = msgs.Error
				response.Status.Msg = "Cluster must be shutdown in order to enable standby mode"
				return response
			}
		case msgs.UpdateClusterStandbyDisable:
			cluster.Spec.Standby = false
		}
		// return an error if attempting to enable standby for a cluster that does not have the
		// required S3 settings
		if cluster.Spec.Standby &&
			!strings.Contains(cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], "s3") {
			response.Status.Code = msgs.Error
			response.Status.Msg = "Backrest storage type 's3' must be enabled in order to enable " +
				"standby mode"
			return response
		}

		// if a startup or shutdown was requested then update the pgcluster spec accordingly
		if request.Startup {
			cluster.Spec.Shutdown = false
		} else if request.Shutdown {
			cluster.Spec.Shutdown = true
		}

		// ensure there is a value for Resources
		if cluster.Spec.Resources == nil {
			cluster.Spec.Resources = v1.ResourceList{}
		}

		// if the CPU or memory values have been modified, update the values in the
		// cluster CRD
		if request.CPURequest != "" {
			quantity, _ := resource.ParseQuantity(request.CPURequest)
			cluster.Spec.Resources[v1.ResourceCPU] = quantity
		}

		if request.MemoryRequest != "" {
			quantity, _ := resource.ParseQuantity(request.MemoryRequest)
			cluster.Spec.Resources[v1.ResourceMemory] = quantity
		}

		// extract the parameters for the TablespaceMounts and put them in the
		// format that is required by the pgcluster CRD
		for _, tablespace := range request.Tablespaces {
			storageSpec, _ := apiserver.Pgo.GetStorageSpec(tablespace.StorageConfig)

			// if a PVCSize is specified, override the value of the Size parameter in
			// storage spec
			if tablespace.PVCSize != "" {
				storageSpec.Size = tablespace.PVCSize
			}

			cluster.Spec.TablespaceMounts[tablespace.Name] = storageSpec
		}

		if err := kubeapi.Updatepgcluster(apiserver.RESTClient, &cluster, cluster.Spec.Name, request.Namespace); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		response.Results = append(response.Results, "updated pgcluster "+cluster.Spec.Name)
	}

	return response
}

func GetPrimaryAndReplicaPods(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)

	selector := config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "," + config.LABEL_DEPLOYMENT_NAME
	log.Debugf("selector for GetPrimaryAndReplicaPods is %s", selector)

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)
		d.PVCName = apiserver.GetPVCName(&p)

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}
	selector = config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "-replica" + "," + config.LABEL_DEPLOYMENT_NAME
	log.Debugf("selector for GetPrimaryAndReplicaPods is %s", selector)

	pods, err = kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)
		d.PVCName = apiserver.GetPVCName(&p)

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}

	return output, err

}

// validateBackrestStorageTypeOnCreate validates the pgbackrest storage type specified when
// a new cluster.  This includes ensuring the type provided is valid, and that the required
// configuration settings (s3 bucket, region, etc.) are also present
func validateBackrestStorageTypeOnCreate(request *msgs.CreateClusterRequest) error {

	requestBackRestStorageType := request.BackrestStorageType

	if requestBackRestStorageType != "" && !util.IsValidBackrestStorageType(requestBackRestStorageType) {
		return fmt.Errorf("Invalid value provided for pgBackRest storage type. The following values are allowed: %s",
			"\""+strings.Join(apiserver.GetBackrestStorageTypes(), "\", \"")+"\"")
	} else if strings.Contains(requestBackRestStorageType, "s3") && isMissingS3Config(request) {
		return errors.New("A configuration setting for AWS S3 storage is missing. Values must be " +
			"provided for the S3 bucket, S3 endpoint and S3 region in order to use the 's3' " +
			"storage type with pgBackRest.")
	}

	return nil
}

// validateClusterTLS validates the parameters that allow a user to enable TLS
// connections to a PostgreSQL cluster
func validateClusterTLS(request *msgs.CreateClusterRequest) error {
	// if TLSOnly is not set and  neither TLSSecret no CASecret are set, just return
	if !request.TLSOnly && request.TLSSecret == "" && request.CASecret == "" {
		return nil
	}

	// if TLS only is set, but there is no TLSSecret nor CASecret, return
	if request.TLSOnly && !(request.TLSSecret != "" && request.CASecret != "") {
		return fmt.Errorf("TLS only clusters requires both a TLS secret and CA secret")
	}
	// if TLSSecret or CASecret is set, but not both are set, return
	if (request.TLSSecret != "" && request.CASecret == "") || (request.TLSSecret == "" && request.CASecret != "") {
		fmt.Errorf("Both TLS secret and CA secret must be set in order to enable TLS for PostgreSQL")
	}

	// now check for the existence of the two secrets
	// First the TLS secret
	if _, _, err := kubeapi.GetSecret(apiserver.Clientset, request.TLSSecret, request.Namespace); err != nil {
		return err
	}

	// then, the CA secret
	if _, _, err := kubeapi.GetSecret(apiserver.Clientset, request.CASecret, request.Namespace); err != nil {
		return err
	}

	// after this, we are validated!
	return nil
}

// validateTablespaces validates the tablespace parameters. if there is an error
// it aborts and returns an error
func validateTablespaces(tablespaces []msgs.ClusterTablespaceDetail) error {
	// iterate through the list of tablespaces and return any erors
	for _, tablespace := range tablespaces {
		if !apiserver.IsValidStorageName(tablespace.StorageConfig) {
			return fmt.Errorf("%s storage config for tablespace %s was not found",
				tablespace.StorageConfig, tablespace.Name)
		}

		if tablespace.PVCSize != "" {
			if err := apiserver.ValidateQuantity(tablespace.PVCSize); err != nil {
				return fmt.Errorf(apiserver.ErrMessagePVCSize,
					tablespace.PVCSize, err.Error())
			}
		}
	}

	return nil
}

// determines if any of the required S3 configuration settings (bucket, endpoint
// and region) are missing from both the incoming request or the pgo.yaml config file
func isMissingS3Config(request *msgs.CreateClusterRequest) bool {
	if request.BackrestS3Bucket == "" && apiserver.Pgo.Cluster.BackrestS3Bucket == "" {
		return true
	}
	if request.BackrestS3Endpoint == "" && apiserver.Pgo.Cluster.BackrestS3Endpoint == "" {
		return true
	}
	if request.BackrestS3Region == "" && apiserver.Pgo.Cluster.BackrestS3Region == "" {
		return true
	}
	return false
}

func validateStandbyCluster(request *msgs.CreateClusterRequest) error {
	switch {
	case !strings.Contains(request.BackrestStorageType, "s3"):
		return errors.New("Backrest storage type 's3' must be selected in order to create a " +
			"standby cluster")
	case request.BackrestRepoPath == "":
		return errors.New("A pgBackRest repository path must be specified when creating a " +
			"standby cluster")
	}
	return nil
}
