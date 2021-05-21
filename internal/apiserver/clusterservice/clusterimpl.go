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
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/apiserver/backupoptions"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
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

	// get the clusters list
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

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			response.Status.Code = msgs.Error
			response.Status.Msg = cluster.Name + msgs.UpgradeError
			return response
		}

		log.Debugf("deleting cluster %s", cluster.Spec.Name)

		// first delete any existing rmdata pgtask with the same name
		err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Delete(ctx, cluster.Name+"-rmdata", metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		if err := util.CreateRMDataTask(apiserver.Clientset, cluster, "", deleteBackups, deleteData, false, false); err != nil {
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

	// get a list of all clusters
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("clusters found len is %d", len(clusterList.Items))

	for i := range clusterList.Items {
		c := clusterList.Items[i]
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

	// get pods, but exclude backup pods and backrest repo
	selector := fmt.Sprintf("%s=%s,%s", config.LABEL_PG_CLUSTER, cluster.GetName(), config.LABEL_PG_DATABASE)
	log.Debugf("selector for GetPods is %s", selector)

	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return output, err
	}

	for i := range pods.Items {
		p := &pods.Items[i]
		d := msgs.ShowClusterPod{
			PVC: []msgs.ShowClusterPodPVC{},
		}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(p)

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
		d.Type = getType(p)
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
		if strings.HasSuffix(p.Name, "-backrest-repo") {
			d.BackrestRepo = true
			d.ClusterName = cluster.Name
		} else if strings.HasSuffix(p.Name, "-pgbouncer") {
			d.Pgbouncer = true
			d.ClusterName = cluster.Name
		} else if strings.HasSuffix(p.Name, "-pgadmin") {
			d.PGAdmin = true
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
	for i := range clusterList.Items {
		c := clusterList.Items[i]
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
			case (strings.HasSuffix(service.Name, "-"+msgs.PodTypeReplica) && strings.Count(service.Name, "-"+msgs.PodTypeReplica) == 1):
				endpoint.InstanceType = msgs.ClusterTestInstanceTypeReplica
			case service.PGAdmin:
				endpoint.InstanceType = msgs.ClusterTestInstanceTypePGAdmin
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

	if err := util.ValidateLabels(request.UserLabels); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
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
	}

	// validate the optional ServiceType parameter
	switch request.ServiceType {
	default:
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("invalid service type %q", request.ServiceType)
		return resp
	case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
		v1.ServiceTypeLoadBalancer, v1.ServiceTypeExternalName, "": // no-op
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
	backrestStorageTypes, err := validateBackrestStorageTypeOnCreate(request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// if a value for BackrestStorageConfig is provided, validate it here
	if request.BackrestStorageConfig != "" && !apiserver.IsValidStorageName(request.BackrestStorageConfig) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("%q storage config was not found", request.BackrestStorageConfig)
		return resp
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
	}

	if request.ReplicaStorageConfig != "" {
		if !apiserver.IsValidStorageName(request.ReplicaStorageConfig) {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.ReplicaStorageConfig + " Storage config was not found "
			return resp
		}
	}

	// determine if the the password type is valid
	if _, err := apiserver.GetPasswordType(request.PasswordType); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else if request.PasswordType == "scram" {
		request.PasswordType = "scram-sha-256"
	}

	// if the pgBouncer flag is set, validate that replicas is set to a
	// nonnegative value and the service type.
	if request.PgbouncerFlag {
		if request.PgBouncerReplicas < 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessageReplicas+" for pgBouncer", 1)
			return resp
		}

		// validate the optional ServiceType parameter
		switch request.PgBouncerServiceType {
		default:
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf("invalid pgBouncer service type %q", request.PgBouncerServiceType)
			return resp
		case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
			v1.ServiceTypeLoadBalancer, v1.ServiceTypeExternalName, "": // no-op
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
	newInstance := getClusterParams(request, clusterName, ns)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	newInstance.Spec.BackrestStorageTypes = backrestStorageTypes

	if request.SecretFrom != "" {
		err = validateSecretFrom(newInstance, request.SecretFrom)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	_ = validateConfigPolicies(clusterName, request.Policies, ns)

	// create the user secrets
	// first, the superuser
	if password, err := createUserSecret(request, newInstance, crv1.PGUserSuperuser, request.PasswordSuperuser); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else {
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
	if password, err := createUserSecret(request, newInstance, crv1.PGUserReplication, request.PasswordReplication); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else {
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
	if password, err := createUserSecret(request, newInstance, newInstance.Spec.User, request.Password); err != nil {
		log.Error(err)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	} else {
		user := msgs.CreateClusterDetailUser{
			Username: newInstance.Spec.User,
			Password: password,
		}

		resp.Result.Users = append(resp.Result.Users, user)
	}

	// Create Backrest secret for GCS/S3/SSH Keys:
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

		// if a GCS key is provided, we need to base64 decode it
		backrestGCSKey := []byte{}
		if request.BackrestGCSKey != "" {
			// try to decode the string
			backrestGCSKey, err = base64.StdEncoding.DecodeString(request.BackrestGCSKey)

			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = fmt.Sprintf("could not decode GCS key: %s", err.Error())
				return resp
			}
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
				util.BackRestRepoSecretKeyAWSS3KeyGCSKey:         backrestGCSKey,
			},
		}

		for k, v := range util.GetCustomLabels(newInstance) {
			secret.ObjectMeta.Labels[k] = v
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

	// create a workflow for this new cluster
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

	// create CRD for new cluster
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
		// create a pgtask to add the policy after the db is ready
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

func getClusterParams(request *msgs.CreateClusterRequest, name string, ns string) *crv1.Pgcluster {
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
		UserLabels: map[string]string{},
	}

	// enable the exporter sidecar based on the what the user passed in or what
	// the default value is. the user value takes precedence, unless it's false,
	// as the legacy check only looked for enablement
	spec.Exporter = request.MetricsFlag || apiserver.MetricsFlag

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

		// additionally if a specific pgBouncer Service Type is set, set that here
		spec.PgBouncer.ServiceType = request.PgBouncerServiceType
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

	// if TLS is enabled for pgBouncer, ensure the secret is specified
	if request.PgBouncerTLSSecret != "" {
		spec.PgBouncer.TLSSecret = request.PgBouncerTLSSecret
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

	// if there is a node label, set the node affinity
	if request.NodeLabel != "" {
		nodeLabel := strings.Split(request.NodeLabel, "=")
		spec.NodeAffinity = crv1.NodeAffinitySpec{
			Default: util.GenerateNodeAffinity(request.NodeAffinityType, nodeLabel[0], []string{nodeLabel[1]}),
		}

		log.Debugf("using node label %s", request.NodeLabel)
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

	spec.ServiceType = request.ServiceType

	if request.UserLabels != nil {
		spec.UserLabels = request.UserLabels
	}
	spec.UserLabels[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION

	// override any values from config file
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

	// set the password type
	spec.PasswordType = request.PasswordType

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

	replicas, _ := strconv.Atoi(spec.Replicas)
	if spec.SyncReplication != nil && *spec.SyncReplication && replicas < 1 {
		spec.Replicas = "1"
		log.Infof("sync replication set. ensuring there is at least one replica.")
	}

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

	// set the pgBackRest GCS settings
	spec.BackrestGCSBucket = apiserver.Pgo.Cluster.BackrestGCSBucket
	if request.BackrestGCSBucket != "" {
		spec.BackrestGCSBucket = request.BackrestGCSBucket
	}

	spec.BackrestGCSEndpoint = apiserver.Pgo.Cluster.BackrestGCSEndpoint
	if request.BackrestGCSEndpoint != "" {
		spec.BackrestGCSEndpoint = request.BackrestGCSEndpoint
	}

	spec.BackrestGCSKeyType = apiserver.Pgo.Cluster.BackrestGCSKeyType
	if request.BackrestGCSKeyType != "" {
		spec.BackrestGCSKeyType = request.BackrestGCSKeyType
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

	// set any tolerations
	spec.Tolerations = request.Tolerations

	labels := make(map[string]string)
	labels[config.LABEL_NAME] = name
	spec.DisableAutofail = !request.AutofailFlag || apiserver.Pgo.Cluster.DisableAutofail
	// set whether or not the cluster will be a standby cluster
	spec.Standby = request.Standby
	// set the pgBackRest repository path
	spec.BackrestRepoPath = request.BackrestRepoPath

	// enable the pgBadger sidecar based on the what the user passed in or what
	// the default value is. the user value takes precedence, unless it's false,
	// as the legacy check only looked for enablement
	spec.PGBadger = request.BadgerFlag || apiserver.BadgerFlag

	newInstance := &crv1.Pgcluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
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

// validateSecretFrom is a legacy method that looks for all of the Secrets from
// a cluster defined by "clusterName" and determines if there are bootstrap
// secrets available, i.e.:
//
// - the Postgres superuser
// - the replication user
// - a user as defined vy the "user" parameter
func validateSecretFrom(cluster *crv1.Pgcluster, secretFromClusterName string) error {
	ctx := context.TODO()

	// grab all of the Secrets from the referenced cluster so we can determine if
	// the Secrets that we are looking for are present
	options := metav1.ListOptions{
		LabelSelector: fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, secretFromClusterName).String(),
	}

	secrets, err := apiserver.Clientset.CoreV1().Secrets(cluster.Namespace).List(ctx, options)
	if err != nil {
		return err
	}

	// if no secrets are found, take an early exit
	if len(secrets.Items) == 0 {
		return fmt.Errorf("no secrets found for %q", secretFromClusterName)
	}

	// see if all three of the secrets exist. this borrows from the legacy method
	// of checking
	found := map[string]bool{
		crv1.PGUserSuperuser:   false,
		crv1.PGUserReplication: false,
		cluster.Spec.User:      false,
	}

	for _, secret := range secrets.Items {
		found[crv1.PGUserSuperuser] = found[crv1.PGUserSuperuser] ||
			(secret.Name == crv1.UserSecretNameFromClusterName(secretFromClusterName, crv1.PGUserSuperuser))
		found[crv1.PGUserReplication] = found[crv1.PGUserReplication] ||
			(secret.Name == crv1.UserSecretNameFromClusterName(secretFromClusterName, crv1.PGUserReplication))
		found[cluster.Spec.User] = found[cluster.Spec.User] ||
			(secret.Name == crv1.UserSecretNameFromClusterName(secretFromClusterName, cluster.Spec.User))
	}

	// if not all of the Secrets were found, return an error
	for secretName, ok := range found {
		if !ok {
			return fmt.Errorf("could not find secret %q in cluster %q", secretName, secretFromClusterName)
		}
	}

	return nil
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

	// create pgtask CRD
	spec := crv1.PgtaskSpec{}
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

func getType(pod *v1.Pod) string {
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
func createUserSecret(request *msgs.CreateClusterRequest, cluster *crv1.Pgcluster, username, password string) (string, error) {
	ctx := context.TODO()
	secretName := crv1.UserSecretName(cluster, username)

	// if the secret already exists, we can perform an early exit
	// if there is an error, we'll ignore it
	if secret, err := apiserver.Clientset.
		CoreV1().Secrets(cluster.Namespace).
		Get(ctx, secretName, metav1.GetOptions{}); err == nil {
		log.Infof("secret exists: [%s] - skipping", secretName)

		return string(secret.Data["password"][:]), nil
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
		secretFromSecretName := fmt.Sprintf("%s-%s-secret", request.SecretFrom, username)

		// now attempt to load said secret
		oldPassword, err := util.GetPasswordFromSecret(apiserver.Clientset, cluster.Namespace, secretFromSecretName)
		// if there is an error, abandon here, otherwise set the oldPassword as the
		// current password
		if err != nil {
			return "", err
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
			return "", err
		}

		password = generatedPassword
	}

	// great, now we can create the secret! if we can't, return an error
	if err := util.CreateSecret(apiserver.Clientset, cluster.Spec.Name, secretName,
		username, password, cluster.Namespace, util.GetCustomLabels(cluster)); err != nil {
		return "", err
	}

	// otherwise, return the secret name, password
	return password, nil
}

// UpdateCluster ...
func UpdateCluster(request *msgs.UpdateClusterRequest) msgs.UpdateClusterResponse {
	ctx := context.TODO()

	response := msgs.UpdateClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("autofail is [%v]\n", request.Autofail)

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

	// if any PVC resizing is occurring, ensure that it is a valid quantity
	if request.PVCSize != "" {
		if err := apiserver.ValidateQuantity(request.PVCSize); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	if request.BackrestPVCSize != "" {
		if err := apiserver.ValidateQuantity(request.BackrestPVCSize); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	if request.WALPVCSize != "" {
		if err := apiserver.ValidateQuantity(request.WALPVCSize); err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	clusterList := crv1.PgclusterList{}

	// get the clusters list
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

	for i := range clusterList.Items {
		cluster := clusterList.Items[i]

		// validate any PVC resizing. If we are resizing a PVC, ensure that we are
		// making it larger
		if request.PVCSize != "" {
			if err := util.ValidatePVCResize(cluster.Spec.PrimaryStorage.Size, request.PVCSize); err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

			cluster.Spec.PrimaryStorage.Size = request.PVCSize
		}

		if request.BackrestPVCSize != "" {
			if err := util.ValidatePVCResize(cluster.Spec.BackrestStorage.Size, request.BackrestPVCSize); err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

			cluster.Spec.BackrestStorage.Size = request.BackrestPVCSize
		}

		if request.WALPVCSize != "" {
			if err := util.ValidatePVCResize(cluster.Spec.WALStorage.Size, request.WALPVCSize); err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

			cluster.Spec.WALStorage.Size = request.WALPVCSize
		}

		// validate an TLS settings. This is fun...to leverage a validation we have
		// on create, we're doing to use a "CreateClusterRequest" struct and fill
		// in the settings from both the request the current cluster spec.
		// though, if we're disabling TLS, then we're skipping that
		if request.DisableTLS {
			cluster.Spec.TLS.CASecret = ""
			cluster.Spec.TLS.ReplicationTLSSecret = ""
			cluster.Spec.TLS.TLSSecret = ""
			cluster.Spec.TLSOnly = false
		} else {
			v := &msgs.CreateClusterRequest{
				CASecret:             cluster.Spec.TLS.CASecret,
				Namespace:            cluster.Namespace,
				ReplicationTLSSecret: cluster.Spec.TLS.ReplicationTLSSecret,
				TLSOnly:              cluster.Spec.TLSOnly,
				TLSSecret:            cluster.Spec.TLS.TLSSecret,
			}

			// while we check for overrides, we can add them to the spec as well. If
			// there is an error during the validation, we're not going to be updating
			// the spec anyway
			if request.CASecret != "" {
				v.CASecret = request.CASecret
				cluster.Spec.TLS.CASecret = request.CASecret
			}

			if request.ReplicationTLSSecret != "" {
				v.ReplicationTLSSecret = request.ReplicationTLSSecret
				cluster.Spec.TLS.ReplicationTLSSecret = request.ReplicationTLSSecret
			}

			if request.TLSSecret != "" {
				v.TLSSecret = request.TLSSecret
				cluster.Spec.TLS.TLSSecret = request.TLSSecret
			}

			switch request.TLSOnly {
			default: // no-op
			case msgs.UpdateClusterTLSOnlyEnable:
				v.TLSOnly = true
				cluster.Spec.TLSOnly = true
			case msgs.UpdateClusterTLSOnlyDisable:
				v.TLSOnly = false
				cluster.Spec.TLSOnly = false
			}

			// validate!
			if err := validateClusterTLS(v); err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
		}

		// set --enable-autofail / --disable-autofail on each pgcluster CRD
		// Make the change based on the value of Autofail vis-a-vis UpdateClusterAutofailStatus
		switch request.Autofail {
		case msgs.UpdateClusterAutofailEnable:
			cluster.Spec.DisableAutofail = false
		case msgs.UpdateClusterAutofailDisable:
			cluster.Spec.DisableAutofail = true
		case msgs.UpdateClusterAutofailDoNothing: // no-op
		}

		// enable or disable the metrics collection sidecar
		switch request.Metrics {
		case msgs.UpdateClusterMetricsEnable:
			cluster.Spec.Exporter = true
		case msgs.UpdateClusterMetricsDisable:
			cluster.Spec.Exporter = false
		case msgs.UpdateClusterMetricsDoNothing: // this is never reached -- no-op
		}

		// enable or disable the pgBadger sidecar
		switch request.PGBadger {
		case msgs.UpdateClusterPGBadgerEnable:
			cluster.Spec.PGBadger = true
		case msgs.UpdateClusterPGBadgerDisable:
			cluster.Spec.PGBadger = false
		case msgs.UpdateClusterPGBadgerDoNothing: // this is never reached -- no-op
		}

		// set the optional ServiceType parameter
		switch request.ServiceType {
		default:
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf("invalid service type %q", request.ServiceType)
			return response
		case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
			v1.ServiceTypeLoadBalancer, v1.ServiceTypeExternalName:
			cluster.Spec.ServiceType = request.ServiceType
		case "": // no-op, well, no change
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
		case msgs.UpdateClusterStandbyDoNothing: // no-op
		}
		// return an error if attempting to enable standby for a cluster that does not have the
		// required S3/GCS settings
		if cluster.Spec.Standby {
			blobEnabled := false
			for _, storageType := range cluster.Spec.BackrestStorageTypes {
				blobEnabled = blobEnabled ||
					(storageType == crv1.BackrestStorageTypeS3 || storageType == crv1.BackrestStorageTypeGCS)
			}

			if !blobEnabled {
				response.Status.Code = msgs.Error
				response.Status.Msg = "Backrest storage type 's3' or 'gcs' must be enabled in order to enable " +
					"standby mode"
				return response
			}
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

		// an odd one...if rotating the password is requested, we can perform this
		// as an operational action and handle it here.
		// if it fails...just put a in the logs.
		if cluster.Spec.Exporter && request.ExporterRotatePassword {
			if err := clusteroperator.RotateExporterPassword(apiserver.Clientset, apiserver.RESTConfig,
				&cluster); err != nil {
				log.Error(err)
			}
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

		// Handle any tolerations. This is fun. So we will have to go through both
		// the toleration addition list as well as the toleration subtraction list.
		//
		// First, we will remove any tolerations that are slated for removal
		if len(request.TolerationsDelete) > 0 {
			tolerations := make([]v1.Toleration, 0)

			for _, toleration := range cluster.Spec.Tolerations {
				delete := false

				for _, tolerationDelete := range request.TolerationsDelete {
					delete = delete || (reflect.DeepEqual(toleration, tolerationDelete))
				}

				// if delete does not match, then we can include this toleration in any
				// updates
				if !delete {
					tolerations = append(tolerations, toleration)
				}
			}

			cluster.Spec.Tolerations = tolerations
		}

		// now, add any new tolerations to the spec
		cluster.Spec.Tolerations = append(cluster.Spec.Tolerations, request.Tolerations...)

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

	// find all of the Pods that represent Postgres primary and replicas.
	// only consider running Pods
	selector := config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "," + config.LABEL_DEPLOYMENT_NAME

	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	pods, err := apiserver.Clientset.CoreV1().Pods(ns).List(ctx, options)
	if err != nil {
		return output, err
	}
	for i := range pods.Items {
		p := &pods.Items[i]
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(p)

		d.Primary = false
		d.Type = getType(p)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}
	selector = config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "-replica" + "," + config.LABEL_DEPLOYMENT_NAME
	options = metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	pods, err = apiserver.Clientset.CoreV1().Pods(ns).List(ctx, options)
	if err != nil {
		return output, err
	}
	for i := range pods.Items {
		p := &pods.Items[i]
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(p)

		d.Primary = false
		d.Type = getType(p)
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
func validateBackrestStorageTypeOnCreate(request *msgs.CreateClusterRequest) ([]crv1.BackrestStorageType, error) {
	storageTypes, err := crv1.ParseBackrestStorageTypes(request.BackrestStorageType)

	if err != nil {
		// if the error is due to no storage types elected, return an empty storage
		// type slice. otherwise return an error
		if errors.Is(err, crv1.ErrStorageTypesEmpty) {
			return []crv1.BackrestStorageType{}, nil
		}

		return nil, err
	}

	// a special check: cannot have both GCS and S3 active at the same time
	i := 0
	for _, storageType := range storageTypes {
		if storageType == "s3" || storageType == "gcs" {
			i += 1
		}
	}

	if i == 2 {
		return nil, fmt.Errorf("Cannot use S3 and GCS at the same time.")
	}

	// a special check -- if S3 or GCS storage is included, check to see if all
	// of the appropriate settings are in place
	for _, storageType := range storageTypes {
		switch storageType {
		default: // no-op
		case crv1.BackrestStorageTypeGCS:
			if isMissingGCSConfig(request) {
				return nil, fmt.Errorf("A configuration settings for GCS storage is missing. " +
					"Values must be provided for the GCS bucket, GCS endpoint, and a GCS key in order " +
					"to use the GCS storage type with pgBackRest.")
			}
		case crv1.BackrestStorageTypeS3:
			if isMissingS3Config(request) {
				return nil, fmt.Errorf("A configuration setting for AWS S3 storage is missing. Values must be " +
					"provided for the S3 bucket, S3 endpoint and S3 region in order to use the 's3' " +
					"storage type with pgBackRest.")
			}

			// a check on the KeyType attribute...if set, it has to be one of two
			// values
			if request.BackrestGCSKeyType != "" &&
				request.BackrestGCSKeyType != "service" && request.BackrestGCSKeyType != "token" {
				return nil, fmt.Errorf("Invalid GCS key type. Must either be \"service\" or \"token\".")
			}
		}
	}

	return storageTypes, nil
}

// validateClusterTLS validates the parameters that allow a user to enable TLS
// connections to a PostgreSQL cluster
func validateClusterTLS(request *msgs.CreateClusterRequest) error {
	ctx := context.TODO()

	// if ReplicationTLSSecret is set, but neither TLSSecret nor CASecret is set
	// then return
	if request.ReplicationTLSSecret != "" && (request.TLSSecret == "" || request.CASecret == "") {
		return fmt.Errorf("Both TLS secret and CA secret must be set in order to enable certificate-based authentication for replication")
	}

	// if PgBouncerTLSSecret is set, return if:
	// a) pgBouncer is not enabled OR
	// b) neither TLSSecret nor CASecret is set
	if request.PgBouncerTLSSecret != "" {
		if !request.PgbouncerFlag {
			return fmt.Errorf("pgBouncer must be enabled in order to enable TLS for pgBouncer")
		}

		if request.TLSSecret == "" || request.CASecret == "" {
			return fmt.Errorf("Both TLS secret and CA secret must be set in order to enable TLS for pgBouncer")
		}
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

	// then, if set, the pgBouncer TLS secret
	if request.PgBouncerTLSSecret != "" {
		if _, err := apiserver.Clientset.
			CoreV1().Secrets(request.Namespace).
			Get(ctx, request.PgBouncerTLSSecret, metav1.GetOptions{}); err != nil {
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

// determines if any of the required GCS configuration settings (bucket) are
// missing from both the incoming request or the pgo.yaml config file
func isMissingGCSConfig(request *msgs.CreateClusterRequest) bool {
	return (request.BackrestGCSBucket == "" && apiserver.Pgo.Cluster.BackrestGCSBucket == "")
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

// isMissingExistingDataSourceGCSConfig determines if any of the required GCS
// configuration settings (bucket, endpoint, key) are missing from the
// annotations in the pgBackRest repo secret as needed to bootstrap a cluster
// from an existing GCS repository
func isMissingExistingDataSourceGCSConfig(backrestRepoSecret *v1.Secret) bool {
	switch {
	case backrestRepoSecret.Annotations[config.ANNOTATION_GCS_BUCKET] == "":
		return true
	case len(backrestRepoSecret.Data[util.BackRestRepoSecretKeyAWSS3KeyGCSKey]) == 0:
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

	restoreClusterName := request.PGDataSource.RestoreFrom
	restoreOpts := request.PGDataSource.RestoreOpts

	var restoreFromNamespace string
	if request.PGDataSource.Namespace != "" {
		restoreFromNamespace = request.PGDataSource.Namespace
	} else {
		restoreFromNamespace = request.Namespace
	}

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
	if _, err := apiserver.Clientset.CoreV1().PersistentVolumeClaims(restoreFromNamespace).Get(ctx,
		fmt.Sprintf(util.BackrestRepoPVCName, restoreClusterName),
		metav1.GetOptions{}); err != nil {
		return fmt.Errorf("Unable to find PVC %s for cluster %s (namespace %s), cannot restore "+
			"from the specified data source",
			fmt.Sprintf(util.BackrestRepoPVCName, restoreClusterName),
			restoreClusterName, restoreFromNamespace)
	}

	// now verify that a pgBackRest repo secret exists for the cluster we are restoring from
	backrestRepoSecret, err := apiserver.Clientset.CoreV1().Secrets(restoreFromNamespace).Get(ctx,
		fmt.Sprintf(util.BackrestRepoSecretName, restoreClusterName), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Unable to find secret %s for cluster %s (namespace %s), cannot restore "+
			"from the specified data source",
			fmt.Sprintf(util.BackrestRepoSecretName, restoreClusterName),
			restoreClusterName, restoreFromNamespace)
	}

	// next perform general validation of the restore options
	if err := backupoptions.ValidateBackupOpts(restoreOpts, request); err != nil {
		return fmt.Errorf("%s: %w", ErrInvalidDataSource, err)
	}

	// now detect if an 's3' repo type was specified via the restore opts, and if so verify that s3
	// settings are present in backrest repo secret for the backup being restored from
	s3Restore := backrest.S3RepoTypeCLIOptionExists(restoreOpts)
	if s3Restore && isMissingExistingDataSourceS3Config(backrestRepoSecret) {
		return fmt.Errorf("Secret %s (namespace %s) is missing the S3 configuration required to "+
			"restore from an S3 repository", backrestRepoSecret.GetName(), restoreFromNamespace)
	}

	// now detect if an 'gcs' repo type was specified via the restore opts, and if
	// so verify that gcs settings are present in backrest repo secret for the
	// backup being restored from
	gcsRestore := backrest.GCSRepoTypeCLIOptionExists(restoreOpts)
	if gcsRestore && isMissingExistingDataSourceGCSConfig(backrestRepoSecret) {
		return fmt.Errorf("Secret %s (namespace %s) is missing the GCS configuration required to "+
			"restore from a GCS repository", backrestRepoSecret.GetName(), restoreFromNamespace)
	}

	// finally, verify that the cluster being restored from is in the proper status, and that no
	// other clusters currently being bootstrapping from the same cluster
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(restoreFromNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", ErrInvalidDataSource, err)
	}
	for _, cl := range clusterList.Items {

		if cl.GetName() == restoreClusterName &&
			cl.Status.State == crv1.PgclusterStateShutdown {
			return fmt.Errorf("Unable to restore from cluster %s (namespace %s) because it has "+
				"a %s status", restoreClusterName, restoreFromNamespace, string(cl.Status.State))
		}

		if cl.Spec.PGDataSource.RestoreFrom == restoreClusterName &&
			cl.Status.State == crv1.PgclusterStateBootstrapping {
			return fmt.Errorf("Cluster %s (namespace %s) is currently bootstrapping from cluster %s, please "+
				"try again once it is completes", cl.GetName(), cl.GetNamespace(), restoreClusterName)
		}
	}

	return nil
}

func validateStandbyCluster(request *msgs.CreateClusterRequest) error {
	switch {
	case !(strings.Contains(request.BackrestStorageType, "s3") || strings.Contains(request.BackrestStorageType, "gcs")):
		return errors.New("Backrest storage type 's3' must be selected in order to create a " +
			"standby cluster")
	case request.BackrestRepoPath == "":
		return errors.New("A pgBackRest repository path must be specified when creating a " +
			"standby cluster")
	}
	return nil
}
