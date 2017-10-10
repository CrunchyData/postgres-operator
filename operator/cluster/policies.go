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

package cluster

import (
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api/v1"
	"k8s.io/api/core/v1"

	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/cache"
	"os"
	"strings"
	"time"
)

func ProcessPolicies(clientset *kubernetes.Clientset, restclient *rest.RESTClient, stopchan chan struct{}, namespace string) {

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster,master"}
	fw, err := clientset.Core().Pods(namespace).Watch(lo)
	if err != nil {
		log.Error("fatal error in ProcessPolicies " + err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infof("got a processpolicies watch event %v\n", event.Type)

		switch event.Type {
		case watch.Added:
			//deployment := event.Object.(*v1beta1.Deployment)
			//log.Infof("deployment processpolicy added=%s\n", dep.Name)
		case watch.Deleted:
			//deployment := event.Object.(*v1beta1.Deployment)
			//log.Infof("deployment processpolicy deleted=%s\n", deployment.Name)
		case watch.Error:
			log.Infof("deployment processpolicy error event")
		case watch.Modified:
			pod := event.Object.(*v1.Pod)
			//log.Infof("deployment processpolicy modified=%s\n", deployment.Name)
			//log.Infof("status available replicas=%d\n", deployment.Status.AvailableReplicas)
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

func applyPolicies(namespace string, clientset *kubernetes.Clientset, restclient *rest.RESTClient, clusterName string) {
	//dep *v1beta1.Deployment
	//get the crv1 which holds the requested labels if any
	cl := crv1.Pgcluster{}
	err := restclient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Name(clusterName).
		Do().
		Into(&cl)
	if err == nil {
	} else if kerrors.IsNotFound(err) {
		log.Error("could not get cluster in policy processing using " + clusterName)
		return
	} else {
		log.Error("error in policy processing " + err.Error())
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

	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY found in policy apply for " + clusterName)
		return
	}

	err = strategy.UpdatePolicyLabels(clientset, clusterName, namespace, labels)

	//err = util.UpdateDeploymentLabels(clientset, clusterName, namespace, labels)
	if err != nil {
		log.Error(err)
	}
}

func AddPolicylog(clientset *kubernetes.Clientset, restclient *rest.RESTClient, policylog *crv1.Pgpolicylog, namespace string) {
	policylogname := policylog.Spec.PolicyName + policylog.Spec.ClusterName
	log.Infof("policylog added=%s\n", policylogname)

	labels := make(map[string]string)

	err := util.ExecPolicy(clientset, restclient, namespace, policylog.Spec.PolicyName, policylog.Spec.ClusterName)
	if err != nil {
		log.Error(err)
	} else {
		labels[policylog.Spec.PolicyName] = "pgpolicy"
	}

	cl := crv1.Pgcluster{}
	err = restclient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Name(policylog.Spec.ClusterName).
		Do().
		Into(&cl)
	if err != nil {
		log.Error("error getting cluster crv1 in addPolicylog " + policylog.Spec.ClusterName)
		return

	}
	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster creation" + cl.Spec.STRATEGY)
		return
	}

	err = strategy.UpdatePolicyLabels(clientset, policylog.Spec.ClusterName, namespace, labels)

	//update the deployment's labels to show applied policies
	//err = util.UpdateDeploymentLabels(clientset, policylog.Spec.ClusterName, namespace, labels)
	if err != nil {
		log.Error(err)
	}

	//update the policylog with applydate and status
	err = util.Patch(restclient, "/spec/status", crv1.UPGRADE_COMPLETED_STATUS, crv1.PgpolicylogResourcePlural, policylogname, namespace)
	if err != nil {
		log.Error("error in policylog status patch " + err.Error())
	}

	t := time.Now()
	err = util.Patch(restclient, "/spec/applydate", t.Format("2006-01-02-15:04:05"), crv1.PgpolicylogResourcePlural, policylogname, namespace)
	if err != nil {
		log.Error("error in policylog applydate patch " + err.Error())
	}

}

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
func getClusterName(pod *v1.Pod) string {
	var clusterName string
	labels := pod.ObjectMeta.Labels
	for k, v := range labels {
		if k == "pg-cluster" {
			clusterName = v
		}
	}

	return clusterName
}
