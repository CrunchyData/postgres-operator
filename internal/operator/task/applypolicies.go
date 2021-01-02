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
	"context"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/util"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// RemoveBackups ...
func ApplyPolicies(clusterName string, clientset kubeapi.Interface, RESTConfig *rest.Config, ns string) {
	ctx := context.TODO()
	taskName := clusterName + "-policies"

	task, err := clientset.CrunchydataV1().Pgtasks(ns).Get(ctx, taskName, metav1.GetOptions{})
	if err == nil {
		// apply those policies
		for k := range task.Spec.Parameters {
			log.Debugf("applying policy %s to %s", k, clusterName)
			applyPolicy(clientset, RESTConfig, k, clusterName, ns)
		}
		// delete the pgtask to not redo this again
		_ = clientset.CrunchydataV1().Pgtasks(ns).Delete(ctx, taskName, metav1.DeleteOptions{})
	}
}

func applyPolicy(clientset kubeapi.Interface, restconfig *rest.Config, policyName, clusterName, ns string) {
	ctx := context.TODO()
	cl, err := clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, clusterName, metav1.GetOptions{})
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

	log.Debugf("patching deployment %s: %s", clusterName, patch)
	_, err = clientset.AppsV1().Deployments(ns).Patch(ctx, clusterName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Error(err)
	}

	// update the pgcluster crd labels with the new policy
	log.Debugf("patching cluster %s: %s", cl.Spec.Name, patch)
	_, err = clientset.CrunchydataV1().Pgclusters(ns).Patch(ctx, cl.Spec.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Error(err)
	}
}
