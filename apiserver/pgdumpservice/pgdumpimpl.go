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
	"strconv"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/backupoptions"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pgDumpCommand = "pgdump"
const pgDumpInfoCommand = "info"
const pgDumpTaskExtension = "-pgdump"
const pgDumpJobExtension = "-pgdump-job"

//  CreateBackup ...
// pgo backup mycluster
// pgo backup --selector=name=mycluster
func CreatepgDump(request *msgs.CreatepgDumpBackupRequest, ns string) msgs.CreatepgDumpBackupResponse {

	resp := msgs.CreatepgDumpBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	// var newInstance *crv1.Pgtask

	log.Debug("CreatePgDump storage config... " + request.StorageConfig)
	if request.StorageConfig != "" {
		if apiserver.IsValidStorageName(request.StorageConfig) == false {
			log.Debug("CreateBackup sc error is found " + request.StorageConfig)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = request.StorageConfig + " Storage config was not found "
			return resp
		}
	}

	if request.BackupOpts != "" {
		err := backupoptions.ValidateBackupOpts(request.BackupOpts, request)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		clusterList := crv1.PgclusterList{}

		err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, request.Selector, ns)
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
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, clusterName, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		RemovePgDumpJob(clusterName+pgDumpJobExtension, ns)

		result := crv1.Pgtask{}

		// error if the task already exists
		found, err = kubeapi.Getpgtask(apiserver.RESTClient, &result, taskName, ns)
		if !found {
			log.Debugf("pgdump pgtask %s was not found so we will create it", taskName)
		} else if err != nil {

			resp.Results = append(resp.Results, "error getting pgtask for "+taskName)
			break
		} else {

			log.Debugf("pgtask %s was found so we will recreate it", taskName)
			//remove the existing pgtask
			err := kubeapi.Deletepgtask(apiserver.RESTClient, taskName, ns)

			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		//get pod name from cluster
		// var podname, deployName string
		var podname string
		podname, err = getPrimaryPodName(&cluster, ns)

		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		// where all the magic happens about the task.
		// TODO: Needs error handling for invalid parameters in the request
		theTask := buildPgTaskForDump(clusterName, taskName, crv1.PgtaskpgDump, podname, "database", request)

		err = kubeapi.Createpgtask(apiserver.RESTClient, theTask, ns)
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
func ShowpgDump(clusterName string, selector string, ns string) msgs.ShowBackupResponse {
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
		&clusterList, selector, ns)
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

		backupItem, error := getPgBackupForTask(c.Name, pgTaskName, ns)

		if backupItem != nil {
			log.Debugf("pgTask %s was found", pgTaskName)
			response.BackupList.Items = append(response.BackupList.Items, *backupItem)

		} else if error != nil {
			log.Debugf("pgTask %s was not found, error", pgTaskName)
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

// builds out a pgTask structure that can be handed to kube
func buildPgTaskForDump(clusterName string, taskName string, action string, podName string,
	containerName string, request *msgs.CreatepgDumpBackupRequest) *crv1.Pgtask {

	var newInstance *crv1.Pgtask
	var storageSpec crv1.PgStorageSpec
	var pvcName string

	backupUser := clusterName + "-postgres-secret"

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

	// get dumpall flag, separate from dumpOpts, validate options
	dumpAllFlag, dumpOpts := parseOptionFlags(request.BackupOpts)

	spec := crv1.PgtaskSpec{}

	spec.Name = taskName
	spec.TaskType = crv1.PgtaskpgDump
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[config.LABEL_PGDUMP_HOST] = clusterName      // same name as service
	spec.Parameters[config.LABEL_CONTAINER_NAME] = containerName // ??
	spec.Parameters[config.LABEL_PGDUMP_COMMAND] = action
	spec.Parameters[config.LABEL_PGDUMP_OPTS] = dumpOpts
	spec.Parameters[config.LABEL_PGDUMP_DB] = "postgres"
	spec.Parameters[config.LABEL_PGDUMP_USER] = backupUser
	spec.Parameters[config.LABEL_PGDUMP_PORT] = apiserver.Pgo.Cluster.Port
	spec.Parameters[config.LABEL_PGDUMP_ALL] = strconv.FormatBool(dumpAllFlag)
	spec.Parameters[config.LABEL_PVC_NAME] = pvcName
	spec.Parameters[config.LABEL_CCP_IMAGE_TAG_KEY] = apiserver.Pgo.Cluster.CCPImageTag
	spec.StorageSpec = storageSpec

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	return newInstance
}

func getDeployName(cluster *crv1.Pgcluster, ns string) (string, error) {
	var depName string

	selector := config.LABEL_PGPOOL + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name

	deps, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
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

func getPrimaryPodName(cluster *crv1.Pgcluster, ns string) (string, error) {
	var podname string

	selector := config.LABEL_PGPOOL + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
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
	if pod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
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

// 	dumpAllFlag, dumpOpts = parseOptionFlags(request.BackupOpt)
func parseOptionFlags(allFlags string) (bool, string) {
	dumpFlag := false

	// error =

	parsedOptions := []string{}

	options := strings.Split(allFlags, " ")

	for _, token := range options {

		// handle dump flag
		if strings.Contains(token, "--dump-all") {
			dumpFlag = true
		} else {
			parsedOptions = append(parsedOptions, token)
		}

	}

	optionString := strings.Join(parsedOptions, " ")

	log.Debugf("pgdump optionFlags: %s, dumpAll: %t", optionString, dumpFlag)

	return dumpFlag, optionString

}

// if backup && err are nil, it simply wasn't found. Otherwise found or an error
func getPgBackupForTask(clusterName string, taskName string, ns string) (*crv1.Pgbackup, error) {

	var err error

	task := crv1.Pgtask{}

	var backup *crv1.Pgbackup

	spec := crv1.PgtaskSpec{}
	// spec.Name = name
	spec.TaskType = crv1.PgtaskpgDump

	found, err := kubeapi.Getpgtask(apiserver.RESTClient, &task, taskName, ns)

	if found {
		backup = buildPgBackupFrompgTask(&task)
	} else if kerrors.IsNotFound(err) {
		err = nil // not found is not really an error.
	} else if err == nil {
		// It simply does not exist
		log.Debugf("pgTask not found for requested pgdump %s", taskName)
	}

	return backup, err
}

// converts pgTask to a pgBackup structure
func buildPgBackupFrompgTask(dumpTask *crv1.Pgtask) *crv1.Pgbackup {

	backup := crv1.Pgbackup{}

	backup.ObjectMeta.CreationTimestamp = dumpTask.ObjectMeta.CreationTimestamp

	spec := dumpTask.Spec

	backup.Spec.Name = spec.Name
	backup.Spec.BackupStatus = spec.Status
	backup.Spec.CCPImageTag = spec.Parameters[config.LABEL_CCP_IMAGE_TAG_KEY]
	backup.Spec.BackupHost = spec.Parameters[config.LABEL_PGDUMP_HOST]
	backup.Spec.BackupUserSecret = spec.Parameters[config.LABEL_PGDUMP_USER]
	backup.Spec.BackupPort = spec.Parameters[config.LABEL_PGDUMP_PORT]
	backup.Spec.BackupPVC = spec.Parameters[config.LABEL_PVC_NAME]
	backup.Spec.StorageSpec.Size = dumpTask.Spec.StorageSpec.Size
	backup.Spec.StorageSpec.AccessMode = dumpTask.Spec.StorageSpec.AccessMode

	// if dump-all flag is set, prepend it to options string since it was separated out before processing.
	if spec.Parameters[config.LABEL_PGDUMP_ALL] == "true" {
		backup.Spec.BackupOpts = "--dump-all " + spec.Parameters[config.LABEL_PGDUMP_OPTS]
	} else {
		backup.Spec.BackupOpts = spec.Parameters[config.LABEL_PGDUMP_OPTS]
	}

	return &backup

}

//  Restore ...
// pgo restore mycluster --to-cluster=restored
func Restore(request *msgs.PgRestoreRequest, ns string) msgs.PgRestoreResponse {
	resp := msgs.PgRestoreResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = "Restore Not Implemented"
	resp.Results = make([]string, 0)

	taskName := "restore-" + request.FromCluster + pgDumpTaskExtension

	log.Debugf("Restore %v\n", request)

	if request.RestoreOpts != "" {
		err := backupoptions.ValidateBackupOpts(request.RestoreOpts, request)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, request.FromCluster, ns)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = request.FromCluster + " was not found, verify cluster name"
		return resp
	} else if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// pgtask := getRestoreParams(cluster)

	pgtask, err := buildPgTaskForRestore(taskName, crv1.PgtaskpgRestore, request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	existingTask := crv1.Pgtask{}

	//delete any existing pgtask with the same name
	found, err = kubeapi.Getpgtask(apiserver.RESTClient,
		&existingTask,
		pgtask.Name,
		ns)
	if found {
		log.Debugf("deleting prior pgtask %s", pgtask.Name)
		err = kubeapi.Deletepgtask(apiserver.RESTClient,
			pgtask.Name,
			ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	//create a pgtask for the restore workflow
	err = kubeapi.Createpgtask(apiserver.RESTClient,
		pgtask,
		ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, "restore performed on "+request.FromCluster+" to "+request.FromPVC+" opts="+request.RestoreOpts+" pitr-target="+request.PITRTarget)

	return resp
}

// builds out a pgTask structure that can be handed to kube
func buildPgTaskForRestore(taskName string, action string, request *msgs.PgRestoreRequest) (*crv1.Pgtask, error) {

	var newInstance *crv1.Pgtask
	var storageSpec crv1.PgStorageSpec

	backupUser := request.FromCluster + "-postgres-secret"

	spec := crv1.PgtaskSpec{}

	spec.Name = taskName
	spec.Namespace = request.Namespace
	spec.TaskType = crv1.PgtaskpgRestore
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_PGRESTORE_DB] = "postgres"
	spec.Parameters[config.LABEL_PGRESTORE_HOST] = request.FromCluster
	spec.Parameters[config.LABEL_PGRESTORE_FROM_CLUSTER] = request.FromCluster
	spec.Parameters[config.LABEL_PGRESTORE_FROM_PVC] = request.FromPVC
	spec.Parameters[config.LABEL_PGRESTORE_PITR_TARGET] = request.PITRTarget
	spec.Parameters[config.LABEL_PGRESTORE_OPTS] = request.RestoreOpts
	spec.Parameters[config.LABEL_PGRESTORE_USER] = backupUser
	spec.Parameters[config.LABEL_PGRESTORE_PITR_TARGET] = request.PITRTarget

	spec.Parameters[config.LABEL_PGRESTORE_COMMAND] = action

	spec.Parameters[config.LABEL_PGRESTORE_PORT] = apiserver.Pgo.Cluster.Port
	spec.Parameters[config.LABEL_CCP_IMAGE_TAG_KEY] = apiserver.Pgo.Cluster.CCPImageTag

	// validate & parse nodeLabel if exists
	if request.NodeLabel != "" {

		if err := apiserver.ValidateNodeLabel(request.NodeLabel); err != nil {
			return nil, err
		}

		parts := strings.Split(request.NodeLabel, "=")
		spec.Parameters[config.LABEL_NODE_LABEL_KEY] = parts[0]
		spec.Parameters[config.LABEL_NODE_LABEL_VALUE] = parts[1]

		log.Debug("Restore node labels used from user entered flag")
	}

	spec.StorageSpec = storageSpec

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	return newInstance, nil
}

// TODO: Needed?
func RemovePgDumpJob(name, ns string) {

	_, found := kubeapi.GetJob(apiserver.Clientset, name, ns)
	if !found {
		return
	}

	log.Debugf("found backup job %s will remove\n", name)

	kubeapi.DeleteJob(apiserver.Clientset, name, ns)
}
