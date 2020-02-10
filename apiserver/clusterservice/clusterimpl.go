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
	"database/sql"
	"errors"
	"fmt"

	"io/ioutil"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	log "github.com/sirupsen/logrus"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"

	_ "github.com/lib/pq"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
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
	selector := config.LABEL_BACKREST_JOB + "!=true," + config.LABEL_BACKREST_RESTORE + "!=true," + config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_PGBACKUP + "!=true," + config.LABEL_PGBACKUP + "!=false," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
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

func query(dbUser, dbHost, dbPort, database, dbPassword string) bool {
	var conn *sql.DB
	var err error

	connString := "sslmode=disable user=" + dbUser + " host=" + dbHost + " port=" + dbPort + " dbname=" + database + " password=" + dbPassword
	//log.Debugf("connString=%s", connString)

	conn, err = sql.Open("postgres", connString)
	if err != nil {
		log.Debug(err.Error())
		return false
	}

	var ts string
	var rows *sql.Rows

	rows, err = conn.Query("select now()::text")
	if err != nil {
		log.Debug(err.Error())
		return false
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()
	for rows.Next() {
		if err = rows.Scan(&ts); err != nil {
			log.Debug(err.Error())
			return false
		}
		log.Debugf("returned %s", ts)
		return true
	}
	return false

}

// CreateCluster ...
// pgo create cluster mycluster
func CreateCluster(request *msgs.CreateClusterRequest, ns, pgouser string) msgs.CreateClusterResponse {
	var id string
	resp := msgs.CreateClusterResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)
	clusterName := request.Name

	// set the generated password length for random password generation
	generatedPasswordLength := util.GeneratedPasswordLength(apiserver.Pgo.Cluster.PasswordLength)

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

	for i := 0; i < request.Series; i++ {
		if request.Series > 1 {
			clusterName = request.Name + strconv.Itoa(i)
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

		// validate the storage type for tablespaces
		// this is in the format "<tablespaceName1>=<storagetype1>,<tablespaceName2>,<storagetype2>,..."
		if request.TablespaceMounts != "" {
			tablespaces := strings.Split(request.TablespaceMounts, ",")

			for _, v := range tablespaces {
				p := strings.Split(v, "=")

				if apiserver.IsValidStorageName(p[1]) == false {
					resp.Status.Code = msgs.Error
					resp.Status.Msg = fmt.Sprintf("%s storage config for a tablespace was not found", request.StorageConfig)
					return resp
				}
			}
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

		//Set archive timeout value
		userLabelsMap[config.LABEL_ARCHIVE_TIMEOUT] = apiserver.Pgo.Cluster.ArchiveTimeout

		if request.PgbouncerFlag {
			// set flag at cluster level later
			// userLabelsMap[config.LABEL_PGBOUNCER] = "true"

			// need to create password to be added to postgres container and pgbouncer credential...
			if len(request.PgbouncerPass) > 0 {
				userLabelsMap[config.LABEL_PGBOUNCER_PASS] = request.PgbouncerPass
			} else {
				userLabelsMap[config.LABEL_PGBOUNCER_PASS] = util.GeneratePassword(generatedPasswordLength)

			}

			// default pgbouncer user to "pgbouncer" - request should be empty until configurable user is implemented.
			if len(request.PgbouncerUser) > 0 {
				userLabelsMap[config.LABEL_PGBOUNCER_USER] = request.PgbouncerUser
			} else {
				userLabelsMap[config.LABEL_PGBOUNCER_USER] = "pgbouncer"
			}

			userLabelsMap[config.LABEL_PGBOUNCER_SECRET] = request.PgbouncerSecret

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
		// it is, then set the user label for pod anti-affinity to the request value.  Otherwise,
		// return the validation error.
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

		t := time.Now()
		newInstance.Spec.PswLastUpdate = t.Format(time.RFC3339)

		//create secrets
		err, newInstance.Spec.RootSecretName, newInstance.Spec.PrimarySecretName, newInstance.Spec.UserSecretName = createSecrets(request, clusterName, ns, newInstance.Spec.User)
		if err != nil {
			resp.Results = append(resp.Results, err.Error())
			return resp
		}
		newInstance.Spec.CollectSecretName = clusterName + crv1.CollectSecretSuffix

		// Create Backrest secret for S3/SSH Keys:
		// We make this regardless if backrest is enabled or not because
		// the deployment template always tries to mount /sshd volume
		secretName := fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET)
		_, _, err = kubeapi.GetSecret(apiserver.Clientset, secretName, request.Namespace)
		if kerrors.IsNotFound(err) {
			err := util.CreateBackrestRepoSecrets(apiserver.Clientset,
				util.BackrestRepoConfig{
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
			resp.Results = append(resp.Results, err.Error())
			return resp
		}
		newInstance.Spec.UserLabels[config.LABEL_WORKFLOW_ID] = id

		//create CRD for new cluster
		err = kubeapi.Createpgcluster(apiserver.RESTClient,
			newInstance, ns)
		if err != nil {
			resp.Results = append(resp.Results, err.Error())
		} else {
			resp.Results = append(resp.Results, "created Pgcluster "+clusterName)
		}
		resp.Results = append(resp.Results, "workflow id "+id)
	}

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

	spec := crv1.PgclusterSpec{}

	if userLabelsMap[config.LABEL_CUSTOM_CONFIG] != "" {
		spec.CustomConfig = userLabelsMap[config.LABEL_CUSTOM_CONFIG]
	}

	if request.ContainerResources != "" {
		spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(request.ContainerResources)
	} else {
		log.Debugf("Pgo.DefaultContainerResources is %s", apiserver.Pgo.DefaultContainerResources)
		defaultContainerResource := apiserver.Pgo.DefaultContainerResources
		if defaultContainerResource != "" {
			spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(defaultContainerResource)
		}
	}

	spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.PrimaryStorage)
	if request.StorageConfig != "" {
		spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
	}

	// extract the parameters for th TablespacEMounts and put them in the format
	// that is required by the pgcluster CRD
	if request.TablespaceMounts != "" {
		tablespaceMountsMap := map[string]crv1.PgStorageSpec{}

		tablespaces := strings.Split(request.TablespaceMounts, ",")

		for _, v := range tablespaces {
			p := strings.Split(v, "=")
			storageSpec, _ := apiserver.Pgo.GetStorageSpec(p[1])
			tablespaceMountsMap[p[0]] = storageSpec
		}

		spec.TablespaceMounts = tablespaceMountsMap
	}

	spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.ReplicaStorage)
	if request.ReplicaStorageConfig != "" {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(request.ReplicaStorageConfig)
	}

	spec.BackrestStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.BackrestStorage)

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

	spec.Database = "userdb"
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
	spec.Strategy = "1"
	spec.UserLabels = userLabelsMap
	spec.UserLabels[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION

	//override any values from config file
	str = apiserver.Pgo.Cluster.Port
	log.Debugf("%s", apiserver.Pgo.Cluster.Port)
	if str != "" {
		spec.Port = str
	}
	str = apiserver.Pgo.Cluster.User
	if str != "" {
		log.Debugf("Pgo.Cluster.User is %s", str)
		spec.User = str
	}
	str = apiserver.Pgo.Cluster.Database
	log.Debugf("Pgo.Cluster.Database is %s", apiserver.Pgo.Cluster.Database)
	if str != "" {
		spec.Database = str
	}
	str = apiserver.Pgo.Cluster.Strategy
	log.Debugf("%s", apiserver.Pgo.Cluster.Strategy)
	if str != "" {
		spec.Strategy = str
	}
	//pass along command line flags for a restore
	if request.SecretFrom != "" {
		spec.SecretFrom = request.SecretFrom
	}

	spec.CustomConfig = request.CustomConfig
	spec.PodAntiAffinity = request.PodAntiAffinity
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

	//pgbadger - set with global flag first then check for a user flag
	labels[config.LABEL_BADGER] = strconv.FormatBool(apiserver.BadgerFlag)
	if request.BadgerFlag {
		labels[config.LABEL_BADGER] = "true"
	}

	// pgBackRest is always set to true. This is here due to a time where
	// pgBackRest was not the only way
	labels[config.LABEL_BACKREST] = "true"

	// pgbouncer
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
	} else if pod.ObjectMeta.Labels[config.LABEL_PGBACKUP] == "true" {
		return msgs.PodTypeBackup
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

// deleteDatabaseSecrets delete secrets that match pg-cluster=somecluster
func deleteDatabaseSecrets(db, ns string) error {
	var err error
	//get all that match pg-cluster=db
	selector := config.LABEL_PG_CLUSTER + "=" + db
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, s := range secrets.Items {
		err := kubeapi.DeleteSecret(apiserver.Clientset, s.ObjectMeta.Name, ns)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}

func deleteConfigMaps(clusterName, ns string) error {
	label := fmt.Sprintf("pg-cluster=%s", clusterName)
	list, ok := kubeapi.ListConfigMap(apiserver.Clientset, label, ns)
	if !ok {
		return fmt.Errorf("No configMaps found for selector: %s", label)
	}

	for _, configmap := range list.Items {
		err := kubeapi.DeleteConfigMap(apiserver.Clientset, configmap.Name, ns)
		if err != nil {
			return err
		}
	}
	return nil
}

func createSecrets(request *msgs.CreateClusterRequest, clusterName, ns, user string) (error, string, string, string) {

	var err error
	var RootPassword, Password, PrimaryPassword string
	var RootSecretName, PrimarySecretName, UserSecretName string

	// set the generated password length for random password generation
	generatedPasswordLength := util.GeneratedPasswordLength(apiserver.Pgo.Cluster.PasswordLength)

	//allows user to override with their own passwords
	if request.Password != "" {
		log.Debug("user has set a password, will use that instead of generated ones or the secret-from settings")
		RootPassword = request.Password
		Password = request.Password
		PrimaryPassword = request.Password
	}

	if request.SecretFrom != "" {
		log.Debugf("secret-from is specified! using %s", request.SecretFrom)
		_, RootPassword, err = util.GetPasswordFromSecret(apiserver.Clientset, ns, request.SecretFrom+crv1.RootSecretSuffix)
		_, Password, err = util.GetPasswordFromSecret(apiserver.Clientset, ns, request.SecretFrom+"-"+user+crv1.UserSecretSuffix)
		_, PrimaryPassword, err = util.GetPasswordFromSecret(apiserver.Clientset, ns, request.SecretFrom+crv1.PrimarySecretSuffix)
		if err != nil {
			log.Error("error getting secrets using SecretFrom " + request.SecretFrom)
			return err, RootSecretName, PrimarySecretName, UserSecretName
		}
	}

	RootSecretName = clusterName + crv1.RootSecretSuffix
	pgPassword := util.GeneratePassword(generatedPasswordLength)
	if RootPassword != "" {
		log.Debugf("using user specified password for secret %s", RootSecretName)
		pgPassword = RootPassword
	}

	PrimarySecretName = clusterName + crv1.PrimarySecretSuffix
	primaryPassword := util.GeneratePassword(generatedPasswordLength)
	if PrimaryPassword != "" {
		log.Debugf("using user specified password for secret %s", PrimarySecretName)
		primaryPassword = PrimaryPassword
	}

	UserSecretName = clusterName + "-" + user + crv1.UserSecretSuffix
	testPassword := util.GeneratePassword(generatedPasswordLength)
	if Password != "" {
		log.Debugf("using user specified password for secret %s", UserSecretName)
		testPassword = Password
	}

	var found bool
	_, found, err = kubeapi.GetSecret(apiserver.Clientset, RootSecretName, ns)
	if found {
		log.Debugf("not creating secrets %s since it already exists", RootSecretName)
		return err, RootSecretName, PrimarySecretName, UserSecretName

	}

	username := "postgres"
	err = util.CreateSecret(apiserver.Clientset, clusterName, RootSecretName, username, pgPassword, ns)
	if err != nil {
		log.Errorf("error creating secret %s %s", RootSecretName, err.Error())
		return err, RootSecretName, PrimarySecretName, UserSecretName
	}

	username = "primaryuser"
	err = util.CreateSecret(apiserver.Clientset, clusterName, PrimarySecretName, username, primaryPassword, ns)
	if err != nil {
		log.Errorf("error creating secret %s %s", PrimarySecretName, err.Error())
		return err, RootSecretName, PrimarySecretName, UserSecretName
	}

	username = user
	err = util.CreateSecret(apiserver.Clientset, clusterName, UserSecretName, username, testPassword, ns)
	if err != nil {
		log.Errorf("error creating secret %s %s", UserSecretName, err.Error())
		return err, RootSecretName, PrimarySecretName, UserSecretName
	}

	return err, RootSecretName, PrimarySecretName, UserSecretName
}

// UpdateCluster ...
func UpdateCluster(request *msgs.UpdateClusterRequest) msgs.UpdateClusterResponse {
	var err error

	response := msgs.UpdateClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("autofail is [%t]\n", request.Autofail)

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

		err = kubeapi.Updatepgcluster(apiserver.RESTClient,
			&cluster, cluster.Spec.Name, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		} else {
			response.Results = append(response.Results, "updated pgcluster "+cluster.Spec.Name)
		}
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
