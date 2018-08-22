package apiserver

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
)

//TODO remove and replace with util.GetSecrets
func GetSecrets(cluster *crv1.Pgcluster) ([]msgs.ShowUserSecret, error) {

	output := make([]msgs.ShowUserSecret, 0)
	selector := "pgpool!=true," + util.LABEL_PG_DATABASE + "=" + cluster.Spec.Name

	secrets, err := kubeapi.GetSecrets(Clientset, selector, Namespace)
	if err != nil {
		return output, err
	}

	log.Debugf("got %d secrets for %s\n", len(secrets.Items), cluster.Spec.Name)
	for _, s := range secrets.Items {
		d := msgs.ShowUserSecret{}
		d.Name = s.Name
		d.Username = string(s.Data["username"][:])
		d.Password = string(s.Data["password"][:])
		output = append(output, d)

	}

	return output, err
}

func GetPodStatus(deployName string) (string, string) {

	//get pods with replica-name=deployName
	pods, err := kubeapi.GetPods(Clientset, util.LABEL_REPLICA_NAME+"="+deployName, Namespace)
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
		if v.Name == "backrestrepo-volume" || v.Name == "pgdata" || v.Name == "pgwal-volume" {
			if v.VolumeSource.PersistentVolumeClaim != nil {
				pvcList[v.Name] = v.VolumeSource.PersistentVolumeClaim.ClaimName
			}
		}
	}

	return pvcList

}
