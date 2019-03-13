package pgpoolservice

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
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePgpool ...
// pgo create pgpool mycluster
// pgo create pgpool --selector=name=mycluster
func CreatePgpool(request *msgs.CreatePgpoolRequest, ns string) msgs.CreatePgpoolResponse {
	var err error
	resp := msgs.CreatePgpoolResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("createPgpool selector is [%s]", request.Selector)

	if request.Selector == "" && len(request.Args) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "either a cluster list or a selector needs to be supplied for this command"
		return resp
	}

	if request.PgpoolSecret != "" {
		var found bool
		_, found, err = kubeapi.GetSecret(apiserver.Clientset, request.PgpoolSecret, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "--pgpool-secret specified secret " + request.PgpoolSecret + " not found"
			return resp
		}

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
				resp.Status.Msg = request.Args[i] + " was not found"
				resp.Status.Code = msgs.Error
				return resp
			}
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
			clusterList.Items = append(clusterList.Items, argCluster)

		}
	}

	log.Debugf("createPgpool clusters found len is %d", len(clusterList.Items))
	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found that match request selector or arguments"
		return resp
	}

	for _, cluster := range clusterList.Items {
		log.Debugf("adding pgpool to cluster [%s]", cluster.Name)

		spec := crv1.PgtaskSpec{}
		spec.Namespace = ns
		spec.Name = config.LABEL_PGPOOL_TASK_ADD + "-" + cluster.Name
		spec.TaskType = crv1.PgtaskAddPgpool
		spec.StorageSpec = crv1.PgStorageSpec{}
		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PGPOOL_TASK_CLUSTER] = cluster.Name
		spec.Parameters[config.LABEL_PGPOOL_SECRET] = request.PgpoolSecret

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: spec.Name,
			},
			Spec: spec,
		}

		newInstance.ObjectMeta.Labels = make(map[string]string)
		newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = cluster.Name
		newInstance.ObjectMeta.Labels[config.LABEL_PGPOOL_TASK_ADD] = "true"

		//check if this cluster already has a pgpool
		if cluster.Spec.UserLabels[config.LABEL_PGPOOL] == "true" {
			resp.Results = append(resp.Results, cluster.Name+" already has pgpool added")
			resp.Status.Code = msgs.Error
		} else {
			err := kubeapi.Createpgtask(apiserver.RESTClient,
				newInstance, ns)
			if err != nil {
				log.Error(err)
				resp.Results = append(resp.Results, "error adding pgpool for "+cluster.Name+err.Error())
			} else {
				resp.Results = append(resp.Results, "pgpool added for "+cluster.Name)
			}
		}

	}

	return resp

}

// DeletePgpool ...
// pgo delete pgpool mycluster
// pgo delete pgpool --selector=name=mycluster
func DeletePgpool(request *msgs.DeletePgpoolRequest, ns string) msgs.DeletePgpoolResponse {
	var err error
	resp := msgs.DeletePgpoolResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("deletePgpool selector is [%s]", request.Selector)

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
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
			clusterList.Items = append(clusterList.Items, argCluster)

		}
	}

	log.Debugf("deletePgpool clusters found len is %d", len(clusterList.Items))
	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found that match request selector or arguments"
		return resp
	}

	for _, cluster := range clusterList.Items {
		log.Debugf("deleting pgpool from cluster [%s]", cluster.Name)

		spec := crv1.PgtaskSpec{}
		spec.Namespace = ns
		spec.Name = config.LABEL_PGPOOL_TASK_DELETE + "-" + cluster.Name
		spec.TaskType = crv1.PgtaskDeletePgpool
		spec.StorageSpec = crv1.PgStorageSpec{}
		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PGPOOL_TASK_CLUSTER] = cluster.Name

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: spec.Name,
			},
			Spec: spec,
		}

		newInstance.ObjectMeta.Labels = make(map[string]string)
		newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = cluster.Name
		newInstance.ObjectMeta.Labels[config.LABEL_PGPOOL_TASK_DELETE] = "true"

		err := kubeapi.Createpgtask(apiserver.RESTClient,
			newInstance, ns)
		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Results = append(resp.Results, cluster.Name+err.Error())
			return resp
		} else {
			resp.Results = append(resp.Results, cluster.Name+" pgpool deleted")
		}

	}

	return resp

}
