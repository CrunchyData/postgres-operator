package failoverservice

/*
Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/labels"
)

//  CreateFailover ...
// pgo failover mycluster
// pgo failover all
// pgo failover --selector=name=mycluster
func CreateFailover(request *msgs.CreateFailoverRequest) msgs.CreateFailoverResponse {
	var err error
	resp := msgs.CreateFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	if request.Target != "" {
		_, err = validateDeploymentName(request.Target, request.ClusterName)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	//get the clusters list
	_, err = validateClusterName(request.ClusterName)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debugf("create failover called for %s", request.ClusterName)

	// Create a pgtask
	spec := crv1.PgtaskSpec{}
	spec.Name = request.ClusterName + "-" + util.LABEL_FAILOVER

	// previous failovers will leave a pgtask so remove it first
	kubeapi.Deletepgtask(apiserver.RESTClient, spec.Name, apiserver.Namespace)

	spec.TaskType = crv1.PgtaskFailover
	spec.Parameters = make(map[string]string)
	spec.Parameters[request.ClusterName] = request.ClusterName

	labels := make(map[string]string)
	labels["target"] = request.Target
	labels[util.LABEL_PG_CLUSTER] = request.ClusterName

	if request.AutofailReplaceReplica != "" {
		if request.AutofailReplaceReplica == "true" ||
			request.AutofailReplaceReplica == "false" {
			labels[util.LABEL_AUTOFAIL_REPLACE_REPLICA] = request.AutofailReplaceReplica
		} else {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "true or false value required for --autofail-replace-replica flag"
			return resp
		}
	}

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	err = kubeapi.Createpgtask(apiserver.RESTClient,
		newInstance, apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, "created Pgtask (failover) for cluster "+request.ClusterName)

	return resp
}

//  QueryFailover ...
// pgo failover mycluster --query
func QueryFailover(name string) msgs.QueryFailoverResponse {
	var err error
	resp := msgs.QueryFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)
	resp.Targets = make([]msgs.FailoverTargetSpec, 0)

	//var deployment *v1beta1.Deployment

	//get the clusters list
	_, err = validateClusterName(name)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debugf("query failover called for %s", name)

	//get pods using selector service-name=clusterName-replica

	selector := util.LABEL_SERVICE_NAME + "=" + name + "-replica"
	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, apiserver.Namespace)
	if kerrors.IsNotFound(err) {
		log.Debug("no replicas found")
		resp.Status.Msg = "no replicas found for " + name
		return resp
	} else if err != nil {
		log.Error("error getting pods " + err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	deploymentNameList := ""
	for _, p := range pods.Items {
		deploymentNameList = deploymentNameList + p.ObjectMeta.Labels[util.LABEL_DEPLOYMENT_NAME] + ","
	}
	log.Debugf("deployment name list is %s", deploymentNameList)

	//get failover targets for this cluster
	//deployments with --selector=primary=false,pg-cluster=ClusterName

	//selector := util.LABEL_PRIMARY + "=false," + util.LABEL_PG_CLUSTER + "=" + name
	selector = util.LABEL_DEPLOYMENT_NAME + " in (" + deploymentNameList + ")"

	var deployments *v1beta1.DeploymentList
	deployments, err = kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
	if kerrors.IsNotFound(err) {
		log.Debug("no replicas found")
		resp.Status.Msg = "no replicas found for " + name
		return resp
	} else if err != nil {
		log.Error("error getting deployments " + err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debugf("deps len %d", len(deployments.Items))
	for _, dep := range deployments.Items {
		log.Debugf("found %s", dep.Name)
		target := msgs.FailoverTargetSpec{}
		target.Name = dep.Name

		target.ReceiveLocation, target.ReplayLocation = util.GetRepStatus(apiserver.RESTClient, apiserver.Clientset, &dep, apiserver.Namespace)
		//get the pod status
		target.ReadyStatus, target.Node = apiserver.GetPodStatus(dep.Name)
		//get the rep status
		resp.Targets = append(resp.Targets, target)
	}

	return resp
}

func validateClusterName(clusterName string) (*crv1.Pgcluster, error) {
	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient,
		&cluster, clusterName, apiserver.Namespace)
	if !found {
		return &cluster, errors.New("no cluster found named " + clusterName)
	}

	return &cluster, err
}

func validateDeploymentName(deployName, clusterName string) (*v1beta1.Deployment, error) {

	deployment, found, err := kubeapi.GetDeployment(apiserver.Clientset, deployName, apiserver.Namespace)
	if !found {
		return deployment, errors.New("no target found named " + deployName)
	}

	//make sure the primary is not being selected by the user
	if deployment.ObjectMeta.Labels[util.LABEL_SERVICE_NAME] == clusterName {
		return deployment, errors.New("deployment primary can not be selected as failover target")
	}

	return deployment, err

}
