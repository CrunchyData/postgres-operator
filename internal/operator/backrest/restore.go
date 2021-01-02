package backrest

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type BackrestRestoreJobTemplateFields struct {
	JobName                string
	ClusterName            string
	WorkflowID             string
	ToClusterPVCName       string
	SecurityContext        string
	PGOImagePrefix         string
	PGOImageTag            string
	CommandOpts            string
	PITRTarget             string
	PgbackrestStanza       string
	PgbackrestDBPath       string
	PgbackrestRepo1Path    string
	PgbackrestRepo1Host    string
	PgbackrestS3EnvVars    string
	NodeSelector           string
	Tablespaces            string
	TablespaceVolumes      string
	TablespaceVolumeMounts string
}

// Restore ...
func Restore(restclient *rest.RESTClient, namespace string, clientset kubernetes.Interface, task *crv1.Pgtask) {

	clusterName := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER]
	log.Debugf("restore workflow: started for cluster %s", clusterName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow error: could not find a pgcluster in Restore Workflow for %s", clusterName)
		return
	}

	// disable autofail if it is currently enabled
	if err = util.ToggleAutoFailover(clientset, false, cluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE],
		namespace); err != nil {
		log.Error(err)
		return
	}

	//create the "to-cluster" PVC to hold the new dataPVC]
	restoreToName := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_TO_PVC]
	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, &cluster, namespace, restoreToName, cluster.Spec.PrimaryStorage)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("restore workflow: created pvc %s for cluster %s", restoreToName, clusterName)
	//delete current primary and all replica deployments
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_PG_DATABASE + "=true"
	depList, err := clientset.
		AppsV1().Deployments(namespace).
		List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Errorf("restore workflow error: could not get depList using %s", selector)
		return
	}

	if len(depList.Items) == 0 {
		log.Debugf("restore workflow: no primary or replicas found using selector %s. Skipping deployment deletion.", selector)
	} else {
		for _, depToDelete := range depList.Items {
			deletePropagation := metav1.DeletePropagationForeground
			err = clientset.
				AppsV1().Deployments(namespace).
				Delete(depToDelete.Name, &metav1.DeleteOptions{
					PropagationPolicy: &deletePropagation,
				})
			if err != nil {
				log.Errorf("restore workflow error: could not delete primary or replica %s", depToDelete.Name)
				return
			}
			log.Debugf("restore workflow: deleted primary or replica %s", depToDelete.Name)
		}
	}

	message := "Cluster is being restored"
	err = kubeapi.PatchpgclusterStatus(restclient, crv1.PgclusterStateRestore, message, &cluster, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	pgreplicaList := &crv1.PgreplicaList{}
	selector = config.LABEL_PG_CLUSTER + "=" + clusterName
	log.Debugf("Restored cluster %s went to ready, patching replicas", clusterName)
	err = kubeapi.GetpgreplicasBySelector(restclient, pgreplicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	for _, pgreplica := range pgreplicaList.Items {
		pgreplica.Status.State = crv1.PgreplicaStatePendingRestore
		pgreplica.Spec.Status = "restore"
		delete(pgreplica.Annotations, config.ANNOTATION_PGHA_BOOTSTRAP_REPLICA)
		err = kubeapi.Updatepgreplica(restclient, &pgreplica, pgreplica.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}

	// set up a map of the names of the tablespaces as well as the storage classes
	tablespaceStorageTypeMap := operator.GetTablespaceStorageTypeMap(cluster.Spec.TablespaceMounts)

	// combine supplemental groups from all volumes
	var supplementalGroups []int64
	supplementalGroups = append(supplementalGroups, dataVolume.SupplementalGroups...)
	for _, v := range tablespaceVolumes {
		supplementalGroups = append(supplementalGroups, v.SupplementalGroups...)
	}

	//sleep for a bit to give the bounce time to take effect and let
	//the backrest repo container come back and be able to service requests
	time.Sleep(time.Second * time.Duration(30))

	//create the Job to run the backrest restore container

	workflowID := task.Spec.Parameters[crv1.PgtaskWorkflowID]
	jobFields := BackrestRestoreJobTemplateFields{
		JobName:                "restore-" + task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER] + "-" + util.RandStringBytesRmndr(4),
		ClusterName:            task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER],
		SecurityContext:        operator.GetPodSecurityContext(supplementalGroups),
		ToClusterPVCName:       restoreToName,
		WorkflowID:             workflowID,
		CommandOpts:            task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_OPTS],
		PITRTarget:             task.Spec.Parameters[config.LABEL_BACKREST_PITR_TARGET],
		PGOImagePrefix:         util.GetValueOrDefault(cluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:            operator.Pgo.Pgo.PGOImageTag,
		PgbackrestStanza:       task.Spec.Parameters[config.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:       task.Spec.Parameters[config.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepo1Path:    util.GetPGBackRestRepoPath(cluster),
		PgbackrestRepo1Host:    task.Spec.Parameters[config.LABEL_PGBACKREST_REPO_HOST],
		NodeSelector:           operator.GetAffinity(task.Spec.Parameters["NodeLabelKey"], task.Spec.Parameters["NodeLabelValue"], "In"),
		PgbackrestS3EnvVars:    operator.GetPgbackrestS3EnvVars(cluster, clientset, namespace),
		TablespaceVolumes:      operator.GetTablespaceVolumesJSON(restoreToName, tablespaceStorageTypeMap),
		TablespaceVolumeMounts: operator.GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap),
	}

	// A recovery target should also have a recovery target action. The PostgreSQL
	// and pgBackRest defaults are `pause` which requires the user to execute SQL
	// before the cluster will accept any writes. If no action has been specified,
	// use `promote` which accepts writes as soon as recovery succeeds.
	//
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html#RUNTIME-CONFIG-WAL-RECOVERY-TARGET
	// - https://pgbackrest.org/command.html#command-restore/category-command/option-target-action
	//
	if jobFields.PITRTarget != "" && !strings.Contains(jobFields.CommandOpts, "--target-action") {
		jobFields.CommandOpts = strings.TrimSpace(jobFields.CommandOpts + " --target-action=promote")
	}

	// If the pgBackRest repo type is set to 's3', pass in the relevant command line argument.
	// This is used in place of the environment variable so that it works as expected with
	// the --no-repo1-s3-verify-tls flag, added below
	pgBackrestRepoType := operator.GetRepoType(task.Spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE])
	if pgBackrestRepoType == "s3" &&
		!strings.Contains(jobFields.CommandOpts, "--repo1-type") &&
		!strings.Contains(jobFields.CommandOpts, "--repo-type") {
		jobFields.CommandOpts = strings.TrimSpace(jobFields.CommandOpts + " --repo1-type=s3")
	}

	// If TLS verification is disabled for this pgcluster, pass in the appropriate
	// flag to the restore command. Otherwise, leave the default behavior, which will
	// perform the normal certificate validation.
	verifyTLS, _ := strconv.ParseBool(operator.GetS3VerifyTLSSetting(&cluster))
	if pgBackrestRepoType == "s3" && !verifyTLS &&
		!strings.Contains(jobFields.CommandOpts, "--no-repo1-s3-verify-tls") {
		jobFields.CommandOpts = strings.TrimSpace(jobFields.CommandOpts + " --no-repo1-s3-verify-tls")
	}

	jobTemplate := bytes.Buffer{}

	if err := config.BackrestRestorejobTemplate.Execute(&jobTemplate, jobFields); err != nil {
		log.Error(err.Error())
		log.Error("restore workflow: error executing job template")
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.BackrestRestorejobTemplate.Execute(os.Stdout, jobFields)
	}

	job := v1batch.Job{}
	if err := json.Unmarshal(jobTemplate.Bytes(), &job); err != nil {
		log.Error("restore workflow: error unmarshalling json into Job " + err.Error())
		return
	}

	if cluster.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToBackRest(&job.Spec.Template.Spec, walVolume)
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_BACKREST_RESTORE,
		&job.Spec.Template.Spec.Containers[0])

	if j, err := clientset.BatchV1().Jobs(namespace).Create(&job); err != nil {
		log.Error(err)
		log.Error("restore workflow: error in creating restore job")
		return
	} else {
		log.Debugf("restore workflow: restore job %s created", j.Name)
	}

	publishRestore(cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clusterName, task.ObjectMeta.Labels[config.LABEL_PGOUSER], namespace)

	err = updateWorkflow(restclient, workflowID, namespace, crv1.PgtaskWorkflowBackrestRestoreJobCreatedStatus)
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in updating workflow status")
	}

}

func UpdateRestoreWorkflow(restclient *rest.RESTClient, clientset kubernetes.Interface, clusterName, status, namespace,
	workflowID, restoreToName string, affinity *v1.Affinity) {
	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType
	log.Debugf("restore workflow phase 2: taskName is %s", taskName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow phase 2 error: could not find a pgclustet in updateRestoreWorkflow for %s", clusterName)
		return
	}

	// set the "init" flag to true in the PGHA configMap for the PG cluster
	operator.UpdatePGHAConfigInitFlag(clientset, true, clusterName, namespace)

	//create the new primary deployment
	createRestoredDeployment(restclient, &cluster, clientset, namespace, restoreToName, workflowID, affinity)

	log.Debugf("restore workflow phase  2: created restored primary was %s now %s", cluster.Spec.Name, restoreToName)

	//update workflow
	if err := updateWorkflow(restclient, workflowID, namespace, crv1.PgtaskWorkflowBackrestRestorePrimaryCreatedStatus); err != nil {
		log.Warn(err)
	}
}

func updateWorkflow(restclient *rest.RESTClient, workflowID, namespace, status string) error {
	//update workflow
	log.Debugf("restore workflow: update workflow %s", workflowID)
	selector := crv1.PgtaskWorkflowID + "=" + workflowID
	taskList := crv1.PgtaskList{}
	err := kubeapi.GetpgtasksBySelector(restclient, &taskList, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get workflow %s", workflowID)
		return err
	}
	if len(taskList.Items) != 1 {
		log.Errorf("restore workflow error: workflow %s not found", workflowID)
		return errors.New("restore workflow error: workflow not found")
	}

	task := taskList.Items[0]
	task.Spec.Parameters[status] = time.Now().Format(time.RFC3339)
	err = kubeapi.Updatepgtask(restclient, &task, task.Name, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not update workflow %s to status %s", workflowID, status)
		return err
	}
	return err
}

func createRestoredDeployment(restclient *rest.RESTClient, cluster *crv1.Pgcluster, clientset kubernetes.Interface,
	namespace, restoreToName, workflowID string, affinity *v1.Affinity) error {

	// interpret the storage specs again. the volumes were already created during
	// the restore job.
	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, cluster, namespace, restoreToName, cluster.Spec.PrimaryStorage)

	//primaryLabels := operator.GetPrimaryLabels(cluster.Spec.Name, cluster.Spec.ClusterName, false, cluster.Spec.UserLabels)

	cluster.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = restoreToName
	cluster.Spec.UserLabels["name"] = cluster.Spec.Name
	cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER] = cluster.Spec.ClusterName

	// Set the Patroni scope to the name of the primary deployment.  Replicas will get scope using the
	// 'crunchy-pgha-scope' label
	cluster.Spec.UserLabels[config.LABEL_PGHA_SCOPE] = restoreToName

	archiveMode := "on"

	var affinityStr string
	if affinity != nil {
		log.Debugf("Affinity found on restore job, and will applied to the restored deployment")
		affinityBytes, err := json.MarshalIndent(affinity, "", "  ")
		if err != nil {
			log.Error("unable to marshall affinity obtained from restore job spec")
		}
		// Since the template for a cluster deployment contains the braces for the json
		// defining any affinity rules, we trim them here from the affinity json obtained
		// directly from the restore job (which also has the same braces)
		affinityStr = strings.Trim(string(affinityBytes), "{}")
	} else {
		affinityStr = operator.GetAffinity(cluster.Spec.UserLabels["NodeLabelKey"], cluster.Spec.UserLabels["NodeLabelValue"], "In")
	}

	// set up a map of the names of the tablespaces as well as the storage classes
	tablespaceStorageTypeMap := operator.GetTablespaceStorageTypeMap(cluster.Spec.TablespaceMounts)

	// combine supplemental groups from all volumes
	var supplementalGroups []int64
	supplementalGroups = append(supplementalGroups, dataVolume.SupplementalGroups...)
	for _, v := range tablespaceVolumes {
		supplementalGroups = append(supplementalGroups, v.SupplementalGroups...)
	}

	deploymentFields := operator.DeploymentTemplateFields{
		Name:              restoreToName,
		IsInit:            true,
		Replicas:          "1",
		ClusterName:       cluster.Spec.Name,
		Port:              cluster.Spec.Port,
		CCPImagePrefix:    util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImage:          cluster.Spec.CCPImage,
		CCPImageTag:       cluster.Spec.CCPImageTag,
		PVCName:           dataVolume.InlineVolumeSource(),
		DeploymentLabels:  operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		PodLabels:         operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		DataPathOverride:  restoreToName,
		Database:          cluster.Spec.Database,
		ArchiveMode:       archiveMode,
		SecurityContext:   operator.GetPodSecurityContext(supplementalGroups),
		RootSecretName:    cluster.Spec.RootSecretName,
		PrimarySecretName: cluster.Spec.PrimarySecretName,
		UserSecretName:    cluster.Spec.UserSecretName,
		NodeSelector:      affinityStr,
		PodAntiAffinity: operator.GetPodAntiAffinity(cluster,
			crv1.PodAntiAffinityDeploymentDefault, cluster.Spec.PodAntiAffinity.Default),
		ContainerResources: operator.GetResourcesJSON(cluster.Spec.Resources, cluster.Spec.Limits),
		ConfVolume:         operator.GetConfVolume(clientset, cluster, namespace),
		CollectAddon:       operator.GetCollectAddon(clientset, namespace, &cluster.Spec),
		CollectVolume:      operator.GetCollectVolume(clientset, cluster, namespace),
		BadgerAddon:        operator.GetBadgerAddon(clientset, namespace, cluster, restoreToName),
		ScopeLabel:         config.LABEL_PGHA_SCOPE,
		Standby:            false, // always disabled since standby clusters cannot be restored
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cluster, cluster.Labels[config.LABEL_BACKREST], restoreToName,
			cluster.Spec.Port, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:      operator.GetPgbackrestS3EnvVars(*cluster, clientset, namespace),
		EnableCrunchyadm:         operator.Pgo.Cluster.EnableCrunchyadm,
		ReplicaReinitOnStartFail: !operator.Pgo.Cluster.DisableReplicaStartFailReinit,
		SyncReplication:          operator.GetSyncReplication(cluster.Spec.SyncReplication),
		Tablespaces:              operator.GetTablespaceNames(cluster.Spec.TablespaceMounts),
		TablespaceVolumes:        operator.GetTablespaceVolumesJSON(restoreToName, tablespaceStorageTypeMap),
		TablespaceVolumeMounts:   operator.GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap),
		TLSEnabled:               cluster.Spec.TLS.IsTLSEnabled(),
		TLSOnly:                  cluster.Spec.TLSOnly,
		TLSSecret:                cluster.Spec.TLS.TLSSecret,
		ReplicationTLSSecret:     cluster.Spec.TLS.ReplicationTLSSecret,
		CASecret:                 cluster.Spec.TLS.CASecret,
	}

	log.Debug("collectaddon value is [" + deploymentFields.CollectAddon + "]")
	var primaryDoc bytes.Buffer
	err = config.DeploymentTemplate.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	//a form of debugging
	if operator.CRUNCHY_DEBUG {
		config.DeploymentTemplate.Execute(os.Stdout, deploymentFields)
	}

	deployment := appsv1.Deployment{}
	err = json.Unmarshal(primaryDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling primary json into Deployment " + err.Error())
		return err
	}

	if cluster.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToPostgreSQL(&deployment.Spec.Template.Spec, walVolume, restoreToName)
	}

	// determine if any of the container images need to be overridden
	operator.OverrideClusterContainerImages(deployment.Spec.Template.Spec.Containers)

	_, err = clientset.AppsV1().Deployments(namespace).Create(&deployment)
	if err != nil {
		return err
	}

	// store the workflowID in a user label
	cluster.Spec.UserLabels[crv1.PgtaskWorkflowID] = workflowID
	// patch the pgcluster CRD with the updated info
	if err = util.PatchClusterCRD(restclient, cluster.Spec.UserLabels, cluster, restoreToName, namespace); err != nil {
		log.Error("could not patch primary crv1 with labels")
		return err
	}
	return err

}

func publishRestore(id, clusterName, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventRestoreClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventRestoreCluster,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}
