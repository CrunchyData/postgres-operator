package pgdumpservice

/*
Copyright 2019 - 2023 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strconv"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/apiserver/backupoptions"
	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pgDumpTaskExtension = "-pgdump"
	pgDumpJobExtension  = "-pgdump-job"
)

//  CreateBackup ...
// pgo backup mycluster
// pgo backup --selector=name=mycluster
func CreatepgDump(request *msgs.CreatepgDumpBackupRequest, ns string) msgs.CreatepgDumpBackupResponse {
	ctx := context.TODO()
	resp := msgs.CreatepgDumpBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	// var newInstance *crv1.Pgtask

	log.Debug("CreatePgDump storage config... " + request.StorageConfig)
	if request.StorageConfig != "" {
		if !apiserver.IsValidStorageName(request.StorageConfig) {
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
		// use the selector instead of an argument list to filter on

		clusterList, err := apiserver.Clientset.
			CrunchydataV1().Pgclusters(ns).
			List(ctx, metav1.ListOptions{LabelSelector: request.Selector})
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

		cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, clusterName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = cluster.Name + msgs.UpgradeError
			return resp
		}

		deletePropagation := metav1.DeletePropagationForeground
		_ = apiserver.Clientset.
			BatchV1().Jobs(ns).
			Delete(ctx, clusterName+pgDumpJobExtension, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})

		// error if the task already exists
		_, err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Get(ctx, taskName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			log.Debugf("pgdump pgtask %s was not found so we will create it", taskName)
		} else if err != nil {

			resp.Results = append(resp.Results, "error getting pgtask for "+taskName)
			break
		} else {

			log.Debugf("pgtask %s was found so we will recreate it", taskName)
			// remove the existing pgtask
			err := apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Delete(ctx, taskName, metav1.DeleteOptions{})
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		// where all the magic happens about the task.
		// TODO: Needs error handling for invalid parameters in the request
		theTask := buildPgTaskForDump(clusterName, taskName, crv1.PgtaskpgDump, "database", request)

		_, err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, theTask, metav1.CreateOptions{})
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
	ctx := context.TODO()
	var err error

	response := msgs.ShowBackupResponse{
		BackupList: msgs.PgbackupList{
			Items: []msgs.Pgbackup{},
		},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	if selector == "" && clusterName == "all" {
		// leave selector empty, retrieves all clusters.
	} else {
		if selector == "" {
			selector = "name=" + clusterName
		}
	}

	// get a list of all clusters
	clusterList, err := apiserver.Clientset.
		CrunchydataV1().Pgclusters(ns).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
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

		backupItem, error := getPgBackupForTask(pgTaskName, ns)

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
			response.Status.Msg = fmt.Sprintf("pgDump %s not found.", pgTaskName)
		}

	}

	return response
}

// builds out a pgTask structure that can be handed to kube
func buildPgTaskForDump(clusterName, taskName, action, containerName string,
	request *msgs.CreatepgDumpBackupRequest) *crv1.Pgtask {
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
		// Set the default PVC name using the pgcluster name and the
		// database name. For example, a pgcluster 'mycluster' with
		// a databsae 'postgres' would have a PVC named
		// backup-mycluster-pgdump-postgres-pvc
		pvcName = taskName + "-" + request.PGDumpDB + "-pvc"
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
	spec.Parameters[config.LABEL_PGDUMP_DB] = request.PGDumpDB
	spec.Parameters[config.LABEL_PGDUMP_USER] = backupUser
	spec.Parameters[config.LABEL_PGDUMP_PORT] = apiserver.Pgo.Cluster.Port
	spec.Parameters[config.LABEL_PGDUMP_ALL] = strconv.FormatBool(dumpAllFlag)
	spec.Parameters[config.LABEL_PVC_NAME] = pvcName
	spec.Parameters[config.LABEL_CCP_IMAGE_TAG_KEY] = apiserver.Pgo.Cluster.CCPImageTag
	spec.StorageSpec = storageSpec

	newInstance = &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	return newInstance
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
func getPgBackupForTask(taskName, ns string) (*msgs.Pgbackup, error) {
	ctx := context.TODO()
	task, err := apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Get(ctx, taskName, metav1.GetOptions{})

	if err == nil {
		return buildPgBackupFrompgTask(task), nil
	} else if kerrors.IsNotFound(err) {
		// keeping in this weird old logic for the moment
		return nil, nil
	}

	return nil, err
}

// converts pgTask to a pgBackup structure
func buildPgBackupFrompgTask(dumpTask *crv1.Pgtask) *msgs.Pgbackup {
	backup := msgs.Pgbackup{}

	spec := dumpTask.Spec

	backup.Name = spec.Name
	backup.CreationTimestamp = dumpTask.ObjectMeta.CreationTimestamp.String()
	backup.BackupStatus = spec.Status
	backup.CCPImageTag = spec.Parameters[config.LABEL_CCP_IMAGE_TAG_KEY]
	backup.BackupHost = spec.Parameters[config.LABEL_PGDUMP_HOST]
	backup.BackupUserSecret = spec.Parameters[config.LABEL_PGDUMP_USER]
	backup.BackupPort = spec.Parameters[config.LABEL_PGDUMP_PORT]
	backup.BackupPVC = spec.Parameters[config.LABEL_PVC_NAME]
	backup.StorageSpec.Size = dumpTask.Spec.StorageSpec.Size
	backup.StorageSpec.AccessMode = dumpTask.Spec.StorageSpec.AccessMode

	// if dump-all flag is set, prepend it to options string since it was separated out before processing.
	if spec.Parameters[config.LABEL_PGDUMP_ALL] == "true" {
		backup.BackupOpts = "--dump-all " + spec.Parameters[config.LABEL_PGDUMP_OPTS]
	} else {
		backup.BackupOpts = spec.Parameters[config.LABEL_PGDUMP_OPTS]
	}

	return &backup
}

//  Restore ...
// pgo restore mycluster --to-cluster=restored
func Restore(request *msgs.PgRestoreRequest, ns string) msgs.PgRestoreResponse {
	ctx := context.TODO()
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

	_, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, request.FromCluster, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = request.FromCluster + " was not found, verify cluster name"
		return resp
	} else if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	_, err = apiserver.Clientset.CoreV1().PersistentVolumeClaims(ns).Get(ctx, request.FromPVC, metav1.GetOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	pgtask, err := buildPgTaskForRestore(taskName, crv1.PgtaskpgRestore, request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// delete any existing pgtask with the same name
	err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Delete(ctx, pgtask.Name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// create a pgtask for the restore workflow
	_, err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, pgtask, metav1.CreateOptions{})
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
	spec.Parameters[config.LABEL_PGRESTORE_DB] = request.PGDumpDB
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

		// determine if any special node affinity type must be set
		spec.Parameters[config.LABEL_NODE_AFFINITY_TYPE] = "preferred"
		if request.NodeAffinityType == crv1.NodeAffinityTypeRequired {
			spec.Parameters[config.LABEL_NODE_AFFINITY_TYPE] = "required"
		}

		log.Debug("Restore node labels used from user entered flag")
	}

	spec.StorageSpec = storageSpec

	newInstance = &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	return newInstance, nil
}
