package upgrade

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
	"github.com/crunchydata/postgres-operator"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddUpgrade creates a pgupgrade job
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgtask, namespace string) {
	cl := crv1.Pgcluster{}

	//not a db so get the pgcluster CRD
	_, err := kubeapi.Getpgcluster(restclient, &cl,
		upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER], namespace)
	if err != nil {
		log.Error("cound not find pgcluster for minor upgrade")
		log.Error(err)
		return
	}

	err = kubeapi.PatchDeployment(clientset, cl.Spec.Name, namespace, "/spec/template/spec/containers/0/image", operator.Pgo.Cluster.CCPImagePrefix+"/"+cl.Spec.CCPImage+":"+upgrade.Spec.Parameters["CCPImageTag"])
	if err != nil {
		log.Error(err)
		log.Error("error in doing minor upgrade")
		return
	}

	log.Info("updating the pg version after cluster upgrade")
	err = util.Patch(clientset, "/spec/ccpimagetag", upgrade.Spec.Parameters["CCPImageTag"], crv1.PgclusterResourcePlural, upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER], namespace)

	if err != nil {
		log.Error("error patching pgcluster in upgrade" + err.Error())
	}

	//update the upgrade CRD status to completed
	log.Debug("update pgtask status %s to %s ", upgrade.Spec.Name, crv1.UpgradeCompletedStatus)
	err = kubeapi.Patchpgupgrade(restclient, upgrade.Spec.Name, "/spec/upgradestatus", crv1.UpgradeCompletedStatus, namespace)
	if err != nil {
		log.Error("error in upgradestatus patch " + err.Error())
	}

}
