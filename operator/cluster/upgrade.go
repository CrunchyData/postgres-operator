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
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Store image names as constants to use later
const (
	postgresImage      = "crunchy-postgres"
	postgresHAImage    = "crunchy-postgres-ha"
	postgresGISImage   = "crunchy-postgres-gis"
	postgresGISHAImage = "crunchy-postgres-gis-ha"
)

// store the replica postfix string
const replicaServicePostfix = "-replica"

// AddUpgrade implements the upgrade workflow in accordance with the received pgtask
// the general process is outlined below:
// 1) get the existing pgcluster CRD instance that matches the name provided in the pgtask
// 2) Patch the existing services
// 3) Determine the current Primary PVC
// 4) Scale down existing replicas and store the number for recreation
// 5) Delete the various resources that will need to be recreated
// 6) Recreate the BackrestRepo secret, since the key encryption algorithm has been updated
// 7) Update the existing pgcluster CRD instance to match the current version
// 8) Submit the pgcluster CRD for recreation
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgtask, namespace string) {

	upgradeTargetClusterName := upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]

	log.Debugf("started upgrade of cluster: %s", upgradeTargetClusterName)

	// publish our upgrade event
	PublishUpgradeEvent(events.EventUpgradeCluster, namespace, upgrade, "")

	// the pgcluster to hold the new pgcluster CR
	pgcluster := crv1.Pgcluster{}
	// get the pgcluster CRD
	if _, err := kubeapi.Getpgcluster(restclient, &pgcluster, upgradeTargetClusterName, namespace); err != nil {
		errormessage := "cound not find pgcluster for pgcluster upgrade"
		log.Errorf("%s Error: %s", errormessage, err)
		PublishUpgradeEvent(events.EventUpgradeClusterFailure, namespace, upgrade, errormessage)
		return
	}

	// update the workflow status to 'in progress' while the upgrade takes place
	updateUpgradeWorkflow(restclient, namespace, upgrade.ObjectMeta.Labels[crv1.PgtaskWorkflowID], crv1.PgtaskUpgradeInProgress)

	// grab the existing pgo version
	oldpgoversion := pgcluster.ObjectMeta.Labels[config.LABEL_PGO_VERSION]

	// patch the existing services to add the potentially missing ports
	patchServices(clientset, pgcluster.Name, namespace)

	// grab the current primary value. In differnet versions of the Operator, this was stored in multiple
	// ways depending on version. If the 'master' role is available on a particular pod (starting in 4.2.0),
	// this is the most authoritative option. Next, in the current version, the current primary value is stored
	// in an annotation on the pgcluster CRD and should be used if available and the master pod cannot be identified.
	// Next, if the current primary label is present (used by previous Operator versions), we will use that.
	// Finally, if none of the above is available, we will set the default pgcluster name as the current primary value
	currentPrimaryFromPod := getMasterPodDeploymentName(clientset, &pgcluster)
	currentPrimaryFromAnnotation := pgcluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY]
	currentPrimaryFromLabel := pgcluster.ObjectMeta.Labels[config.LABEL_CURRENT_PRIMARY]

	// compare the three values, and return the correct current primary value
	currentPrimary := getCurrentPrimary(pgcluster.Name, currentPrimaryFromPod, currentPrimaryFromAnnotation, currentPrimaryFromLabel)

	// remove and count the existing replicas
	replicas := handleReplicas(clientset, restclient, pgcluster.Name, currentPrimary, namespace)
	SetReplicaNumber(&pgcluster, replicas)

	// create the 'pgha-config' configmap while taking the init value from any existing 'pgha-default-config' configmap
	createUpgradePGHAConfigMap(clientset, &pgcluster, namespace)

	// delete the existing pgcluster CRDs and other resources that will be recreated
	deleteBeforeUpgrade(clientset, restclient, pgcluster.Name, currentPrimary, namespace, pgcluster.Spec.Standby)

	// recreate new Backrest Repo secret that was just deleted
	recreateBackrestRepoSecret(clientset, upgradeTargetClusterName, namespace, operator.PgoNamespace)

	// set proper values for the pgcluster that are updated between CR versions
	preparePgclusterForUpgrade(&pgcluster, upgrade.Spec.Parameters, oldpgoversion, currentPrimary)

	// create a new workflow for this recreated cluster
	workflowid, err := createClusterRecreateWorkflowTask(restclient, pgcluster.Name, namespace, upgrade.Spec.Parameters[config.LABEL_PGOUSER])
	if err != nil {
		// we will log any errors here, but will attempt to continue to submit the cluster for recreation regardless
		log.Errorf("error generating a new workflow task for the recreation of the upgraded cluster %s, Error: %s", pgcluster.Name, err)
	}

	// update pgcluster CRD workflow ID
	pgcluster.Spec.UserLabels[config.LABEL_WORKFLOW_ID] = workflowid

	err = kubeapi.Createpgcluster(restclient, &pgcluster, namespace)
	if err != nil {
		log.Errorf("error submitting upgraded pgcluster CRD for cluster recreation of cluster %s, Error: %v", pgcluster.Name, err)
	} else {
		log.Debugf("upgraded cluster %s submitted for recreation, workflowid: %s", pgcluster.Name)
	}

	// submit an event now that the new pgcluster has been submitted to the cluster creation process
	PublishUpgradeEvent(events.EventUpgradeClusterCreateSubmitted, namespace, upgrade, "")

	log.Debugf("finished main upgrade workflow for cluster: %s", upgradeTargetClusterName)

}

// patchServices updates the existing services to include the necessary ports for this
// version of the Postgres Operator
func patchServices(clientset *kubernetes.Clientset, serviceName, namespace string) {

	// add the Patroni port mapping to both the primary and replica services
	portPatch := kubeapi.PortPatch("patroni", "TCP", 8009, 8009)

	// for both service definitions, use a JSON patch 'add' if found
	// look for primary service
	primaryService, found, _ := kubeapi.GetService(clientset, serviceName, namespace)
	if found {
		// check through service ports and see if the Patroni port is already set
		var patroniFound bool
		for _, ports := range primaryService.Spec.Ports {
			if ports.Name == "patroni" {
				patroniFound = true
			}
		}
		// if the Patroni port does not exist, patch it into the service definition
		if !patroniFound {
			// update ports if found
			kubeapi.PatchServicePort(clientset, serviceName, namespace, "add", "/spec/ports/-", portPatch[0])
		}

		// replace the selector section of the master service definition to match the Patroni required values
		masterSelectorPatches := kubeapi.SelectorPatches(serviceName, "master")
		kubeapi.PatchServiceSelector(clientset, serviceName, namespace, "replace", "/spec/selector", masterSelectorPatches[0])
	}

	// look for replica service, update if found
	replicaService, found, _ := kubeapi.GetService(clientset, serviceName+replicaServicePostfix, namespace)
	if found {
		// check through service ports and see if the Patroni port is already set
		var patroniFound bool
		for _, ports := range replicaService.Spec.Ports {
			if ports.Name == "patroni" {
				patroniFound = true
			}
		}
		// if the Patroni port does not exist, patch it into the service definition
		if !patroniFound {
			// update ports if found
			kubeapi.PatchServicePort(clientset, serviceName+replicaServicePostfix, namespace, "add", "/spec/ports/-", portPatch[0])
		}

		// replace the selector section of the replica service definition to match the Patroni required values
		replicaSelectorPatches := kubeapi.SelectorPatches(serviceName, "replica")
		kubeapi.PatchServiceSelector(clientset, serviceName+replicaServicePostfix, namespace, "replace", "/spec/selector", replicaSelectorPatches[0])
	}

}

// getMasterPod searches through the pods associated with this pgcluster for the 'master' pod,
// if set. This will not be applicable to releases before the Operator 4.2.0 HA features were
// added. If this label does not exist or is otherwise not set as expected, return an empty
// string value and call an alternate function to determine the current primary pod.
func getMasterPodDeploymentName(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster) string {
	// first look for a 'master' role label on the current primary deployment
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGHA_ROLE, "master")
	pods, err := kubeapi.GetPods(clientset, selector, cluster.Namespace)
	if err != nil {
		log.Errorf("no pod with the master role label was found for cluster %s. Error: ", cluster.Name, err)
		return ""
	}

	// if no pod with that role is found, return an empty string since the current master pod
	// cannot be established
	if len(pods.Items) < 1 {
		log.Debugf("no pod with the master role label was found for cluster %s", cluster.Name)
		return ""
	}
	// similarly, if more than one pod with that role is found, return an empty string since the
	// true master pod cannot be determined
	if len(pods.Items) > 1 {
		log.Errorf("%v pods with the master role label were found for cluster %s. There should only be one.",
			len(pods.Items), cluster.Name)
		return ""
	}
	// if only one pod was returned, this is the proper primary pod
	primaryPod := pods.Items[0]
	// now return the master pod's deployment name
	return primaryPod.Labels[config.LABEL_DEPLOYMENT_NAME]
}

// getCurrentPrimary returns the correct current primary value to use for the upgrade.
// the deployment name of the pod with the 'master' role is considered the most authoritative,
// followed by the CRD's 'current-primary' annotation, followed then by the current primary
// label. If none of these values are set, return the default name.
func getCurrentPrimary(clusterName, podPrimary, crPrimary, labelPrimary string) string {
	// the master pod is the preferred source of truth, as it will be correct
	// for 4.2 pgclusters and beyond, regardless of failover method
	if podPrimary != "" {
		return podPrimary
	}

	// the CRD annotation is the next preferred value
	if crPrimary != "" {
		return crPrimary
	}

	// the current primary label should be used if the spec value and master pod
	// values are missing
	if labelPrimary != "" {
		return labelPrimary
	}

	// if none of these are set, return the pgcluster name as the default
	return clusterName
}

// handleReplicas deletes all pgreplicas related to the pgcluster to be ugpraded, then returns the number
// of pgreplicas that were found. This will delete any PVCs that match the existing pgreplica CRs, but
// will leave any other PVCs, whether they are from the current primary, previous primaries that are now
// unassociated because of a failover or the backrest-shared-repo PVC. The total number of current replicas
// will also be captured during this process so that the correct number of replicas can be recreated.
func handleReplicas(clientset *kubernetes.Clientset, restclient *rest.RESTClient, clusterName, currentPrimaryPVC, namespace string) string {
	log.Debugf("deleting pgreplicas and noting the number found for cluster %s", clusterName)
	replicaList := crv1.PgreplicaList{}
	// Save the number of found replicas for this cluster
	numReps := 0
	if err := kubeapi.Getpgreplicas(restclient, &replicaList, namespace); err != nil {
		log.Errorf("unable to get pgreplicas. Error: %s", err)
	}

	// go through the list of found replicas
	for index := range replicaList.Items {
		if replicaList.Items[index].Spec.ClusterName == clusterName {
			log.Debugf("scaling down pgreplica: %s", replicaList.Items[index].Name)
			ScaleDownBase(clientset, restclient, &replicaList.Items[index], namespace)
			log.Debugf("deleting pgreplica CRD: %s", replicaList.Items[index].Name)
			kubeapi.Deletepgreplica(restclient, replicaList.Items[index].Name, namespace)
			// if the existing replica PVC is not being used as the primary PVC, delete
			// note this will not remove any leftover PVCs from previous failovers,
			// those will require manual deletion so as to avoid any accidental
			// deletion of valid PVCs.
			if replicaList.Items[index].Name != currentPrimaryPVC {
				kubeapi.DeletePVC(clientset, replicaList.Items[index].Name, namespace)
				log.Debugf("deleting replica pvc: %s", replicaList.Items[index].Name)
			}

			// regardless of whether the pgreplica PVC is being used as the primary or not, we still
			// want to count it toward the number of replicas to create
			numReps++
		}
	}
	// return the number of pgreplicas as a string
	return strconv.Itoa(numReps)
}

// SetReplicaNumber sets the pgcluster's replica value based off of the number of pgreplicas
// discovered during the deletion process. This is necessary because the pgcluser will only
// include the number of replicas created when the pgcluster was first generated
// (e.g. pgo create cluster hippo --replica-count=2) but will not included any replicas
// created using the 'pgo scale' command
func SetReplicaNumber(pgcluster *crv1.Pgcluster, numReplicas string) {

	pgcluster.Spec.Replicas = numReplicas
}

// deleteBeforeUpgrade deletes the deployments, services, pgcluster, jobs, tasks and default configmaps before attempting
// to upgrade the pgcluster deployment. This preserves existing secrets, non-standard configmaps and service definitions
// for use in the newly upgraded cluster.
func deleteBeforeUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, clusterName, currentPrimary, namespace string, isStandby bool) {

	// first, get all deployments for the pgcluster in question
	deployments, err := kubeapi.GetDeployments(clientset, config.LABEL_PG_CLUSTER+"="+clusterName, namespace)
	if err != nil {
		log.Errorf("unable to get deployments. Error: %s", err)
	}

	// next, delete those deployments
	for index := range deployments.Items {
		kubeapi.DeleteDeployment(clientset, deployments.Items[index].Name, namespace)
	}

	// wait until the backrest shared repo pod deployment has been deleted before continuing
	waitStatus := deploymentWait(clientset, namespace, clusterName+"-backrest-shared-repo", 180, 10)
	log.Debug(waitStatus)
	// wait until the primary pod deployment has been deleted before continuing
	waitStatus = deploymentWait(clientset, namespace, currentPrimary, 180, 10)
	log.Debug(waitStatus)

	// delete the pgcluster
	kubeapi.Deletepgcluster(restclient, clusterName, namespace)

	// delete all existing job references
	kubeapi.DeleteJobs(clientset, config.LABEL_PG_CLUSTER+"="+clusterName, namespace)

	// delete all existing pgtask references except for the upgrade task
	// Note: this will be deleted by the existing pgcluster creation process once the
	// updated pgcluster created and processed by the cluster controller
	if err = deleteNonupgradePgtasks(restclient, config.LABEL_PG_CLUSTER+"="+clusterName, namespace); err != nil {
		log.Errorf("error while deleting pgtasks for cluster %s, Error: %v", clusterName, err)
	}

	// delete the leader configmap used by the Postgres Operator since this information may change after
	// the upgrade is complete
	// Note: deletion is required for cluster recreation
	checkDeleteConfigmap(clientset, clusterName+"-leader", namespace)

	// delete the '<cluster-name>-pgha-default-config' configmap, if it exists so the config syncer
	// will not try to use it instead of '<cluster-name>-pgha-config'
	checkDeleteConfigmap(clientset, clusterName+"-pgha-default-config", namespace)

	// delete the backrest repo config secret, since key encryption has been updated from RSA to EdDSA
	kubeapi.DeleteSecret(clientset, clusterName+"-backrest-repo-config", namespace)
}

// deploymentWait is modified from cluster.waitForDeploymentDelete. It simply waits for the current primary deployment
// deletion to complete before proceding with the rest of the pgcluster upgrade.
func deploymentWait(clientset *kubernetes.Clientset, namespace, deploymentName string, timeoutSecs, periodSecs time.Duration) string {
	timeout := time.After(timeoutSecs * time.Second)
	tick := time.Tick(periodSecs * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Sprintf("Timed out waiting for deployment to be deleted: [%s]", deploymentName)
		case <-tick:
			_, deploymentFound, _ := kubeapi.GetDeployment(clientset, deploymentName, namespace)
			if !(deploymentFound) {
				return fmt.Sprintf("Deploment %s has been deleted.", deploymentName)
			}
			log.Debugf("deployment deleted: %t", !deploymentFound)
		}
	}
}

// deleteConfigmap will delete a configmap after checking if it exists
// this is done to avoid unnecessary errors during attempted deletes of
// configmaps that don't exist
func checkDeleteConfigmap(clientset *kubernetes.Clientset, configmap, namespace string) {
	// check for configmap
	_, found := kubeapi.GetConfigMap(clientset, configmap, namespace)
	// if found, delete it
	if found {
		kubeapi.DeleteConfigMap(clientset, configmap, namespace)
	}
}

// deleteNonupgradePgtasks deletes all existing pgtasks by selector with the exception of the
// upgrade task itself
func deleteNonupgradePgtasks(client *rest.RESTClient, selector, namespace string) error {
	taskList := crv1.PgtaskList{}
	err := kubeapi.GetpgtasksBySelector(client, &taskList, selector, namespace)
	if err != nil {
		return err
	}

	// get the pgtask list
	for _, v := range taskList.Items {
		// if the pgtask is not for the upgrade, delete it
		if v.ObjectMeta.Name != v.Name+"-"+config.LABEL_UPGRADE {
			err = kubeapi.Deletepgtask(client, v.ObjectMeta.Name, namespace)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// createUpgradePGHAConfigMap is a modified copy of CreatePGHAConfigMap from operator/clusterutilities.go
// It also creates a configMap that will be utilized to store configuration settings for a PostgreSQL,
// cluster, but with the added step of looking for an existing configmap,
// "<clustername>-pgha-default-config". If that configmap exists, it will get the init value, as this is
// needed for the proper reinitialziation of Patroni.
func createUpgradePGHAConfigMap(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster,
	namespace string) error {

	labels := make(map[string]string)
	labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	labels[config.LABEL_PG_CLUSTER] = cluster.Name
	labels[config.LABEL_PGHA_CONFIGMAP] = "true"

	data := make(map[string]string)

	// if the "pgha-default-config" config map exists, this cluster is being upgraded
	// and should use the initialization value from this existing configmap
	defaultConfigmap, found := kubeapi.GetConfigMap(clientset, cluster.Name+"-pgha-default-config", namespace)
	if found {
		data[operator.PGHAConfigInitSetting] = defaultConfigmap.Data[operator.PGHAConfigInitSetting]
	} else {
		// set "init" to true in the postgres-ha configMap
		data[operator.PGHAConfigInitSetting] = "true"
	}

	// if a standby cluster then we want to create replicas using the S3 pgBackRest repository
	// (and not the local in-cluster pgBackRest repository)
	if cluster.Spec.Standby {
		data[operator.PGHAConfigReplicaBootstrapRepoType] = "s3"
	}

	configmap := &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   cluster.Name + "-" + operator.PGHAConfigMapSuffix,
			Labels: labels,
		},
		Data: data,
	}

	if err := kubeapi.CreateConfigMap(clientset, configmap, namespace); err != nil {
		return err
	}

	return nil
}

// recreateBackrestRepoSecret deletes and recreates the secret for the pgBackRest repo. This is needed
// because the key encryption algorithm has been updated from RSA to EdDSA
func recreateBackrestRepoSecret(clientset *kubernetes.Clientset, clustername, namespace, operatorNamespace string) {
	if err := util.CreateBackrestRepoSecrets(clientset,
		util.BackrestRepoConfig{
			BackrestS3Key:       "", // these are set to empty so that it can be generated
			BackrestS3KeySecret: "",
			ClusterName:         clustername,
			ClusterNamespace:    namespace,
			OperatorNamespace:   operatorNamespace,
		}); err != nil {
		log.Errorf("error generating new backrest repo secrets during pgcluster ugprade: %v", err)
	}
}

// preparePgclusterForUpgrade specifically updates the existing CRD instance to set correct values
// for the current Postgres Operator version, updating or deleting values where appropriate, and sets
// an expected status so that the CRD object can be recreated.
func preparePgclusterForUpgrade(pgcluster *crv1.Pgcluster, parameters map[string]string, oldpgoversion, currentPrimary string) {

	// first, update the PGO version references to the current Postgres Operator version
	pgcluster.ObjectMeta.Labels[config.LABEL_PGO_VERSION] = parameters[config.LABEL_PGO_VERSION]
	pgcluster.Spec.UserLabels[config.LABEL_PGO_VERSION] = parameters[config.LABEL_PGO_VERSION]

	// since the current primary label is not used in this version of the Postgres Operator,
	// delete it before moving on to other upgrade tasks
	delete(pgcluster.ObjectMeta.Labels, config.LABEL_CURRENT_PRIMARY)

	// next, update the image name to the appropriate image
	if pgcluster.Spec.CCPImage == postgresImage {
		pgcluster.Spec.CCPImage = postgresHAImage
	}

	if pgcluster.Spec.CCPImage == postgresGISImage {
		pgcluster.Spec.CCPImage = postgresGISHAImage
	}

	// if there are not any annotations on the current pgcluster (which may be the case depending on
	// which version we are upgrading from), create a new map to hold them
	if pgcluster.Annotations == nil {
		pgcluster.Annotations = make(map[string]string)
	}
	// update our pgcluster annotation with the correct current primary value
	pgcluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY] = currentPrimary

	// if the current primary value is set to a different value than the default deployment label, a failover has occurred.
	// update the deployment label to match this updated value so that the deployment will match the underlying PVC name.
	// since we cannot assume the state of the original primary's PVC is valid after the upgrade, this ensures the new
	// base primary name will match the deployment name. Please note, going forward, failovers to other replicas will
	// result in a new currentprimary value in the CRD annotations, but the deployment label will stay the same, in keeping with
	// the current deployment naming method. In simpler terms, this deployment value is the 'primary deployment' name
	// for this cluster.
	pgcluster.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME] = currentPrimary

	// update the image tag to the standard value set in the Postgres Operator's main
	// configuration (which has already been verified to match the MAJOR PostgreSQL version)
	pgcluster.Spec.CCPImageTag = parameters[config.LABEL_CCP_IMAGE_KEY]

	// set a default autofail value of "true" to enable Patroni's replication. If left to an existing
	// value of "false," Patroni will be in a paused state and unable to sync all replicas to the
	// current timeline
	pgcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] = "true"

	// Don't think we'll need to do this, but leaving the comment for now....
	// pgcluster.ObjectMeta.Labels[config.LABEL_POD_ANTI_AFFINITY] = ""

	// set pgouser to match the default configuration currently in use after the Operator upgrade
	pgcluster.ObjectMeta.Labels[config.LABEL_PGOUSER] = parameters[config.LABEL_PGOUSER]

	// if the exporter port is not set, set to the configuration value for the default configuration
	if pgcluster.Spec.ExporterPort == "" {
		pgcluster.Spec.ExporterPort = operator.Pgo.Cluster.ExporterPort
	}

	// if the pgbadger port is not set, set to the configuration value for the default configuration
	if pgcluster.Spec.PGBadgerPort == "" {
		pgcluster.Spec.PGBadgerPort = operator.Pgo.Cluster.PGBadgerPort
	}

	// ensure that the pgo-backrest label is set to 'true' since pgbackrest is required for normal
	// cluster operations in this version of the Postgres Operator
	pgcluster.ObjectMeta.Labels[config.LABEL_BACKREST] = "true"

	// add a label with the PGO version upgraded from and to
	pgcluster.Annotations[config.ANNOTATION_UPGRADE_INFO] = "From_" + oldpgoversion + "_to_" + parameters[config.LABEL_PGO_VERSION]
	// update the "is upgraded" label to indicate cluster has been upgraded
	pgcluster.Annotations[config.ANNOTATION_IS_UPGRADED] = "true"

	// set the default memory resource values
	setMemoryResources(pgcluster)

	// set the default CCPImagePrefix, if empty
	if pgcluster.Spec.CCPImagePrefix == "" {
		pgcluster.Spec.CCPImagePrefix = operator.Pgo.Cluster.CCPImagePrefix
	}

	// set the default PGOImagePrefix, if empty
	if pgcluster.Spec.PGOImagePrefix == "" {
		pgcluster.Spec.PGOImagePrefix = operator.Pgo.Pgo.PGOImagePrefix
	}

	// finally, clear the resource version and status messages, and set to the appropriate
	// state for use by the pgcluster controller
	pgcluster.ObjectMeta.ResourceVersion = ""
	pgcluster.Spec.Status = ""
	pgcluster.Status.State = crv1.PgclusterStateCreated
	pgcluster.Status.Message = "Created, not processed yet"
}

// setMemoryResources sets the default memory values for pgBackrest, pgBouncer and the
// PostgreSQL instance in a cluster if they are not already set
func setMemoryResources(pgcluster *crv1.Pgcluster) {
	// check if BackrestResources exists, if not create the ResourceList
	if pgcluster.Spec.BackrestResources == nil {
		pgcluster.Spec.BackrestResources = v1.ResourceList{}
	}

	// if the memory value is either not set or is zero, set the default configuration value
	if pgcluster.Spec.BackrestResources.Memory() == nil || pgcluster.Spec.BackrestResources.Memory().IsZero() {
		pgcluster.Spec.BackrestResources[v1.ResourceMemory] = config.DefaultBackrestResourceMemory
	}

	// check if PgBouncer.Resources exists, if not create the ResourceList
	if pgcluster.Spec.PgBouncer.Resources == nil {
		pgcluster.Spec.PgBouncer.Resources = v1.ResourceList{}
	}

	// if the memory value is either not set or is zero, set the default configuration value
	if pgcluster.Spec.PgBouncer.Resources.Memory() == nil || pgcluster.Spec.PgBouncer.Resources.Memory().IsZero() {
		pgcluster.Spec.PgBouncer.Resources[v1.ResourceMemory] = config.DefaultPgBouncerResourceMemory
	}

	// check if PgBouncer.Resources exists, if not create the ResourceList
	if pgcluster.Spec.PgBouncer.Resources == nil {
		pgcluster.Spec.PgBouncer.Resources = v1.ResourceList{}
	}

	// check if BackrestResources exists, if not create the ResourceList
	if pgcluster.Spec.Resources == nil {
		pgcluster.Spec.Resources = v1.ResourceList{}
	}

	// if the memory value is either not set or is zero, set the default configuration value
	if pgcluster.Spec.Resources.Memory() == nil || pgcluster.Spec.Resources.Memory().IsZero() {
		pgcluster.Spec.Resources[v1.ResourceMemory] = config.DefaultInstanceResourceMemory
	}
}

// createClusterRecreateWorkflowTask creates a cluster creation task for the upgraded cluster's recreation
// to maintain the expected workflow and tasking
func createClusterRecreateWorkflowTask(restclient *rest.RESTClient, clusterName, ns, pgouser string) (string, error) {
	// create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = clusterName + "-" + crv1.PgtaskWorkflowCreateClusterType
	spec.TaskType = crv1.PgtaskWorkflow

	spec.Parameters = make(map[string]string)
	spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format(time.RFC3339)
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
		return "", err
	}
	spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	err = kubeapi.Createpgtask(restclient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err
}

// updateUpgradeWorkflow updates a Workflow with the current state of the pgcluster upgrade task
// modified from the cluster.UpdateCloneWorkflow function
func updateUpgradeWorkflow(client *rest.RESTClient, namespace, workflowID, status string) error {
	log.Debugf("pgcluster upgrade workflow: update workflow [%s]", workflowID)

	// we have to look up the name of the workflow bt the workflow ID, which
	// involves using a selector
	selector := fmt.Sprintf("%s=%s", crv1.PgtaskWorkflowID, workflowID)
	taskList := crv1.PgtaskList{}

	if err := kubeapi.GetpgtasksBySelector(client, &taskList, selector, namespace); err != nil {
		log.Errorf("pgcluster upgrade workflow: could not get workflow [%s]", workflowID)
		return err
	}

	// if there is not one unique result, then we should display an error here
	if len(taskList.Items) != 1 {
		errorMsg := fmt.Sprintf("pgcluster upgrade workflow: workflow [%s] not found", workflowID)
		log.Errorf(errorMsg)
		return errors.New(errorMsg)
	}

	// get the first task and update on the current status based on how it is
	// progressing
	task := taskList.Items[0]
	task.Spec.Parameters[status] = time.Now().Format(time.RFC3339)

	if err := kubeapi.Updatepgtask(client, &task, task.Name, namespace); err != nil {
		log.Errorf("pgcluster upgrade workflow: could not update workflow [%s] to status [%s]", workflowID, status)
		return err
	}

	return nil
}

// PublishUpgradeEvent lets one publish an event related to the upgrade process
func PublishUpgradeEvent(eventType string, namespace string, task *crv1.Pgtask, errorMessage string) {
	// get the boilerplate identifiers
	clusterName, workflowID := getUpgradeTaskIdentifiers(task)
	// set up the event header
	eventHeader := events.EventHeader{
		Namespace: namespace,
		Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
		Topic:     []string{events.EventTopicCluster, events.EventTopicUpgrade},
		Timestamp: time.Now(),
		EventType: eventType,
	}
	// get the event format itself and publish it based on the event type
	switch eventType {
	case events.EventUpgradeCluster:
		publishUpgradeClusterEvent(eventHeader, clusterName, workflowID)
	case events.EventUpgradeClusterCreateSubmitted:
		publishUpgradeClusterCreateEvent(eventHeader, clusterName, workflowID)
	case events.EventUpgradeClusterFailure:
		publishUpgradeClusterFailureEvent(eventHeader, clusterName, workflowID, errorMessage)
	}
}

// getUpgradeTaskIdentifiers returns the cluster name and the workflow ID
func getUpgradeTaskIdentifiers(task *crv1.Pgtask) (string, string) {
	return task.Spec.Parameters[config.LABEL_PG_CLUSTER],
		task.Spec.Parameters[crv1.PgtaskWorkflowID]
}

// publishUpgradeClusterEvent publishes the event when the cluster Upgrade process
// has started
func publishUpgradeClusterEvent(eventHeader events.EventHeader, clustername, workflowID string) {
	// set up the event
	event := events.EventUpgradeClusterFormat{
		EventHeader: eventHeader,
		Clustername: clustername,
		WorkflowID:  workflowID,
	}
	// attempt to publish the event; if it fails, log the error, but keep moving on
	if err := events.Publish(event); err != nil {
		log.Errorf("error publishing event. Error: ", err)
	}
}

// publishUpgradeClusterCreateEvent publishes the event when the cluster Upgrade process
// has reached the point where the upgrade pgcluster CRD is submitted for cluster recreation
func publishUpgradeClusterCreateEvent(eventHeader events.EventHeader, clustername, workflowID string) {
	// set up the event
	event := events.EventUpgradeClusterCreateFormat{
		EventHeader: eventHeader,
		Clustername: clustername,
		WorkflowID:  workflowID,
	}
	// attempt to publish the event; if it fails, log the error, but keep moving on
	if err := events.Publish(event); err != nil {
		log.Errorf("error publishing event. Error: ", err)
	}
}

// publishUpgradeClusterFailureEvent publishes the event when the cluster upgrade process
// has failed, including the error message
func publishUpgradeClusterFailureEvent(eventHeader events.EventHeader, clustername, workflowID, errorMessage string) {
	// set up the event
	event := events.EventUpgradeClusterFailureFormat{
		EventHeader:  eventHeader,
		ErrorMessage: errorMessage,
		Clustername:  clustername,
		WorkflowID:   workflowID,
	}
	// attempt to publish the event; if it fails, log the error, but keep moving on
	if err := events.Publish(event); err != nil {
		log.Errorf("error publishing event. Error: ", err)
	}
}
