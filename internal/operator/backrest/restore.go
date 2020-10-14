package backrest

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"

	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// restoreTargetRegex defines a regex to detect if a restore target has been specified
// for pgBackRest using the '--target' option
var restoreTargetRegex = regexp.MustCompile("--target(=| +)")

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

// UpdatePGClusterSpecForRestore updates the spec for pgcluster resource provided as need to
// perform a restore
func UpdatePGClusterSpecForRestore(clientset kubeapi.Interface, cluster *crv1.Pgcluster,
	task *crv1.Pgtask) {

	cluster.Spec.PGDataSource.RestoreFrom = cluster.GetName()

	restoreOpts := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_OPTS]

	// set the proper target for the restore job
	pitrTarget := task.Spec.Parameters[config.LABEL_BACKREST_PITR_TARGET]
	if pitrTarget != "" && !restoreTargetRegex.MatchString(restoreOpts) {
		restoreOpts = fmt.Sprintf("%s --target=%s", restoreOpts, strconv.Quote(pitrTarget))
	}

	// set the proper backrest storage type for the restore job
	storageType := task.Spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE]
	if storageType != "" && !strings.Contains(restoreOpts, "--repo-type") {
		restoreOpts = fmt.Sprintf("%s --repo-type=%s", restoreOpts, storageType)
	}

	cluster.Spec.PGDataSource.RestoreOpts = restoreOpts

	// set the proper node affinity for the restore job
	cluster.Spec.UserLabels[config.LABEL_NODE_LABEL_KEY] =
		task.Spec.Parameters[config.LABEL_NODE_LABEL_KEY]
	cluster.Spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] =
		task.Spec.Parameters[config.LABEL_NODE_LABEL_VALUE]

	return
}

// PrepareClusterForRestore prepares a PostgreSQL cluster for a restore.  This includes deleting
// variousresources (Deployments, Jobs, PVCs & pgtasks) while also patching various custome
// resources (pgreplicas) as needed to perform a restore.
func PrepareClusterForRestore(clientset kubeapi.Interface, cluster *crv1.Pgcluster,
	task *crv1.Pgtask) (*crv1.Pgcluster, error) {

	var err error
	var patchedCluster *crv1.Pgcluster
	namespace := cluster.Namespace
	clusterName := cluster.Name
	log.Debugf("restore workflow: started for cluster %s", clusterName)

	// prepare the pgcluster CR for restore
	patch, err := kubeapi.NewMergePatch().
		Add("metadata", "annotations")(map[string]string{
		config.ANNOTATION_BACKREST_RESTORE: "",
		config.ANNOTATION_CURRENT_PRIMARY:  clusterName,
	}).
		Add("metadata", "labels")(map[string]string{
		config.LABEL_DEPLOYMENT_NAME: clusterName,
	}).
		Add("spec", "status")("").
		Add("status")(crv1.PgclusterStatus{
		Message: "Cluster is being restored",
		State:   crv1.PgclusterStateRestore,
	}).
		Bytes()
	if err == nil {
		log.Debugf("patching cluster %s: %s", clusterName, patch)
		patchedCluster, err = clientset.CrunchydataV1().
			Pgclusters(namespace).Patch(clusterName, types.MergePatchType, patch)
	}
	if err != nil {
		log.Errorf("pgtask Controller: " + err.Error())
		return nil, err
	}
	log.Debugf("restore workflow: patched pgcluster %s for restore", clusterName)

	// find all pgreplica CR's
	replicas, err := clientset.CrunchydataV1().Pgreplicas(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, clusterName),
	})
	if err != nil {
		return nil, err
	}

	// prepare pgreplica CR's for restore
	patch, err = kubeapi.NewMergePatch().
		Remove("metadata", "annotations", config.ANNOTATION_PGHA_BOOTSTRAP_REPLICA).
		Add("spec", "status")("").
		Add("status")(crv1.PgclusterStatus{
		Message: "Cluster is being restored",
		State:   crv1.PgclusterStateRestore,
	}).
		Bytes()
	if err != nil {
		return nil, err
	}
	for _, r := range replicas.Items {
		log.Debugf("patching replica %s: %s", r.GetName(), patch)
		_, err := clientset.CrunchydataV1().
			Pgreplicas(namespace).Patch(r.GetName(), types.MergePatchType, patch)
		if err != nil {
			return nil, err
		}
	}
	log.Debugf("restore workflow: patched replicas in cluster %s for restore", clusterName)

	// find all current pg deployments
	pgInstances, err := clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s", config.LABEL_PG_CLUSTER, clusterName,
			config.LABEL_PG_DATABASE),
	})
	if err != nil {
		return nil, err
	}

	// delete all the primary and replica deployments
	if err := clientset.AppsV1().Deployments(namespace).DeleteCollection(&metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s", config.LABEL_PG_CLUSTER, clusterName,
				config.LABEL_PG_DATABASE),
		}); err != nil {
		return nil, err
	}
	log.Debugf("restore workflow: deleted primary and replicas %v", pgInstances)

	// delete all existing jobs
	deletePropagation := metav1.DeletePropagationBackground
	if err := clientset.BatchV1().Jobs(namespace).DeleteCollection(
		&metav1.DeleteOptions{PropagationPolicy: &deletePropagation},
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, clusterName),
		}); err != nil {
		return nil, err
	}
	log.Debugf("restore workflow: deleted all existing jobs for cluster %s", clusterName)

	// delete all PostgreSQL PVCs (the primary and all replica PVCs)
	for _, deployment := range pgInstances.Items {
		err := clientset.
			CoreV1().PersistentVolumeClaims(namespace).
			Delete(deployment.GetName(), &metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return nil, err
		}
		log.Debugf("restore workflow: deleted primary or replica PVC %s", deployment.GetName())
	}

	// Wait for all PG PVCs to be removed.  If unable to verify that all PVCs have been
	// removed, then the restore cannot proceed the function returns.
	if err := wait.Poll(time.Second/2, time.Minute*3, func() (bool, error) {
		notFound := true
		for _, deployment := range pgInstances.Items {
			if _, err := clientset.CoreV1().PersistentVolumeClaims(namespace).
				Get(deployment.GetName(), metav1.GetOptions{}); err == nil {
				notFound = false
			}
		}
		return notFound, nil
	}); err != nil {
		return nil, err
	}
	log.Debugf("restore workflow: finished waiting for PVCs for cluster %s to be removed",
		clusterName)

	// Delete the DCS and leader ConfigMaps.  These will be recreated during the restore.
	configMaps := []string{fmt.Sprintf("%s-config", clusterName),
		fmt.Sprintf("%s-leader", clusterName)}
	for _, c := range configMaps {
		if err := clientset.CoreV1().ConfigMaps(namespace).Delete(c,
			&metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
			return nil, err
		}
	}
	log.Debugf("restore workflow: deleted 'config' and 'leader' ConfigMaps for cluster %s",
		clusterName)

	patch, err = kubeapi.NewMergePatch().Add("data", "init")("true").Bytes()
	if err == nil {
		name := fmt.Sprintf("%s-pgha-config", clusterName)
		log.Debugf("patching configmap %s: %s", name, patch)
		_, err = clientset.CoreV1().
			ConfigMaps(namespace).Patch(name, types.MergePatchType, patch)
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("restore workflow: set 'init' flag to 'true' for cluster %s",
		clusterName)

	return patchedCluster, nil
}

// UpdateWorkflow is responsible for updating the workflow for a restore
func UpdateWorkflow(clientset pgo.Interface, workflowID, namespace, status string) error {
	//update workflow
	log.Debugf("restore workflow: update workflow %s", workflowID)
	selector := crv1.PgtaskWorkflowID + "=" + workflowID
	taskList, err := clientset.CrunchydataV1().Pgtasks(namespace).List(metav1.ListOptions{LabelSelector: selector})
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
	_, err = clientset.CrunchydataV1().Pgtasks(namespace).Update(&task)
	if err != nil {
		log.Errorf("restore workflow error: could not update workflow %s to status %s", workflowID, status)
		return err
	}
	return err
}

// PublishRestore is responsible for publishing the 'RestoreCluster' event for a restore
func PublishRestore(id, clusterName, username, namespace string) {
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
