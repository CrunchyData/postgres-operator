package apiserver

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
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/core/v1"
)

var backrestStorageTypes = []string{"local", "s3"}

func GetSecrets(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowUserSecret, error) {

	output := make([]msgs.ShowUserSecret, 0)
	selector := "!" + config.LABEL_PGO_BACKREST_REPO + "," + config.LABEL_PGBOUNCER + "!=true," + config.LABEL_PGPOOL + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

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

func GetPodStatus(deployName, ns string) (string, string) {

	//get pods with replica-name=deployName
	pods, err := kubeapi.GetPods(Clientset, config.LABEL_REPLICA_NAME+"="+deployName, ns)
	if err != nil {
		return "error", "error"
	}

	p := pods.Items[0]
	nodeName := p.Spec.NodeName
	for _, c := range p.Status.ContainerStatuses {
		if c.Name == "database" {
			if c.Ready {
				return "Ready", nodeName
			} else {
				return "Not Ready", nodeName
			}
		}
	}

	return "error2", nodeName

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

func CreateRMDataTask(storageSpec crv1.PgStorageSpec, clusterName, pvcName string, dataRoots []string, taskName, ns string) error {
	var err error

	log.Debugf("a CreateRMDataTask dataRoots=%v ns=%s", dataRoots, ns)
	//create a pgtask for each root at this volume/pvc
	for i := 0; i < len(dataRoots); i++ {

		//create pgtask CRD
		spec := crv1.PgtaskSpec{}
		spec.Namespace = ns
		spec.Name = taskName
		spec.TaskType = crv1.PgtaskDeleteData
		spec.StorageSpec = storageSpec

		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PVC_NAME] = pvcName
		spec.Parameters[config.LABEL_DATA_ROOT] = dataRoots[i]
		spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: taskName,
			},
			Spec: spec,
		}
		newInstance.ObjectMeta.Labels = make(map[string]string)
		newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
		newInstance.ObjectMeta.Labels[config.LABEL_RMDATA] = "true"

		err := kubeapi.Createpgtask(RESTClient, newInstance, ns)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err

}

// IsValidBackrestStorageType determines if the storageType string contains valid pgBackRest
// storage type values
func IsValidBackrestStorageType(storageType string) bool {
	isValid := true
	for _, storageType := range strings.Split(storageType, ",") {
		if !IsStringOneOf(storageType, GetBackrestStorageTypes()...) {
			isValid = false
			break
		}
	}
	return isValid
}

// IsStringOneOf tests to see string testVal is included in the list
// of strings provided using acceptedVals
func IsStringOneOf(testVal string, acceptedVals ...string) bool {
	isOneOf := false
	for _, val := range acceptedVals {
		if testVal == val {
			isOneOf = true
			break
		}
	}
	return isOneOf
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
