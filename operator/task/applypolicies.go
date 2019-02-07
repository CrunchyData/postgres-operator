package task

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
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
)

// RemoveBackups ...
func ApplyPolicies(clusterName string, Clientset *kubernetes.Clientset, RESTClient *rest.RESTClient, ns string) {

	taskName := clusterName + "-policies"
	task := crv1.Pgtask{}
	task.Spec = crv1.PgtaskSpec{}
	task.Spec.Name = taskName

	found, err := kubeapi.Getpgtask(RESTClient, &task, taskName, ns)
	if found && err == nil {
		//apply those policies
		for k, _ := range task.Spec.Parameters {
			log.Debugf("applying policy %s to %s", k, clusterName)
			applyPolicy(Clientset, RESTClient, k, clusterName, ns)
		}
		//delete the pgtask to not redo this again
		kubeapi.Deletepgtask(RESTClient, taskName, ns)
	}
}

func applyPolicy(clientset *kubernetes.Clientset, restclient *rest.RESTClient, policyName, clusterName, ns string) {
	err := util.ExecPolicy(clientset, restclient, ns, policyName, clusterName)
	if err != nil {
		log.Error(err)
		return
	}

	labels := make(map[string]string)
	labels[policyName] = "pgpolicy"

	//look up the cluster CRD to get the strategy
	cl := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(restclient, &cl, clusterName, ns)
	if err != nil {
		log.Error(err)
		return

	}

	var strategyMap map[string]cluster.Strategy
	strategyMap = make(map[string]cluster.Strategy)
	strategyMap["1"] = cluster.Strategy1{}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if !ok {
		log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
		return
	}

	err = strategy.UpdatePolicyLabels(clientset, clusterName, ns, labels)
	if err != nil {
		log.Error(err)
	}

	//update the pgcluster crd labels with the new policy
	err = PatchPgcluster(restclient, policyName+"=pgpolicy", cl, ns)
	if err != nil {
		log.Error(err)
	}

}

func PatchPgcluster(restclient *rest.RESTClient, newLabel string, oldCRD crv1.Pgcluster, ns string) error {

	fields := strings.Split(newLabel, "=")
	labelKey := fields[0]
	labelValue := fields[1]
	oldData, err := json.Marshal(oldCRD)
	if err != nil {
		return err
	}
	if oldCRD.ObjectMeta.Labels == nil {
		oldCRD.ObjectMeta.Labels = make(map[string]string)
	}
	oldCRD.ObjectMeta.Labels[labelKey] = labelValue
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCRD)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))
	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(ns).
		Resource(crv1.PgclusterResourcePlural).
		Name(oldCRD.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}
