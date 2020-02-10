package cluster

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"

	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddUpgrade implements the upgrade workflow for cluster minor upgrade
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgtask, namespace string) {

	upgradeTargetClusterName := upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]

	cl := crv1.Pgcluster{}
	_, err := kubeapi.Getpgcluster(restclient, &cl, upgradeTargetClusterName, namespace)
	if err != nil {
		log.Error("cound not find pgcluster for minor upgrade")
		log.Error(err)
		return
	}

	// get current primary label from cluster.
	targetPrimary := cl.ObjectMeta.Labels[config.LABEL_CURRENT_PRIMARY]
	log.Debug("Minor Upgrade Primary set to: ", targetPrimary)

	// label cluster with minor-upgrade label
	labelpgClusterForMinorUpgrade(clientset, restclient, upgradeTargetClusterName, namespace)

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
		log.Debugf("MinorUpgrade: adding deployment to list: ", replica.Spec.Name)
		replicaDeploymentList = append(replicaDeploymentList, replica.Spec.Name)
	}
	log.Debug("MinorUpgrade: replica count for upgrade is ", len(replicaDeploymentList))

	replist := strings.Join(replicaDeploymentList, ",") // string delimited list of replica deployments

	// get the latest version of the task in case it changed
	currentTask := crv1.Pgtask{}
	currentTask.Spec.Parameters = make(map[string]string)
	found, terr := kubeapi.Getpgtask(restclient, &currentTask, upgrade.Spec.Name, namespace)

	if !found {
		log.Error("cound not find pgtask for minor upgrade")
		log.Error(terr)
	}

	currentTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY] = targetPrimary
	currentTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA] = replist

	// Presently, backrest upgrade will not be done by minor upgrade as it uses a container release with the operator itself
	// and not the one that is a part of the container suite.

	// get backrest deployment info here.
	// backRestDeploymentName := cl.Spec.Name + "-backrest-shared-repo"
	//	currentTask.Spec.Parameters[config.LABEL_UPGRADE_BACKREST] = backRestDeploymentName
	currentTask.Spec.Parameters[config.LABEL_UPGRADE_BACKREST] = ""

	//update the upgrade CRD status to in progress
	log.Debugf("update pgtask status %s to %s ", currentTask.Spec.Name, crv1.InProgressStatus)
	currentTask.Spec.Status = crv1.InProgressStatus
	err = kubeapi.Updatepgtask(restclient, &currentTask, currentTask.Spec.Name, namespace)
	if err != nil {
		log.Error("error in updating minor upgrade pgtask to in progress status " + err.Error())
	}

	publishMinorUpgradeStartedEvent(&currentTask, &cl, namespace)

	// if autofail is enabled, then patch the replicas, followed by the primary, sequentially.
	// this process is started by calling UpgradeWithAutofailEnabled to start the upgrade
	// otherwise if autofail is disabled, upgrade the primary and all replicas at the same time
	// by calling UpgradeWithAutofailDisabled
	if util.IsAutofailEnabled(&cl) {
		UpgradeWithAutofailEnabled(clientset, restclient, cl, currentTask.Name, namespace)
	} else {
		UpgradeWithAutofailDisabled(clientset, restclient, cl, currentTask.Name, namespace)
	}
}

// UpgradeWithAutofailEnabled is called when a minor upgrade is initiated for a PG cluster that
// has autofail enabled.  Being that all replicas, and then the primary, are patched
// sequentially for a minor upgrade during which autofail is enabled for the cluster, this
// function simply calls the ProcessNextUpgradeItem function to trigger the upgrade the first
// Deployment in the sequence of primary and replica Deployments (ProcessNextUpgradeItem will
// then be called for any remaining Deployments from the Pod controller).
func UpgradeWithAutofailEnabled(clientset *kubernetes.Clientset, restclient *rest.RESTClient,
	cluster crv1.Pgcluster, upgradeTaskName, namespace string) {
	ProcessNextUpgradeItem(clientset, restclient, cluster, upgradeTaskName, namespace)
}

// ProcessNextUpgradeItem - processes the next deployment for a cluster being upgraded
// One deployment is done per call in the following order: replicas, backrest, primary
// If more than one replica is in the list, they are done one at a time, once per call
// with an item getting removed from the list each time. This method should get called
// after the pod goes ready from the previous item, which is handled by the pod controller.
func ProcessNextUpgradeItem(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cluster crv1.Pgcluster, upgradeTaskName, namespace string) {

	log.Debug("Upgrade: ProcessNextUpgradeItem.... ", upgradeTaskName)

	// get the upgrade task
	upgradeTask := crv1.Pgtask{}
	found, err := kubeapi.Getpgtask(restclient, &upgradeTask, upgradeTaskName, namespace)

	if !found {
		log.Error("cound not find pgtask for minor upgrade")
		log.Error(err)
	}

	replicaTargets := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA]
	primaryTargetName := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY]

	// check for replica's to upgrade
	if len(replicaTargets) > 0 {

		// parse replica target
		repList := strings.Split(replicaTargets, ",")
		replicaTargetName := repList[0] // get last element in list
		log.Debug("Minor Upgrade: processing replica ", replicaTargetName)

		repList = repList[1:] // new targets are all but first (0) element
		log.Debug("Minor Upgrade: remaining replicas: ", repList)
		updatedTargetList := strings.Join(repList, ",")

		// bounce deployment

		//this effectively bounces the Deployment's pod to pick up
		//the new image tag.

		log.Debug("About to patch replica: ", replicaTargetName)

		imageNamePatch, err := createImageNamePatch(cluster, operator.Pgo.Cluster.CCPImagePrefix,
			upgradeTask.Spec.Parameters["CCPImageTag"])
		if err != nil {
			log.Errorf("error creating container image name patch during minor upgrade of cluster %s",
				cluster.Name)
			log.Error(err)
			return
		}

		err = kubeapi.PatchDeploymentStrategicMerge(clientset, replicaTargetName, namespace,
			imageNamePatch)
		if err != nil {
			log.Error(err)
			log.Error("error in doing replica minor upgrade")
			return
		}

		// upgrade replica targets in task.
		upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA] = updatedTargetList

		err = kubeapi.Updatepgtask(restclient, &upgradeTask, upgradeTask.Spec.Name, namespace)
		if err != nil {
			log.Error("error in updating minor upgrade pgtask to in progress status " + err.Error())
		}

	} else if len(primaryTargetName) > 0 {
		// primary to upgrade - should always be one of these.

		log.Debug("Minor Upgrade: primary")
		log.Debug("About to patch primary: ", primaryTargetName)

		imageNamePatch, err := createImageNamePatch(cluster, operator.Pgo.Cluster.CCPImagePrefix,
			upgradeTask.Spec.Parameters["CCPImageTag"])
		if err != nil {
			log.Errorf("error creating container image name patch during minor upgrade of cluster %s",
				cluster.Name)
			log.Error(err)
			return
		}

		err = kubeapi.PatchDeploymentStrategicMerge(clientset, primaryTargetName, namespace,
			imageNamePatch)
		if err != nil {
			log.Error(err)
			log.Error("error in doing primary minor upgrade")
			return
		}

		// upgrade primary target in task.
		upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY] = ""

		err = kubeapi.Updatepgtask(restclient, &upgradeTask, upgradeTask.Spec.Name, namespace)
		if err != nil {
			log.Error("error in updating minor upgrade pgtask to in progress status " + err.Error())
		}

	} else {
		// No other deployments  left to upgrade, complete the upgrade
		completeUpgrade(clientset, restclient, &upgradeTask, true, cluster.Spec.Name, namespace)

		publishMinorUpgradeCompleteEvent(&upgradeTask, &cluster, namespace)
	}
}

// UpgradeWithAutofailDisabled patches the primary and all replicas at the same time, and then
// immediately mark the upgrade as complete.  When performing a minor upgrade with autofail
// disabled, once the upgrade is complete and new pods have been created for the primary and
// each replica, any "database" containers will remain in an unready state, i.e. readiness
// probes will fail for those containers. This is because when autofail is disabled (i.e.
// Patroni is "paused"), Patroni will not attempt to start the PG database on a primary or
// replica.  Therefore, the user must re-enable autofail following a minor upgrade during which
// autofail was disabled in order to fully bring the cluster (which includes the primary and
// all replicas) back online.
func UpgradeWithAutofailDisabled(clientset *kubernetes.Clientset, restclient *rest.RESTClient,
	cluster crv1.Pgcluster, upgradeTaskName, namespace string) {

	log.Debug("Upgrade: Autofail disabled.... ", upgradeTaskName)

	// get the upgrade task
	upgradeTask := crv1.Pgtask{}
	found, err := kubeapi.Getpgtask(restclient, &upgradeTask, upgradeTaskName, namespace)
	if err != nil {
		log.Error(err)
		return
	} else if !found {
		log.Error("cound not find pgtask for minor upgrade")
		return
	}

	replicaTargets := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA]
	primaryTargetName := upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY]

	// upgrade everything at once and then mark as complete
	imageNamePatch, err := createImageNamePatch(cluster, operator.Pgo.Cluster.CCPImagePrefix,
		upgradeTask.Spec.Parameters["CCPImageTag"])
	if err != nil {
		log.Errorf("error creating container image name patch during minor upgrade of cluster %s",
			cluster.Name)
		log.Error(err)
		return
	}

	repList := strings.Split(replicaTargets, ",")
	for _, replicaTargetName := range repList {
		err = kubeapi.PatchDeploymentStrategicMerge(clientset, replicaTargetName, namespace,
			imageNamePatch)
		if err != nil {
			log.Error(err)
			log.Error("error in doing replica minor upgrade")
			return
		}
	}

	err = kubeapi.PatchDeploymentStrategicMerge(clientset, primaryTargetName, namespace,
		imageNamePatch)
	if err != nil {
		log.Error(err)
		log.Error("error in doing primary minor upgrade")
		return
	}

	// upgrade replica targets in task.
	upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_REPLICA] = ""
	// upgrade primary target in task.
	upgradeTask.Spec.Parameters[config.LABEL_UPGRADE_PRIMARY] = ""

	// No other deployments left to upgrade, complete the upgrade
	completeUpgrade(clientset, restclient, &upgradeTask, false, cluster.Spec.Name, namespace)

	publishMinorUpgradeCompleteEvent(&upgradeTask, &cluster, namespace)

	return
}

// createImageNamePatch creates and returns a string containing the JSON structure needed to
// patch a Deployment specification in order update the image name for the database container, as
// well as any sidecar containers (e.g. collect, pgbadger and/or crunchyadm) that are enabled
// for the cluster.  An example of a string that would be generated by this function, in this case
// to update the "database" container only, would be as follows:
//
//  '{"spec":{"template":{"spec":{"containers":[{"name":"database","image":"new-image-name"}'
//
func createImageNamePatch(cluster crv1.Pgcluster, ccpImagePrefix, ccpImageTag string) (string, error) {

	// These types represent json structure will be utilized by this function to update
	// (i.e. patch) the image names of any/all containers in a PG cluster Deployment:
	type patchDeploymentContainers struct {
		Name  string `json:"name"`
		Image string `json:"image"`
	}
	type patchDeploymentPodSpec struct {
		Containers []patchDeploymentContainers `json:"containers"`
	}
	type patchDeploymentTemplate struct {
		PatchDeploymentPodSpec patchDeploymentPodSpec `json:"spec"`
	}
	type patchDeploymentSpec struct {
		PatchDeploymentTemplate patchDeploymentTemplate `json:"template"`
	}
	type patchImageName struct {
		PatchDeploymentSpec patchDeploymentSpec `json:"spec"`
	}

	var containersToPatch []patchDeploymentContainers

	// add the database container, which will always be patched
	databaseContainer := patchDeploymentContainers{
		Name:  "database",
		Image: ccpImagePrefix + "/" + cluster.Spec.CCPImage + ":" + ccpImageTag,
	}

	// see if the image name is overridden
	if strings.Contains(cluster.Spec.CCPImage, "gis-ha") &&
		operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA] != "" {
		databaseContainer.Image = operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA]
	} else if operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA] != "" {
		databaseContainer.Image = operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA]
	}

	containersToPatch = append(containersToPatch, databaseContainer)

	// determine if pgbadger is enabled for the cluster using label "crunchy_collect"
	collectEnabled, err := strconv.ParseBool(cluster.Labels[config.LABEL_COLLECT])
	if err != nil {
		return "", err
	} else if collectEnabled {
		collectContainer := patchDeploymentContainers{
			Name:  "collect",
			Image: ccpImagePrefix + "/" + collectCCPImage + ":" + ccpImageTag,
		}
		// see if the image name is overridden
		if operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_COLLECT] != "" {
			collectContainer.Image = operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_COLLECT]
		}
		containersToPatch = append(containersToPatch, collectContainer)
	}

	// determine if pgbadger is enabled for the cluster using label "crunchy-pgbadger"
	badgerEnabled, err := strconv.ParseBool(cluster.Labels[config.LABEL_BADGER])
	if err != nil {
		return "", err
	} else if badgerEnabled {
		badgerContainer := patchDeploymentContainers{
			Name:  "pgbadger",
			Image: ccpImagePrefix + "/" + pgBadgerCCPImage + ":" + ccpImageTag,
		}
		// see if the image name is overridden
		if operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_PGBADGER] != "" {
			badgerContainer.Image = operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_PGBADGER]
		}
		containersToPatch = append(containersToPatch, badgerContainer)
	}

	// check pgo.yaml to see if the crunchyadm sidecar is enabled for the current pgo installation
	if operator.Pgo.Cluster.EnableCrunchyadm {
		crunchyadmContainer := patchDeploymentContainers{
			Name:  "crunchyadm",
			Image: ccpImagePrefix + "/" + crunchyadmCCPImage + ":" + ccpImageTag,
		}
		// see if the image name is overridden
		if operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_ADMIN] != "" {
			crunchyadmContainer.Image = operator.ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_ADMIN]
		}
		containersToPatch = append(containersToPatch, crunchyadmContainer)
	}

	// create the json structure required to patch (i.e. update) the images defined in
	// containersToPatch
	patchDeployPodSpec := patchDeploymentPodSpec{
		Containers: containersToPatch,
	}
	patchDeployTemplate := patchDeploymentTemplate{
		PatchDeploymentPodSpec: patchDeployPodSpec,
	}
	patchDeploySpec := patchDeploymentSpec{
		PatchDeploymentTemplate: patchDeployTemplate,
	}
	patchImgName := patchImageName{
		PatchDeploymentSpec: patchDeploySpec,
	}

	data, err := json.Marshal(patchImgName)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// completeUpgrade - makes any finishing changes required to complete the upgrade and
// does final updates to the task and cluster.
func completeUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgradeTask *crv1.Pgtask, autoFail bool, clusterName, namespace string) {

	log.Debug("Minor Upgrade: Completing...")

	// update cluster image tag (CCPImageTag) once the upgrade is complete
	upgradedImageTag := upgradeTask.Spec.Parameters["CCPImageTag"]
	updateClusterCCPImage(restclient, upgradedImageTag, clusterName, namespace)

	removeMinorUpgradeLabelFromCluster(clientset, restclient, clusterName, namespace)

	//update the upgrade CRD status to completed
	log.Debugf("update pgtask status %s to %s ", upgradeTask.Spec.Name, crv1.CompletedStatus)
	upgradeTask.Spec.Status = crv1.CompletedStatus
	err := kubeapi.Updatepgtask(restclient, upgradeTask, upgradeTask.Spec.Name, namespace)
	if err != nil {
		log.Error("error in updating minor upgrade pgtask to completed status " + err.Error())
	}

}

func updateClusterCCPImage(restclient *rest.RESTClient, upgradedCCPImageTag, clusterName, namespace string) {

	//update the CRD with the new image tag to maintain the truth
	log.Info("updating the ccpimagetag in the pgcluster CR.")
	err := util.Patch(restclient, "/spec/ccpimagetag", upgradedCCPImageTag, crv1.PgclusterResourcePlural, clusterName, namespace)

	if err != nil {
		log.Error("error patching pgcluster in upgrade" + err.Error())
	}

}

// labelClusterForMinorUpgrade - applies a minor upgrade label to userlabels collection on pgcluster
func labelpgClusterForMinorUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, clusterName, namespace string) error {

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found {
		log.Errorf("could not find pgcluster %s with labels", clusterName)
		return err
	}

	cluster.Spec.UserLabels[config.LABEL_MINOR_UPGRADE] = config.LABEL_UPGRADE_IN_PROGRESS
	err = util.PatchClusterCRD(restclient, cluster.Spec.UserLabels, &cluster, namespace)
	if err != nil {
		log.Errorf("Minor Upgrade: could not patch pgcluster %s with labels", clusterName)
		return err
	}

	return err
}

func removeMinorUpgradeLabelFromCluster(clientset *kubernetes.Clientset, restclient *rest.RESTClient, clusterName, namespace string) error {

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found {
		log.Errorf("could not find pgcluster %s with labels", clusterName)
		return err
	}

	// update minor upgade to complete.
	cluster.Spec.UserLabels[config.LABEL_MINOR_UPGRADE] = config.LABEL_UPGRADE_COMPLETED

	err = util.PatchClusterCRD(restclient, cluster.Spec.UserLabels, &cluster, namespace)
	if err != nil {
		log.Errorf("Minor Upgrade: could not patch pgcluster %s with labels", clusterName)
		return err
	}

	return err
}

// publishMinorUpgradeStartedEvent - indicates the upgrade has started.
func publishMinorUpgradeStartedEvent(upgradeTask *crv1.Pgtask, cluster *crv1.Pgcluster, namespace string) {

	//publish event for failover
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventUpgradeClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  upgradeTask.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventUpgradeCluster,
		},
		Clustername: cluster.Name,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err)
	}

}

// publishMinorUpgradeCompleteEvent - indicates that a minor upgrade has successfully completed
func publishMinorUpgradeCompleteEvent(upgradeTask *crv1.Pgtask, cluster *crv1.Pgcluster, namespace string) {

	//capture the cluster creation event
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventUpgradeClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  cluster.Spec.UserLabels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventUpgradeClusterCompleted,
		},
		Clustername: cluster.Name,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}
