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

// Package cluster holds the cluster TPR logic and definitions
// A cluster is comprised of a master service, replica service,
// master deployment, and replica deployment
package cluster

import (
	log "github.com/Sirupsen/logrus"
	"text/template"

	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var UpgradeTemplate1 *template.Template

type JobTemplateFields struct {
	Name              string
	OLD_PVC_NAME      string
	NEW_PVC_NAME      string
	CCP_IMAGE_TAG     string
	OLD_DATABASE_NAME string
	NEW_DATABASE_NAME string
	OLD_VERSION       string
	NEW_VERSION       string
}

const CLUSTER_UPGRADE_TEMPLATE_PATH = "/pgconf/postgres-operator/cluster/1/cluster-upgrade-job.json"

func init() {

	UpgradeTemplate1 = util.LoadTemplate(CLUSTER_UPGRADE_TEMPLATE_PATH)
}

func (r ClusterStrategy1) MinorUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, cluster *tpr.PgCluster, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error

	log.Info("minor cluster upgrade using Strategy 1 in namespace " + namespace)

	return err

}

func (r ClusterStrategy1) MajorUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, cluster *tpr.PgCluster, upgrade *tpr.PgUpgrade, namespace string) error {
	var err error

	log.Info("major cluster upgrade using Strategy 1 in namespace " + namespace)

	return err

}
