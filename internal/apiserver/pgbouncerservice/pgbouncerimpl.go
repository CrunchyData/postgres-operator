package pgbouncerservice

/*
Copyright 2018 - 2023 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pgBouncerServiceSuffix = "-pgbouncer"

// CreatePgbouncer ...
// pgo create pgbouncer mycluster
// pgo create pgbouncer --selector=name=mycluster
func CreatePgbouncer(request *msgs.CreatePgbouncerRequest, ns, pgouser string) msgs.CreatePgbouncerResponse {
	ctx := context.TODO()
	var err error
	resp := msgs.CreatePgbouncerResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	// validate the CPU/Memory request parameters, if they are passed in
	if err := apiserver.ValidateResourceRequestLimit(request.CPURequest, request.CPULimit, resource.Quantity{}); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if err := apiserver.ValidateResourceRequestLimit(request.MemoryRequest, request.MemoryLimit,
		apiserver.Pgo.Cluster.DefaultPgBouncerResourceMemory); err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// validate the number of replicas being requested
	if request.Replicas < 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessageReplicas, 1)
		return resp
	}

	log.Debugf("createPgbouncer selector is [%s]", request.Selector)

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.Args, request.Selector)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	for i := range clusterList.Items {
		cluster := clusterList.Items[i]
		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = cluster.Name + msgs.UpgradeError
			return resp
		}

		// validate the TLS settings
		if err := validateTLS(cluster, request); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		log.Debugf("adding pgbouncer to cluster [%s]", cluster.Name)

		resources := v1.ResourceList{}
		limits := v1.ResourceList{}

		// Set the value that enables the pgBouncer, which is the replicas
		// Set the default value, and if there is a custom number of replicas
		// provided, set it to that
		cluster.Spec.PgBouncer.Replicas = config.DefaultPgBouncerReplicas

		if request.Replicas > 0 {
			cluster.Spec.PgBouncer.Replicas = request.Replicas
		}

		// set the optional ServiceType parameter
		switch request.ServiceType {
		default:
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf("invalid service type %q", request.ServiceType)
			return resp
		case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
			v1.ServiceTypeLoadBalancer, v1.ServiceTypeExternalName, "":
			cluster.Spec.PgBouncer.ServiceType = request.ServiceType
		}

		// if the request has overriding CPU/memory parameters,
		// these will take precedence over the defaults
		if request.CPULimit != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.CPULimit)
			limits[v1.ResourceCPU] = quantity
		}

		if request.CPURequest != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.CPURequest)
			resources[v1.ResourceCPU] = quantity
		}

		if request.MemoryLimit != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.MemoryLimit)
			limits[v1.ResourceMemory] = quantity
		}

		if request.MemoryRequest != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.MemoryRequest)
			resources[v1.ResourceMemory] = quantity
		} else {
			resources[v1.ResourceMemory] = apiserver.Pgo.Cluster.DefaultPgBouncerResourceMemory
		}

		cluster.Spec.PgBouncer.Resources = resources
		cluster.Spec.PgBouncer.Limits = limits
		cluster.Spec.PgBouncer.TLSSecret = request.TLSSecret

		// update the cluster CRD with these udpates. If there is an error
		if _, err := apiserver.Clientset.CrunchydataV1().Pgclusters(request.Namespace).
			Update(ctx, &cluster, metav1.UpdateOptions{}); err != nil {
			log.Error(err)
			resp.Results = append(resp.Results, err.Error())
			continue
		}

		resp.Results = append(resp.Results, fmt.Sprintf("%s pgbouncer added", cluster.Name))
	}

	return resp
}

// DeletePgbouncer ...
// pgo delete pgbouncer mycluster
// pgo delete pgbouncer --selector=name=mycluster
func DeletePgbouncer(request *msgs.DeletePgbouncerRequest, ns string) msgs.DeletePgbouncerResponse {
	ctx := context.TODO()
	var err error
	resp := msgs.DeletePgbouncerResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("deletePgbouncer selector is [%s]", request.Selector)

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.Args, request.Selector)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// Return an error if any clusters identified to have pgbouncer fully deleted (as specified
	// using the uninstall parameter) have standby mode enabled and the 'uninstall' option selected.
	// This because while in standby mode the cluster is read-only, preventing the execution of the
	// SQL required to remove pgBouncer.
	if hasStandby, standbyClusters := apiserver.PGClusterListHasStandby(clusterList); hasStandby &&
		request.Uninstall {

		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("Request rejected, unable to delete pgbouncer using the "+
			"'uninstall' parameter for clusters %s: %s.", strings.Join(standbyClusters, ","),
			apiserver.ErrStandbyNotAllowed.Error())
		return resp
	}

	for i := range clusterList.Items {
		cluster := clusterList.Items[i]
		log.Debugf("deleting pgbouncer from cluster [%s]", cluster.Name)

		// check to see if the uninstall flag was set. If it was, apply the update
		// inline
		if request.Uninstall {
			if err := clusteroperator.UninstallPgBouncer(apiserver.Clientset, apiserver.RESTConfig, &cluster); err != nil {
				log.Error(err)
				resp.Status.Code = msgs.Error
				resp.Results = append(resp.Results, err.Error())
				return resp
			}
		}

		// Disable the pgBouncer Deploymnet, which means setting Replicas to 0
		cluster.Spec.PgBouncer.Replicas = 0
		// Set the resources/limits to their default values
		cluster.Spec.PgBouncer.Resources = v1.ResourceList{}
		cluster.Spec.PgBouncer.Limits = v1.ResourceList{}

		// update the cluster CRD with these udpates. If there is an error
		if _, err := apiserver.Clientset.CrunchydataV1().Pgclusters(request.Namespace).
			Update(ctx, &cluster, metav1.UpdateOptions{}); err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Results = append(resp.Results, err.Error())
			return resp
		}

		// follow the legacy format for returning this information
		result := fmt.Sprintf("%s pgbouncer deleted", cluster.Name)
		resp.Results = append(resp.Results, result)
	}

	return resp
}

// ShowPgBouncer gets information about a PostgreSQL cluster's pgBouncer
// deployment
//
// pgo show pgbouncer
// pgo show pgbouncer --selector
func ShowPgBouncer(request *msgs.ShowPgBouncerRequest, namespace string) msgs.ShowPgBouncerResponse {
	// set up a dummy response
	response := msgs.ShowPgBouncerResponse{
		Results: []msgs.ShowPgBouncerDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	log.Debugf("show pgbouncer called, cluster [%v], selector [%s]", request.ClusterNames, request.Selector)

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.ClusterNames, request.Selector)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// iterate through the list of clusters to get the relevant pgBouncer
	// information about them
	for _, cluster := range clusterList.Items {
		result := msgs.ShowPgBouncerDetail{
			ClusterName:  cluster.Spec.Name,
			HasPgBouncer: true,
		}
		// first, check if the cluster has pgBouncer enabled
		if !cluster.Spec.PgBouncer.Enabled() {
			result.HasPgBouncer = false
			response.Results = append(response.Results, result)
			continue
		}

		// only set the pgBouncer user if we know this is a pgBouncer enabled
		// cluster...even though, yes, this is a constant
		result.Username = crv1.PGUserPgBouncer

		// set the pgBouncer service information on this record
		setPgBouncerServiceDetail(cluster, &result)

		// get the user information about the pgBouncer deployment
		setPgBouncerPasswordDetail(cluster, &result)

		// append the result to the list
		response.Results = append(response.Results, result)
	}

	return response
}

// UpdatePgBouncer updates a cluster's pgBouncer deployment based on the
// parameters passed in. This includes:
//
// - password rotation
// - updating CPU/memory resources
func UpdatePgBouncer(request *msgs.UpdatePgBouncerRequest, namespace, pgouser string) msgs.UpdatePgBouncerResponse {
	ctx := context.TODO()
	// set up a dummy response
	response := msgs.UpdatePgBouncerResponse{
		// Results: []msgs.ShowPgBouncerDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	// validate the CPU/Memory parameters, if they are passed in
	zeroQuantity := resource.Quantity{}

	if err := apiserver.ValidateResourceRequestLimit(request.CPURequest, request.CPULimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Don't check the default value as pgBouncer is already deployed
	if err := apiserver.ValidateResourceRequestLimit(request.MemoryRequest, request.MemoryLimit, zeroQuantity); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// validate the number of replicas being requested
	if request.Replicas < 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf(apiserver.ErrMessageReplicas, 1)
		return response
	}

	log.Debugf("update pgbouncer called, cluster [%v], selector [%s]", request.ClusterNames, request.Selector)

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.ClusterNames, request.Selector)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Return an error if any clusters selected to have pgbouncer updated have standby mode enabled.
	// This is because while in standby mode the cluster is read-only, preventing the execution of the
	// SQL required to update pgbouncer.
	if hasStandby, standbyClusters := apiserver.PGClusterListHasStandby(clusterList); hasStandby {

		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Request rejected, unable to update pgbouncer for "+
			"clusters %s: %s.", strings.Join(standbyClusters, ","),
			apiserver.ErrStandbyNotAllowed.Error())
		return response
	}

	// iterate through the list of clusters to get the relevant pgBouncer
	// information about them
	for i := range clusterList.Items {
		cluster := clusterList.Items[i]
		result := msgs.UpdatePgBouncerDetail{
			ClusterName:  cluster.Spec.Name,
			HasPgBouncer: true,
		}

		// first, check if the cluster has pgBouncer enabled
		if !cluster.Spec.PgBouncer.Enabled() {
			result.HasPgBouncer = false
			response.Results = append(response.Results, result)
			continue
		}

		// if we are rotating the password, perform the request inline
		if request.RotatePassword {
			if err := clusteroperator.RotatePgBouncerPassword(apiserver.Clientset, apiserver.RESTConfig, &cluster); err != nil {
				log.Error(err)
				result.Error = true
				result.ErrorMessage = err.Error()
				response.Results = append(response.Results, result)
			}
		}

		// set the optional ServiceType parameter
		switch request.ServiceType {
		default:
			result.Error = true
			result.ErrorMessage = fmt.Sprintf("invalid service type %q", request.ServiceType)
			response.Results = append(response.Results, result)
			continue
		case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
			v1.ServiceTypeLoadBalancer, v1.ServiceTypeExternalName:
			cluster.Spec.PgBouncer.ServiceType = request.ServiceType
		case "": // no-op, well, no change
		}

		// ensure the Resources/Limits are non-nil
		if cluster.Spec.PgBouncer.Resources == nil {
			cluster.Spec.PgBouncer.Resources = v1.ResourceList{}
		}

		if cluster.Spec.PgBouncer.Limits == nil {
			cluster.Spec.PgBouncer.Limits = v1.ResourceList{}
		}

		// if the request has overriding CPU/Memory parameters,
		// add them to the cluster's pgbouncer resource list
		if request.CPULimit != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.CPULimit)
			cluster.Spec.PgBouncer.Limits[v1.ResourceCPU] = quantity
		}

		if request.CPURequest != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.CPURequest)
			cluster.Spec.PgBouncer.Resources[v1.ResourceCPU] = quantity
		}

		if request.MemoryLimit != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.MemoryLimit)
			cluster.Spec.PgBouncer.Limits[v1.ResourceMemory] = quantity
		}

		if request.MemoryRequest != "" {
			// as this was already validated, we can ignore the error
			quantity, _ := resource.ParseQuantity(request.MemoryRequest)
			cluster.Spec.PgBouncer.Resources[v1.ResourceMemory] = quantity
		}

		// apply the replica count number if there is a change, i.e. replicas is not
		// 0
		if request.Replicas > 0 {
			cluster.Spec.PgBouncer.Replicas = request.Replicas
		}

		if _, err := apiserver.Clientset.CrunchydataV1().Pgclusters(cluster.Namespace).
			Update(ctx, &cluster, metav1.UpdateOptions{}); err != nil {
			log.Error(err)
			result.Error = true
			result.ErrorMessage = err.Error()
			response.Results = append(response.Results, result)
			continue
		}

		// append the result to the list
		response.Results = append(response.Results, result)
	}

	return response
}

// getClusterList tries to return a list of clusters based on either having an
// argument list of cluster names, or a Kubernetes selector
func getClusterList(namespace string, clusterNames []string, selector string) (crv1.PgclusterList, error) {
	ctx := context.TODO()
	clusterList := crv1.PgclusterList{}

	// see if there are any values in the cluster name list or in the selector
	// if nothing exists, return an error
	if len(clusterNames) == 0 && selector == "" {
		err := fmt.Errorf("either a list of cluster names or a selector needs to be supplied for this comment")
		return clusterList, err
	}

	// try to build the cluster list based on either the selector or the list
	// of arguments...or both. First, start with the selector
	if selector != "" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		// if there is an error, return here with an empty cluster list
		if err != nil {
			return crv1.PgclusterList{}, err
		}
		clusterList = *cl
	}

	// now try to get clusters based specific cluster names
	for _, clusterName := range clusterNames {
		cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
		// if there is an error, capture it here and return here with an empty list
		if err != nil {
			return crv1.PgclusterList{}, err
		}

		// if successful, append to the cluster list
		clusterList.Items = append(clusterList.Items, *cluster)
	}

	log.Debugf("clusters founds: [%d]", len(clusterList.Items))

	// if after all this, there are no clusters found, return an error
	if len(clusterList.Items) == 0 {
		err := fmt.Errorf("no clusters found")
		return clusterList, err
	}

	// all set! return the cluster list with error
	return clusterList, nil
}

// setPgBouncerPasswordDetail applies the password that is used by the pgbouncer
// service account
func setPgBouncerPasswordDetail(cluster crv1.Pgcluster, result *msgs.ShowPgBouncerDetail) {
	pgBouncerSecretName := util.GeneratePgBouncerSecretName(cluster.Spec.Name)

	// attempt to get the secret, but only get the password
	password, err := util.GetPasswordFromSecret(apiserver.Clientset,
		cluster.Namespace, pgBouncerSecretName)
	if err != nil {
		log.Warn(err)
	}

	// and set the password. Easy!
	result.Password = password
}

// setPgBouncerServiceDetail applies the information about the pgBouncer service
// to the result for the pgBouncer show
func setPgBouncerServiceDetail(cluster crv1.Pgcluster, result *msgs.ShowPgBouncerDetail) {
	ctx := context.TODO()
	// get the service information about the pgBouncer deployment
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, cluster.Spec.Name)

	// have to go through a bunch of services because "current design"
	services, err := apiserver.Clientset.
		CoreV1().Services(cluster.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
		// if there is an error, return without making any adjustments
	if err != nil {
		log.Warn(err)
		return
	}

	log.Debugf("cluster [%s] has [%d] services", cluster.Spec.Name, len(services.Items))

	// adding the service information was borrowed from the ShowCluster
	// resource
	for _, service := range services.Items {
		// if this service is not for pgBouncer, then skip
		if !strings.HasSuffix(service.Name, pgBouncerServiceSuffix) {
			continue
		}

		// this is the pgBouncer service!
		result.ServiceClusterIP = service.Spec.ClusterIP
		result.ServiceName = service.Name

		// try to get the exterinal IP based on the formula used in show cluster
		if len(service.Spec.ExternalIPs) > 0 {
			result.ServiceExternalIP = service.Spec.ExternalIPs[0]
		}

		if len(service.Status.LoadBalancer.Ingress) > 0 {
			result.ServiceExternalIP = service.Status.LoadBalancer.Ingress[0].IP
		}
	}
}

// validateTLS validates the parameters that allow a user to enable TLS
// connections to a pgBouncer cluster. In essence, it requires both the
// TLSSecret to be set for pgBouncer as well as a CASecret/TLSSecret for the
// cluster itself
func validateTLS(cluster crv1.Pgcluster, request *msgs.CreatePgbouncerRequest) error {
	ctx := context.TODO()

	// if TLSSecret is not set, well, this is valid
	if request.TLSSecret == "" {
		return nil
	}

	// if ReplicationTLSSecret is set, but neither TLSSecret nor CASecret is not
	// set, then return
	if request.TLSSecret != "" && (cluster.Spec.TLS.TLSSecret == "" || cluster.Spec.TLS.CASecret == "") {
		return fmt.Errorf("%s: both TLS secret and CA secret must be set on the cluster in order to enable TLS for pgBouncer", cluster.Name)
	}

	// ensure the TLSSecret and CASecret for the cluster are actually present
	// now check for the existence of the two secrets
	// First the TLS secret
	if _, err := apiserver.Clientset.
		CoreV1().Secrets(cluster.Namespace).
		Get(ctx, cluster.Spec.TLS.TLSSecret, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("%s: cannot find TLS secret for cluster: %w", cluster.Name, err)
	}

	if _, err := apiserver.Clientset.
		CoreV1().Secrets(cluster.Namespace).
		Get(ctx, cluster.Spec.TLS.CASecret, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("%s: cannot find CA secret for cluster: %w", cluster.Name, err)
	}

	// after this, we are validated!
	return nil
}
