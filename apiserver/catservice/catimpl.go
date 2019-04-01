package catservice

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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
)

// pgo cat mycluster /pgdata/mycluster/postgresql.conf
func Cat(request *msgs.CatRequest, ns string) msgs.CatResponse {
	resp := msgs.CatResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("Cat %v", request)

	if len(request.Args) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no cluster name was passed"
		return resp
	}

	clusterName := request.Args[0]

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, clusterName, ns)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = clusterName + " was not found, verify cluster name"
		return resp
	}

	err = validateArgs(request.Args)
	if err != nil {
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

	clusterName = request.Args[0]
	log.Debugf("cat called for cluster %s", clusterName)

	var results string
	results, err = cat(&podList.Items[0], apiserver.Clientset, apiserver.RESTClient, apiserver.RESTConfig, ns, request.Args)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, results)

	return resp
}

// run cat on the postgres pod, remember we are assuming
// first container in the pod is always the postgres container.
func cat(
	pod *v1.Pod,
	clientset *kubernetes.Clientset,
	client *rest.RESTClient, restconfig *rest.Config, ns string, args []string) (string, error) {

	command := make([]string, 0)
	command = append(command, "cat")
	for i := 1; i < len(args); i++ {
		command = append(command, args[i])
	}

	log.Debugf("running Exec in namespace=[%s] podname=[%s] container name=[%s] command=[%v]", ns, pod.Name, pod.Spec.Containers[0].Name, command)

	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, apiserver.Clientset, command, pod.Spec.Containers[0].Name, pod.Name, ns, nil)
	if err != nil {
		log.Error(err)
		return "error in exec to pod", err
	}
	log.Debugf("stdout=[%s] stderr=[%s]", stdout, stderr)

	return stdout, err
}

//make sure the parameters to the cat command dont' container mischief
func validateArgs(args []string) error {
	var err error
	var bad = "&|;>"

	for i := 1; i < len(args); i++ {
		if strings.ContainsAny(args[i], bad) {
			return errors.New(args[i] + " contains non-allowed characters [" + bad + "]")
		}
	}
	return err
}
