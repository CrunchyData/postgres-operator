package pgbouncerservice

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePgbouncer ...
// pgo create pgbouncer mycluster
// pgo create pgbouncer --selector=name=mycluster
func CreatePgbouncer(request *msgs.CreatePgbouncerRequest, ns string) msgs.CreatePgbouncerResponse {
	var err error
	resp := msgs.CreatePgbouncerResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("createPgbouncer selector is [%s]", request.Selector)

	if request.Selector == "" && len(request.Args) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "either a cluster list or a selector needs to be supplied for this command"
		return resp
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	if request.Selector != "" {
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	if len(request.Args) > 0 {
		argCluster := crv1.Pgcluster{}

		for i := 0; i < len(request.Args); i++ {
			found, err := kubeapi.Getpgcluster(apiserver.RESTClient,
				&argCluster, request.Args[i], ns)

			if !found {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.Args[i] + " not found"
				return resp
			}
			if !found {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
			clusterList.Items = append(clusterList.Items, argCluster)

		}
	}

	log.Debugf("createPgbouncer clusters found len is %d", len(clusterList.Items))
	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found that match request selector or arguments"
		return resp
	}

	for _, cluster := range clusterList.Items {
		log.Debugf("adding pgbouncer to cluster [%s]", cluster.Name)

		pgbouncerpass := request.PgbouncerPass

		// create pgbouncer password if not specified by user.
		if !(len(pgbouncerpass) > 0) {
			pgbouncerpass = util.GeneratePassword(10)
		}

		spec := crv1.PgtaskSpec{}
		spec.Namespace = ns
		spec.Name = config.LABEL_PGBOUNCER_TASK_ADD + "-" + cluster.Name
		spec.TaskType = crv1.PgtaskAddPgbouncer
		spec.StorageSpec = crv1.PgStorageSpec{}
		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER] = cluster.Name
		spec.Parameters[config.LABEL_PGBOUNCER_USER] = request.PgbouncerUser
		spec.Parameters[config.LABEL_PGBOUNCER_PASS] = request.PgbouncerPass

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: spec.Name,
			},
			Spec: spec,
		}

		newInstance.ObjectMeta.Labels = make(map[string]string)
		newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = cluster.Name
		newInstance.ObjectMeta.Labels[config.LABEL_PGBOUNCER_TASK_ADD] = "true"

		//check if this cluster already has a pgbouncer
		// if cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true" {
		if cluster.Labels[config.LABEL_PGBOUNCER] == "true" {
			resp.Results = append(resp.Results, cluster.Name+" already has pgbouncer added")
			resp.Status.Code = msgs.Error
		} else {
			err := kubeapi.Createpgtask(apiserver.RESTClient,
				newInstance, ns)
			if err != nil {
				log.Error(err)
				resp.Results = append(resp.Results, err.Error())
				return resp
			} else {
				resp.Results = append(resp.Results, cluster.Name+" pgbouncer added")
			}
		}

	}

	return resp

}

// DeletePgbouncer ...
// pgo delete pgbouncer mycluster
// pgo delete pgbouncer --selector=name=mycluster
func DeletePgbouncer(request *msgs.DeletePgbouncerRequest, ns string) msgs.DeletePgbouncerResponse {
	var err error
	resp := msgs.DeletePgbouncerResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("deletePgbouncer selector is [%s]", request.Selector)

	if request.Selector == "" && len(request.Args) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "either a cluster list or a selector needs to be supplied for this command"
		return resp
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	if request.Selector != "" {
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	if len(request.Args) > 0 {
		argCluster := crv1.Pgcluster{}

		for i := 0; i < len(request.Args); i++ {
			found, err := kubeapi.Getpgcluster(apiserver.RESTClient,
				&argCluster, request.Args[i], ns)

			if !found || err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
			clusterList.Items = append(clusterList.Items, argCluster)

		}
	}

	log.Debugf("deletePgbouncer clusters found len is %d", len(clusterList.Items))
	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found that match request selector or arguments"
		return resp
	}

	for _, cluster := range clusterList.Items {
		log.Debugf("deleting pgbouncer from cluster [%s]", cluster.Name)

		spec := crv1.PgtaskSpec{}
		spec.Namespace = ns
		spec.Name = config.LABEL_PGBOUNCER_TASK_DELETE + "-" + cluster.Name
		spec.TaskType = crv1.PgtaskDeletePgbouncer
		spec.StorageSpec = crv1.PgStorageSpec{}
		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER] = cluster.Name

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: spec.Name,
			},
			Spec: spec,
		}

		newInstance.ObjectMeta.Labels = make(map[string]string)
		newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = cluster.Name
		newInstance.ObjectMeta.Labels[config.LABEL_PGBOUNCER_TASK_DELETE] = "true"

		err := kubeapi.Createpgtask(apiserver.RESTClient,
			newInstance, ns)
		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Results = append(resp.Results, err.Error())
			return resp
		} else {
			resp.Results = append(resp.Results, cluster.Name+" pgbouncer deleted")
		}

	}

	return resp

}
