package rmdata

/*
Copyright 2019 Crunchy Data
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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	"time"
)

const (
	MAX_TRIES            = 16
	pgBackRestPathFormat = "/backrestrepo/%s"
	pgDataPathFormat     = "/pgdata/%s"
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
		err = kubeapi.Deletepgreplica(request.RESTClient, request.ReplicaName, request.Namespace)
		if err != nil {
			log.Error(err)
			return
		}

		if request.RemoveData {
			removeOnlyReplicaData(request)
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
		//the case of removing a backup using
		//pgo delete backup, only applies to
		//backup-type=pgbasebackup
		//currently we only support removing the PVC
		//and not the backup contents
		removeBackupJobs(request)
		pvcList := make([]string, 0)
		pvcList = append(pvcList, request.ClusterName+"-backup")
		removePVCs(pvcList, request)

		//removing a backup pvc is its own case, leave when done
		return
	}

	log.Info("rmdata.Process cluster use case")

	// first, clear out any of the scheduled jobs that may occur, as this would be
	// executing asynchronously against any stale data
	removeSchedules(request)

	//the user had done something like:
	//pgo delete cluster mycluster --delete-data
	if request.RemoveData {
		removeData(request)
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
		pvcList, err := getPVCs(request)
		if err != nil {
			log.Error(err)
		}
		removePVCs(pvcList, request)
	}

	// backups have to be the last thing we remove. We want to ensure that all
	// the clusters (well, really, the primary) have stopped. This means that no
	// more WAL archives are being pushed, and at this point it is safe for us to
	// remove the pgBackRest repo
	//
	// The actual check for removing the pgBackRest files occurs in the
	// removeBackRestRepo function. The rest of the function calls are related
	// to the current deployment, i.e. this PostgreSQL cluster, and are safe to
	// remove
	removeBackrestRepo(request)
	removeBackupJobs(request)
	// NOTE: removeBackups will be removed in a later commit -- it is ok to not
	// guard it
	removeBackups(request)
	removeBackupSecrets(request)
}

// deleteClusterData calls a series of commands to attempt to "safely" delete
// the PGDATA on the server
//
/// First, it ensures PostgreSQL has stopped, to avoid any unpredictable
// behavior.
//
// Next, it calls "rm -rf" on the directory. This was implemented prior to this
// "refactor" and may change in the future to be a bit safer in this environment
func deletePGDATA(request Request, pod v1.Pod) error {
	clusterName := pod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]
	containerName := pod.Spec.Containers[0].Name
	pgDataPath := fmt.Sprintf(pgDataPathFormat, clusterName)

	log.Debugf("delete pgdata: [%s]", clusterName)

	// first, ensure the PostgreSQL cluster is stopped
	stopCommand := []string{"pg_ctl", "stop", "-m", "immediate", "-D", pgDataPath}
	// execute the command here. While not ideal, it's OK if it errors out.
	if stdout, stderr, err := kubeapi.ExecToPodThroughAPI(
		request.RESTConfig,
		request.Clientset,
		stopCommand,
		containerName,
		pod.Name,
		request.Namespace, nil); err != nil {
		log.Errorf("error execing into remove data pod %s: command %s error %s", pod.Name, stopCommand, err.Error())
	} else {
		log.Infof("stopped postgresql [%s]: stdout=[%s] stderr=[%s]", clusterName, stdout, stderr)
	}

	// execute the rm command here. If it errors, return
	if stdout, stderr, err := rmDataDir(request, pod.Name, containerName, pgDataPath); err != nil {
		return err
	} else {
		log.Infof("rm pgdata [%s]: stdout=[%s] stderr=[%s]", clusterName, stdout, stderr)
	}

	return nil
}

// removeBackRestRepo removes the pgBackRest repo that is associated with the
// PostgreSQL cluster
func removeBackrestRepo(request Request) {
	deploymentName := fmt.Sprintf("%s-backrest-shared-repo", request.ClusterName)
	repoPath := fmt.Sprintf(pgBackRestPathFormat, deploymentName)

	log.Debugf("deleting the pgbackrest repo [%s]", deploymentName)

	// first, delete the backup data directory
	// we'll determine from the request if we ewant to remove the backup directory
	if request.RemoveBackup {
		// NOTE: need to use the ClusterName for the Pod selector
		selector := fmt.Sprintf("%s=%s,%s=true",
			config.LABEL_PG_CLUSTER, request.ClusterName, config.LABEL_PGO_BACKREST_REPO)
		// even if we error, we can move on with deleting the deployments and servies
		if pods, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace); err != nil {
			log.Error(err)
		} else {
			// iterate through any pod matching this query, and remove the data
			// directory
			for _, pod := range pods.Items {
				log.Debugf("remove pgbackrest repo from pod [%s]", pod.Name)
				// take the first available container
				containerName := pod.Spec.Containers[0].Name

				if stdout, stderr, err := rmDataDir(request, pod.Name, containerName, repoPath); err != nil {
					log.Error(err)
				} else {
					log.Infof("rm pgbackrest repo [%s]: stdout=[%s] stderr=[%s]", deploymentName, stdout, stderr)
				}
			}
		}
	}

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

func removeBackups(request Request) {

	//see if a pgbasebackup PVC exists
	backupPVCName := request.ClusterName + "-backup"
	log.Infof("pgbasebackup backup pvc: %s", backupPVCName)
	pvc, found, err := kubeapi.GetPVC(request.Clientset, request.ClusterName, request.Namespace)
	if found {
		log.Infof("pgbasebackup backup pvc: found")
		err = kubeapi.DeletePVC(request.Clientset, pvc.Name, request.Namespace)
		if err != nil {
			log.Errorf("error removing pgbasebackup pvc %s %s", backupPVCName, err.Error())
		} else {
			log.Infof("removed pgbasebackup pvc %s", backupPVCName)
		}
	} else {
		log.Infof("pgbasebackup backup pvc: NOT found")
	}

	//delete pgbackrest PVC if it exists

	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName
	log.Infof("remove backrest pvc selector [%s]", selector)

	var pvcList *v1.PersistentVolumeClaimList
	pvcList, err = kubeapi.GetPVCs(request.Clientset, selector, request.Namespace)
	if len(pvcList.Items) > 0 {
		for _, v := range pvcList.Items {
			err = kubeapi.DeletePVC(request.Clientset, v.Name, request.Namespace)
			if err != nil {
				log.Errorf("error removing backrest pvc %s %s", v.Name, err.Error())
			} else {
				log.Infof("removed backrest pvc %s", v.Name)
			}
		}
	}

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
// <cluster-name>-pgha-default-config (stores a Patroni config file in YAML format)
func removeClusterConfigmaps(request Request) {
	// Store the derived names of the three configmaps in an array
	clusterConfigmaps := [3]string{
		// first, derive the name of the PG HA default configmap, which is
		// "`clusterName`-`LABEL_PGHA_DEFAULT_CONFIGMAP`"
		fmt.Sprintf("%s-%s", request.ClusterName, config.LABEL_PGHA_DEFAULT_CONFIGMAP),
		// next, the name of the leader configmap, which is
		// "`clusterName`-leader"
		fmt.Sprintf("%s-%s", request.ClusterName, "leader"),
		// finally, the name of the general configuration settings configmap, which is
		// "`clusterName`-config"
		fmt.Sprintf("%s-%s", request.ClusterName, "config")}

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

func removeData(request Request) {
	//get the replicas
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName + "," + config.LABEL_SERVICE_NAME + "=" + request.ClusterName + "-replica"
	log.Debugf("removeData selector %s", selector)
	pods, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Errorf("error selecting replica pods %s %s", selector, err.Error())
	}

	//replicas should have a label on their pod of the
	//form deployment-name=somedeploymentname

	log.Debugf("removeData %d replica pods", len(pods.Items))
	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if err := deletePGDATA(request, pod); err != nil {
				log.Errorf("error execing into remove data pod %s: %s", pod.Name, err.Error())
			}
		}
	}

	//get the primary

	//primaries should have the label of
	//the form deployment-name=somedeploymentname and service-name=somecluster
	selector = config.LABEL_PG_CLUSTER + "=" + request.ClusterName + "," + config.LABEL_SERVICE_NAME + "=" + request.ClusterName
	pods, err = kubeapi.GetPods(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Errorf("error selecting primary pod %s %s", selector, err.Error())
	}

	if len(pods.Items) > 0 {
		pod := pods.Items[0]
		if err := deletePGDATA(request, pod); err != nil {
			log.Errorf("error execing into remove data pod %s: error %s", pod.Name, err.Error())
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

func removeOnlyReplicaData(request Request) {
	//get the replica pod only, this is the case where
	//a user scales down a replica, in this case the DeploymentName
	//is used to identify the correct pod
	selector := config.LABEL_DEPLOYMENT_NAME + "=" + request.ReplicaName

	pods, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Errorf("error selecting replica pods %s %s", selector, err.Error())
	}

	//replicas should have a label on their pod of the
	//form deployment-name=somedeploymentname

	if len(pods.Items) > 0 {
		for _, v := range pods.Items {
			command := make([]string, 0)
			command = append(command, "rm")
			command = append(command, "-rf")
			command = append(command, "/pgdata/"+v.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME])
			stdout, stderr, err := kubeapi.ExecToPodThroughAPI(request.RESTConfig, request.Clientset, command, v.Spec.Containers[0].Name, v.Name, request.Namespace, nil)
			if err != nil {
				log.Errorf("error execing into remove data pod %s command %s error %s", v.Name, command, err.Error())
			}
			log.Infof("stdout=[%s] stderr=[%s]", stdout, stderr)
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
		err = kubeapi.Deletepgreplica(request.RESTClient, r.Spec.Name, request.Namespace)
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
		err = kubeapi.Deletepgtask(request.RESTClient, r.Spec.Name, request.Namespace)
	}

}

//get the pvc's for this cluster leaving out the backrest repo pvc
func getPVCs(request Request) ([]string, error) {
	pvcList := make([]string, 0)
	deployments, err := kubeapi.GetDeployments(request.Clientset,
		config.LABEL_PG_CLUSTER+"="+request.ClusterName, request.Namespace)
	if err != nil {
		log.Error(err)
		return pvcList, err
	}

	for _, d := range deployments.Items {
		if d.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "" {
			pvcList = append(pvcList, d.ObjectMeta.Name)
		}
	}

	return pvcList, nil

}

//get the pvc for this replica deployment
func getReplicaPVC(request Request) ([]string, error) {
	pvcList := make([]string, 0)

	//at this point, the naming convention is useful
	//and ClusterName is the replica deployment name
	//when isReplica=true
	pvcList = append(pvcList, request.ReplicaName)
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

func removeBackupJobs(request Request) {
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName + "," + config.LABEL_PGBACKUP + "=true"
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

	var completed bool
	for i := 0; i < MAX_TRIES; i++ {
		jobs, err := kubeapi.GetJobs(request.Clientset, selector, request.Namespace)
		if len(jobs.Items) > 0 || err != nil {
			log.Info("sleeping to wait for backup jobs to fully terminate")
			time.Sleep(time.Second * time.Duration(4))
		} else {
			completed = true
			break
		}
	}
	if !completed {
		log.Error("could not remove all backup jobs")
	}
}

func removeReplicaServices(request Request) {

	//remove the replica service if there is only a single replica
	//which means we are scaling down the only replica

	var err error
	var replicaList *appsv1.DeploymentList
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName + "," + config.LABEL_SERVICE_NAME + "=" + request.ClusterName + "-replica"
	replicaList, err = kubeapi.GetDeployments(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}
	if len(replicaList.Items) == 0 {
		log.Error("no replicas found for this cluster")
		return
	}

	if len(replicaList.Items) == 1 {
		log.Debug("removing replica service when scaling down to 0 replicas")
		err = kubeapi.DeleteService(request.Clientset, request.ClusterName+"-replica", request.Namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}

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

// rmDataDir removes a data directory, such as a PGDATA directory, or a pgBackRest
// repo
func rmDataDir(request Request, podName, containerName, path string) (string, string, error) {
	command := []string{"rm", "-rf", path}

	return kubeapi.ExecToPodThroughAPI(
		request.RESTConfig,
		request.Clientset,
		command,
		containerName,
		podName,
		request.Namespace, nil)
}
