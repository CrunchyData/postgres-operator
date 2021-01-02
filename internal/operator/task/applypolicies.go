package task

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"strings"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RemoveBackups ...
func ApplyPolicies(clusterName string, Clientset kubernetes.Interface, RESTClient *rest.RESTClient, RESTConfig *rest.Config, ns string) {

	taskName := clusterName + "-policies"
	task := crv1.Pgtask{}
	task.Spec = crv1.PgtaskSpec{}
	task.Spec.Name = taskName

	found, err := kubeapi.Getpgtask(RESTClient, &task, taskName, ns)
	if found && err == nil {
		//apply those policies
		for k := range task.Spec.Parameters {
			log.Debugf("applying policy %s to %s", k, clusterName)
			applyPolicy(Clientset, RESTClient, RESTConfig, k, clusterName, ns)
		}
		//delete the pgtask to not redo this again
		kubeapi.Deletepgtask(RESTClient, taskName, ns)
	}
}

func applyPolicy(clientset kubernetes.Interface, restclient *rest.RESTClient, restconfig *rest.Config, policyName, clusterName, ns string) {
	cl := crv1.Pgcluster{}

	if _, err := kubeapi.Getpgcluster(restclient, &cl, clusterName, ns); err != nil {
		log.Error(err)
		return
	}

	if err := util.ExecPolicy(clientset, restclient, restconfig, ns, policyName, clusterName, cl.Spec.Port); err != nil {
		log.Error(err)
		return
	}

	labels := make(map[string]string)
	labels[policyName] = "pgpolicy"

	if err := util.UpdatePolicyLabels(clientset, clusterName, ns, labels); err != nil {
		log.Error(err)
	}

	//update the pgcluster crd labels with the new policy
	if err := PatchPgcluster(restclient, policyName+"=pgpolicy", cl, ns); err != nil {
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
