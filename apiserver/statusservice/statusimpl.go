package statusservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"sort"
)

func Status() msgs.StatusResponse {
	var err error

	response := msgs.StatusResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	err = getStatus(&response.Result)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = "error getting status report"
	}

	return response
}

func getStatus(results *msgs.StatusDetail) error {

	var err error
	results.OperatorStartTime = getOperatorStart()
	results.NumBackups = getNumBackups()
	results.NumClaims = getNumClaims()
	results.NumDatabases = getNumDatabases()
	results.VolumeCap = getVolumeCap()
	results.DbTags = getDBTags()
	results.NotReady = getNotReady()
	results.Nodes = getNodes()
	results.Labels = getLabels()
	return err
}

func getOperatorStart() string {
	pods, err := kubeapi.GetPods(apiserver.Clientset, "name="+util.LABEL_OPERATOR, apiserver.Namespace)
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

func getNumBackups() int {
	//count the number of Jobs with pgbackup=true and completionTime not nil
	jobs, err := kubeapi.GetJobs(apiserver.Clientset, util.LABEL_PGBACKUP, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(jobs.Items)
}

func getNumClaims() int {
	//count number of PVCs with pgremove=true
	pvcs, err := kubeapi.GetPVCs(apiserver.Clientset, util.LABEL_PGREMOVE, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(pvcs.Items)
}

func getNumDatabases() int {
	//count number of Deployments with pg-cluster
	deps, err := kubeapi.GetDeployments(apiserver.Clientset, util.LABEL_PG_CLUSTER, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(deps.Items)
}

func getVolumeCap() string {
	//sum all PVCs storage capacity
	pvcs, err := kubeapi.GetPVCs(apiserver.Clientset, util.LABEL_PGREMOVE, apiserver.Namespace)
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
	log.Infof("capTotal string is %s\n", q.String())
	return q.String()
}

func getDBTags() map[string]int {
	results := make(map[string]int)
	//count all pods with pg-cluster, sum by image tag value
	pods, err := kubeapi.GetPods(apiserver.Clientset, util.LABEL_PG_CLUSTER, apiserver.Namespace)
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

func getNotReady() []string {
	//show all pods with pg-cluster that have status.Phase of not Running
	agg := make([]string, 0)

	pods, err := kubeapi.GetPods(apiserver.Clientset, util.LABEL_PG_CLUSTER, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return agg
	}
	for _, p := range pods.Items {
		for _, stat := range p.Status.ContainerStatuses {
			if !stat.Ready {
				agg = append(agg, p.ObjectMeta.Name)
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

	nodes, err := kubeapi.GetNodes(apiserver.Clientset)
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

func getLabels() []msgs.KeyValue {
	var ss []msgs.KeyValue
	results := make(map[string]int)
	// GetDeployments gets a list of deployments using a label selector
	deps, err := kubeapi.GetDeployments(apiserver.Clientset, "", apiserver.Namespace)
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
