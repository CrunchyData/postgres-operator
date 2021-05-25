package cluster

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	pgoconfig "github.com/crunchydata/postgres-operator/internal/operator/config"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
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

// nssWrapperForceCommand is the string that should be appended to the sshd_config file as
// needed for nss_wrapper support when upgrading from versions prior to v4.7
const nssWrapperForceCommand = `# ensure nss_wrapper env vars are set when executing commands as needed for OpenShift compatibility
ForceCommand NSS_WRAPPER_SUBDIR=ssh . /opt/crunchy/bin/nss_wrapper_env.sh && $SSH_ORIGINAL_COMMAND`

// the following regex expressions are used when upgrading the sshd_config file for a PG cluster
var (
	// nssWrapperRegex is the regular expression that is utilized to determine if the nss_wrapper
	// ForceCommand setting is missing from the sshd_config (as it would be for versions prior to
	// v4.7)
	nssWrapperRegex = regexp.MustCompile(nssWrapperForceCommand)

	// nssWrapperRegex is the regular expression that is utilized to determine if the UsePAM
	// setting is set to 'yes' in the sshd_config (as it might be for versions up to v4.6.1,
	// v4.5.2 and v4.4.3)
	usePAMRegex = regexp.MustCompile(`(?im)^UsePAM\s*yes`)
)

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
	ctx := context.TODO()
	upgradeTargetClusterName := upgrade.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]

	log.Debugf("started upgrade of cluster: %s", upgradeTargetClusterName)

	// publish our upgrade event
	PublishUpgradeEvent(events.EventUpgradeCluster, namespace, upgrade, "")

	pgcluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, upgradeTargetClusterName, metav1.GetOptions{})
	if err != nil {
		errormessage := "cound not find pgcluster for pgcluster upgrade"
		log.Errorf("%s Error: %s", errormessage, err)
		PublishUpgradeEvent(events.EventUpgradeClusterFailure, namespace, upgrade, errormessage)
		return
	}

	// update the workflow status to 'in progress' while the upgrade takes place
	_ = updateUpgradeWorkflow(clientset, namespace, upgrade.ObjectMeta.Labels[crv1.PgtaskWorkflowID], crv1.PgtaskUpgradeInProgress)

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
	_ = createUpgradePGHAConfigMap(clientset, pgcluster, namespace)

	// delete the existing pgcluster CRDs and other resources that will be recreated
	if err := deleteBeforeUpgrade(clientset, pgcluster, currentPrimary, namespace); err != nil {
		log.Error("refusing to upgrade due to unsuccessful resource removal")
		PublishUpgradeEvent(events.EventUpgradeClusterFailure, namespace, upgrade, err.Error())
		return
	}

	// recreate new Backrest Repo secret that was just deleted
	recreateBackrestRepoSecret(clientset, upgradeTargetClusterName, namespace, operator.PgoNamespace)

	// set proper values for the pgcluster that are updated between CR versions
	preparePgclusterForUpgrade(pgcluster, upgrade.Spec.Parameters, oldpgoversion, currentPrimary)

	// update the unix socket directories parameter so it no longer include /crunchyadm and
	// set any path references to the /opt/crunchy... paths
	if err = updateClusterConfig(clientset, pgcluster, namespace); err != nil {
		log.Errorf("error updating %s-pgha-config configmap during upgrade of cluster %s, Error: %v", pgcluster.Name, pgcluster.Name, err)
	}

	// create a new workflow for this recreated cluster
	workflowid, err := createClusterRecreateWorkflowTask(clientset, pgcluster.Name, namespace, upgrade.Spec.Parameters[config.LABEL_PGOUSER])
	if err != nil {
		// we will log any errors here, but will attempt to continue to submit the cluster for recreation regardless
		log.Errorf("error generating a new workflow task for the recreation of the upgraded cluster %s, Error: %s", pgcluster.Name, err)
	}

	// update pgcluster CRD workflow ID
	pgcluster.Spec.UserLabels[config.LABEL_WORKFLOW_ID] = workflowid

	_, err = clientset.CrunchydataV1().Pgclusters(namespace).Create(ctx, pgcluster, metav1.CreateOptions{})
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
	ctx := context.TODO()
	// first look for a 'primary' role label on the current primary deployment
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)

	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	// only consider pods that are running
	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(ctx, options)
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
	ctx := context.TODO()
	log.Debugf("deleting pgreplicas and noting the number found for cluster %s", clusterName)
	// Save the number of found replicas for this cluster
	numReps := 0
	replicaList, err := clientset.CrunchydataV1().Pgreplicas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("unable to get pgreplicas. Error: %s", err)
	}

	// go through the list of found replicas
	for index := range replicaList.Items {
		if replicaList.Items[index].Spec.ClusterName == clusterName {
			log.Debugf("scaling down pgreplica: %s", replicaList.Items[index].Name)
			ScaleDownBase(clientset, &replicaList.Items[index], namespace)
			log.Debugf("deleting pgreplica CRD: %s", replicaList.Items[index].Name)
			_ = clientset.CrunchydataV1().Pgreplicas(namespace).Delete(ctx, replicaList.Items[index].Name, metav1.DeleteOptions{})
			// if the existing replica PVC is not being used as the primary PVC, delete
			// note this will not remove any leftover PVCs from previous failovers,
			// those will require manual deletion so as to avoid any accidental
			// deletion of valid PVCs.
			if replicaList.Items[index].Name != currentPrimaryPVC {
				deletePropagation := metav1.DeletePropagationForeground
				_ = clientset.
					CoreV1().PersistentVolumeClaims(namespace).
					Delete(ctx, replicaList.Items[index].Name, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
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
func deleteBeforeUpgrade(clientset kubeapi.Interface, pgcluster *crv1.Pgcluster, currentPrimary, namespace string) error {
	ctx := context.TODO()

	// first, indicate that there is an upgrade occurring on this custom resource
	// this will prevent the rmdata job from firing off
	annotations := pgcluster.ObjectMeta.GetAnnotations()
	annotations[config.ANNOTATION_UPGRADE_IN_PROGRESS] = config.LABEL_TRUE
	pgcluster.ObjectMeta.SetAnnotations(annotations)

	if _, err := clientset.CrunchydataV1().Pgclusters(namespace).Update(ctx,
		pgcluster, metav1.UpdateOptions{}); err != nil {
		log.Errorf("unable to set annotations to keep backups and data: %s", err)
		return err
	}

	// next, get all deployments for the pgcluster in question
	deployments, err := clientset.
		AppsV1().Deployments(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER + "=" + pgcluster.Name})
	if err != nil {
		log.Errorf("unable to get deployments. Error: %s", err)
		return err
	}

	// next, delete those deployments
	for index := range deployments.Items {
		deletePropagation := metav1.DeletePropagationForeground
		_ = clientset.
			AppsV1().Deployments(namespace).
			Delete(ctx, deployments.Items[index].Name, metav1.DeleteOptions{
				PropagationPolicy: &deletePropagation,
			})
	}

	// wait until the backrest shared repo pod deployment has been deleted before continuing
	waitStatus := deploymentWait(clientset, namespace, pgcluster.Name+"-backrest-shared-repo",
		180*time.Second, 10*time.Second)
	log.Debug(waitStatus)
	// wait until the primary pod deployment has been deleted before continuing
	waitStatus = deploymentWait(clientset, namespace, currentPrimary,
		180*time.Second, 10*time.Second)
	log.Debug(waitStatus)

	// delete the pgcluster
	_ = clientset.CrunchydataV1().Pgclusters(namespace).Delete(ctx, pgcluster.Name, metav1.DeleteOptions{})

	// delete all existing job references
	deletePropagation := metav1.DeletePropagationForeground
	_ = clientset.
		BatchV1().Jobs(namespace).
		DeleteCollection(ctx,
			metav1.DeleteOptions{PropagationPolicy: &deletePropagation},
			metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER + "=" + pgcluster.Name})

	// delete all existing pgtask references except for the upgrade task
	// Note: this will be deleted by the existing pgcluster creation process once the
	// updated pgcluster created and processed by the cluster controller
	if err = deleteNonupgradePgtasks(clientset, config.LABEL_PG_CLUSTER+"="+pgcluster.Name, namespace); err != nil {
		log.Errorf("error while deleting pgtasks for cluster %s, Error: %v", pgcluster.Name, err)
	}

	// delete the leader configmap used by the Postgres Operator since this information may change after
	// the upgrade is complete
	// Note: deletion is required for cluster recreation
	_ = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, pgcluster.Name+"-leader", metav1.DeleteOptions{})

	// delete the '<cluster-name>-pgha-default-config' configmap, if it exists so the config syncer
	// will not try to use it instead of '<cluster-name>-pgha-config'
	_ = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, pgcluster.Name+"-pgha-default-config", metav1.DeleteOptions{})

	return nil
}

// deploymentWait is modified from cluster.waitForDeploymentDelete. It simply waits for the current primary deployment
// deletion to complete before proceeding with the rest of the pgcluster upgrade.
func deploymentWait(clientset kubernetes.Interface, namespace, deploymentName string, timeoutSecs, periodSecs time.Duration) string {
	ctx := context.TODO()

	if err := wait.Poll(periodSecs, timeoutSecs, func() (bool, error) {
		_, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
		return err != nil, nil
	}); err != nil {
		return fmt.Sprintf("Timed out waiting for deployment to be deleted: [%s]", deploymentName)
	}

	return fmt.Sprintf("Deployment %s has been deleted.", deploymentName)
}

// deleteNonupgradePgtasks deletes all existing pgtasks by selector with the exception of the
// upgrade task itself
func deleteNonupgradePgtasks(clientset pgo.Interface, selector, namespace string) error {
	ctx := context.TODO()
	taskList, err := clientset.CrunchydataV1().Pgtasks(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}

	// get the pgtask list
	for _, v := range taskList.Items {
		// if the pgtask is not for the upgrade, delete it
		if v.ObjectMeta.Name != v.Name+"-"+config.LABEL_UPGRADE {
			err = clientset.CrunchydataV1().Pgtasks(namespace).Delete(ctx, v.Name, metav1.DeleteOptions{})
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
	ctx := context.TODO()

	labels := make(map[string]string)
	labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	labels[config.LABEL_PG_CLUSTER] = cluster.Name
	labels[config.LABEL_PGHA_CONFIGMAP] = "true"

	data := make(map[string]string)

	// if the "pgha-default-config" config map exists, this cluster is being upgraded
	// and should use the initialization value from this existing configmap
	defaultConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, cluster.Name+"-pgha-default-config", metav1.GetOptions{})
	if err == nil {
		data[operator.PGHAConfigInitSetting] = defaultConfigmap.Data[operator.PGHAConfigInitSetting]
	} else {
		// set "init" to true in the postgres-ha configMap
		data[operator.PGHAConfigInitSetting] = "true"
	}

	// if a standby cluster then we want to create replicas using the S3 or GCS
	// pgBackRest repository (and not the local in-cluster pgBackRest repository)
	if cluster.Spec.Standby {
		repoType := crv1.BackrestStorageTypeS3

		for _, r := range cluster.Spec.BackrestStorageTypes {
			if r == crv1.BackrestStorageTypeGCS {
				repoType = crv1.BackrestStorageTypeGCS
			}
		}
		data[operator.PGHAConfigReplicaBootstrapRepoType] = string(repoType)
	}

	configmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cluster.Name + "-" + operator.PGHAConfigMapSuffix,
			Labels: labels,
		},
		Data: data,
	}

	if _, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configmap, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// recreateBackrestRepoSecret overwrites the secret for the pgBackRest repo. This is needed
// because the key encryption algorithm has been updated from RSA to EdDSA
func recreateBackrestRepoSecret(clientset kubernetes.Interface, clustername, namespace, operatorNamespace string) {
	ctx := context.TODO()
	config := util.BackrestRepoConfig{
		ClusterName:       clustername,
		ClusterNamespace:  namespace,
		OperatorNamespace: operatorNamespace,
	}

	secretName := clustername + "-backrest-repo-config"
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})

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
		}
		if b, ok := secret.Data["aws-s3-key"]; ok {
			config.BackrestS3Key = string(b)
		}
		if b, ok := secret.Data["aws-s3-key-secret"]; ok {
			config.BackrestS3KeySecret = string(b)
		}
	}

	var repoSecret *v1.Secret
	if err == nil {
		repoSecret, err = util.CreateBackrestRepoSecrets(clientset, config)
	}
	if err != nil {
		log.Errorf("error generating new backrest repo secrets during pgcluster upgrade: %v", err)
	}

	if err := updatePGBackRestSSHDConfig(clientset, repoSecret, namespace); err != nil {
		log.Errorf("error upgrading pgBackRest sshd_config: %v", err)
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
	// 4.6.0 added this value to the spec as "Exporter", so the next step ensure
	// that the value is migrated over
	if value, ok := pgcluster.ObjectMeta.Labels["crunchy_collect"]; ok {
		pgcluster.ObjectMeta.Labels[config.LABEL_EXPORTER] = value
	}
	delete(pgcluster.ObjectMeta.Labels, "crunchy_collect")

	// Note that this is the *user labels*, the above is in the metadata labels
	if value, ok := pgcluster.Spec.UserLabels["crunchy_collect"]; ok {
		pgcluster.Spec.UserLabels[config.LABEL_EXPORTER] = value
	}
	delete(pgcluster.Spec.UserLabels, "crunchy_collect")

	// convert the metrics label over to using a proper definition. Give the user
	// label precedence.
	if value, ok := pgcluster.ObjectMeta.Labels[config.LABEL_EXPORTER]; ok {
		pgcluster.Spec.Exporter, _ = strconv.ParseBool(value)
	}
	delete(pgcluster.ObjectMeta.Labels, config.LABEL_EXPORTER)

	// again, note this is *user* labels, the above are the metadata labels
	if value, ok := pgcluster.Spec.UserLabels[config.LABEL_EXPORTER]; ok {
		pgcluster.Spec.Exporter, _ = strconv.ParseBool(value)
	}
	delete(pgcluster.Spec.UserLabels, config.LABEL_EXPORTER)

	// 4.6.0 moved pgBadger to use an attribute instead of a label. If this label
	// exists on the current CRD, move the value to the attribute.
	if ok, _ := strconv.ParseBool(pgcluster.ObjectMeta.GetLabels()["crunchy-pgbadger"]); ok {
		pgcluster.Spec.PGBadger = true
	}
	delete(pgcluster.ObjectMeta.Labels, "crunchy-pgbadger")

	// 4.6.0 moved the format "service-type" label into the ServiceType CRD
	// attribute, so we may need to do the same
	if val, ok := pgcluster.Spec.UserLabels["service-type"]; ok {
		pgcluster.Spec.ServiceType = v1.ServiceType(val)
	}
	delete(pgcluster.Spec.UserLabels, "service-type")

	// 4.6.0 removed the "pg-pod-anti-affinity" label from user labels, as this is
	// superfluous and handled through other processes. We can explicitly
	// eliminate it
	delete(pgcluster.Spec.UserLabels, "pg-pod-anti-affinity")

	// 4.6.0 moved the "autofail" label to the DisableAutofail attribute. Given
	// by default we need to start in an autofailover state, we just delete the
	// legacy attribute
	delete(pgcluster.ObjectMeta.Labels, "autofail")

	// 4.6.0 moved the node labels to the custom resource objects in a more
	// structure way. if we have a node label, then let's migrate it to that
	// format
	if pgcluster.Spec.UserLabels["NodeLabelKey"] != "" && pgcluster.Spec.UserLabels["NodeLabelValue"] != "" {
		// transition to using the native NodeAffinity objects. In the previous
		// setup, this was, by default, preferred node affinity. Designed to match
		// a standard setup.
		requirement := v1.NodeSelectorRequirement{
			Key:      pgcluster.Spec.UserLabels["NodeLabelKey"],
			Values:   []string{pgcluster.Spec.UserLabels["NodeLabelValue"]},
			Operator: v1.NodeSelectorOpIn,
		}
		term := v1.PreferredSchedulingTerm{
			Weight: crv1.NodeAffinityDefaultWeight, // taking this from the former template
			Preference: v1.NodeSelectorTerm{
				MatchExpressions: []v1.NodeSelectorRequirement{requirement},
			},
		}

		// and here is our default node affinity rule
		pgcluster.Spec.NodeAffinity = crv1.NodeAffinitySpec{
			Default: &v1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{term},
			},
		}
	}
	// erase all trace of this
	delete(pgcluster.Spec.UserLabels, "NodeLabelKey")
	delete(pgcluster.Spec.UserLabels, "NodeLabelValue")

	// 4.6.0 moved the "backrest-storage-type" label to a CRD attribute, well,
	// really an array of CRD attributes, which we need to map the various
	// attributes to. "local" will be mapped the "posix" to match the pgBackRest
	// nomenclature
	//
	// If we come back with an empty array, we will default it to posix
	if val, ok := pgcluster.Spec.UserLabels["backrest-storage-type"]; ok {
		pgcluster.Spec.BackrestStorageTypes = make([]crv1.BackrestStorageType, 0)
		storageTypes := strings.Split(val, ",")

		// loop through each of the storage types processed and determine which of
		// the standard storage types it matches
		for _, s := range storageTypes {
			for _, storageType := range crv1.BackrestStorageTypes {
				// if this is not the storage type, continue looping
				if crv1.BackrestStorageType(s) != storageType {
					continue
				}

				// so this is the storage type. However, if it's "local" let's update
				// it to be posix
				if storageType == crv1.BackrestStorageTypeLocal {
					pgcluster.Spec.BackrestStorageTypes = append(pgcluster.Spec.BackrestStorageTypes,
						crv1.BackrestStorageTypePosix)
				} else {
					pgcluster.Spec.BackrestStorageTypes = append(pgcluster.Spec.BackrestStorageTypes, storageType)
				}

				// we can break the inner loop
				break
			}
		}

		// remember: if somehow this is empty, add "posix"
		if len(pgcluster.Spec.BackrestStorageTypes) == 0 {
			pgcluster.Spec.BackrestStorageTypes = append(pgcluster.Spec.BackrestStorageTypes,
				crv1.BackrestStorageTypePosix)
		}
	}
	// and delete the label
	delete(pgcluster.Spec.UserLabels, "backrest-storage-type")

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

	// set a default disable autofail value of "false" to enable Patroni's replication.
	// If left to an existing value of "true," Patroni will be in a paused state
	// and unable to sync all replicas to the current timeline
	pgcluster.Spec.DisableAutofail = false

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
	ctx := context.TODO()

	// create pgtask CRD
	spec := crv1.PgtaskSpec{}
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

	_, err = clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, newInstance, metav1.CreateOptions{})
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err
}

// updateUpgradeWorkflow updates a Workflow with the current state of the pgcluster upgrade task
// modified from the cluster.UpdateCloneWorkflow function
func updateUpgradeWorkflow(clientset pgo.Interface, namespace, workflowID, status string) error {
	ctx := context.TODO()
	log.Debugf("pgcluster upgrade workflow: update workflow [%s]", workflowID)

	// we have to look up the name of the workflow bt the workflow ID, which
	// involves using a selector
	selector := fmt.Sprintf("%s=%s", crv1.PgtaskWorkflowID, workflowID)
	taskList, err := clientset.CrunchydataV1().Pgtasks(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
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

	if _, err := clientset.CrunchydataV1().Pgtasks(namespace).Update(ctx, &task, metav1.UpdateOptions{}); err != nil {
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

// updateClusterConfig updates PG configuration for cluster via its Distributed Configuration Store
// (DCS) according to the key/value pairs defined in the pgConfig map, specifically by updating
// the <clusterName>-pgha-config ConfigMap.  The configuration settings specified are
// applied to the entire cluster via the DCS configuration included within this the
// <clusterName>-pgha-config ConfigMap.
func updateClusterConfig(clientset kubeapi.Interface, pgcluster *crv1.Pgcluster, namespace string) error {

	// first, define the names for the two main sections of the <clustername>-pgha-config configmap

	// <clustername>-dcs-config
	dcsConfigName := fmt.Sprintf(pgoconfig.PGHADCSConfigName, pgcluster.Name)
	// <clustername>-local-config
	localConfigName := fmt.Sprintf(pgoconfig.PGHALocalConfigName, pgcluster.Name)

	// next, get the <clustername>-pgha-config configmap
	clusterConfig, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), fmt.Sprintf("%s-pgha-config", pgcluster.Name), metav1.GetOptions{})
	if err != nil {
		return err
	}

	// prepare DCS config struct
	dcsConf := &pgoconfig.DCSConfig{}
	if err := yaml.Unmarshal([]byte(clusterConfig.Data[dcsConfigName]), dcsConf); err != nil {
		return err
	}

	// prepare LocalDB config struct
	localDBConf := &pgoconfig.LocalDBConfig{}
	if err := yaml.Unmarshal([]byte(clusterConfig.Data[localConfigName]), localDBConf); err != nil {
		return err
	}

	// set the updated path values for both DCS and LocalDB configs, if the fields and maps exist
	// as of version 4.6, the /crunchyadm directory no longer exists (previously set as a unix socket directory)
	// and the /opt/cpm... directories are now set under /opt/crunchy
	if dcsConf.PostgreSQL != nil && dcsConf.PostgreSQL.Parameters != nil {
		dcsConf.PostgreSQL.Parameters["unix_socket_directories"] = "/tmp"

		// ensure the proper archive_command is set according to the BackrestStorageTypes defined for
		// the pgcluster
		switch {
		case operator.IsLocalAndS3Storage(pgcluster):
			dcsConf.PostgreSQL.Parameters["archive_command"] = `source /opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-archive-push-local-s3.sh %p`
		case operator.IsLocalAndGCSStorage(pgcluster):
			dcsConf.PostgreSQL.Parameters["archive_command"] = `source /opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-archive-push-local-gcs.sh %p`
		default:
			dcsConf.PostgreSQL.Parameters["archive_command"] = `source /opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-set-env.sh && pgbackrest archive-push "%p"`
		}

		dcsConf.PostgreSQL.RecoveryConf["restore_command"] = `source /opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-set-env.sh && pgbackrest archive-get %f "%p"`
	}

	if localDBConf.PostgreSQL.Callbacks != nil {
		localDBConf.PostgreSQL.Callbacks.OnRoleChange = "/opt/crunchy/bin/postgres-ha/callbacks/pgha-on-role-change.sh"
	}
	if localDBConf.PostgreSQL.PGBackRest != nil {
		localDBConf.PostgreSQL.PGBackRest.Command = "/opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-create-replica.sh replica"
	}
	if localDBConf.PostgreSQL.PGBackRestStandby != nil {
		localDBConf.PostgreSQL.PGBackRestStandby.Command = "/opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-create-replica.sh standby"
	}

	// set up content and patch DCS config
	dcsContent, err := yaml.Marshal(dcsConf)
	if err != nil {
		return err
	}

	// patch the configmap with the DCS config updates
	if err := pgoconfig.PatchConfigMapData(clientset, clusterConfig, dcsConfigName, dcsContent); err != nil {
		return err
	}

	// set up content and patch localDB config
	localDBContent, err := yaml.Marshal(localDBConf)
	if err != nil {
		return err
	}

	// patch the configmap with the localDB config updates
	if err := pgoconfig.PatchConfigMapData(clientset, clusterConfig, localConfigName, localDBContent); err != nil {
		return err
	}

	// get the newly patched <clustername>-pgha-config configmap
	patchedClusterConfig, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), fmt.Sprintf("%s-pgha-config", pgcluster.Name), metav1.GetOptions{})
	if err != nil {
		return err
	}

	// sync the changes to the configmap to the DCS
	return pgoconfig.NewDCS(patchedClusterConfig, clientset, pgcluster.Name).Sync()
}

// updatePGBackRestSSHDConfig is responsible for upgrading the sshd_config file as needed across
// operator versions to ensure proper functionality with pgBackRest
func updatePGBackRestSSHDConfig(clientset kubernetes.Interface, repoSecret *v1.Secret,
	namespace string) error {

	ctx := context.TODO()
	var err error
	var secretRequiresUpdate bool
	updatedRepoSecret := repoSecret.DeepCopy()

	// For versions prior to v4.7, the 'ForceCommand' will be missing from the sshd_config as
	// as needed for nss_wrapper support.  Therefore, check to see if the proper ForceCommand
	// setting exists in the sshd_config, and if not, add it.
	if !nssWrapperRegex.MatchString(string(updatedRepoSecret.Data["sshd_config"])) {
		secretRequiresUpdate = true
		updatedRepoSecret.Data["sshd_config"] =
			[]byte(fmt.Sprintf("%s\n%s\n", string(updatedRepoSecret.Data["sshd_config"]),
				nssWrapperForceCommand))
	}

	// For versions prior to v4.6.2, the UsePAM setting might be set to 'yes' as previously
	// required to workaround a known Docker issue.  Since this issue has since been resolved,
	// we now want to ensure this setting is set to 'no'.
	if usePAMRegex.MatchString(string(updatedRepoSecret.Data["sshd_config"])) {
		secretRequiresUpdate = true
		updatedRepoSecret.Data["sshd_config"] =
			[]byte(usePAMRegex.ReplaceAllString(string(updatedRepoSecret.Data["sshd_config"]),
				"UsePAM no"))
	}

	if secretRequiresUpdate {
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, updatedRepoSecret,
			metav1.UpdateOptions{})
	}

	return err
}
