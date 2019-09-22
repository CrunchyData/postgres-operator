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
	"strings"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
//	"github.com/crunchydata/postgres-operator/operator"
//	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddUpgrade implements the upgrade workflow for cluster minor upgrade
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgtask, namespace string) {
	cl := crv1.Pgcluster{}



	upgradeTarget := upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	//not a db so get the pgcluster CRD
	_, err := kubeapi.Getpgcluster(restclient, &cl, upgradeTarget, namespace)
	if err != nil {
		log.Error("cound not find pgcluster for minor upgrade")
		log.Error(err)
		return
	}

	// 

	// get replicalist and the deployments that need to be updated (which are also the name of the replicas)
	replicaList := crv1.PgreplicaList{}
	selector := config.LABEL_PG_CLUSTER + "=" + cl.Spec.Name
	err = kubeapi.GetpgreplicasBySelector(restclient, &replicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	replicaDeploymentList := []string{}
	for _, replica := range replicaList.Items {
		log.Debug("MinorUpgrade: adding deployment %s to list", replica.Spec.Name)
		replicaDeploymentList = append(replicaDeploymentList, replica.Spec.Name)
	}
	log.Debug("MinorUpgrade: replica count for upgrade is %d", len(replicaDeploymentList, ))

	replist := strings.Join(replicaDeploymentList, ",") // string delimited list of replica deployments

	// get backrest deployment info here.
	backRestDeploymentName := cl.Spec.Name + "-backrest-shared-repo"


	// get the latest version of the task in case it changed
	currentTask := crv1.Pgtask{}
	currentTask.Spec.Parameters = make(map[string]string)
	found, terr := kubeapi.Getpgtask(restclient, &currentTask, upgrade.Spec.Name, namespace)
	
	if !found {
		log.Error("cound not find pgtask for minor upgrade")
		log.Error(terr)
	}

	currentTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY] = upgradeTarget
	currentTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA] = replist
	currentTask.Spec.Parameters[config.LABEL_UPGRADE_BACKREST] = backRestDeploymentName


	//this effectively bounces the primary Deployment's pod to pick up
	//the new image tag
	// err = kubeapi.PatchDeployment(clientset, cl.Spec.Name, namespace, "/spec/template/spec/containers/0/image", operator.Pgo.Cluster.CCPImagePrefix+"/"+cl.Spec.CCPImage+":"+upgrade.Spec.Parameters["CCPImageTag"])
	// if err != nil {
	// 	log.Error(err)
	// 	log.Error("error in doing minor upgrade")
	// 	return
	// }

	// patch replica deployments
	// for _, deployment := range replicaList.Items {
	// 	err = kubeapi.PatchDeployment(clientset, deployment.Spec.Name, namespace, "/spec/template/spec/containers/0/image", operator.Pgo.Cluster.CCPImagePrefix+"/"+cl.Spec.CCPImage+":"+upgrade.Spec.Parameters["CCPImageTag"])
	// 	if err != nil {
	// 		log.Error(err)
	// 		log.Error("error in doing minor upgrade")
	// 		return
	// 	}	
	// }



	// do this last, once all deployments have been updated.

	//update the CRD with the new image tag to maintain the truth
//	log.Info("updating the pg version after cluster upgrade")
	// err = util.Patch(restclient, "/spec/ccpimagetag", upgrade.Spec.Parameters["CCPImageTag"], crv1.PgclusterResourcePlural, upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER], namespace)

	// if err != nil {
	// 	log.Error("error patching pgcluster in upgrade" + err.Error())
	// }

	//update the upgrade CRD status to completed
	log.Debug("update pgtask status %s to %s ", currentTask.Spec.Name, crv1.InProgressStatus)
	currentTask.Spec.Status = crv1.InProgressStatus
	err = kubeapi.Updatepgtask(restclient, &currentTask, currentTask.Spec.Name, namespace)
	if err != nil {
		log.Error("error in updating minor upgrade pgtask to in progress status " + err.Error())
	}

}
