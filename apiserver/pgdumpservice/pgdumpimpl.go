package pgdumpservice

/*
Copyright 2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pgDumpCommand = "pgdump"
const pgDumpInfoCommand = "info"
const pgDumpTaskExtension = "-pgdump"
const pgDumpJobExtension = "-pgdump-job"

// const containername = "database" //TODO: is this correct?

//  CreateBackup ...
// pgo backup mycluster
// pgo backup --selector=name=mycluster
func CreatepgDump(request *msgs.CreatepgDumpBackupRequest) msgs.CreatepgDumpBackupResponse {

	resp := msgs.CreatepgDumpBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	// var newInstance *crv1.Pgtask

	log.Info("CreatePgDump storage config... " + request.StorageConfig)
	if request.StorageConfig != "" {
		if apiserver.IsValidStorageName(request.StorageConfig) == false {
			log.Info("CreateBackup sc error is found " + request.StorageConfig)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.StorageConfig + " Storage config was not found "
			return resp
		}
	}

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		clusterList := crv1.PgclusterList{}

		err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, request.Selector, apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
			resp.Results = append(resp.Results, "no clusters found with that selector")
			return resp
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			request.Args = newargs
		}

	}

	for _, clusterName := range request.Args {
		log.Debugf("create pgdump called for %s", clusterName)
		taskName := "backup-" + clusterName + pgDumpTaskExtension

		cluster := crv1.Pgcluster{}
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, clusterName, apiserver.Namespace)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		RemovePgDumpJob(clusterName + pgDumpJobExtension)

		result := crv1.Pgtask{}

		// error if the task already exists
		found, err = kubeapi.Getpgtask(apiserver.RESTClient, &result, taskName, apiserver.Namespace)
		if !found {
			log.Debugf("pgdump pgtask %s was not found so we will create it", taskName)
		} else if err != nil {

			resp.Results = append(resp.Results, "error getting pgtask for "+taskName)
			break
		} else {

			log.Debugf("pgtask %s was found so we will recreate it", taskName)
			//remove the existing pgtask
			err := kubeapi.Deletepgtask(apiserver.RESTClient, taskName, apiserver.Namespace)

			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		//get pod name from cluster
		// var podname, deployName string
		var podname string
		podname, err = getPrimaryPodName(&cluster)

		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		theTask := buildPgTaskForDump(clusterName, taskName, crv1.PgtaskpgDump, podname, "database", request)

		err = kubeapi.Createpgtask(apiserver.RESTClient, theTask, apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Pgtask "+taskName)

	}

	return resp
}

// ShowpgDump ...
func ShowpgDump(clusterName string, selector string) msgs.ShowBackupResponse {
	var err error

	response := msgs.ShowBackupResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.BackupList.Items = make([]crv1.Pgbackup, 0)

	if selector == "" && clusterName == "all" {
		// leave selector empty, retrieves all clusters.
	} else {
		if selector == "" {
			selector = "name=" + clusterName
		}
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, apiserver.Namespace)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {

		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		pgTaskName := "backup-" + c.Name + pgDumpTaskExtension

		backupItem, error := getPgDumpForTask(c.Name, pgTaskName)

		if backupItem != nil {
			log.Debug("pgTask %s was found", pgTaskName)
			response.BackupList.Items = append(response.BackupList.Items, *backupItem)

		} else if error != nil {

			response.Status.Code = msgs.Error
			response.Status.Msg = error.Error()

		} else {
			// nothing found, no error
			log.Debugf("pgTask %s not found, no erros", pgTaskName)
			response.Status.Code = msgs.Ok
			response.Status.Msg = fmt.Sprintln("pgDump %s not found.", pgTaskName)
		}

	}

	return response

}

func buildPgTaskForDump(clusterName string, taskName string, action string, podName string,
	containerName string, request *msgs.CreatepgDumpBackupRequest) *crv1.Pgtask {

	var newInstance *crv1.Pgtask
	var storageSpec crv1.PgStorageSpec
	var pvcName string

	if request.StorageConfig != "" {
		storageSpec, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
	} else {
		storageSpec, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.BackupStorage)
	}

	// specify PVC name if not set by user.
	if len(request.PVCName) > 0 {
		pvcName = request.PVCName
	} else {
		pvcName = taskName + "-pvc"
	}

	// storageSpec.Name =

	spec := crv1.PgtaskSpec{}

	spec.Name = taskName
	spec.TaskType = crv1.PgtaskpgDump
	spec.Parameters = make(map[string]string)
	spec.Parameters[util.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[util.LABEL_PGDUMP_HOST] = clusterName      // same name as service
	spec.Parameters[util.LABEL_CONTAINER_NAME] = containerName // ??
	spec.Parameters[util.LABEL_PGDUMP_COMMAND] = action
	spec.Parameters[util.LABEL_PGDUMP_OPTS] = request.BackupOpts
	spec.Parameters[util.LABEL_PGDUMP_DB] = "postgres"
	spec.Parameters[util.LABEL_PGDUMP_USER] = clusterName + "-primaryuser-secret"
	spec.Parameters[util.LABEL_PGDUMP_PORT] = "5432"
	spec.Parameters[util.LABEL_PGDUMP_ALL] = "false"
	spec.Parameters[util.LABEL_PVC_NAME] = pvcName
	spec.Parameters[util.LABEL_CCP_IMAGE_TAG_KEY] = apiserver.Pgo.Cluster.CCPImageTag
	spec.StorageSpec = storageSpec

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	return newInstance
}

func getDeployName(cluster *crv1.Pgcluster) (string, error) {
	var depName string

	selector := util.LABEL_PGPOOL + "!=true," + util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + util.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name

	deps, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return depName, err
	}

	if len(deps.Items) != 1 {
		return depName, errors.New("error:  deployment count is wrong for pgdump backup " + cluster.Spec.Name)
	}
	for _, d := range deps.Items {
		return d.Name, err
	}

	return depName, errors.New("unknown error in pgdump backup")
}

func getPrimaryPodName(cluster *crv1.Pgcluster) (string, error) {
	var podname string

	selector := util.LABEL_PGPOOL + "!=true," + util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + util.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return podname, err
	}

	for _, p := range pods.Items {
		if isPrimary(&p, cluster.Spec.Name) && isReady(&p) {
			return p.Name, err
		}
	}

	return podname, errors.New("primary pod is not in Ready state")
}

func isPrimary(pod *v1.Pod, clusterName string) bool {
	if pod.ObjectMeta.Labels[util.LABEL_SERVICE_NAME] == clusterName {
		return true
	}
	return false

}

func isReady(pod *v1.Pod) bool {
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	if readyCount != containerCount {
		return false
	}
	return true

}

// if backup && err are nil, it simply wasn't found. Otherwise found or an error
func getPgDumpForTask(clusterName string, taskName string) (*crv1.Pgbackup, error) {

	task := crv1.Pgtask{}

	var backup *crv1.Pgbackup

	spec := crv1.PgtaskSpec{}
	// spec.Name = name
	spec.TaskType = crv1.PgtaskpgDump

	found, err := kubeapi.Getpgtask(apiserver.RESTClient, &task, taskName, apiserver.Namespace)

	if found {
		backup = convertDumpTaskToPgBackup(&task)
	} else if err == nil {
		// It simply does not exist
		log.Debugf("pgTask not found for requested pgdump %s", taskName)
	}

	return backup, err
}

func convertDumpTaskToPgBackup(dumpTask *crv1.Pgtask) *crv1.Pgbackup {

	backup := crv1.Pgbackup{}

	backup.ObjectMeta.CreationTimestamp = dumpTask.ObjectMeta.CreationTimestamp

	spec := dumpTask.Spec

	backup.Spec.Name = spec.Name
	backup.Spec.BackupStatus = spec.Status
	backup.Spec.CCPImageTag = spec.Parameters[util.LABEL_CCP_IMAGE_TAG_KEY]
	backup.Spec.BackupHost = spec.Parameters[util.LABEL_PGDUMP_HOST]
	backup.Spec.BackupUserSecret = spec.Parameters[util.LABEL_PGDUMP_USER]
	backup.Spec.BackupPort = spec.Parameters[util.LABEL_PGDUMP_PORT]
	backup.Spec.DumpAll = spec.Parameters[util.LABEL_PGDUMP_ALL]

	if backup.Spec.DumpAll == "" {
		backup.Spec.DumpAll = "false"
	}

	backup.Spec.BackupPVC = spec.Parameters[util.LABEL_PVC_NAME]
	backup.Spec.StorageSpec.Size = dumpTask.Spec.StorageSpec.Size
	backup.Spec.StorageSpec.AccessMode = dumpTask.Spec.StorageSpec.AccessMode
	backup.Spec.BackupOpts = spec.Parameters[util.LABEL_PGDUMP_OPTS]

	return &backup

}

//  Restore ...
// pgo restore mycluster --to-cluster=restored
func Restore(request *msgs.RestoreRequest) msgs.RestoreResponse {
	resp := msgs.RestoreResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("Restore %v\n", request)

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, request.FromCluster, apiserver.Namespace)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = request.FromCluster + " was not found, verify cluster name"
		return resp
	} else if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	//verify that the cluster we are restoring from has backrest enabled
	// if cluster.Spec.UserLabels[util.LABEL_BACKREST] != "true" {
	// 	resp.Status.Code = msgs.Error
	// 	resp.Status.Msg = "can't restore, cluster restoring from does not have backrest enabled"
	// 	return resp
	// }

	pgtask := getRestoreParams(request)
	existingTask := crv1.Pgtask{}

	//delete any existing pgtask with the same name
	found, err = kubeapi.Getpgtask(apiserver.RESTClient,
		&existingTask,
		pgtask.Name,
		apiserver.Namespace)
	if found {
		log.Debugf("deleting prior pgtask %s", pgtask.Name)
		err = kubeapi.Deletepgtask(apiserver.RESTClient,
			pgtask.Name,
			apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	//create a pgtask for the restore workflow
	err = kubeapi.Createpgtask(apiserver.RESTClient,
		pgtask,
		apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, "restore performed on "+request.FromCluster+" to "+request.ToPVC+" opts="+request.RestoreOpts+" pitr-target="+request.PITRTarget)

	return resp
}

//TODO: Need to update this for pgdump
func getRestoreParams(request *msgs.RestoreRequest) *crv1.Pgtask {
	var newInstance *crv1.Pgtask

	spec := crv1.PgtaskSpec{}
	spec.Name = "backrest-restore-" + request.FromCluster + "-to-" + request.ToPVC
	spec.TaskType = crv1.PgtaskBackrestRestore
	spec.Parameters = make(map[string]string)
	spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER] = request.FromCluster
	spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC] = request.ToPVC
	spec.Parameters[util.LABEL_BACKREST_RESTORE_OPTS] = request.RestoreOpts
	spec.Parameters[util.LABEL_BACKREST_PITR_TARGET] = request.PITRTarget
	spec.Parameters[util.LABEL_PGBACKREST_STANZA] = "db"
	spec.Parameters[util.LABEL_PGBACKREST_DB_PATH] = "/pgdata/" + request.ToPVC
	spec.Parameters[util.LABEL_PGBACKREST_REPO_PATH] = "/backrestrepo/" + request.FromCluster + "-backups"

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	return newInstance
}

// TODO: Needed?
func RemovePgDumpJob(name string) {

	_, found := kubeapi.GetJob(apiserver.Clientset, name, apiserver.Namespace)
	if !found {
		return
	}

	log.Debugf("found backup job %s will remove\n", name)

	kubeapi.DeleteJob(apiserver.Clientset, name, apiserver.Namespace)
}
