package main

/*
Copyright 2019 - 2020 Crunchy Data
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
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/util"

	log "github.com/sirupsen/logrus"
	kerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maximumTries      = 16
	pgBackRestRepoPVC = "%s-pgbr-repo"
	pgDumpPVCPrefix   = "backup-%s-pgdump"
	// the tablespace on a replcia follows the pattern "<replicaName-tablespace-.."
	tablespaceReplicaPVCPattern = "%s-tablespace-"
	// the WAL PVC on a replcia follows the pattern "<replicaName-wal>"
	walReplicaPVCPattern = "%s-wal"

	// the following constants define the suffixes for the various configMaps created by Patroni
	configConfigMapSuffix   = "config"
	leaderConfigMapSuffix   = "leader"
	failoverConfigMapSuffix = "failover"
)

func Delete(request Request) {
	ctx := context.TODO()
	log.Infof("rmdata.Process %v", request)

	// if, check to see if this is a full cluster removal...i.e. "IsReplica"
	// and "IsBackup" is set to false
	//
	// if this is a full cluster removal, first disable autofailover
	if !(request.IsReplica || request.IsBackup) {
		log.Debug("disabling autofailover for cluster removal")
		util.ToggleAutoFailover(request.Clientset, false, request.ClusterPGHAScope, request.Namespace)
	}

	//the case of 'pgo scaledown'
	if request.IsReplica {
		log.Info("rmdata.Process scaledown replica use case")
		removeReplicaServices(request)
		pvcList, err := getReplicaPVC(request)
		if err != nil {
			log.Error(err)
		}
		//delete the pgreplica CRD
		if err := request.Clientset.
			CrunchydataV1().Pgreplicas(request.Namespace).
			Delete(ctx, request.ReplicaName, metav1.DeleteOptions{}); err != nil {
			// If the name of the replica being deleted matches the scope for the cluster, then
			// we assume it was the original primary and the pgreplica deletion will fail with
			// a not found error.  In this case we allow the rmdata process to continue despite
			// the error.  This allows for the original primary to be scaled down once it is
			// is no longer a primary, and has become a replica.
			if !(request.ReplicaName == request.ClusterPGHAScope && kerror.IsNotFound(err)) {
				log.Error(err)
				return
			}
			log.Debug("replica name matches PGHA scope, assuming scale down of original primary" +
				"and therefore ignoring error attempting to delete nonexistent pgreplica")
		}

		err = removeReplica(request)
		if err != nil {
			log.Error(err)
		}

		if request.RemoveData {
			removePVCs(pvcList, request)
		}

		//scale down is its own use case so we leave when done
		return
	}

	if request.IsBackup {
		log.Info("rmdata.Process backup use case")
		//the case of removing a backup using `pgo delete backup`, only applies to
		// "backup-type=pgdump"
		removeBackupJobs(request)
		removeLogicalBackupPVCs(request)
		// this is the special case of removing an ad hoc backup removal, so we can
		// exit here
		return
	}

	log.Info("rmdata.Process cluster use case")

	//the user had done something like:
	//pgo delete cluster mycluster --delete-data
	if request.RemoveData {
		removeUserSecrets(request)
	}

	//handle the case of 'pgo delete cluster mycluster'
	removeCluster(request)
	if err := request.Clientset.
		CrunchydataV1().Pgclusters(request.Namespace).
		Delete(ctx, request.ClusterName, metav1.DeleteOptions{}); err != nil {
		log.Error(err)
	}
	removeServices(request)
	removeAddons(request)
	removePgreplicas(request)
	removePgtasks(request)
	removeClusterConfigmaps(request)
	//removeClusterJobs(request)
	if request.RemoveData {
		if pvcList, err := getInstancePVCs(request); err != nil {
			log.Error(err)
		} else {
			log.Debugf("rmdata pvc list: [%v]", pvcList)

			removePVCs(pvcList, request)
		}
	}

	// backups have to be the last thing we remove. We want to ensure that all
	// the clusters (well, really, the primary) have stopped. This means that no
	// more WAL archives are being pushed, and at this point it is safe for us to
	// remove the pgBackRest repo if we have opted to remove all of the backups.
	//
	// Regardless of the choice the user made, we want to remove all of the
	// backup jobs, as those take up space
	removeBackupJobs(request)
	// Now, even though it appears we are removing the pgBackRest repo here, we
	// are **not** removing the physical data unless request.RemoveBackup is true.
	// In that case, only the deployment/services for the pgBackRest repo are
	// removed
	removeBackrestRepo(request)
	// now, check to see if the user wants the remainder of the physical data and
	// PVCs to be removed
	if request.RemoveBackup {
		removeBackupSecrets(request)
		removeAllBackupPVCs(request)
	}
}

// removeBackRestRepo removes the pgBackRest repo that is associated with the
// PostgreSQL cluster
func removeBackrestRepo(request Request) {
	ctx := context.TODO()
	deploymentName := fmt.Sprintf("%s-backrest-shared-repo", request.ClusterName)

	log.Debugf("deleting the pgbackrest repo [%s]", deploymentName)

	// now delete the deployment and services
	deletePropagation := metav1.DeletePropagationForeground
	err := request.Clientset.
		AppsV1().Deployments(request.Namespace).
		Delete(ctx, deploymentName, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
	if err != nil {
		log.Error(err)
	}

	//delete the service for the backrest repo
	err = request.Clientset.
		CoreV1().Services(request.Namespace).
		Delete(ctx, deploymentName, metav1.DeleteOptions{})
	if err != nil {
		log.Error(err)
	}
}

// removeAllBackupPVCs removes all of the PVCs associated with any kind of
// backup
func removeAllBackupPVCs(request Request) {
	// first, ensure that logical backups are removed
	removeLogicalBackupPVCs(request)
	// finally, we will remove the pgBackRest repo PVC...or PVCs?
	removePgBackRestRepoPVCs(request)
}

// removeBackupSecrets removes any secrets that are associated with backups
// for this cluster, in particular, the secret that is used by the pgBackRest
// repository that is available for this cluster.
func removeBackupSecrets(request Request) {
	ctx := context.TODO()
	// first, derive the secrename of the pgBackRest repo, which is the
	// "`clusterName`-`LABEL_BACKREST_REPO_SECRET`"
	secretName := fmt.Sprintf("%s-%s",
		request.ClusterName, config.LABEL_BACKREST_REPO_SECRET)
	log.Debugf("removeBackupSecrets: %s", secretName)

	// we can attempt to delete the secret directly without making any further
	// API calls. Even if we did a "get", there could still be a race with some
	// independent process (e.g. an external user) deleting the secret before we
	// get to it. The main goal is to have the secret deleted
	//
	// we'll also check to see if there was an error, but if there is we'll only
	// log the fact there was an error; this function is just a pass through
	if err := request.Clientset.CoreV1().Secrets(request.Namespace).Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil {
		log.Error(err)
	}
}

// removeClusterConfigmaps deletes the configmaps that are created for each
// cluster. The first two are created by Patroni when it initializes a new cluster:
// <cluster-name>-leader (stores data pertinent to the leader election process)
// <cluster-name>-config (stores global/cluster-wide configuration settings)
// Additionally, the Postgres Operator also creates a configMap for each cluster
// containing a default Patroni configuration file:
// <cluster-name>-pgha-config (stores a Patroni config file in YAML format)
func removeClusterConfigmaps(request Request) {
	ctx := context.TODO()
	// Store the derived names of the three configmaps in an array
	clusterConfigmaps := []string{
		// first, derive the name of the PG HA default configmap, which is
		// "`clusterName`-`LABEL_PGHA_CONFIGMAP`"
		fmt.Sprintf("%s-%s", request.ClusterName, config.LABEL_PGHA_CONFIGMAP),
		// next, the name of the leader configmap, which is
		// "`clusterName`-leader"
		fmt.Sprintf("%s-%s", request.ClusterName, leaderConfigMapSuffix),
		// next, the name of the general configuration settings configmap, which is
		// "`clusterName`-config"
		fmt.Sprintf("%s-%s", request.ClusterName, configConfigMapSuffix),
		// next, the name of the failover configmap, which is
		// "`clusterName`-failover"
		fmt.Sprintf("%s-%s", request.ClusterName, failoverConfigMapSuffix),
		// finally, if there is a pgbouncer, remove the pgbouncer configmap
		util.GeneratePgBouncerConfigMapName(request.ClusterName),
	}

	// As with similar resources, we can attempt to delete the configmaps directly without
	// making any further API calls since the goal is simply to delete the configmap. Race
	// conditions are more or less unavoidable but should not cause any additional problems.
	// We'll also check to see if there was an error, but if there is we'll only
	// log the fact there was an error; this function is just a pass through
	for _, cm := range clusterConfigmaps {
		if err := request.Clientset.CoreV1().ConfigMaps(request.Namespace).Delete(ctx, cm, metav1.DeleteOptions{}); err != nil && !kerror.IsNotFound(err) {
			log.Error(err)
		}
	}
}

// removeCluster removes the cluster deployments EXCEPT for the pgBackRest repo
func removeCluster(request Request) {
	ctx := context.TODO()
	// ensure we are deleting every deployment EXCEPT for the pgBackRest repo,
	// which needs to happen in a separate step to ensure we clear out all the
	// data
	selector := fmt.Sprintf("%s=%s,%s!=true",
		config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_PGO_BACKREST_REPO)

	deployments, err := request.Clientset.
		AppsV1().Deployments(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})

	// if there is an error here, return as we cannot iterate over the deployment
	// list
	if err != nil {
		log.Error(err)
		return
	}

	// iterate through each deployment and delete it
	for _, d := range deployments.Items {
		deletePropagation := metav1.DeletePropagationForeground
		err := request.Clientset.
			AppsV1().Deployments(request.Namespace).
			Delete(ctx, d.Name, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
		if err != nil {
			log.Error(err)
		}
	}

	// this was here before...this looks like it ensures that deployments are
	// deleted. the only thing I'm modifying is the selector
	var completed bool
	for i := 0; i < maximumTries; i++ {
		deployments, err := request.Clientset.
			AppsV1().Deployments(request.Namespace).
			List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			log.Error(err)
		}
		if len(deployments.Items) > 0 {
			log.Info("sleeping to wait for Deployments to fully terminate")
			time.Sleep(time.Second * time.Duration(4))
		} else {
			completed = true
		}
	}
	if !completed {
		log.Error("could not terminate all cluster deployments")
	}
}

func removeReplica(request Request) error {
	ctx := context.TODO()
	deletePropagation := metav1.DeletePropagationForeground
	err := request.Clientset.
		AppsV1().Deployments(request.Namespace).
		Delete(ctx, request.ReplicaName, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
	if err != nil {
		log.Error(err)
		return err
	}

	//wait for the deployment to go away fully
	var completed bool
	for i := 0; i < maximumTries; i++ {
		_, err = request.Clientset.
			AppsV1().Deployments(request.Namespace).
			Get(ctx, request.ReplicaName, metav1.GetOptions{})
		if err == nil {
			log.Info("sleeping to wait for Deployments to fully terminate")
			time.Sleep(time.Second * time.Duration(4))
		} else {
			completed = true
			break
		}
	}
	if !completed {
		return errors.New("could not delete replica deployment within max tries")
	}
	return nil
}

func removeUserSecrets(request Request) {
	ctx := context.TODO()
	//get all that match pg-cluster=db
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName

	secrets, err := request.Clientset.
		CoreV1().Secrets(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error(err)
		return
	}

	for _, s := range secrets.Items {
		if s.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "" {
			err := request.Clientset.CoreV1().Secrets(request.Namespace).Delete(ctx, s.ObjectMeta.Name, metav1.DeleteOptions{})
			if err != nil {
				log.Error(err)
			}
		}
	}

}

func removeAddons(request Request) {
	ctx := context.TODO()
	//remove pgbouncer

	pgbouncerDepName := request.ClusterName + "-pgbouncer"

	deletePropagation := metav1.DeletePropagationForeground
	_ = request.Clientset.
		AppsV1().Deployments(request.Namespace).
		Delete(ctx, pgbouncerDepName, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})

	//delete the service name=<clustename>-pgbouncer

	_ = request.Clientset.
		CoreV1().Services(request.Namespace).
		Delete(ctx, pgbouncerDepName, metav1.DeleteOptions{})
}

func removeServices(request Request) {
	ctx := context.TODO()

	//remove any service for this cluster

	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName

	services, err := request.Clientset.
		CoreV1().Services(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error(err)
		return
	}

	for i := 0; i < len(services.Items); i++ {
		err := request.Clientset.
			CoreV1().Services(request.Namespace).
			Delete(ctx, services.Items[i].Name, metav1.DeleteOptions{})
		if err != nil {
			log.Error(err)
		}
	}

}

func removePgreplicas(request Request) {
	ctx := context.TODO()

	//get a list of pgreplicas for this cluster
	replicaList, err := request.Clientset.CrunchydataV1().Pgreplicas(request.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: config.LABEL_PG_CLUSTER + "=" + request.ClusterName,
	})
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("pgreplicas found len is %d\n", len(replicaList.Items))

	for _, r := range replicaList.Items {
		if err := request.Clientset.
			CrunchydataV1().Pgreplicas(request.Namespace).
			Delete(ctx, r.Spec.Name, metav1.DeleteOptions{}); err != nil {
			log.Warn(err)
		}
	}

}

func removePgtasks(request Request) {
	ctx := context.TODO()

	//get a list of pgtasks for this cluster
	taskList, err := request.Clientset.
		CrunchydataV1().Pgtasks(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER + "=" + request.ClusterName})
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("pgtasks to remove is %d\n", len(taskList.Items))

	for _, r := range taskList.Items {
		if err := request.Clientset.CrunchydataV1().Pgtasks(request.Namespace).Delete(ctx, r.Spec.Name, metav1.DeleteOptions{}); err != nil {
			log.Warn(err)
		}
	}

}

// getInstancePVCs gets all the PVCs that are associated with PostgreSQL
// instances (at least to the best of our knowledge)
func getInstancePVCs(request Request) ([]string, error) {
	ctx := context.TODO()
	pvcList := make([]string, 0)
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)
	pgDump, pgBackRest := fmt.Sprintf(pgDumpPVCPrefix, request.ClusterName),
		fmt.Sprintf(pgBackRestRepoPVC, request.ClusterName)

	log.Debugf("instance pvcs overall selector: [%s]", selector)

	// get all of the PVCs to analyze (see the step below)
	pvcs, err := request.Clientset.
		CoreV1().PersistentVolumeClaims(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})

	// if there is an error, return here and log the error in the calling function
	if err != nil {
		return pvcList, err
	}

	// ...this will be a bit janky.
	//
	// ...we are going to go through all of the PVCs that are associated with this
	// cluster. We will then compare them against the names of the backup types
	// of PVCs. If they do not match any of those names, then we will add them
	// to the list.
	//
	// ...process of elimination until we tighten up the labeling
	for _, pvc := range pvcs.Items {
		pvcName := pvc.ObjectMeta.Name

		log.Debugf("found pvc: [%s]", pvcName)

		if strings.HasPrefix(pvcName, pgDump) || pvcName == pgBackRest {
			log.Debug("skipping...")
			continue
		}

		pvcList = append(pvcList, pvcName)
	}

	log.Debugf("instance pvcs found: [%v]", pvcList)

	return pvcList, nil
}

//get the pvc for this replica deployment
func getReplicaPVC(request Request) ([]string, error) {
	ctx := context.TODO()
	pvcList := make([]string, 0)

	//at this point, the naming convention is useful
	//and ClusterName is the replica deployment name
	//when isReplica=true
	pvcList = append(pvcList, request.ReplicaName)

	// see if there are any tablespaces or WAL volumes assigned to this replica,
	// and add them to the list.
	//
	// ...this is a bit janky, as we have to iterate through ALL the PVCs
	// associated with this managed cluster, and pull out anyones that have a name
	// with the pattern "<replicaName-tablespace>" or "<replicaName-wal>"
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)

	// get all of the PVCs that are specific to this replica and remove them
	pvcs, err := request.Clientset.
		CoreV1().PersistentVolumeClaims(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})

	// if there is an error, return here and log the error in the calling function
	if err != nil {
		return pvcList, err
	}

	// ...and where the fun begins
	tablespaceReplicaPVCPrefix := fmt.Sprintf(tablespaceReplicaPVCPattern, request.ReplicaName)
	walReplicaPVCName := fmt.Sprintf(walReplicaPVCPattern, request.ReplicaName)

	// iterate over the PVC list and append the tablespace PVCs
	for _, pvc := range pvcs.Items {
		pvcName := pvc.ObjectMeta.Name

		// if it does not start with the tablespace replica PVC pattern and does not equal the WAL
		// PVC pattern then continue
		if !(strings.HasPrefix(pvcName, tablespaceReplicaPVCPrefix) ||
			pvcName == walReplicaPVCName) {
			continue
		}

		log.Debugf("found pvc: [%s]", pvcName)

		pvcList = append(pvcList, pvcName)
	}

	return pvcList, nil
}

func removePVCs(pvcList []string, request Request) error {
	ctx := context.TODO()

	for _, p := range pvcList {
		log.Infof("deleting pvc %s", p)
		deletePropagation := metav1.DeletePropagationForeground
		err := request.Clientset.
			CoreV1().PersistentVolumeClaims(request.Namespace).
			Delete(ctx, p, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
		if err != nil {
			log.Error(err)
		}
	}

	return nil

}

// removeBackupJobs removes any job associated with a backup. These include:
//
// - pgBackRest
// - pg_dump (logical)
func removeBackupJobs(request Request) {
	ctx := context.TODO()

	// Some mild cleanup for this function...going to make a list of selectors
	// for the different kinds of backup jobs so they can be deleted, but cannot
	// do a full cleanup of this process just yet
	selectors := []string{
		// pgBackRest
		fmt.Sprintf("%s=%s,%s=true", config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_BACKREST_JOB),
		// pg_dump
		fmt.Sprintf("%s=%s,%s=true", config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_BACKUP_TYPE_PGDUMP),
	}

	// iterate through each type of selector and attempt to get all of the jobs
	// that are associated with it
	for _, selector := range selectors {
		log.Debugf("backup job selector: [%s]", selector)

		// find all the jobs associated with this selector
		jobs, err := request.Clientset.
			BatchV1().Jobs(request.Namespace).
			List(ctx, metav1.ListOptions{LabelSelector: selector})

		if err != nil {
			log.Error(err)
			continue
		}

		// iterate through the list of jobs and attempt to delete them
		for i := 0; i < len(jobs.Items); i++ {
			deletePropagation := metav1.DeletePropagationForeground
			err := request.Clientset.
				BatchV1().Jobs(request.Namespace).
				Delete(ctx, jobs.Items[i].Name, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
			if err != nil {
				log.Error(err)
			}
		}

		// ...ensure all the jobs are deleted
		var completed bool

		for i := 0; i < maximumTries; i++ {
			jobs, err := request.Clientset.
				BatchV1().Jobs(request.Namespace).
				List(ctx, metav1.ListOptions{LabelSelector: selector})

			if len(jobs.Items) > 0 || err != nil {
				log.Debug("sleeping to wait for backup jobs to fully terminate")
				time.Sleep(time.Second * time.Duration(4))
			} else {
				completed = true
				break
			}
		}

		if !completed {
			log.Errorf("could not remove all backup jobs for [%s]", selector)
		}
	}
}

// removeLogicalBackupPVCs removes the logical backups associated with a cluster
// this is an "all-or-nothing" solution: as right now it will only remove the
// PVC, it will remove **all** logical backups
//
// Additionally, as these backups are nota actually mounted anywhere, except
// during one-off jobs, we cannot perform a delete of the filesystem (i.e.
// "rm -rf" like in other commands). Well, we could...we could write a job to do
// this, but that will be saved for future work
func removeLogicalBackupPVCs(request Request) {
	ctx := context.TODO()
	pvcList := make([]string, 0)
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)
	dumpPrefix := fmt.Sprintf(pgDumpPVCPrefix, request.ClusterName)

	// get all of the PVCs to analyze (see the step below)
	pvcs, err := request.Clientset.
		CoreV1().PersistentVolumeClaims(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error(err)
		return
	}

	// Now iterate through all the PVCs to identify those that are for a logical backup and add
	// them to the PVC list for deletion.  This pattern matching will be utilized until better
	// labeling is in place to uniquely identify logical backup PVCs.
	for _, pvc := range pvcs.Items {
		pvcName := pvc.GetName()

		if !strings.HasPrefix(pvcName, dumpPrefix) {
			continue
		}

		pvcList = append(pvcList, pvcName)
	}

	log.Debugf("logical backup pvcs found: [%v]", pvcList)

	removePVCs(pvcList, request)
}

// removePgBackRestRepoPVCs removes any PVCs that are used by a pgBackRest repo
func removePgBackRestRepoPVCs(request Request) {
	// there is only a single PVC for a pgBackRest repo, and it has a well-defined
	// name
	pvcName := fmt.Sprintf(pgBackRestRepoPVC, request.ClusterName)

	log.Debugf("remove backrest pvc name [%s]", pvcName)

	// make a simple of the PVCs that can be removed by the removePVC command
	pvcList := []string{pvcName}
	removePVCs(pvcList, request)
}

// removeReplicaServices removes the replica service if there is currently only a single replica
// in the cluster, i.e. if the last/final replica is being being removed with the current rmdata
// job.  If more than one replica still exists, then no action is taken.
func removeReplicaServices(request Request) {
	ctx := context.TODO()

	// selector in the format "pg-cluster=<cluster-name>,role=replica"
	// which will grab any/all replicas
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName,
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_REPLICA)
	replicaList, err := request.Clientset.
		CoreV1().Pods(request.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error(err)
		return
	}

	switch len(replicaList.Items) {
	case 0:
		log.Error("no replicas found for this cluster")
		return
	case 1:
		log.Debug("removing replica service when scaling down to 0 replicas")
		err := request.Clientset.
			CoreV1().Services(request.Namespace).
			Delete(ctx, request.ClusterName+"-replica", metav1.DeleteOptions{})
		if err != nil {
			log.Error(err)
			return
		}
	}

	log.Debug("more than one replica detected, replica service will not be deleted")
}
