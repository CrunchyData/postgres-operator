package pgadminservice

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/pgadmin"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pgAdminServiceSuffix = "-pgadmin"

// CreatePgAdmin ...
// pgo create pgadmin mycluster
// pgo create pgadmin --selector=name=mycluster
func CreatePgAdmin(request *msgs.CreatePgAdminRequest, ns, pgouser string) msgs.CreatePgAdminResponse {
	ctx := context.TODO()
	var err error
	resp := msgs.CreatePgAdminResponse{
		Status:  msgs.Status{Code: msgs.Ok},
		Results: []string{},
	}

	log.Debugf("createPgAdmin selector is [%s]", request.Selector)

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.Args, request.Selector)
	if err != nil {
		resp.SetError(err.Error())
		return resp
	}

	for _, cluster := range clusterList.Items {
		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = cluster.Name + msgs.UpgradeError
			return resp
		}

		// set the default pgadmin storage value first (i.e. the primary storage value)
		if cluster.Spec.PGAdminStorage == (crv1.PgStorageSpec{}) {
			cluster.Spec.PGAdminStorage = cluster.Spec.PrimaryStorage
		}

		// if a value for pgAdmin storage config is provided with the request, validate it here
		// if it is not valid, return now
		if request.StorageConfig != "" && !apiserver.IsValidStorageName(request.StorageConfig) {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf("%q storage config provided invalid", request.StorageConfig)
			return resp
		}

		// Extract parameters for the optional pgAdmin storage. server configuration and
		// request parameters are all optional.
		// Now, set the main configured value
		cluster.Spec.PGAdminStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.PGAdminStorage)

		// set the requested value, if provided
		if request.StorageConfig != "" {
			cluster.Spec.PGAdminStorage, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
		}

		// if the pgAdmin PVCSize is overwritten, update the cluster spec with this value
		if request.PVCSize != "" {
			// if the PVCSize is set to a customized value, ensure that it is recognizable by Kubernetes
			if err := apiserver.ValidateQuantity(request.PVCSize); err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = fmt.Sprintf(apiserver.ErrMessagePVCSize, request.PVCSize, err.Error())
				return resp
			}
			log.Debugf("pgAdmin PVC Size is overwritten to be [%s]", request.PVCSize)
			cluster.Spec.PGAdminStorage.Size = request.PVCSize
		}

		log.Debugf("adding pgAdmin to cluster [%s]", cluster.Name)

		// generate the pgtask, starting with spec
		spec := crv1.PgtaskSpec{
			Name:        fmt.Sprintf("%s-%s", config.LABEL_PGADMIN_TASK_ADD, cluster.Name),
			TaskType:    crv1.PgtaskPgAdminAdd,
			StorageSpec: cluster.Spec.PGAdminStorage,
			Parameters: map[string]string{
				config.LABEL_PGADMIN_TASK_CLUSTER: cluster.Name,
			},
		}

		task := &crv1.Pgtask{
			ObjectMeta: metav1.ObjectMeta{
				Name: spec.Name,
				Labels: map[string]string{
					config.LABEL_PG_CLUSTER:       cluster.Name,
					config.LABEL_PGADMIN_TASK_ADD: "true",
					config.LABEL_PGOUSER:          pgouser,
				},
			},
			Spec: spec,
		}

		if _, err := apiserver.Clientset.CrunchydataV1().Pgtasks(cluster.Namespace).Create(ctx, task, metav1.CreateOptions{}); err != nil {
			log.Error(err)
			resp.SetError("error creating tasks for one or more clusters")
			resp.Results = append(resp.Results, fmt.Sprintf("%s: error - %s", cluster.Name, err.Error()))
			continue
		} else {
			resp.Results = append(resp.Results, fmt.Sprintf("%s pgAdmin addition scheduled", cluster.Name))
		}
	}

	return resp
}

// DeletePgAdmin ...
// pgo delete pgadmin mycluster
// pgo delete pgadmin --selector=name=mycluster
func DeletePgAdmin(request *msgs.DeletePgAdminRequest, ns string) msgs.DeletePgAdminResponse {
	ctx := context.TODO()
	var err error
	resp := msgs.DeletePgAdminResponse{
		Status:  msgs.Status{Code: msgs.Ok},
		Results: []string{},
	}

	log.Debugf("deletePgAdmin selector is [%s]", request.Selector)

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.Args, request.Selector)
	if err != nil {
		resp.SetError(err.Error())
		return resp
	}

	for _, cluster := range clusterList.Items {
		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = cluster.Name + msgs.UpgradeError
			return resp
		}

		log.Debugf("deleting pgAdmin from cluster [%s]", cluster.Name)

		// generate the pgtask, starting with spec
		spec := crv1.PgtaskSpec{
			Name:     config.LABEL_PGADMIN_TASK_DELETE + "-" + cluster.Name,
			TaskType: crv1.PgtaskPgAdminDelete,
			Parameters: map[string]string{
				config.LABEL_PGADMIN_TASK_CLUSTER: cluster.Name,
			},
		}

		task := &crv1.Pgtask{
			ObjectMeta: metav1.ObjectMeta{
				Name: spec.Name,
				Labels: map[string]string{
					config.LABEL_PG_CLUSTER:          cluster.Name,
					config.LABEL_PGADMIN_TASK_DELETE: "true",
				},
			},
			Spec: spec,
		}

		if _, err := apiserver.Clientset.CrunchydataV1().Pgtasks(cluster.Namespace).Create(ctx, task, metav1.CreateOptions{}); err != nil {
			log.Error(err)
			resp.SetError("error creating tasks for one or more clusters")
			resp.Results = append(resp.Results, fmt.Sprintf("%s: error - %s", cluster.Name, err.Error()))
			return resp
		} else {
			resp.Results = append(resp.Results, cluster.Name+" pgAdmin delete scheduled")
		}

	}

	return resp
}

// ShowPgAdmin gets information about a PostgreSQL cluster's pgAdmin
// deployment
//
// pgo show pgadmin
// pgo show pgadmin --selector
func ShowPgAdmin(request *msgs.ShowPgAdminRequest, namespace string) msgs.ShowPgAdminResponse {
	ctx := context.TODO()
	log.Debugf("show pgAdmin called, cluster [%v], selector [%s]", request.ClusterNames, request.Selector)

	response := msgs.ShowPgAdminResponse{
		Results: []msgs.ShowPgAdminDetail{},
		Status:  msgs.Status{Code: msgs.Ok},
	}

	// try to get the list of clusters. if there is an error, put it into the
	// status and return
	clusterList, err := getClusterList(request.Namespace, request.ClusterNames, request.Selector)
	if err != nil {
		response.SetError(err.Error())
		return response
	}

	// iterate through the list of clusters to get the relevant pgAdmin
	// information about them
	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]
		result := msgs.ShowPgAdminDetail{
			ClusterName: cluster.Spec.Name,
			HasPgAdmin:  true,
		}

		// first, check if the cluster has the pgAdmin label. If it does not, we
		// add it to the list and keep iterating
		clusterLabels := cluster.GetLabels()

		if clusterLabels[config.LABEL_PGADMIN] != "true" {
			result.HasPgAdmin = false
			response.Results = append(response.Results, result)
			continue
		}

		// This takes advantage of pgadmin deployment and pgadmin service
		// sharing a name that is clustername + pgAdminServiceSuffix
		service, err := apiserver.Clientset.
			CoreV1().Services(cluster.Namespace).
			Get(ctx, cluster.Name+pgAdminServiceSuffix, metav1.GetOptions{})
		if err != nil {
			response.SetError(err.Error())
			return response
		}

		result.ServiceClusterIP = service.Spec.ClusterIP
		result.ServiceName = service.Name
		if len(service.Spec.ExternalIPs) > 0 {
			result.ServiceExternalIP = service.Spec.ExternalIPs[0]
		}
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			result.ServiceExternalIP = service.Status.LoadBalancer.Ingress[0].IP
		}

		// In the future, construct results to contain individual error stati
		// for now log and return empty content if encountered
		qr, err := pgadmin.GetPgAdminQueryRunner(apiserver.Clientset, apiserver.RESTConfig, cluster)
		if err != nil {
			log.Error(err)
			continue
		} else if qr != nil {
			names, err := pgadmin.GetUsernames(qr)
			if err != nil {
				log.Error(err)
				continue
			}
			result.Users = names
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
		cl, err := apiserver.Clientset.
			CrunchydataV1().Pgclusters(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: selector})
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
