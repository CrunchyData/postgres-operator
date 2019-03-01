// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// MinorUpgrade ..
func (r Strategy1) MinorUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, upgrade *crv1.Pgupgrade, namespace string) error {
	var err error

	log.Info("minor cluster upgrade using Strategy 1 in namespace " + namespace)

	//do this instead of deleting the deployment and creating a new one
	//kubectl patch deploy mango --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"crunchydata/crunchy-postgres:centos7-10.4-1.8.4"}]'
	//also patch the CRD
	//kubectl patch pgcluster fango --type='json' -p='[{"op": "replace", "path": "/spec/ccpimagetag", "value":"centos7-10.4-1.8.4"}]'

	err = kubeapi.PatchDeployment(clientset, cl.Spec.Name, namespace, "/spec/template/spec/containers/0/image", operator.Pgo.Cluster.CCPImagePrefix+"/"+upgrade.Spec.CCPImage+":"+upgrade.Spec.CCPImageTag)

	err = util.Patch(restclient, "/spec/ccpimagetag", upgrade.Spec.CCPImageTag, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)

	//update the upgrade CRD status to completed
	log.Debug("jeff patch pgupgrade %s to %s here in upgrade_strategy", upgrade.Spec.Name, crv1.UpgradeCompletedStatus)
	err = kubeapi.Patchpgupgrade(restclient, upgrade.Spec.Name, "/spec/upgradestatus", crv1.UpgradeCompletedStatus, namespace)
	if err != nil {
		log.Error("error in upgradestatus patch " + err.Error())
	}

	return err

}
