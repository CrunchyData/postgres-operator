package rmdata

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
	"errors"
	"fmt"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
	kerror "k8s.io/apimachinery/pkg/api/errors"

	"time"
)

const (
	MAX_TRIES            = 16
	pgBackRestPathFormat = "/backrestrepo/%s"
	pgBackRestRepoPVC    = "%s-pgbr-repo"
	pgDumpPVC            = "backup-%s-pgdump-pvc"
	pgDataPathFormat     = "/pgdata/%s"
	tablespacePathFormat = "/tablespaces/%s/%s"
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
		if err = kubeapi.Deletepgreplica(request.RESTClient, request.ReplicaName,
			request.Namespace); err != nil {
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

	// first, clear out any of the scheduled jobs that may occur, as this would be
	// executing asynchronously against any stale data
	removeSchedules(request)

	//the user had done something like:
	//pgo delete cluster mycluster --delete-data
	if request.RemoveData {
		removeUserSecrets(request)
	}

	//handle the case of 'pgo delete cluster mycluster'
	removeCluster(request)
	if err := kubeapi.Deletepgcluster(request.RESTClient, request.ClusterName, request.Namespace); err != nil {
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
	deploymentName := fmt.Sprintf("%s-backrest-shared-repo", request.ClusterName)

	log.Debugf("deleting the pgbackrest repo [%s]", deploymentName)

	// now delete the deployment and services
	err := kubeapi.DeleteDeployment(request.Clientset, deploymentName, request.Namespace)
	if err != nil {
		log.Error(err)
	}

	//delete the service for the backrest repo
	err = kubeapi.DeleteService(request.Clientset, deploymentName, request.Namespace)
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
	if err := kubeapi.DeleteSecret(request.Clientset, secretName, request.Namespace); err != nil {
		log.Error(err)
	}

	// and done!
	return
}

// removeClusterConfigmaps deletes the three configmaps that are created
// for each cluster. The first two are created by Patroni when it initializes a new cluster:
// <cluster-name>-leader (stores data pertinent to the leader election process)
// <cluster-name>-config (stores global/cluster-wide configuration settings)
// Additionally, the Postgres Operator also creates a configMap for each cluster
// containing a default Patroni configuration file:
// <cluster-name>-pgha-config (stores a Patroni config file in YAML format)
func removeClusterConfigmaps(request Request) {
	// Store the derived names of the three configmaps in an array
	clusterConfigmaps := [4]string{
		// first, derive the name of the PG HA default configmap, which is
		// "`clusterName`-`LABEL_PGHA_CONFIGMAP`"
		fmt.Sprintf("%s-%s", request.ClusterName, config.LABEL_PGHA_CONFIGMAP),
		// next, the name of the leader configmap, which is
		// "`clusterName`-leader"
		fmt.Sprintf("%s-%s", request.ClusterName, leaderConfigMapSuffix),
		// next, the name of the general configuration settings configmap, which is
		// "`clusterName`-config"
		fmt.Sprintf("%s-%s", request.ClusterName, configConfigMapSuffix),
		// finally, the name of the failover configmap, which is
		// "`clusterName`-failover"
		fmt.Sprintf("%s-%s", request.ClusterName, failoverConfigMapSuffix),
	}

	// As with similar resources, we can attempt to delete the configmaps directly without
	// making any further API calls since the goal is simply to delete the configmap. Race
	// conditions are more or less unavoidable but should not cause any additional problems.
	// We'll also check to see if there was an error, but if there is we'll only
	// log the fact there was an error; this function is just a pass through
	for _, cm := range clusterConfigmaps {
		if err := kubeapi.DeleteConfigMap(request.Clientset, cm, request.Namespace); err != nil {
			log.Error(err)
		}
	}
}

func removeClusterJobs(request Request) {
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName
	jobs, err := kubeapi.GetJobs(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}
	for i := 0; i < len(jobs.Items); i++ {
		job := jobs.Items[i]
		err := kubeapi.DeleteJob(request.Clientset, job.Name, request.Namespace)
		if err != nil {
			log.Error(err)
		}
	}
}

// removeCluster removes the cluster deployments EXCEPT for the pgBackRest repo
func removeCluster(request Request) {
	// ensure we are deleting every deployment EXCEPT for the pgBackRest repo,
	// which needs to happen in a separate step to ensure we clear out all the
	// data
	selector := fmt.Sprintf("%s=%s,%s!=true",
		config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_PGO_BACKREST_REPO)

	deployments, err := kubeapi.GetDeployments(request.Clientset, selector, request.Namespace)

	// if there is an error here, return as we cannot iterate over the deployment
	// list
	if err != nil {
		log.Error(err)
		return
	}

	// iterate through each deployment and delete it
	for _, d := range deployments.Items {
		if err := kubeapi.DeleteDeployment(request.Clientset, d.ObjectMeta.Name, request.Namespace); err != nil {
			log.Error(err)
		}
	}

	// this was here before...this looks like it ensures that deployments are
	// deleted. the only thing I'm modifying is the selector
	var completed bool
	for i := 0; i < MAX_TRIES; i++ {
		deployments, err := kubeapi.GetDeployments(request.Clientset, selector,
			request.Namespace)
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

	d, found, err := kubeapi.GetDeployment(request.Clientset,
		request.ReplicaName, request.Namespace)
	if !found || err != nil {
		log.Error(err)
		return err
	}

	err = kubeapi.DeleteDeployment(request.Clientset, d.ObjectMeta.Name, request.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	//wait for the deployment to go away fully
	var completed bool
	for i := 0; i < MAX_TRIES; i++ {
		_, found, _ := kubeapi.GetDeployment(request.Clientset,
			request.ReplicaName, request.Namespace)
		if found {
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
	//get all that match pg-cluster=db
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName

	secrets, err := kubeapi.GetSecrets(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	for _, s := range secrets.Items {
		if s.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "" {
			err := kubeapi.DeleteSecret(request.Clientset, s.ObjectMeta.Name, request.Namespace)
			if err != nil {
				log.Error(err)
			}
		}
	}

}

func removeAddons(request Request) {
	//remove pgbouncer

	pgbouncerDepName := request.ClusterName + "-pgbouncer"

	_, found, _ := kubeapi.GetDeployment(request.Clientset, pgbouncerDepName, request.Namespace)
	if found {

		kubeapi.DeleteDeployment(request.Clientset, pgbouncerDepName, request.Namespace)
	}

	//delete the service name=<clustename>-pgbouncer

	_, found, _ = kubeapi.GetService(request.Clientset, pgbouncerDepName, request.Namespace)
	if found {
		kubeapi.DeleteService(request.Clientset, pgbouncerDepName, request.Namespace)
	}

}

func removeServices(request Request) {

	//remove any service for this cluster

	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName

	services, err := kubeapi.GetServices(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	for i := 0; i < len(services.Items); i++ {
		svc := services.Items[i]
		err := kubeapi.DeleteService(request.Clientset, svc.Name, request.Namespace)
		if err != nil {
			log.Error(err)
		}
	}

}

func removePgreplicas(request Request) {
	replicaList := crv1.PgreplicaList{}

	//get a list of pgreplicas for this cluster
	err := kubeapi.GetpgreplicasBySelector(request.RESTClient,
		&replicaList, config.LABEL_PG_CLUSTER+"="+request.ClusterName,
		request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("pgreplicas found len is %d\n", len(replicaList.Items))

	for _, r := range replicaList.Items {
		if err := kubeapi.Deletepgreplica(request.RESTClient, r.Spec.Name, request.Namespace); err != nil {
			log.Warn(err)
		}
	}

}

func removePgtasks(request Request) {
	taskList := crv1.PgtaskList{}

	//get a list of pgtasks for this cluster
	err := kubeapi.GetpgtasksBySelector(request.RESTClient,
		&taskList, config.LABEL_PG_CLUSTER+"="+request.ClusterName,
		request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("pgtasks to remove is %d\n", len(taskList.Items))

	for _, r := range taskList.Items {
		if err := kubeapi.Deletepgtask(request.RESTClient, r.Spec.Name, request.Namespace); err != nil {
			log.Warn(err)
		}
	}

}

// getInstancePVCs gets all the PVCs that are associated with PostgreSQL
// instances (at least to the best of our knowledge)
func getInstancePVCs(request Request) ([]string, error) {
	pvcList := make([]string, 0)
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)
	pgDump, pgBackRest := fmt.Sprintf(pgDumpPVC, request.ClusterName),
		fmt.Sprintf(pgBackRestRepoPVC, request.ClusterName)

	log.Debugf("instance pvcs overall selector: [%s]", selector)

	// get all of the PVCs to analyze (see the step below)
	pvcs, err := kubeapi.GetPVCs(request.Clientset, selector, request.Namespace)

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

		if pvcName == pgDump || pvcName == pgBackRest {
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
	pvcs, err := kubeapi.GetPVCs(request.Clientset, selector, request.Namespace)

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

	for _, p := range pvcList {
		log.Infof("deleting pvc %s", p)
		err := kubeapi.DeletePVC(request.Clientset, p, request.Namespace)
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
		jobs, err := kubeapi.GetJobs(request.Clientset, selector, request.Namespace)

		if err != nil {
			log.Error(err)
			continue
		}

		// iterate through the list of jobs and attempt to delete them
		for i := 0; i < len(jobs.Items); i++ {
			job := jobs.Items[i]

			if err := kubeapi.DeleteJob(request.Clientset, job.Name, request.Namespace); err != nil {
				log.Error(err)
			}
		}

		// ...ensure all the jobs are deleted
		var completed bool

		for i := 0; i < MAX_TRIES; i++ {
			jobs, err := kubeapi.GetJobs(request.Clientset, selector, request.Namespace)

			if len(jobs.Items) > 0 || err != nil {
				log.Debug("sleeping to wait for backup jobs to fully terminate")
				time.Sleep(time.Second * time.Duration(4))
			} else {
				completed = true
				break
			}
		}

		if !completed {
			log.Error("could not remove all backup jobs for [%s]", selector)
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
	// get the name of the PVC, which uses a format that is fixed
	pvcName := fmt.Sprintf(pgDumpPVC, request.ClusterName)

	log.Debugf("remove pgdump pvc name [%s]", pvcName)

	// make a simple list of the PVCs that can be applied to the "removePVC"
	// command
	pvcList := []string{pvcName}
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

	// selector in the format "pg-cluster=<cluster-name>,role=replica"
	// which will grab any/all replicas
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName,
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_REPLICA)
	replicaList, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace)
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
		if kubeapi.DeleteService(request.Clientset, request.ClusterName+"-replica",
			request.Namespace); err != nil {
			log.Error(err)
			return
		}
	}

	log.Debug("more than one replica detected, replica service will not be deleted")
}

// removeSchedules removes any of the ConfigMap objects that were created to
// execute schedule tasks, such as backups
// As these are consistently labeled, we can leverage Kuernetes selectors to
// delete all of them
func removeSchedules(request Request) {
	log.Debugf("removing schedules for '%s'", request.ClusterName)

	// a ConfigMap used for the schedule uses the following label selector:
	// crunchy-scheduler=true,<config.LABEL_PG_CLUSTER>=<request.ClusterName>
	selector := fmt.Sprintf("crunchy-scheduler=true,%s=%s",
		config.LABEL_PG_CLUSTER, request.ClusterName)

	// run the query the deletes all of the scheduled configmaps
	// if there is an error, log it, but continue on without making a big stink
	if err := kubeapi.DeleteConfigMaps(request.Clientset, selector, request.Namespace); err != nil {
		log.Error(err)
	}
}
