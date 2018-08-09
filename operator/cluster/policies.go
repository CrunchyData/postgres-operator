package cluster

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"strings"
)

// ProcessPolicies ...
func ProcessPolicies(clientset *kubernetes.Clientset, restclient *rest.RESTClient, stopchan chan struct{}, namespace string) {

	lo := meta_v1.ListOptions{LabelSelector: util.LABEL_PG_CLUSTER + ",primary"}
	fw, err := clientset.Core().Pods(namespace).Watch(lo)
	if err != nil {
		log.Error("fatal error in ProcessPolicies " + err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infof("got a processpolicies watch event %v\n", event.Type)

		switch event.Type {
		case watch.Added:
		case watch.Deleted:
		case watch.Error:
			log.Infof("deployment processpolicy error event")
		case watch.Modified:
			pod := event.Object.(*v1.Pod)
			ready, restarts := podReady(pod)
			if restarts > 0 {
				log.Info("restarts > 0, will not apply policies again to " + pod.Name)
			} else if ready {
				clusterName := getClusterName(pod)
				applyPolicies(namespace, clientset, restclient, clusterName)
			}

		default:
			log.Infoln("processpolices unknown watch event %v\n", event.Type)
		}

		return false, nil
	})

	if err4 != nil {
		log.Error("error in ProcessPolicies " + err4.Error())
	}

}

// applyPolicies ...
func applyPolicies(namespace string, clientset *kubernetes.Clientset, restclient *rest.RESTClient, clusterName string) {
	//get the crv1 which holds the requested labels if any
	cl := crv1.Pgcluster{}
	_, err := kubeapi.Getpgcluster(restclient, &cl, clusterName, namespace)
	if err != nil {
		return
	}

	if cl.Spec.Policies == "" {
		log.Debug("no policies to apply to " + clusterName)
		return
	}
	log.Debug("policies to apply to " + clusterName + " are " + cl.Spec.Policies)
	policies := strings.Split(cl.Spec.Policies, ",")

	//apply the policies
	labels := make(map[string]string)

	for _, v := range policies {
		err = util.ExecPolicy(clientset, restclient, namespace, v, cl.Spec.Name)
		if err != nil {
			log.Error(err)
		} else {
			labels[v] = "pgpolicy"
		}

	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy found in policy apply for " + clusterName)
		return
	}

	err = strategy.UpdatePolicyLabels(clientset, clusterName, namespace, labels)

	//err = util.UpdateDeploymentLabels(clientset, clusterName, namespace, labels)
	if err != nil {
		log.Error(err)
	}
}

// podReady ...
func podReady(pod *v1.Pod) (bool, int32) {
	var restartCount int32
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		restartCount = restartCount + stat.RestartCount
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	log.Debugf(" %s %d/%d", pod.Name, readyCount, containerCount)
	if readyCount > 0 && readyCount == containerCount {
		return true, restartCount
	}
	return false, restartCount

}

// getClusterName ...
func getClusterName(pod *v1.Pod) string {
	var clusterName string
	labels := pod.ObjectMeta.Labels
	for k, v := range labels {
		if k == util.LABEL_PG_CLUSTER {
			clusterName = v
		}
	}

	return clusterName
}
