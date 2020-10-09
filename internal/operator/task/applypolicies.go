package task

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/util"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// RemoveBackups ...
func ApplyPolicies(clusterName string, clientset kubeapi.Interface, RESTConfig *rest.Config, ns string) {

	taskName := clusterName + "-policies"

	task, err := clientset.CrunchydataV1().Pgtasks(ns).Get(taskName, metav1.GetOptions{})
	if err == nil {
		//apply those policies
		for k := range task.Spec.Parameters {
			log.Debugf("applying policy %s to %s", k, clusterName)
			applyPolicy(clientset, RESTConfig, k, clusterName, ns)
		}
		//delete the pgtask to not redo this again
		clientset.CrunchydataV1().Pgtasks(ns).Delete(taskName, &metav1.DeleteOptions{})
	}
}

func applyPolicy(clientset kubeapi.Interface, restconfig *rest.Config, policyName, clusterName, ns string) {

	cl, err := clientset.CrunchydataV1().Pgclusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return
	}

	if err := util.ExecPolicy(clientset, restconfig, ns, policyName, clusterName, cl.Spec.Port); err != nil {
		log.Error(err)
		return
	}

	labels := make(map[string]string)
	labels[policyName] = "pgpolicy"

	patch, err := kubeapi.NewMergePatch().Add("metadata", "labels")(labels).Bytes()
	if err != nil {
		log.Error(err)
	}

	_, err = clientset.AppsV1().Deployments(ns).Patch(clusterName, types.MergePatchType, patch)
	if err != nil {
		log.Error(err)
	}

	//update the pgcluster crd labels with the new policy
	_, err = clientset.CrunchydataV1().Pgclusters(ns).Patch(cl.Spec.Name, types.MergePatchType, patch)
	if err != nil {
		log.Error(err)
	}

}
