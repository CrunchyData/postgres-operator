package cluster

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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddUpgrade creates a pgupgrade job
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgupgrade, namespace string) {
	cl := crv1.Pgcluster{}

	//not a db so get the pgcluster CRD
	_, err := kubeapi.Getpgcluster(restclient, &cl,
		upgrade.Spec.Name, namespace)
	if err != nil {
		return
	}

	err = AddUpgradeBase(clientset, restclient, upgrade, namespace, &cl)
	if err != nil {
		log.Error("error adding upgrade" + err.Error())
	} else {
		//update the upgrade CRD status to submitted
		upgrade.Spec.UpgradeStatus = crv1.UpgradeSubmittedStatus
		kubeapi.Updatepgupgrade(restclient, upgrade, upgrade.Spec.Name, namespace)
		//		err = util.Patch(restclient, "/spec/upgradestatus", crv1.UpgradeSubmittedStatus, "pgupgrades", upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error("error setting upgrade status " + err.Error())
		}
	}

}

// DeleteUpgrade deletes a pgupgrade job
func DeleteUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgupgrade, namespace string) {
	var jobName = "upgrade-" + upgrade.Spec.Name
	log.Debugf("deleting Job with Name= %s in namespace %s", jobName, namespace)

	//delete the job
	kubeapi.DeleteJob(clientset, jobName, namespace)
}
