package statusservice

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
	"fmt"
	"sort"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Status(ns string) msgs.StatusResponse {
	return msgs.StatusResponse{
		Result: msgs.StatusDetail{
			DbTags:       getDBTags(ns),
			NotReady:     getNotReady(ns),
			NumClaims:    getNumClaims(ns),
			NumDatabases: getNumDatabases(ns),
			Labels:       getLabels(ns),
			VolumeCap:    getVolumeCap(ns),
		},
		Status: msgs.Status{Code: msgs.Ok, Msg: ""},
	}
}

func getNumClaims(ns string) int {
	ctx := context.TODO()

	pvcs, err := apiserver.Clientset.
		CoreV1().PersistentVolumeClaims(ns).
		List(ctx, metav1.ListOptions{LabelSelector: config.LABEL_VENDOR + "=" + config.LABEL_CRUNCHY})
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(pvcs.Items)
}

func getNumDatabases(ns string) int {
	ctx := context.TODO()
	// count number of Deployments with pg-cluster
	deps, err := apiserver.Clientset.
		AppsV1().Deployments(ns).
		List(ctx, metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER})
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(deps.Items)
}

func getVolumeCap(ns string) string {
	ctx := context.TODO()
	// sum all PVCs storage capacity
	pvcs, err := apiserver.Clientset.
		CoreV1().PersistentVolumeClaims(ns).
		List(ctx, metav1.ListOptions{LabelSelector: config.LABEL_VENDOR + "=" + config.LABEL_CRUNCHY})
	if err != nil {
		log.Error(err)
		return "error"
	}

	var capTotal int64
	capTotal = 0
	for i := range pvcs.Items {
		capTotal = capTotal + getClaimCapacity(&pvcs.Items[i])
	}
	q := resource.NewQuantity(capTotal, resource.BinarySI)
	// log.Infof("capTotal string is %s\n", q.String())
	return q.String()
}

func getDBTags(ns string) map[string]int {
	ctx := context.TODO()
	results := make(map[string]int)
	// count all pods with pg-cluster, sum by image tag value
	pods, err := apiserver.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER})
	if err != nil {
		log.Error(err)
		return results
	}
	for _, p := range pods.Items {
		for _, c := range p.Spec.Containers {
			results[c.Image]++
		}
	}

	return results
}

func getNotReady(ns string) []string {
	ctx := context.TODO()
	// show all database pods for each pgcluster that are not yet running
	agg := make([]string, 0)
	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		clusterList = &crv1.PgclusterList{}
	}

	for _, cluster := range clusterList.Items {

		selector := fmt.Sprintf("%s=crunchydata,name=%s", config.LABEL_VENDOR, cluster.Spec.ClusterName)
		pods, err := apiserver.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			log.Error(err)
			return agg
		} else if len(pods.Items) > 1 {
			log.Error(fmt.Errorf("Multiple database pods found with the same name using selector while searching for "+
				"databases that are not ready using selector %s", selector))
			return agg
		} else if len(pods.Items) == 0 {
			log.Error(fmt.Errorf("No database pods found while searching for database pods that are not ready using "+
				"selector %s", selector))
			return agg
		}

		pod := pods.Items[0]
		for _, stat := range pod.Status.ContainerStatuses {
			if !stat.Ready {
				agg = append(agg, pod.ObjectMeta.Name)
			}
		}
	}

	return agg
}

func getClaimCapacity(pvc *v1.PersistentVolumeClaim) int64 {
	qty := pvc.Status.Capacity[v1.ResourceStorage]
	diskSize := resource.MustParse(qty.String())
	diskSizeInt64, _ := diskSize.AsInt64()

	return diskSizeInt64
}

func getLabels(ns string) []msgs.KeyValue {
	ctx := context.TODO()
	var ss []msgs.KeyValue
	results := make(map[string]int)
	deps, err := apiserver.Clientset.
		AppsV1().Deployments(ns).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err)
		return ss
	}

	for _, dep := range deps.Items {
		for k, v := range dep.ObjectMeta.Labels {
			lv := k + "=" + v
			if results[lv] == 0 {
				results[lv] = 1
			} else {
				results[lv] = results[lv] + 1
			}
		}
	}

	for k, v := range results {
		ss = append(ss, msgs.KeyValue{Key: k, Value: v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	return ss
}
