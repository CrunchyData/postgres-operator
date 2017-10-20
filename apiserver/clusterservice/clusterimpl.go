package clusterservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"k8s.io/apimachinery/pkg/labels"
)

// ShowCluster ...
func ShowCluster(namespace, name, selector string) msgs.ShowClusterResponse {
	var err error

	response := msgs.ShowClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	myselector := labels.Everything()
	log.Debug("selector is " + selector)
	if selector != "" {
		name = "all"
		myselector, err = labels.Parse(selector)
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	log.Debugf("label selector is [%v]\n", myselector)

	if name == "all" {
		//get a list of all clusters
		err := apiserver.RestClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(namespace).
			LabelsSelectorParam(myselector).
			Do().Into(&response.ClusterList)
		if err != nil {
			log.Error("error getting list of clusters" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debug("clusters found len is %d\n", len(response.ClusterList.Items))
	} else {
		cluster := crv1.Pgcluster{}
		err := apiserver.RestClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(namespace).
			Name(name).
			Do().Into(&cluster)
		if err != nil {
			log.Error("error getting cluster" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.ClusterList.Items = make([]crv1.Pgcluster, 1)
		response.ClusterList.Items[0] = cluster
	}

	return response

}
