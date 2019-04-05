package backupservice

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/backupoptions"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// ShowBackup ...
func ShowBackup(name, ns string) msgs.ShowBackupResponse {
	var err error
	response := msgs.ShowBackupResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all backups
		err = kubeapi.Getpgbackups(apiserver.RESTClient, &response.BackupList, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("backups found len is %d\n", len(response.BackupList.Items))
	} else {
		backup := crv1.Pgbackup{}
		found, err := kubeapi.Getpgbackup(apiserver.RESTClient, &backup, name, ns)
		if !found {
			response.Status.Code = msgs.Error
			response.Status.Msg = "backup not found for " + name
			return response
		}
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.BackupList.Items = make([]crv1.Pgbackup, 1)
		response.BackupList.Items[0] = backup
	}

	return response

}

// DeleteBackup ...
func DeleteBackup(backupName, ns string) msgs.DeleteBackupResponse {
	resp := msgs.DeleteBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var err error

	if backupName == "all" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "all not a valid cluster name"
		return resp
	}

	err = kubeapi.Deletepgbackup(apiserver.RESTClient, backupName, ns)

	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}
	resp.Results = append(resp.Results, backupName)

	//create a pgtask to remove the PVC and its data
	pvcName := backupName + "-backup"
	dataRoots := []string{backupName + "-backups"}

	storageSpec := crv1.PgStorageSpec{}
	err = apiserver.CreateRMDataTask(storageSpec, backupName, pvcName, dataRoots, backupName+"-backup", ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

//  CreateBackup ...
// pgo backup mycluster
// pgo backup all
// pgo backup --selector=name=mycluster
func CreateBackup(request *msgs.CreateBackupRequest, ns string) msgs.CreateBackupResponse {
	resp := msgs.CreateBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var newInstance *crv1.Pgbackup
	var wfId string

	log.Info("CreateBackup sc " + request.StorageConfig)
	if request.StorageConfig != "" {
		if apiserver.IsValidStorageName(request.StorageConfig) == false {
			log.Info("CreateBackup sc error is found " + request.StorageConfig)
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

	for _, arg := range request.Args {
		log.Debugf("create backup called for %s", arg)

		cluster := crv1.Pgcluster{}
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, arg, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = arg + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		result := crv1.Pgbackup{}

		wfId, err = createBackupWorkflowTask(cluster.Spec.Name, ns)

		found, err = kubeapi.Getpgbackup(apiserver.RESTClient, &result, arg, ns)
		if !found {
			log.Debugf("pgbackup %s was not found so we will create it", arg)
			// Create an instance of our CRD
			newInstance, err = getBackupParams(arg, request, ns)
			if err != nil {
				msg := "error creating backup for " + arg
				log.Error(err)
				resp.Results = append(resp.Results, msg)
				break
			}
			if request.PVCName != "" {
				log.Debugf("backuppvc is %s", request.PVCName)
				newInstance.Spec.BackupPVC = request.PVCName
			}

			log.Debugf("CreateBackup BackupOpts=%s", newInstance.Spec.BackupOpts)

			err = kubeapi.Createpgbackup(apiserver.RESTClient, newInstance, ns)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		} else if err != nil {
			resp.Results = append(resp.Results, "error getting pgbackup for "+arg)
			break
		} else {
			log.Debugf("pgbackup %s was found so we will update it with a re-add status", arg)
			result.Spec.BackupStatus = crv1.PgBackupJobReSubmitted
			result.Spec.BackupOpts = request.BackupOpts

			err = kubeapi.Updatepgbackup(apiserver.RESTClient, &result, arg, ns)

			if err != nil {
				log.Error(err)
				resp.Results = append(resp.Results, "error updating pgbackup for "+arg)
				break
			}
		}

		resp.Results = append(resp.Results, "created backup Job for "+arg)

		resp.Results = append(resp.Results, "workflow id "+wfId)

	}

	return resp
}

func getBackupParams(name string, request *msgs.CreateBackupRequest, ns string) (*crv1.Pgbackup, error) {
	var err error
	var newInstance *crv1.Pgbackup

	spec := crv1.PgbackupSpec{}
	spec.Namespace = ns
	spec.Name = name
	if request.StorageConfig != "" {
		spec.StorageSpec, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
	} else {
		spec.StorageSpec, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.BackupStorage)
	}
	spec.CCPImageTag = apiserver.Pgo.Cluster.CCPImageTag
	spec.BackupStatus = "initial"
	spec.BackupHost = "basic"
	spec.BackupUserSecret = "primaryuser"
	spec.BackupPort = apiserver.Pgo.Cluster.Port
	spec.BackupOpts = request.BackupOpts
	spec.Toc = make(map[string]string)

	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, name, ns)
	if err == nil {
		spec.BackupHost = cluster.Spec.Name
		spec.BackupUserSecret = cluster.Spec.Name + crv1.PrimarySecretSuffix
		_, err = util.GetSecretPassword(apiserver.Clientset, cluster.Spec.Name, crv1.PrimarySecretSuffix, ns)
		if err != nil {
			return newInstance, err
		}
		spec.BackupPort = cluster.Spec.Port
	} else {
		return newInstance, err
	}

	newInstance = &crv1.Pgbackup{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, nil
}

func createBackupWorkflowTask(clusterName, ns string) (string, error) {

	existingTask := crv1.Pgtask{}

	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackupType

	//delete any existing pgtask with the same name
	found, err := kubeapi.Getpgtask(apiserver.RESTClient,
		&existingTask, taskName, ns)
	if found {
		log.Debugf("deleting prior pgtask %s", taskName)
		err = kubeapi.Deletepgtask(apiserver.RESTClient, taskName, ns)
		if err != nil {
			return "", err
		}
	}

	//create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = clusterName + "-" + crv1.PgtaskWorkflowBackupType
	spec.TaskType = crv1.PgtaskWorkflow

	spec.Parameters = make(map[string]string)
	spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format("2006-01-02.15.04.05")
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
		return "", err
	}

	log.Debugf("Backup workflow id: %s", u)

	spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	err = kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err

}
