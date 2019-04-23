package statusservice

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
	"fmt"
	"sort"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

func Status(ns string) msgs.StatusResponse {
	var err error

	response := msgs.StatusResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	err = getStatus(&response.Result, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = "error getting status report"
	}

	return response
}

func getStatus(results *msgs.StatusDetail, ns string) error {

	var err error
	results.OperatorStartTime = getOperatorStart(apiserver.PgoNamespace)
	results.NumBackups = getNumBackups(ns)
	results.NumClaims = getNumClaims(ns)
	results.NumDatabases = getNumDatabases(ns)
	results.VolumeCap = getVolumeCap(ns)
	results.DbTags = getDBTags(ns)
	results.NotReady = getNotReady(ns)
	results.Nodes = getNodes()
	results.Labels = getLabels(ns)
	return err
}

func getOperatorStart(ns string) string {
	pods, err := kubeapi.GetPods(apiserver.Clientset, "name="+config.LABEL_OPERATOR, ns)
	if err != nil {
		log.Error(err)
		return "error"
	}

	for _, p := range pods.Items {
		if string(p.Status.Phase) == "Running" {
			return p.Status.StartTime.String()
		}
	}
	log.Error("a running postgres-operator pod is not found")

	return "error"
}

func getNumBackups(ns string) int {
	//count the number of Jobs with pgbackup=true and completionTime not nil
	jobs, err := kubeapi.GetJobs(apiserver.Clientset, config.LABEL_PGBACKUP, ns)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(jobs.Items)
}

func getNumClaims(ns string) int {
	//count number of PVCs with pgremove=true
	pvcs, err := kubeapi.GetPVCs(apiserver.Clientset, config.LABEL_PGREMOVE, ns)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(pvcs.Items)
}

func getNumDatabases(ns string) int {
	//count number of Deployments with pg-cluster
	deps, err := kubeapi.GetDeployments(apiserver.Clientset, config.LABEL_PG_CLUSTER, ns)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(deps.Items)
}

func getVolumeCap(ns string) string {
	//sum all PVCs storage capacity
	pvcs, err := kubeapi.GetPVCs(apiserver.Clientset, config.LABEL_PGREMOVE, ns)
	if err != nil {
		log.Error(err)
		return "error"
	}

	var capTotal int64
	capTotal = 0
	for _, p := range pvcs.Items {
		capTotal = capTotal + getClaimCapacity(apiserver.Clientset, &p)
	}
	q := resource.NewQuantity(capTotal, resource.BinarySI)
	//log.Infof("capTotal string is %s\n", q.String())
	return q.String()
}

func getDBTags(ns string) map[string]int {
	results := make(map[string]int)
	//count all pods with pg-cluster, sum by image tag value
	pods, err := kubeapi.GetPods(apiserver.Clientset, config.LABEL_PG_CLUSTER, ns)
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
	//show all database pods for each pgcluster that are not yet running
	agg := make([]string, 0)
	clusterList := crv1.PgclusterList{}
	kubeapi.Getpgclusters(apiserver.RESTClient, &clusterList, ns)

	for _, cluster := range clusterList.Items {

		selector := fmt.Sprintf("%s=crunchydata,name=%s", config.LABEL_VENDOR, cluster.Spec.ClusterName)
		pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
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

func getClaimCapacity(clientset *kubernetes.Clientset, pvc *v1.PersistentVolumeClaim) int64 {
	qty := pvc.Status.Capacity[v1.ResourceStorage]
	diskSize := resource.MustParse(qty.String())
	diskSizeInt64, _ := diskSize.AsInt64()

	return diskSizeInt64

}

func getNodes() []msgs.NodeInfo {
	result := make([]msgs.NodeInfo, 0)

	nodes, err := kubeapi.GetAllNodes(apiserver.Clientset)
	if err != nil {
		log.Error(err)
		return result
	}

	for _, node := range nodes.Items {
		r := msgs.NodeInfo{}
		r.Labels = node.ObjectMeta.Labels
		r.Name = node.ObjectMeta.Name
		clen := len(node.Status.Conditions)
		if clen > 0 {
			r.Status = string(node.Status.Conditions[clen-1].Type)
		}
		result = append(result, r)

	}

	return result
}

func getLabels(ns string) []msgs.KeyValue {
	var ss []msgs.KeyValue
	results := make(map[string]int)
	// GetDeployments gets a list of deployments using a label selector
	deps, err := kubeapi.GetDeployments(apiserver.Clientset, "", ns)
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
		ss = append(ss, msgs.KeyValue{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	return ss

}
