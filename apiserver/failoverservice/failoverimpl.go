package failoverservice

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
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//  CreateFailover ...
// pgo failover mycluster
// pgo failover all
// pgo failover --selector=name=mycluster
func CreateFailover(request *msgs.CreateFailoverRequest, ns string) msgs.CreateFailoverResponse {
	var err error
	resp := msgs.CreateFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	if request.Target != "" {
		_, err = validateDeploymentName(request.Target, request.ClusterName, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	//get the clusters list
	var theCRD *crv1.Pgcluster
	theCRD, err = validateClusterName(request.ClusterName, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	err = checkAutofail(theCRD)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debugf("create failover called for %s", request.ClusterName)

	// Create a pgtask
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = request.ClusterName + "-" + config.LABEL_FAILOVER

	// previous failovers will leave a pgtask so remove it first
	kubeapi.Deletepgtask(apiserver.RESTClient, spec.Name, ns)

	spec.TaskType = crv1.PgtaskFailover
	spec.Parameters = make(map[string]string)
	spec.Parameters[request.ClusterName] = request.ClusterName

	labels := make(map[string]string)
	labels["target"] = request.Target
	labels[config.LABEL_PG_CLUSTER] = request.ClusterName

	if request.AutofailReplaceReplica != "" {
		if request.AutofailReplaceReplica == "true" ||
			request.AutofailReplaceReplica == "false" {
			labels[config.LABEL_AUTOFAIL_REPLACE_REPLICA] = request.AutofailReplaceReplica
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
		newInstance, ns)
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
func QueryFailover(name, ns string) msgs.QueryFailoverResponse {
	var err error
	resp := msgs.QueryFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)
	resp.Targets = make([]msgs.FailoverTargetSpec, 0)

	//get the clusters list
	var theCRD *crv1.Pgcluster

	theCRD, err = validateClusterName(name, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}
	err = checkAutofail(theCRD)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debugf("query failover called for %s", name)

	var nodes []string

	if apiserver.Pgo.Pgo.PreferredFailoverNode != "" {
		log.Debug("PreferredFailoverNode is set to %s", apiserver.Pgo.Pgo.PreferredFailoverNode)
		nodes, err = util.GetPreferredNodes(apiserver.Clientset, apiserver.Pgo.Pgo.PreferredFailoverNode, ns)
		if err != nil {
			log.Error("error getting preferred nodes " + err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	} else {
		log.Debug("PreferredFailoverNode is not set ")
	}

	//get pods using selector service-name=clusterName-replica

	selector := config.LABEL_SERVICE_NAME + "=" + name + "-replica"
	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
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
		deploymentNameList = deploymentNameList + p.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME] + ","
	}
	log.Debugf("deployment name list is %s", deploymentNameList)

	//get failover targets for this cluster
	//deployments with --selector=primary=false,pg-cluster=ClusterName

	selector = config.LABEL_DEPLOYMENT_NAME + " in (" + deploymentNameList + ")"

	var deployments *v1.DeploymentList
	deployments, err = kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
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

		target.ReceiveLocation, target.ReplayLocation, target.Node, err = util.GetRepStatus(apiserver.RESTClient, apiserver.Clientset, &dep, ns, apiserver.Pgo.Cluster.Port)
		if err != nil {
			log.Error("error getting rep status " + err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		//get the pod status
		target.ReadyStatus, target.Node = apiserver.GetPodStatus(dep.Name, ns)
		if preferredNode(nodes, target.Node) {
			target.PreferredNode = true
		}
		//get the rep status
		resp.Targets = append(resp.Targets, target)
	}

	return resp
}

func validateClusterName(clusterName, ns string) (*crv1.Pgcluster, error) {
	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient,
		&cluster, clusterName, ns)
	if !found {
		return &cluster, errors.New("no cluster found named " + clusterName)
	}

	return &cluster, err
}

func validateDeploymentName(deployName, clusterName, ns string) (*v1.Deployment, error) {

	deployment, found, err := kubeapi.GetDeployment(apiserver.Clientset, deployName, ns)
	if !found {
		return deployment, errors.New("no target found named " + deployName)
	}

	//make sure the primary is not being selected by the user
	if deployment.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
		return deployment, errors.New("deployment primary can not be selected as failover target")
	}

	return deployment, err

}
func preferredNode(nodes []string, targetNode string) bool {
	for _, n := range nodes {
		if n == targetNode {
			return true
		}
	}
	return false
}

func checkAutofail(cluster *crv1.Pgcluster) error {
	var err error
	labels := cluster.ObjectMeta.Labels
	failLabel := labels[config.LABEL_AUTOFAIL]
	if failLabel == "true" {
		return errors.New("autofail flag is set to true, manual failover requires autofail to be set to false, use pgo update to disable autofail.")
	}
	return err
}
