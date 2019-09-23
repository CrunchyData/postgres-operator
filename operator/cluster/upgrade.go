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
	kerrors "k8s.io/apimachinery/pkg/api/errors"

)

// AddUpgrade implements the upgrade workflow for cluster minor upgrade
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgtask, namespace string) {
	cl := crv1.Pgcluster{}

	// TODO: persist original and target CCP_IMAGE_TAGs in pgtask
	// TODO: persist autofail state of cluster in pgtask(?)

	upgradeTargetClusterName := upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]

	_, err := kubeapi.Getpgcluster(restclient, &cl, upgradeTargetClusterName, namespace)
	if err != nil {
		log.Error("cound not find pgcluster for minor upgrade")
		log.Error(err)
		return
	}

	// label cluster with minor-upgrade label
	addMinorUpgradeLabelToDeployments(clientset, upgradeTargetClusterName, namespace)

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

	currentTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY] = upgradeTargetClusterName // same name as primary
	currentTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA] = replist
	currentTask.Spec.Parameters[config.LABEL_UPGRADE_BACKREST] = backRestDeploymentName




	// patch replica deployments
	// for _, deployment := range replicaList.Items {
	// 	err = kubeapi.PatchDeployment(clientset, deployment.Spec.Name, namespace, "/spec/template/spec/containers/0/image", operator.Pgo.Cluster.CCPImagePrefix+"/"+cl.Spec.CCPImage+":"+upgrade.Spec.Parameters["CCPImageTag"])
	// 	if err != nil {
	// 		log.Error(err)
	// 		log.Error("error in doing minor upgrade")
	// 		return
	// 	}	
	// }





	//update the upgrade CRD status to completed
	log.Debug("update pgtask status %s to %s ", currentTask.Spec.Name, crv1.InProgressStatus)
	currentTask.Spec.Status = crv1.InProgressStatus
	err = kubeapi.Updatepgtask(restclient, &currentTask, currentTask.Spec.Name, namespace)
	if err != nil {
		log.Error("error in updating minor upgrade pgtask to in progress status " + err.Error())
	}


	// start the upgrade
	ProcessNextUpgradeItem(clientset, restclient, currentTask.Spec.Name, namespace)

}

// ProcessNextUpgradeItem - processes the next deployment for a cluster being upgraded
// One deployment is done per call in the following order: replicas, backrest, primary
// If more than one replica is in the list, they are done one at a time, once per call
// with the item getting removed from the list each time. This method should get called
// after the pod goes ready from the previous item.
func ProcessNextUpgradeItem(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgradeTaskName, namespace string) {

	log.Debug("Upgrade: ProcessNextUpgradeItem.... ", upgradeTaskName)
	
	// get the upgrade task
	upgradeTask := crv1.Pgtask{}
	found, err := kubeapi.Getpgtask(restclient, &upgradeTask, upgradeTaskName, namespace)
	
	if !found {
		log.Error("cound not find pgtask for minor upgrade")
		log.Error(err)

		FailUpgradeWithError(upgradeTaskName, "Upgrade task not found. Should not happen.")
	}

	replicaTargets := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA]
	backrestTarget := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_BACKREST]
	primaryTarget := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY]

	// check for replica's to upgrade
	if len(replicaTargets) > 0 {

		// parse replica target
		repList := strings.Split(replicaTargets, ",")
		repTargetName := repList[0] // get last element in list
		log.Debug("MinorUPgrade: processing replica ", repTargetName)

		repList = repList[1:] // new targets are all but first (0) element
		log.Debug("Minor Upgrade: remaining replicas: ", repList)
		updatedTargetList := strings.Join(repList, ",")

		// bounce deployment
	
		//this effectively bounces the Deployment's pod to pick up
		//the new image tag.
		
		// err = kubeapi.PatchDeployment(clientset, cl.Spec.Name, namespace, "/spec/template/spec/containers/0/image", operator.Pgo.Cluster.CCPImagePrefix+"/"+cl.Spec.CCPImage+":"+upgrade.Spec.Parameters["CCPImageTag"])
		// if err != nil {
		// 	log.Error(err)
		// 	log.Error("error in doing minor upgrade")
		// 	return
		// }
		
		// upgrade replica targets in task.
		upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA] = updatedTargetList

		err = kubeapi.Updatepgtask(restclient, &upgradeTask, upgradeTask.Spec.Name, namespace)
		if err != nil {
			log.Error("error in updating minor upgrade pgtask to in progress status " + err.Error())
		}



	} else if len (backrestTarget) > 0 {
	// backrest to upgrade
	log.Debug("Minor Upgrade: backrest")

	} else if len (primaryTarget) > 0 {
	// primary to upgrade
	log.Debug("Minor Upgrade: primary")

	} else {
		// complete upgrade
		log.Debug("Minor Upgrade: completing")

	}







}


// CompleteUpgrade - makes any finishing changes required to complete the upgrade and
// does final update to the task. 
func CompleteUpgrade() {

	// do this last, once all deployments have been updated.

	//update the CRD with the new image tag to maintain the truth
//	log.Info("updating the pg version after cluster upgrade")
	// err = util.Patch(restclient, "/spec/ccpimagetag", upgrade.Spec.Parameters["CCPImageTag"], crv1.PgclusterResourcePlural, upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER], namespace)

	// if err != nil {
	// 	log.Error("error patching pgcluster in upgrade" + err.Error())
	// }


}

// FailUpgradeWithError - called when it is not possible to continue the upgrade process
// 
func FailUpgradeWithError(upgradeTaskName, errorText string) {

	log.Error("Minor Upgrade unable to complete: ", errorText)

}

// addMinorUpgradeLabelToDeployments - labels the deployments with a minor-upgrade=true label
func addMinorUpgradeLabelToDeployments(clientset *kubernetes.Clientset, clusterName, namespace string)  {

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName

	deployments, err := kubeapi.GetDeployments(clientset, selector, namespace)

	if kerrors.IsNotFound(err) {
		log.Debug("minor upgrade no deployments found - should not have happened")
	} else if err != nil {
		log.Error("error getting deployments " + err.Error())
	} else {

		for _, deployment := range deployments.Items {
			//add the minor upgrade label to the Deployment
			err = kubeapi.AddLabelToDeployment(clientset, &deployment, config.LABEL_MINOR_UPGRADE, "true", namespace)

			if err != nil {
				log.Error("could not add label to deployment for minor upgrade ", deployment.Name)
			}
		}
	}

}

func removeMinorUpgradeLabelFromDeployments(clusterName, namespace string) {

	// TODO - when upgrade completes, this gets called to remove the label.
}