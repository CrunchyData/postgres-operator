package upgradeservice

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
	"k8s.io/apimachinery/pkg/labels"
)

// CreateUpgrade ...
func CreateUpgrade(request *msgs.CreateUpgradeRequest, ns string) msgs.CreateUpgradeResponse {
	response := msgs.CreateUpgradeResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 1)

	log.Debugf("createUpgrade called %v", request)

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		myselector, err := labels.Parse(request.Selector)
		if err != nil {
			log.Error("could not parse selector flag")
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("myselector is %s", myselector.String())

		//get the clusters list
		clusterList := crv1.PgclusterList{}
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
			response.Status.Msg = "no clusters found"
			return response
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			request.Args = newargs
		}
	}

	for _, clusterName := range request.Args {
		log.Debugf("create upgrade called for %s", clusterName)

		//build the pgtask for the minor upgrade
		spec := crv1.PgtaskSpec{}
		spec.TaskType = crv1.PgtaskMinorUpgrade
		spec.Status = "requested"
		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
		spec.Name = clusterName + "-minor-upgrade"
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

		// remove any existing pgtask for this minor upgrade
		result := crv1.Pgtask{}
		found, err := kubeapi.Getpgtask(apiserver.RESTClient,
			&result, spec.Name, ns)
		if found {
			err := kubeapi.Deletepgtask(apiserver.RESTClient, spec.Name, ns)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
		}

		//validate the cluster name
		cl := crv1.Pgcluster{}
		found, err = kubeapi.Getpgcluster(apiserver.RESTClient,
			&cl, clusterName, ns)
		if !found {
			response.Status.Code = msgs.Error
			response.Status.Msg = clusterName + " is not a valid pgcluster"
			return response
		}

		//figure out what version we are upgrading to
		imageToUpgradeTo := apiserver.Pgo.Cluster.CCPImageTag
		if request.CCPImageTag != "" {
			imageToUpgradeTo = request.CCPImageTag
		}
		if imageToUpgradeTo == cl.Spec.CCPImageTag {
			response.Status.Code = msgs.Error
			response.Status.Msg = "can not upgrade to the same image tag " + imageToUpgradeTo + " " + cl.Spec.CCPImageTag
			return response
		}
		log.Debugf("upgrading to image tag %s", imageToUpgradeTo)
		spec.Parameters["CCPImageTag"] = imageToUpgradeTo

		// Create an instance of our CRD
		err = kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		msg := "created minor upgrade task for " + clusterName
		response.Results = append(response.Results, msg)

	}

	return response
}
