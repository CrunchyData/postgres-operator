package reloadservice

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
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
)

//  Reload ...
// pgo reload mycluster
// pgo reload all
// pgo reload --selector=name=mycluster
func Reload(request *msgs.ReloadRequest, ns string) msgs.ReloadResponse {
	resp := msgs.ReloadResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("Reload %v", request)

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		clusterList := crv1.PgclusterList{}

		err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, request.Selector, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
			resp.Results = append(resp.Results, "no clusters found with that selector")
			return resp
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			request.Args = newargs
		}

	}

	for _, arg := range request.Args {
		log.Debugf("reload called for %s", arg)

		cluster := crv1.Pgcluster{}
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, arg, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = arg + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		var podList *v1.PodList
		selector := config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name
		podList, err = kubeapi.GetPods(apiserver.Clientset, selector, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		if len(podList.Items) == 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "no pods found using " + selector
			return resp
		}

		err = reload(&podList.Items[0], apiserver.Clientset, apiserver.RESTClient, ns, apiserver.RESTConfig, ns)
		if err != nil {
			if !strings.Contains(err.Error(), "normal") {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		resp.Results = append(resp.Results, "reload performed on "+arg)
	}

	return resp
}

// run reload.sh on the postgres pod, remember we are assuming
// first container in the pod is always the postgres container.
func reload(
	pod *v1.Pod,
	clientset *kubernetes.Clientset,
	client *rest.RESTClient, namespace string, restconfig *rest.Config, ns string) error {
	//get the target pod that matches the replica-name=target

	command := make([]string, 1)
	command[0] = "/opt/cpm/bin/reload.sh"

	log.Debugf("running Exec with namespace=[%s] podname=[%s] container name=[%s]", namespace, pod.Name, pod.Spec.Containers[0].Name)
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, apiserver.Clientset, command, pod.Spec.Containers[0].Name, pod.Name, ns, nil)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("stdout=[%s] stderr=[%s]", stdout, stderr)

	return err
}
