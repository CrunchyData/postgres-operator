package cluster

/*
 Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
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

// legacyS3CASHA256Digest informs us if we should override the S3 CA with the
// new bundle
const legacyS3CASHA256Digest = "d1c290ea1e4544dec1934931fbfa1fb2060eb3a0f2239ba191f444ecbce35cbb"

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
func AddUpgrade(clientset kubeapi.Interface, upgrade *crv1.Pgtask, namespace string) {

	upgradeTargetClusterName := upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]

	log.Debugf("started upgrade of cluster: %s", upgradeTargetClusterName)

	// publish our upgrade event
	PublishUpgradeEvent(events.EventUpgradeCluster, namespace, upgrade, "")

	pgcluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(upgradeTargetClusterName, metav1.GetOptions{})
	if err != nil {
		errormessage := "cound not find pgcluster for pgcluster upgrade"
		log.Errorf("%s Error: %s", errormessage, err)
		PublishUpgradeEvent(events.EventUpgradeClusterFailure, namespace, upgrade, errormessage)
		return
	}

	// update the workflow status to 'in progress' while the upgrade takes place
	updateUpgradeWorkflow(clientset, namespace, upgrade.ObjectMeta.Labels[crv1.PgtaskWorkflowID], crv1.PgtaskUpgradeInProgress)

	// grab the existing pgo version
	oldpgoversion := pgcluster.ObjectMeta.Labels[config.LABEL_PGO_VERSION]

	// grab the current primary value. In differnet versions of the Operator, this was stored in multiple
	// ways depending on version. If the 'primary' role is available on a particular pod (starting in 4.2.0),
	// this is the most authoritative option. Next, in the current version, the current primary value is stored
	// in an annotation on the pgcluster CRD and should be used if available and the primary pod cannot be identified.
	// Next, if the current primary label is present (used by previous Operator versions), we will use that.
	// Finally, if none of the above is available, we will set the default pgcluster name as the current primary value
	currentPrimaryFromPod := getPrimaryPodDeploymentName(clientset, pgcluster)
	currentPrimaryFromAnnotation := pgcluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY]
	currentPrimaryFromLabel := pgcluster.ObjectMeta.Labels[config.LABEL_CURRENT_PRIMARY]

	// compare the three values, and return the correct current primary value
	currentPrimary := getCurrentPrimary(pgcluster.Name, currentPrimaryFromPod, currentPrimaryFromAnnotation, currentPrimaryFromLabel)

	// remove and count the existing replicas
	replicas := handleReplicas(clientset, pgcluster.Name, currentPrimary, namespace)
	SetReplicaNumber(pgcluster, replicas)

	// create the 'pgha-config' configmap while taking the init value from any existing 'pgha-default-config' configmap
	createUpgradePGHAConfigMap(clientset, pgcluster, namespace)

	// delete the existing pgcluster CRDs and other resources that will be recreated
	deleteBeforeUpgrade(clientset, pgcluster.Name, currentPrimary, namespace, pgcluster.Spec.Standby)

	// recreate new Backrest Repo secret that was just deleted
	recreateBackrestRepoSecret(clientset, upgradeTargetClusterName, namespace, operator.PgoNamespace)

	// set proper values for the pgcluster that are updated between CR versions
	preparePgclusterForUpgrade(pgcluster, upgrade.Spec.Parameters, oldpgoversion, currentPrimary)

	// create a new workflow for this recreated cluster
	workflowid, err := createClusterRecreateWorkflowTask(clientset, pgcluster.Name, namespace, upgrade.Spec.Parameters[config.LABEL_PGOUSER])
	if err != nil {
		// we will log any errors here, but will attempt to continue to submit the cluster for recreation regardless
		log.Errorf("error generating a new workflow task for the recreation of the upgraded cluster %s, Error: %s", pgcluster.Name, err)
	}

	// update pgcluster CRD workflow ID
	pgcluster.Spec.UserLabels[config.LABEL_WORKFLOW_ID] = workflowid

	_, err = clientset.CrunchydataV1().Pgclusters(namespace).Create(pgcluster)
	if err != nil {
		log.Errorf("error submitting upgraded pgcluster CRD for cluster recreation of cluster %s, Error: %v", pgcluster.Name, err)
	} else {
		log.Debugf("upgraded cluster %s submitted for recreation", pgcluster.Name)
	}

	// submit an event now that the new pgcluster has been submitted to the cluster creation process
	PublishUpgradeEvent(events.EventUpgradeClusterCreateSubmitted, namespace, upgrade, "")

	log.Debugf("finished main upgrade workflow for cluster: %s", upgradeTargetClusterName)

}

// getPrimaryPodDeploymentName searches through the pods associated with this pgcluster for the 'primary' pod,
// if set. This will not be applicable to releases before the Operator 4.2.0 HA features were
// added. If this label does not exist or is otherwise not set as expected, return an empty
// string value and call an alternate function to determine the current primary pod.
func getPrimaryPodDeploymentName(clientset kubernetes.Interface, cluster *crv1.Pgcluster) string {
	// first look for a 'primary' role label on the current primary deployment
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)

	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	// only consider pods that are running
	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(options)

	if err != nil {
		log.Errorf("no pod with the primary role label was found for cluster %s. Error: %s", cluster.Name, err.Error())
		return ""
	}

	// if no pod with that role is found, return an empty string since the current primary pod
	// cannot be established
	if len(pods.Items) < 1 {
		log.Debugf("no pod with the primary role label was found for cluster %s", cluster.Name)
		return ""
	}
	// similarly, if more than one pod with that role is found, return an empty string since the
	// true primary pod cannot be determined
	if len(pods.Items) > 1 {
		log.Errorf("%v pods with the primary role label were found for cluster %s. There should only be one.",
			len(pods.Items), cluster.Name)
		return ""
	}
	// if only one pod was returned, this is the proper primary pod
	primaryPod := pods.Items[0]
	// now return the primary pod's deployment name
	return primaryPod.Labels[config.LABEL_DEPLOYMENT_NAME]
}

// getCurrentPrimary returns the correct current primary value to use for the upgrade.
// the deployment name of the pod with the 'primary' role is considered the most authoritative,
// followed by the CRD's 'current-primary' annotation, followed then by the current primary
// label. If none of these values are set, return the default name.
func getCurrentPrimary(clusterName, podPrimary, crPrimary, labelPrimary string) string {
	// the primary pod is the preferred source of truth, as it will be correct
	// for 4.2 pgclusters and beyond, regardless of failover method
	if podPrimary != "" {
		return podPrimary
	}

	// the CRD annotation is the next preferred value
	if crPrimary != "" {
		return crPrimary
	}

	// the current primary label should be used if the spec value and primary pod
	// values are missing
	if labelPrimary != "" {
		return labelPrimary
	}

	// if none of these are set, return the pgcluster name as the default
	return clusterName
}

// handleReplicas deletes all pgreplicas related to the pgcluster to be upgraded, then returns the number
// of pgreplicas that were found. This will delete any PVCs that match the existing pgreplica CRs, but
// will leave any other PVCs, whether they are from the current primary, previous primaries that are now
// unassociated because of a failover or the backrest-shared-repo PVC. The total number of current replicas
// will also be captured during this process so that the correct number of replicas can be recreated.
func handleReplicas(clientset kubeapi.Interface, clusterName, currentPrimaryPVC, namespace string) string {
	log.Debugf("deleting pgreplicas and noting the number found for cluster %s", clusterName)
	// Save the number of found replicas for this cluster
	numReps := 0
	replicaList, err := clientset.CrunchydataV1().Pgreplicas(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("unable to get pgreplicas. Error: %s", err)
	}

	// go through the list of found replicas
	for index := range replicaList.Items {
		if replicaList.Items[index].Spec.ClusterName == clusterName {
			log.Debugf("scaling down pgreplica: %s", replicaList.Items[index].Name)
			ScaleDownBase(clientset, &replicaList.Items[index], namespace)
			log.Debugf("deleting pgreplica CRD: %s", replicaList.Items[index].Name)
			clientset.CrunchydataV1().Pgreplicas(namespace).Delete(replicaList.Items[index].Name, &metav1.DeleteOptions{})
			// if the existing replica PVC is not being used as the primary PVC, delete
			// note this will not remove any leftover PVCs from previous failovers,
			// those will require manual deletion so as to avoid any accidental
			// deletion of valid PVCs.
			if replicaList.Items[index].Name != currentPrimaryPVC {
				deletePropagation := metav1.DeletePropagationForeground
				clientset.
					CoreV1().PersistentVolumeClaims(namespace).
					Delete(replicaList.Items[index].Name, &metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
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
func deleteBeforeUpgrade(clientset kubeapi.Interface, clusterName, currentPrimary, namespace string, isStandby bool) {

	// first, get all deployments for the pgcluster in question
	deployments, err := clientset.
		AppsV1().Deployments(namespace).
		List(metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER + "=" + clusterName})
	if err != nil {
		log.Errorf("unable to get deployments. Error: %s", err)
	}

	// next, delete those deployments
	for index := range deployments.Items {
		deletePropagation := metav1.DeletePropagationForeground
		_ = clientset.
			AppsV1().Deployments(namespace).
			Delete(deployments.Items[index].Name, &metav1.DeleteOptions{
				PropagationPolicy: &deletePropagation,
			})
	}

	// wait until the backrest shared repo pod deployment has been deleted before continuing
	waitStatus := deploymentWait(clientset, namespace, clusterName+"-backrest-shared-repo", 180, 10)
	log.Debug(waitStatus)
	// wait until the primary pod deployment has been deleted before continuing
	waitStatus = deploymentWait(clientset, namespace, currentPrimary, 180, 10)
	log.Debug(waitStatus)

	// delete the pgcluster
	clientset.CrunchydataV1().Pgclusters(namespace).Delete(clusterName, &metav1.DeleteOptions{})

	// delete all existing job references
	deletePropagation := metav1.DeletePropagationForeground
	clientset.
		BatchV1().Jobs(namespace).
		DeleteCollection(
			&metav1.DeleteOptions{PropagationPolicy: &deletePropagation},
			metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER + "=" + clusterName})

	// delete all existing pgtask references except for the upgrade task
	// Note: this will be deleted by the existing pgcluster creation process once the
	// updated pgcluster created and processed by the cluster controller
	if err = deleteNonupgradePgtasks(clientset, config.LABEL_PG_CLUSTER+"="+clusterName, namespace); err != nil {
		log.Errorf("error while deleting pgtasks for cluster %s, Error: %v", clusterName, err)
	}

	// delete the leader configmap used by the Postgres Operator since this information may change after
	// the upgrade is complete
	// Note: deletion is required for cluster recreation
	clientset.CoreV1().ConfigMaps(namespace).Delete(clusterName+"-leader", &metav1.DeleteOptions{})

	// delete the '<cluster-name>-pgha-default-config' configmap, if it exists so the config syncer
	// will not try to use it instead of '<cluster-name>-pgha-config'
	clientset.CoreV1().ConfigMaps(namespace).Delete(clusterName+"-pgha-default-config", &metav1.DeleteOptions{})
}

// deploymentWait is modified from cluster.waitForDeploymentDelete. It simply waits for the current primary deployment
// deletion to complete before proceeding with the rest of the pgcluster upgrade.
func deploymentWait(clientset kubernetes.Interface, namespace, deploymentName string, timeoutSecs, periodSecs time.Duration) string {
	timeout := time.After(timeoutSecs * time.Second)
	tick := time.NewTicker(periodSecs * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Sprintf("Timed out waiting for deployment to be deleted: [%s]", deploymentName)
		case <-tick.C:
			_, err := clientset.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
			if err != nil {
				return fmt.Sprintf("Deployment %s has been deleted.", deploymentName)
			}
		}
	}
}

// deleteNonupgradePgtasks deletes all existing pgtasks by selector with the exception of the
// upgrade task itself
func deleteNonupgradePgtasks(clientset pgo.Interface, selector, namespace string) error {
	taskList, err := clientset.CrunchydataV1().Pgtasks(namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}

	// get the pgtask list
	for _, v := range taskList.Items {
		// if the pgtask is not for the upgrade, delete it
		if v.ObjectMeta.Name != v.Name+"-"+config.LABEL_UPGRADE {
			err = clientset.CrunchydataV1().Pgtasks(namespace).Delete(v.Name, &metav1.DeleteOptions{})
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
func createUpgradePGHAConfigMap(clientset kubernetes.Interface, cluster *crv1.Pgcluster,
	namespace string) error {

	labels := make(map[string]string)
	labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	labels[config.LABEL_PG_CLUSTER] = cluster.Name
	labels[config.LABEL_PGHA_CONFIGMAP] = "true"

	data := make(map[string]string)

	// if the "pgha-default-config" config map exists, this cluster is being upgraded
	// and should use the initialization value from this existing configmap
	defaultConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(cluster.Name+"-pgha-default-config", metav1.GetOptions{})
	if err == nil {
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
		ObjectMeta: metav1.ObjectMeta{
			Name:   cluster.Name + "-" + operator.PGHAConfigMapSuffix,
			Labels: labels,
		},
		Data: data,
	}

	if _, err := clientset.CoreV1().ConfigMaps(namespace).Create(configmap); err != nil {
		return err
	}

	return nil
}

// recreateBackrestRepoSecret overwrites the secret for the pgBackRest repo. This is needed
// because the key encryption algorithm has been updated from RSA to EdDSA
func recreateBackrestRepoSecret(clientset kubernetes.Interface, clustername, namespace, operatorNamespace string) {
	config := util.BackrestRepoConfig{
		ClusterName:       clustername,
		ClusterNamespace:  namespace,
		OperatorNamespace: operatorNamespace,
	}

	secretName := clustername + "-backrest-repo-config"
	secret, err := clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})

	// 4.1, 4.2
	if err == nil {
		if b, ok := secret.Data["aws-s3-ca.crt"]; ok {
			config.BackrestS3CA = b
		}
		if b, ok := secret.Data["aws-s3-credentials.yaml"]; ok {
			var parsed struct {
				Key       string `yaml:"aws-s3-key"`
				KeySecret string `yaml:"aws-s3-key-secret"`
			}
			if err = yaml.Unmarshal(b, &parsed); err == nil {
				config.BackrestS3Key = parsed.Key
				config.BackrestS3KeySecret = parsed.KeySecret
			}
		}
	}

	// >= 4.3
	if err == nil {
		if b, ok := secret.Data["aws-s3-ca.crt"]; ok {
			config.BackrestS3CA = b

			// if this matches the old AWS S3 CA bundle, update to the new one.
			if fmt.Sprintf("%x", sha256.Sum256(config.BackrestS3CA)) == legacyS3CASHA256Digest {
				file := path.Join("/default-pgo-backrest-repo/aws-s3-ca.crt")

				// if we can't read the contents of the file for whatever reason, warn,
				// otherwise, update the entry in the Secret
				if contents, err := ioutil.ReadFile(file); err != nil {
					log.Warn(err)
				} else {
					config.BackrestS3CA = contents
				}
			}
		}
		if b, ok := secret.Data["aws-s3-key"]; ok {
			config.BackrestS3Key = string(b)
		}
		if b, ok := secret.Data["aws-s3-key-secret"]; ok {
			config.BackrestS3KeySecret = string(b)
		}
	}

	if err == nil {
		err = util.CreateBackrestRepoSecrets(clientset, config)
	}
	if err != nil {
		log.Errorf("error generating new backrest repo secrets during pgcluster upgrade: %v", err)
	}
}

// preparePgclusterForUpgrade specifically updates the existing CRD instance to set correct values
// for the current Postgres Operator version, updating or deleting values where appropriate, and sets
// an expected status so that the CRD object can be recreated.
func preparePgclusterForUpgrade(pgcluster *crv1.Pgcluster, parameters map[string]string, oldpgoversion, currentPrimary string) {

	// first, update the PGO version references to the current Postgres Operator version
	pgcluster.ObjectMeta.Labels[config.LABEL_PGO_VERSION] = parameters[config.LABEL_PGO_VERSION]
	pgcluster.Spec.UserLabels[config.LABEL_PGO_VERSION] = parameters[config.LABEL_PGO_VERSION]

	// next, capture the existing Crunchy Postgres Exporter configuration settings (previous to version
	// 4.5.0 referred to as Crunchy Collect), if they exist, and store them in the current labels
	if value, ok := pgcluster.ObjectMeta.Labels["crunchy_collect"]; ok {
		pgcluster.ObjectMeta.Labels[config.LABEL_EXPORTER] = value
		delete(pgcluster.ObjectMeta.Labels, "crunchy_collect")
	}

	if value, ok := pgcluster.Spec.UserLabels["crunchy_collect"]; ok {
		pgcluster.Spec.UserLabels[config.LABEL_EXPORTER] = value
		delete(pgcluster.Spec.UserLabels, "crunchy_collect")
	}

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

	// update the image tag to the value provided with the upgrade task. This will either be
	// the standard value set in the Postgres Operator's main configuration (which will have already
	// been verified to match the MAJOR PostgreSQL version) or the value provided by the user for
	// use with PostGIS enabled pgclusters
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

	// added in 4.2 and copied from configuration in 4.4
	if pgcluster.Spec.BackrestS3Bucket == "" {
		pgcluster.Spec.BackrestS3Bucket = operator.Pgo.Cluster.BackrestS3Bucket
	}
	if pgcluster.Spec.BackrestS3Endpoint == "" {
		pgcluster.Spec.BackrestS3Endpoint = operator.Pgo.Cluster.BackrestS3Endpoint
	}
	if pgcluster.Spec.BackrestS3Region == "" {
		pgcluster.Spec.BackrestS3Region = operator.Pgo.Cluster.BackrestS3Region
	}

	// added in 4.4
	if pgcluster.Spec.BackrestS3VerifyTLS == "" {
		pgcluster.Spec.BackrestS3VerifyTLS = operator.Pgo.Cluster.BackrestS3VerifyTLS
	}

	// add a label with the PGO version upgraded from and to
	pgcluster.Annotations[config.ANNOTATION_UPGRADE_INFO] = "From_" + oldpgoversion + "_to_" + parameters[config.LABEL_PGO_VERSION]
	// update the "is upgraded" label to indicate cluster has been upgraded
	pgcluster.Annotations[config.ANNOTATION_IS_UPGRADED] = "true"

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

// createClusterRecreateWorkflowTask creates a cluster creation task for the upgraded cluster's recreation
// to maintain the expected workflow and tasking
func createClusterRecreateWorkflowTask(clientset pgo.Interface, clusterName, ns, pgouser string) (string, error) {
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
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	_, err = clientset.CrunchydataV1().Pgtasks(ns).Create(newInstance)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err
}

// updateUpgradeWorkflow updates a Workflow with the current state of the pgcluster upgrade task
// modified from the cluster.UpdateCloneWorkflow function
func updateUpgradeWorkflow(clientset pgo.Interface, namespace, workflowID, status string) error {
	log.Debugf("pgcluster upgrade workflow: update workflow [%s]", workflowID)

	// we have to look up the name of the workflow bt the workflow ID, which
	// involves using a selector
	selector := fmt.Sprintf("%s=%s", crv1.PgtaskWorkflowID, workflowID)
	taskList, err := clientset.CrunchydataV1().Pgtasks(namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
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

	if _, err := clientset.CrunchydataV1().Pgtasks(namespace).Update(&task); err != nil {
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
		log.Errorf("error publishing event. Error: %s", err.Error())
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
		log.Errorf("error publishing event. Error: %s", err.Error())
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
		log.Errorf("error publishing event. Error: %s", err.Error())
	}
}
