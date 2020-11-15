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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/apiserver/backupoptions"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const (
	// ErrInvalidDataSource defines the error string that is displayed when the data source
	// parameters for a create cluster request are invalid
	ErrInvalidDataSource = "Unable to validate data source"
)

// DeleteCluster ...
func DeleteCluster(name, selector string, deleteData, deleteBackups bool, ns, pgouser string) msgs.DeleteClusterResponse {
	ctx := context.TODO()
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

	//get the clusters list
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
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

		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			response.Status.Code = msgs.Error
			response.Status.Msg = cluster.Name + msgs.UpgradeError
			return response
		}

		log.Debugf("deleting cluster %s", cluster.Spec.Name)
		taskName := cluster.Spec.Name + "-rmdata"
		log.Debugf("creating taskName %s", taskName)
		isBackup := false
		isReplica := false
		replicaName := ""
		clusterPGHAScope := cluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE]

		// first delete any existing rmdata pgtask with the same name
		err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Delete(ctx, taskName, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
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
	ctx := context.TODO()
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

	//get a list of all clusters
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
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
		detail.Pods, err = GetPods(apiserver.Clientset, &c)
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
	ctx := context.TODO()
	output := make([]msgs.ShowClusterDeployment, 0)

	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
	deployments, err := apiserver.Clientset.
		AppsV1().Deployments(ns).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
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

func GetPods(clientset kubernetes.Interface, cluster *crv1.Pgcluster) ([]msgs.ShowClusterPod, error) {
	ctx := context.TODO()
	output := []msgs.ShowClusterPod{}

	//get pods, but exclude backup pods and backrest repo
	selector := fmt.Sprintf("%s=%s,%s", config.LABEL_PG_CLUSTER, cluster.GetName(), config.LABEL_PG_DATABASE)
	log.Debugf("selector for GetPods is %s", selector)

	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return output, err
	}

	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{
			PVC: []msgs.ShowClusterPodPVC{},
		}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)

		// get information about several of the PVCs. This borrows from a legacy
		// method to get this information
		for _, v := range p.Spec.Volumes {
			// if this volume is not a PVC, continue
			if v.VolumeSource.PersistentVolumeClaim == nil {
				continue
			}

			// if this is not any of the 3 mounted PVCs to a PostgreSQL Pod, continue
			if !(v.Name == "pgdata" || v.Name == "pgwal-volume" || strings.HasPrefix(v.Name, "tablespace")) {
				continue
			}

			pvcName := v.VolumeSource.PersistentVolumeClaim.ClaimName
			// query the PVC to get the storage capacity
			pvc, err := clientset.CoreV1().PersistentVolumeClaims(cluster.Namespace).Get(ctx, pvcName, metav1.GetOptions{})

			// if there is an error, ignore it, and move on to the next one
			if err != nil {
				log.Warn(err)
				continue
			}

			capacity := pvc.Status.Capacity[v1.ResourceStorage]

			clusterPVCDetail := msgs.ShowClusterPodPVC{
				Capacity: capacity.String(),
				Name:     pvcName,
			}

			d.PVC = append(d.PVC, clusterPVCDetail)
		}

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
	ctx := context.TODO()
	output := make([]msgs.ShowClusterService, 0)
	selector := config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	services, err := apiserver.Clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
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
		for _, port := range p.Spec.Ports {
			d.ClusterPorts = append(d.ClusterPorts, strconv.Itoa(int(port.Port))+"/"+string(port.Protocol))
		}
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
	ctx := context.TODO()
	var err error

	log.Debugf("TestCluster(%s,%s,%s,%s,%v): Called",
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
			log.Debugf("selector is : all clusters in %s", ns)
		} else {
			selector = "name=" + name
			log.Debugf("selector is: %s", selector)
		}
	}

	// Find a list of a clusters that match the given selector
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})

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
			log.Debugf("Instance found with attributes: (%s, %s, %v)",
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
			case strings.HasSuffix(service.Name, msgs.PodTypeReplica):
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

			log.Debugf("Endpoint found with attributes: (%s, %s, %v)",
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
	ctx := context.TODO()

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

	// error if it already exists
	_, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, clusterName, metav1.GetOptions{})
	if err == nil {
		log.Debugf("pgcluster %s was found so we will not create it", clusterName)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "pgcluster " + clusterName + " was found so we will not create it"
		return resp
	} else if kerrors.IsNotFound(err) {
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

	// validate any parameters provided to bootstrap the cluster from an existing data source
	if err := validateDataSourceParms(request); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// if any of the the PVCSizes are set to a customized value, ensure that they
	// are recognizable by Kubernetes
	// first, the primary/replica PVC size
	if err := apiserver.ValidateQuantity(request.PVCSize); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessagePVCSize, request.PVCSize, err.Error())
		return resp
	}

	// next, the pgBackRest repo PVC size
	if err := apiserver.ValidateQuantity(request.BackrestPVCSize); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessagePVCSize, request.BackrestPVCSize, err.Error())
		return resp
	}

	if err := apiserver.ValidateQuantity(request.WALPVCSize); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessagePVCSize, request.WALPVCSize, err.Error())
		return resp
	}

	// evaluate if the CPU / Memory have been set to custom values and ensure the
	// limit is set to valid bounds
	zeroQuantity := resource.Quantity{}

	if err := apiserver.ValidateResourceRequestLimit(request.CPURequest, request.CPULimit, zeroQuantity); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if err := apiserver.ValidateResourceRequestLimit(request.MemoryRequest, request.MemoryLimit,
		apiserver.Pgo.Cluster.DefaultInstanceResourceMemory); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// similarly, if any of the pgBackRest repo CPU / Memory values have been set,
	// evaluate those as well
	if err := apiserver.ValidateResourceRequestLimit(request.BackrestCPURequest, request.BackrestCPULimit, zeroQuantity); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if err := apiserver.ValidateResourceRequestLimit(request.BackrestMemoryRequest, request.BackrestMemoryLimit,
		apiserver.Pgo.Cluster.DefaultBackrestResourceMemory); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// similarly, if any of the pgBouncer CPU / Memory values have been set,
	// evaluate those as well
	if err := apiserver.ValidateResourceRequestLimit(request.PgBouncerCPURequest, request.PgBouncerCPULimit, zeroQuantity); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if err := apiserver.ValidateResourceRequestLimit(request.PgBouncerMemoryRequest, request.PgBouncerMemoryLimit,
		apiserver.Pgo.Cluster.DefaultPgBouncerResourceMemory); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// similarly, if any of the Crunchy Postgres Exporter CPU / Memory values have been set,
	// evaluate those as well
	if err := apiserver.ValidateResourceRequestLimit(request.ExporterCPURequest, request.ExporterCPULimit, zeroQuantity); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if err := apiserver.ValidateResourceRequestLimit(request.ExporterMemoryRequest, request.ExporterMemoryLimit,
		apiserver.Pgo.Cluster.DefaultPgBouncerResourceMemory); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
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
	userLabelsMap[config.LABEL_EXPORTER] = strconv.FormatBool(apiserver.MetricsFlag)
	if err != nil {
		log.Error(err)
	}

	//if metrics is chosen on the pgo command, stick it into the user labels
	if request.MetricsFlag {
		userLabelsMap[config.LABEL_EXPORTER] = "true"
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
	// been properly specified as required to create a standby cluster
	if request.Standby {
		if err := validateStandbyCluster(request); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	// check that the specified ConfigMap exists
	if request.BackrestConfig != "" {
		_, err := apiserver.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, request.BackrestConfig, metav1.GetOptions{})
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	// ensure the backrest storage type specified for the cluster is valid, and that the
	// configuration required to use that storage type (e.g. a bucket, endpoint and region
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
		resp.Status.Msg = fmt.Sprintf("%q storage config was not found", request.BackrestStorageConfig)
		return resp
	}

	log.Debug("userLabelsMap")
	log.Debugf("%v", userLabelsMap)

	if existsGlobalConfig(ns) {
		userLabelsMap[config.LABEL_CUSTOM_CONFIG] = config.GLOBAL_CUSTOM_CONFIGMAP
	}

	if request.StorageConfig != "" && !apiserver.IsValidStorageName(request.StorageConfig) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("%q storage config was not found", request.StorageConfig)
		return resp
	}

	if request.WALStorageConfig != "" && !apiserver.IsValidStorageName(request.WALStorageConfig) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("%q storage config was not found", request.WALStorageConfig)
		return resp
	}

	if request.WALPVCSize != "" && request.WALStorageConfig == "" && apiserver.Pgo.WALStorage == "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "WAL size requires WAL storage"
		return resp
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

	// if the pgBouncer flag is set, validate that replicas is set to a
	// nonnegative value
	if request.PgbouncerFlag && request.PgBouncerReplicas < 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessageReplicas+" for pgBouncer", 1)
		return resp
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

	// pgBackRest URI style must be set to either 'path' or 'host'. If it is neither,
	// log an error and stop the cluster from being created.
	if request.BackrestS3URIStyle != "" {
		if request.BackrestS3URIStyle != "path" && request.BackrestS3URIStyle != "host" {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "pgBackRest S3 URI style must be set to either \"path\" or \"host\"."
			return resp
		}
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
	newInstance.Spec.CollectSecretName = clusterName + crv1.ExporterSecretSuffix

	// Create Backrest secret for S3/SSH Keys:
	// We make this regardless if backrest is enabled or not because
	// the deployment template always tries to mount /sshd volume
	secretName := fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET)

	if _, err := apiserver.Clientset.
		CoreV1().Secrets(request.Namespace).
		Get(ctx, secretName, metav1.GetOptions{}); kubeapi.IsNotFound(err) {
		// determine if a custom CA secret should be used
		backrestS3CACert := []byte{}

		if request.BackrestS3CASecretName != "" {
			backrestSecret, err := apiserver.Clientset.
				CoreV1().Secrets(request.Namespace).
				Get(ctx, request.BackrestS3CASecretName, metav1.GetOptions{})

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

		// set up the secret for the cluster that contains the pgBackRest
		// information
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
				Labels: map[string]string{
					config.LABEL_VENDOR:            config.LABEL_CRUNCHY,
					config.LABEL_PG_CLUSTER:        clusterName,
					config.LABEL_PGO_BACKREST_REPO: "true",
				},
			},
			Data: map[string][]byte{
				util.BackRestRepoSecretKeyAWSS3KeyAWSS3CACert:    backrestS3CACert,
				util.BackRestRepoSecretKeyAWSS3KeyAWSS3Key:       []byte(request.BackrestS3Key),
				util.BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret: []byte(request.BackrestS3KeySecret),
			},
		}

		if _, err := apiserver.Clientset.CoreV1().Secrets(ns).Create(ctx, secret, metav1.CreateOptions{}); err != nil && !kubeapi.IsAlreadyExists(err) {
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
	_, err = apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Create(ctx, newInstance, metav1.CreateOptions{})
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
	ctx := context.TODO()
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
		// error if it already exists
		_, err := apiserver.Clientset.CrunchydataV1().Pgpolicies(ns).Get(ctx, v, metav1.GetOptions{})
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
		ObjectMeta: metav1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	_, err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, newInstance, metav1.CreateOptions{})

	return err
}

func getClusterParams(request *msgs.CreateClusterRequest, name string, userLabelsMap map[string]string, ns string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{
		Annotations: crv1.ClusterAnnotations{
			Backrest:  map[string]string{},
			Global:    map[string]string{},
			PgBouncer: map[string]string{},
			Postgres:  map[string]string{},
		},
		BackrestResources: v1.ResourceList{},
		BackrestLimits:    v1.ResourceList{},
		Limits:            v1.ResourceList{},
		Resources:         v1.ResourceList{},
		ExporterResources: v1.ResourceList{},
		ExporterLimits:    v1.ResourceList{},
		PgBouncer: crv1.PgBouncerSpec{
			Limits:    v1.ResourceList{},
			Resources: v1.ResourceList{},
		},
	}

	if userLabelsMap[config.LABEL_CUSTOM_CONFIG] != "" {
		spec.CustomConfig = userLabelsMap[config.LABEL_CUSTOM_CONFIG]
	}

	// if the request has overriding CPU/Memory requests/limits parameters,
	// these will take precedence over the defaults
	if request.CPULimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.CPULimit)
		spec.Limits[v1.ResourceCPU] = quantity
	}

	if request.CPURequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.CPURequest)
		spec.Resources[v1.ResourceCPU] = quantity
	}

	if request.MemoryLimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.MemoryLimit)
		spec.Limits[v1.ResourceMemory] = quantity
	}

	if request.MemoryRequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.MemoryRequest)
		spec.Resources[v1.ResourceMemory] = quantity
	} else {
		spec.Resources[v1.ResourceMemory] = apiserver.Pgo.Cluster.DefaultInstanceResourceMemory
	}

	// similarly, if there are any overriding pgBackRest repository container
	// resource request values, set them here
	if request.BackrestCPULimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.BackrestCPULimit)
		spec.BackrestLimits[v1.ResourceCPU] = quantity
	}

	if request.BackrestCPURequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.BackrestCPURequest)
		spec.BackrestResources[v1.ResourceCPU] = quantity
	}

	if request.BackrestMemoryLimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.BackrestMemoryLimit)
		spec.BackrestLimits[v1.ResourceMemory] = quantity
	}

	if request.BackrestMemoryRequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.BackrestMemoryRequest)
		spec.BackrestResources[v1.ResourceMemory] = quantity
	} else {
		spec.BackrestResources[v1.ResourceMemory] = apiserver.Pgo.Cluster.DefaultBackrestResourceMemory
	}

	// similarly, if there are any overriding pgBackRest repository container
	// resource request values, set them here
	if request.ExporterCPULimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.ExporterCPULimit)
		spec.ExporterLimits[v1.ResourceCPU] = quantity
	}

	if request.ExporterCPURequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.ExporterCPURequest)
		spec.ExporterResources[v1.ResourceCPU] = quantity
	}

	if request.ExporterMemoryLimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.ExporterMemoryLimit)
		spec.ExporterLimits[v1.ResourceMemory] = quantity
	}

	if request.ExporterMemoryRequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.ExporterMemoryRequest)
		spec.ExporterResources[v1.ResourceMemory] = quantity
	} else {
		spec.ExporterResources[v1.ResourceMemory] = apiserver.Pgo.Cluster.DefaultExporterResourceMemory
	}

	// if the pgBouncer flag is set to true, indicate that the pgBouncer
	// deployment should be made available in this cluster
	if request.PgbouncerFlag {
		spec.PgBouncer.Replicas = config.DefaultPgBouncerReplicas

		// if the user requests a custom amount of pgBouncer replicas, pass them in
		// here
		if request.PgBouncerReplicas > 0 {
			spec.PgBouncer.Replicas = request.PgBouncerReplicas
		}
	}

	// similarly, if there are any overriding pgBouncer container resource request
	// values, set them here
	if request.PgBouncerCPULimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.PgBouncerCPULimit)
		spec.PgBouncer.Limits[v1.ResourceCPU] = quantity
	}

	if request.PgBouncerCPURequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.PgBouncerCPURequest)
		spec.PgBouncer.Resources[v1.ResourceCPU] = quantity
	}

	if request.PgBouncerMemoryLimit != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.PgBouncerMemoryLimit)
		spec.PgBouncer.Limits[v1.ResourceMemory] = quantity
	}

	if request.PgBouncerMemoryRequest != "" {
		// as this was already validated, we can ignore the error
		quantity, _ := resource.ParseQuantity(request.PgBouncerMemoryRequest)
		spec.PgBouncer.Resources[v1.ResourceMemory] = quantity
	} else {
		spec.PgBouncer.Resources[v1.ResourceMemory] = apiserver.Pgo.Cluster.DefaultPgBouncerResourceMemory
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

	// extract parameters for optional WAL storage. server configuration and
	// request parameters are all optional.
	if apiserver.Pgo.WALStorage != "" {
		spec.WALStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.WALStorage)
	}
	if request.WALStorageConfig != "" {
		spec.WALStorage, _ = apiserver.Pgo.GetStorageSpec(request.WALStorageConfig)
	}
	if request.WALPVCSize != "" {
		spec.WALStorage.Size = request.WALPVCSize
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
	// given we've validated this storage configuration exists, this call should
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

	// update the CRD spec to use the custom CCPImagePrefix, if given
	// otherwise, set the value from the global configuration
	spec.CCPImagePrefix = util.GetValueOrDefault(request.CCPImagePrefix, apiserver.Pgo.Cluster.CCPImagePrefix)

	// update the CRD spec to use the custom PGOImagePrefix, if given
	// otherwise, set the value from the global configuration
	spec.PGOImagePrefix = util.GetValueOrDefault(request.PGOImagePrefix, apiserver.Pgo.Pgo.PGOImagePrefix)

	spec.Namespace = ns
	spec.Name = name
	spec.ClusterName = name
	spec.Port = apiserver.Pgo.Cluster.Port
	spec.PGBadgerPort = apiserver.Pgo.Cluster.PGBadgerPort
	spec.ExporterPort = apiserver.Pgo.Cluster.ExporterPort
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
	spec.TLS.ReplicationTLSSecret = request.ReplicationTLSSecret

	spec.CustomConfig = request.CustomConfig
	spec.SyncReplication = request.SyncReplication

	if request.BackrestConfig != "" {
		configmap := v1.ConfigMapProjection{}
		configmap.Name = request.BackrestConfig
		spec.BackrestConfig = append(spec.BackrestConfig, v1.VolumeProjection{ConfigMap: &configmap})
	}

	// set pgBackRest S3 settings in the spec if included in the request
	// otherwise set to the default configuration value
	if request.BackrestS3Bucket != "" {
		spec.BackrestS3Bucket = request.BackrestS3Bucket
	} else {
		spec.BackrestS3Bucket = apiserver.Pgo.Cluster.BackrestS3Bucket
	}
	if request.BackrestS3Endpoint != "" {
		spec.BackrestS3Endpoint = request.BackrestS3Endpoint
	} else {
		spec.BackrestS3Endpoint = apiserver.Pgo.Cluster.BackrestS3Endpoint
	}
	if request.BackrestS3Region != "" {
		spec.BackrestS3Region = request.BackrestS3Region
	} else {
		spec.BackrestS3Region = apiserver.Pgo.Cluster.BackrestS3Region
	}
	if request.BackrestS3URIStyle != "" {
		spec.BackrestS3URIStyle = request.BackrestS3URIStyle
	} else {
		spec.BackrestS3URIStyle = apiserver.Pgo.Cluster.BackrestS3URIStyle
	}

	// if the pgbackrest-s3-verify-tls flag was set, update the CR spec
	// value accordingly, otherwise, do not set
	if request.BackrestS3VerifyTLS != msgs.UpdateBackrestS3VerifyTLSDoNothing {
		if request.BackrestS3VerifyTLS == msgs.UpdateBackrestS3VerifyTLSDisable {
			spec.BackrestS3VerifyTLS = "false"
		} else {
			spec.BackrestS3VerifyTLS = "true"
		}
	} else {
		spec.BackrestS3VerifyTLS = apiserver.Pgo.Cluster.BackrestS3VerifyTLS
	}

	// set the data source that should be utilized to bootstrap the cluster
	spec.PGDataSource = request.PGDataSource

	// create a map for the CR specific annotations
	annotations := map[string]string{}
	// store the default current primary value as an annotation
	annotations[config.ANNOTATION_CURRENT_PRIMARY] = spec.Name
	// store the initial deployment value, which will match the
	// cluster name initially
	annotations[config.ANNOTATION_PRIMARY_DEPLOYMENT] = spec.Name

	// set the user-defined annotations
	// go through each annotation grouping and make the appropriate changes in the
	// equivalent cluster annotation group
	setClusterAnnotationGroup(spec.Annotations.Global, request.Annotations.Global)
	setClusterAnnotationGroup(spec.Annotations.Postgres, request.Annotations.Postgres)
	setClusterAnnotationGroup(spec.Annotations.Backrest, request.Annotations.Backrest)
	setClusterAnnotationGroup(spec.Annotations.PgBouncer, request.Annotations.PgBouncer)

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

	// pgbadger - set with global flag first then check for a user flag
	labels[config.LABEL_BADGER] = strconv.FormatBool(apiserver.BadgerFlag)
	if request.BadgerFlag {
		labels[config.LABEL_BADGER] = "true"
	}

	// pgBackRest is always set to true. This is here due to a time where
	// pgBackRest was not the only way
	labels[config.LABEL_BACKREST] = "true"

	newInstance := &crv1.Pgcluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
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
	ctx := context.TODO()

	var err error
	selector := config.LABEL_PG_CLUSTER + "=" + secretname
	secrets, err := apiserver.Clientset.
		CoreV1().Secrets(ns).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
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

func createWorkflowTask(clusterName, ns, pgouser string) (string, error) {
	ctx := context.TODO()

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
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	_, err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, newInstance, metav1.CreateOptions{})
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
	} else if pod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == config.LABEL_PGHA_ROLE_PRIMARY {
		return msgs.PodTypePrimary
	} else if pod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == config.LABEL_PGHA_ROLE_REPLICA {
		return msgs.PodTypeReplica
	}
	return msgs.PodTypeUnknown

}

func validateCustomConfig(configmapname, ns string) (bool, error) {
	ctx := context.TODO()
	_, err := apiserver.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, configmapname, metav1.GetOptions{})
	return err == nil, err
}

func existsGlobalConfig(ns string) bool {
	ctx := context.TODO()
	_, err := apiserver.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, config.GLOBAL_CUSTOM_CONFIGMAP, metav1.GetOptions{})
	return err == nil
}

func getReplicas(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterReplica, error) {
	ctx := context.TODO()

	output := make([]msgs.ShowClusterReplica, 0)

	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	replicaList, err := apiserver.Clientset.CrunchydataV1().Pgreplicas(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
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
	ctx := context.TODO()

	// the secretName is just the combination cluster name and the secretNameSuffix
	secretName := fmt.Sprintf("%s%s", cluster.Spec.Name, secretNameSuffix)

	// if the secret already exists, we can perform an early exit
	// if there is an error, we'll ignore it
	if secret, err := apiserver.Clientset.
		CoreV1().Secrets(cluster.Spec.Namespace).
		Get(ctx, secretName, metav1.GetOptions{}); err == nil {
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

		generatedPassword, err := util.GeneratePassword(passwordLength)

		// if the password fails to generate, return the error
		if err != nil {
			return "", "", err
		}

		password = generatedPassword
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
	ctx := context.TODO()

	response := msgs.UpdateClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("autofail is [%v]\n", request.Autofail)

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
	zeroQuantity := resource.Quantity{}

	if err := apiserver.ValidateResourceRequestLimit(request.CPURequest, request.CPULimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Note: we don't consider the default value here because the cluster is
	// already deployed. Additionally, this does not check to see if the
	// request/limits are inline with what's already deployed in a pgcluster. That
	// just becomes too complicated
	if err := apiserver.ValidateResourceRequestLimit(request.MemoryRequest, request.MemoryLimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// similarly, if any of the pgBackRest repo CPU / Memory values have been set,
	// evaluate those as well
	if err := apiserver.ValidateResourceRequestLimit(request.BackrestCPURequest, request.BackrestCPULimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Note: we don't consider the default value here because the cluster is
	// already deployed. Additionally, this does not check to see if the
	// request/limits are inline with what's already deployed for pgBackRest. That
	// just becomes too complicated
	if err := apiserver.ValidateResourceRequestLimit(request.BackrestMemoryRequest, request.BackrestMemoryLimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// similarly, if any of the Crunchy Postgres Exporter repo CPU / Memory values have been set,
	// evaluate those as well
	if err := apiserver.ValidateResourceRequestLimit(request.ExporterCPURequest, request.ExporterCPULimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Note: we don't consider the default value here because the cluster is
	// already deployed. Additionally, this does not check to see if the
	// request/limits are inline with what's already deployed for Crunchy Postgres
	// Exporter. That just becomes too complicated
	if err := apiserver.ValidateResourceRequestLimit(request.ExporterMemoryRequest, request.ExporterMemoryLimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
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
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(request.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		clusterList = *cl
	} else if request.Selector != "" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(request.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: request.Selector,
		})
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		clusterList = *cl
	} else {
		for _, v := range request.Clustername {
			cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(request.Namespace).Get(ctx, v, metav1.GetOptions{})
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
			clusterList.Items = append(clusterList.Items, *cl)
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
			cluster.Spec.Limits = v1.ResourceList{}
			cluster.Spec.Resources = v1.ResourceList{}
		}

		// if the CPU or memory values have been modified, update the values in the
		// cluster CRD
		if request.CPULimit != "" {
			quantity, _ := resource.ParseQuantity(request.CPULimit)
			cluster.Spec.Limits[v1.ResourceCPU] = quantity
		}

		if request.CPURequest != "" {
			quantity, _ := resource.ParseQuantity(request.CPURequest)
			cluster.Spec.Resources[v1.ResourceCPU] = quantity
		}

		if request.MemoryLimit != "" {
			quantity, _ := resource.ParseQuantity(request.MemoryLimit)
			cluster.Spec.Limits[v1.ResourceMemory] = quantity
		}

		if request.MemoryRequest != "" {
			quantity, _ := resource.ParseQuantity(request.MemoryRequest)
			cluster.Spec.Resources[v1.ResourceMemory] = quantity
		}

		// ensure there is a value for BackrestResources
		if cluster.Spec.BackrestResources == nil {
			cluster.Spec.BackrestLimits = v1.ResourceList{}
			cluster.Spec.BackrestResources = v1.ResourceList{}
		}

		// if the pgBackRest repository CPU or memory values have been modified,
		// update the values in the cluster CRD
		if request.BackrestCPULimit != "" {
			quantity, _ := resource.ParseQuantity(request.BackrestCPULimit)
			cluster.Spec.BackrestLimits[v1.ResourceCPU] = quantity
		}

		if request.BackrestCPURequest != "" {
			quantity, _ := resource.ParseQuantity(request.BackrestCPURequest)
			cluster.Spec.BackrestResources[v1.ResourceCPU] = quantity
		}

		if request.BackrestMemoryLimit != "" {
			quantity, _ := resource.ParseQuantity(request.BackrestMemoryLimit)
			cluster.Spec.BackrestLimits[v1.ResourceMemory] = quantity
		}

		if request.BackrestMemoryRequest != "" {
			quantity, _ := resource.ParseQuantity(request.BackrestMemoryRequest)
			cluster.Spec.BackrestResources[v1.ResourceMemory] = quantity
		}

		// ensure there is a value for ExporterResources
		if cluster.Spec.ExporterResources == nil {
			cluster.Spec.ExporterLimits = v1.ResourceList{}
			cluster.Spec.ExporterResources = v1.ResourceList{}
		}

		// if the Exporter CPU or memory values have been modified,
		// update the values in the cluster CRD
		if request.ExporterCPULimit != "" {
			quantity, _ := resource.ParseQuantity(request.ExporterCPULimit)
			cluster.Spec.ExporterLimits[v1.ResourceCPU] = quantity
		}

		if request.ExporterCPURequest != "" {
			quantity, _ := resource.ParseQuantity(request.ExporterCPURequest)
			cluster.Spec.ExporterResources[v1.ResourceCPU] = quantity
		}

		if request.ExporterMemoryLimit != "" {
			quantity, _ := resource.ParseQuantity(request.ExporterMemoryLimit)
			cluster.Spec.ExporterLimits[v1.ResourceMemory] = quantity
		}

		if request.ExporterMemoryRequest != "" {
			quantity, _ := resource.ParseQuantity(request.ExporterMemoryRequest)
			cluster.Spec.ExporterResources[v1.ResourceMemory] = quantity
		}

		// set any user-defined annotations
		// go through each annotation grouping and make the appropriate changes in the
		// equivalent cluster annotation group
		setClusterAnnotationGroup(cluster.Spec.Annotations.Global, request.Annotations.Global)
		setClusterAnnotationGroup(cluster.Spec.Annotations.Postgres, request.Annotations.Postgres)
		setClusterAnnotationGroup(cluster.Spec.Annotations.Backrest, request.Annotations.Backrest)
		setClusterAnnotationGroup(cluster.Spec.Annotations.PgBouncer, request.Annotations.PgBouncer)

		// if TablespaceMounts happens to be nil (e.g. an upgraded cluster), and
		// the tablespaces are being updated, set it here
		if len(request.Tablespaces) > 0 && cluster.Spec.TablespaceMounts == nil {
			cluster.Spec.TablespaceMounts = map[string]crv1.PgStorageSpec{}
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

		if _, err := apiserver.Clientset.CrunchydataV1().Pgclusters(request.Namespace).Update(ctx, &cluster, metav1.UpdateOptions{}); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		response.Results = append(response.Results, "updated pgcluster "+cluster.Spec.Name)
	}

	return response
}

func GetPrimaryAndReplicaPods(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterPod, error) {
	ctx := context.TODO()
	output := make([]msgs.ShowClusterPod, 0)

	selector := config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "," + config.LABEL_DEPLOYMENT_NAME
	log.Debugf("selector for GetPrimaryAndReplicaPods is %s", selector)

	pods, err := apiserver.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}
	selector = config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "-replica" + "," + config.LABEL_DEPLOYMENT_NAME
	log.Debugf("selector for GetPrimaryAndReplicaPods is %s", selector)

	pods, err = apiserver.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}

	return output, err

}

// setClusterAnnotationGroup helps with setting the specific annotation group
func setClusterAnnotationGroup(annotationGroup, annotations map[string]string) {
	for k, v := range annotations {
		switch v {
		default:
			annotationGroup[k] = v
		case "":
			delete(annotationGroup, k)
		}
	}
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
	ctx := context.TODO()

	// if ReplicationTLSSecret is set, but neither TLSSecret nor CASecret is not
	// set, then return
	if request.ReplicationTLSSecret != "" && (request.TLSSecret == "" || request.CASecret == "") {
		return fmt.Errorf("Both TLS secret and CA secret must be set in order to enable certificate-based authentication for replication")
	}

	// if TLSOnly is not set and  neither TLSSecret no CASecret are set, just return
	if !request.TLSOnly && request.TLSSecret == "" && request.CASecret == "" {
		return nil
	}

	// if TLS only is set, but there is no TLSSecret nor CASecret, return
	if request.TLSOnly && !(request.TLSSecret != "" && request.CASecret != "") {
		return errors.New("TLS only clusters requires both a TLS secret and CA secret")
	}
	// if TLSSecret or CASecret is set, but not both are set, return
	if (request.TLSSecret != "" && request.CASecret == "") || (request.TLSSecret == "" && request.CASecret != "") {
		return errors.New("Both TLS secret and CA secret must be set in order to enable TLS for PostgreSQL")
	}

	// now check for the existence of the two secrets
	// First the TLS secret
	if _, err := apiserver.Clientset.
		CoreV1().Secrets(request.Namespace).
		Get(ctx, request.TLSSecret, metav1.GetOptions{}); err != nil {
		return err
	}

	// then, the CA secret
	if _, err := apiserver.Clientset.
		CoreV1().Secrets(request.Namespace).
		Get(ctx, request.CASecret, metav1.GetOptions{}); err != nil {
		return err
	}

	// then, if set, the Replication TLS secret
	if request.ReplicationTLSSecret != "" {
		if _, err := apiserver.Clientset.
			CoreV1().Secrets(request.Namespace).
			Get(ctx, request.ReplicationTLSSecret, metav1.GetOptions{}); err != nil {
			return err
		}
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

		if err := apiserver.ValidateQuantity(tablespace.PVCSize); err != nil {
			return fmt.Errorf(apiserver.ErrMessagePVCSize,
				tablespace.PVCSize, err.Error())
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

// isMissingExistingDataSourceS3Config determines if any of the required S3 configuration
// settings (bucket, endpoint, region, key and key secret) are missing from the annotations
// in the pgBackRest repo secret as needed to bootstrap a cluster from an existing S3 repository
func isMissingExistingDataSourceS3Config(backrestRepoSecret *v1.Secret) bool {
	switch {
	case backrestRepoSecret.Annotations[config.ANNOTATION_S3_BUCKET] == "":
		return true
	case backrestRepoSecret.Annotations[config.ANNOTATION_S3_ENDPOINT] == "":
		return true
	case backrestRepoSecret.Annotations[config.ANNOTATION_S3_REGION] == "":
		return true
	case len(backrestRepoSecret.Data[util.BackRestRepoSecretKeyAWSS3KeyAWSS3Key]) == 0:
		return true
	case len(backrestRepoSecret.Data[util.BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret]) == 0:
		return true
	}
	return false
}

// validateDataSourceParms performs validation of any data source parameters included in a request
// to create a new cluster
func validateDataSourceParms(request *msgs.CreateClusterRequest) error {
	ctx := context.TODO()
	namespace := request.Namespace
	restoreClusterName := request.PGDataSource.RestoreFrom
	restoreOpts := request.PGDataSource.RestoreOpts

	if restoreClusterName == "" && restoreOpts == "" {
		return nil
	}

	// first verify that a "restore from" parameter was specified if the restore options
	// are not empty
	if restoreOpts != "" && restoreClusterName == "" {
		return fmt.Errorf("A cluster to restore from must be specified when providing restore " +
			"options")
	}

	// next verify whether or not a PVC exists for the cluster we are restoring from
	if _, err := apiserver.Clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx,
		fmt.Sprintf(util.BackrestRepoPVCName, restoreClusterName),
		metav1.GetOptions{}); err != nil {
		return fmt.Errorf("Unable to find PVC %s for cluster %s, cannot to restore from the "+
			"specified data source", fmt.Sprintf(util.BackrestRepoPVCName, restoreClusterName),
			restoreClusterName)
	}

	// now verify that a pgBackRest repo secret exists for the cluster we are restoring from
	backrestRepoSecret, err := apiserver.Clientset.CoreV1().Secrets(namespace).Get(ctx,
		fmt.Sprintf(util.BackrestRepoSecretName, restoreClusterName), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Unable to find secret %s for cluster %s, cannot restore from the "+
			"specified data source",
			fmt.Sprintf(util.BackrestRepoSecretName, restoreClusterName), restoreClusterName)
	}

	// next perform general validation of the restore options
	if err := backupoptions.ValidateBackupOpts(restoreOpts, request); err != nil {
		return fmt.Errorf("%s: %w", ErrInvalidDataSource, err)
	}

	// now detect if an 's3' repo type was specified via the restore opts, and if so verify that s3
	// settings are present in backrest repo secret for the backup being restored from
	s3Restore := backrest.S3RepoTypeCLIOptionExists(restoreOpts)
	if s3Restore && isMissingExistingDataSourceS3Config(backrestRepoSecret) {
		return fmt.Errorf("Secret %s is missing the S3 configuration required to restore "+
			"from an S3 repository", backrestRepoSecret.GetName())
	}

	// finally, verify that the cluster being restored from is in the proper status, and that no
	// other clusters currently being bootstrapping from the same cluster
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", ErrInvalidDataSource, err)
	}
	for _, cl := range clusterList.Items {

		if cl.GetName() == restoreClusterName &&
			cl.Status.State == crv1.PgclusterStateShutdown {
			return fmt.Errorf("Unable to restore from cluster %s because it has a %s "+
				"status", restoreClusterName, string(cl.Status.State))
		}

		if cl.Spec.PGDataSource.RestoreFrom == restoreClusterName &&
			cl.Status.State == crv1.PgclusterStateBootstrapping {
			return fmt.Errorf("Cluster %s is currently bootstrapping from cluster %s, please "+
				"try again once it is completes", cl.GetName(), restoreClusterName)
		}
	}

	return nil
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
