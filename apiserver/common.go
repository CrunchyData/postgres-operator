package apiserver

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
	"errors"
	"strconv"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	backrestStorageTypes = []string{"local", "s3"}
	// ErrDBContainerNotFound is an error that indicates that a "database" container
	// could not be found in a specific pod
	ErrDBContainerNotFound = errors.New("\"database\" container not found in pod")
)

// ReplicaPodStatus stores the name of the node a replica pod is assigned to, as well
// as whether or not the pod is considered "Ready" in the Kubernetes cluster
type ReplicaPodStatus struct {
	NodeName    string
	ReadyStatus string
}

func GetSecrets(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowUserSecret, error) {

	output := make([]msgs.ShowUserSecret, 0)
	selector := "!" + config.LABEL_PGO_BACKREST_REPO + "," + config.LABEL_PGBOUNCER + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	secrets, err := kubeapi.GetSecrets(Clientset, selector, ns)
	if err != nil {
		return output, err
	}

	log.Debugf("got %d secrets for %s", len(secrets.Items), cluster.Spec.Name)
	for _, s := range secrets.Items {
		d := msgs.ShowUserSecret{}
		d.Name = s.Name
		d.Username = string(s.Data["username"][:])
		d.Password = string(s.Data["password"][:])
		output = append(output, d)

	}

	return output, err
}

// GetReplicaPodStatus gets the status of all replica pods in the cluster. Specifically, using
// the provided cluster name and namespace, it looks up all replica pod in the cluster, and then
// provides a status for each pod ("Ready" or "Not Ready")
func GetReplicaPodStatus(clusterName, ns string) (*ReplicaPodStatus, error) {

	//get pods with pg-cluster=<cluster-name>,role=replica
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_PGHA_ROLE + "=replica"
	pods, err := kubeapi.GetPods(Clientset, selector, ns)
	if err != nil {
		return nil, err
	}

	p := pods.Items[0]
	for _, c := range p.Status.ContainerStatuses {
		if c.Name == "database" {
			var readyStatus string
			if c.Ready {
				readyStatus = "Ready"
			} else {
				readyStatus = "Not Ready"
			}
			return &ReplicaPodStatus{NodeName: p.Spec.NodeName, ReadyStatus: readyStatus}, nil
		}
	}

	log.Error(ErrDBContainerNotFound)
	return nil, ErrDBContainerNotFound
}

func GetPVCName(pod *v1.Pod) map[string]string {
	pvcList := make(map[string]string)

	for _, v := range pod.Spec.Volumes {
		if v.Name == "backrestrepo" || v.Name == "pgdata" || v.Name == "pgwal-volume" {
			if v.VolumeSource.PersistentVolumeClaim != nil {
				pvcList[v.Name] = v.VolumeSource.PersistentVolumeClaim.ClaimName
			}
		}
	}

	return pvcList

}

func CreateRMDataTask(clusterName, replicaName, taskName string, deleteBackups, deleteData, isReplica, isBackup bool, ns, clusterPGHAScope string) error {
	var err error

	//create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = taskName
	spec.TaskType = crv1.PgtaskDeleteData

	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_DELETE_DATA] = strconv.FormatBool(deleteData)
	spec.Parameters[config.LABEL_DELETE_BACKUPS] = strconv.FormatBool(deleteBackups)
	spec.Parameters[config.LABEL_IS_REPLICA] = strconv.FormatBool(isReplica)
	spec.Parameters[config.LABEL_IS_BACKUP] = strconv.FormatBool(isBackup)
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[config.LABEL_REPLICA_NAME] = replicaName
	spec.Parameters[config.LABEL_PGHA_SCOPE] = clusterPGHAScope

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_RMDATA] = "true"

	err = kubeapi.Createpgtask(RESTClient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	return err

}

func GetBackrestStorageTypes() []string {
	return backrestStorageTypes
}

// IsValidPVC determines if a PVC with the name provided exits
func IsValidPVC(pvcName, ns string) bool {
	_, pvcFound, err := kubeapi.GetPVC(Clientset, pvcName, ns)
	if err != nil {
		log.Error(err)
		return false
	} else if !pvcFound {
		return false
	}

	return true
}
