package backupservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShowBackup ...
func ShowBackup(name string) msgs.ShowBackupResponse {
	var err error
	response := msgs.ShowBackupResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all backups
		err = kubeapi.Getpgbackups(apiserver.RESTClient, &response.BackupList, apiserver.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("backups found len is %d\n", len(response.BackupList.Items))
	} else {
		backup := crv1.Pgbackup{}
		_, err := kubeapi.Getpgbackup(apiserver.RESTClient, &backup, name, apiserver.Namespace)
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
func DeleteBackup(backupName string) msgs.DeleteBackupResponse {
	resp := msgs.DeleteBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var err error

	if backupName == "all" {
		err = kubeapi.DeleteAllpgbackup(apiserver.RESTClient, apiserver.Namespace)
		resp.Results = append(resp.Results, "all")
	} else {
		err = kubeapi.Deletepgbackup(apiserver.RESTClient, backupName, apiserver.Namespace)
		resp.Results = append(resp.Results, backupName)
	}

	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	return resp

}

//  CreateBackup ...
// pgo backup mycluster
// pgo backup all
// pgo backup --selector=name=mycluster
func CreateBackup(request *msgs.CreateBackupRequest) msgs.CreateBackupResponse {
	resp := msgs.CreateBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var newInstance *crv1.Pgbackup

	log.Info("CreateBackup sc " + request.StorageConfig)
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

	for _, arg := range request.Args {
		log.Debug("create backup called for " + arg)

		cluster := crv1.Pgcluster{}
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, arg, apiserver.Namespace)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = arg + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		//remove any existing backup job
		RemoveBackupJob("backup-" + arg)

		result := crv1.Pgbackup{}

		// error if it already exists
		found, err = kubeapi.Getpgbackup(apiserver.RESTClient, &result, arg, apiserver.Namespace)
		if !found {
			log.Debug("pgbackup " + arg + " not found so we create it")
		} else if err != nil {
			resp.Results = append(resp.Results, "error getting pgbackup for "+arg)
			break
		} else {
			log.Debug("pgbackup " + arg + " was found so we recreate it")
			dels := make([]string, 1)
			dels[0] = arg
			DeleteBackup(arg)
		}

		// Create an instance of our CRD
		newInstance, err = getBackupParams(arg, request.StorageConfig)
		if err != nil {
			msg := "error creating backup for " + arg
			log.Error(err)
			resp.Results = append(resp.Results, msg)
			break
		}

		err = kubeapi.Createpgbackup(apiserver.RESTClient, newInstance, apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Pgbackup "+arg)

	}

	return resp
}

func getBackupParams(name, storageConfig string) (*crv1.Pgbackup, error) {
	var err error
	var newInstance *crv1.Pgbackup

	spec := crv1.PgbackupSpec{}
	spec.Name = name
	if storageConfig != "" {
		spec.StorageSpec, _ = apiserver.Pgo.GetStorageSpec(storageConfig)
	} else {
		spec.StorageSpec, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.BackupStorage)
	}
	spec.CCPImageTag = apiserver.Pgo.Cluster.CCPImageTag
	spec.BackupStatus = "initial"
	spec.BackupHost = "basic"
	spec.BackupUser = "primaryuser"
	spec.BackupPass = "password"
	spec.BackupPort = "5432"

	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, name, apiserver.Namespace)
	if err == nil {
		spec.BackupHost = cluster.Spec.Name
		spec.BackupPass, err = util.GetSecretPassword(apiserver.Clientset, cluster.Spec.Name, crv1.PrimarySecretSuffix, apiserver.Namespace)
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

func RemoveBackupJob(name string) {

	_, found := kubeapi.GetJob(apiserver.Clientset, name, apiserver.Namespace)
	if !found {
		return
	}

	log.Debugf("found backup job %s will remove\n", name)

	kubeapi.DeleteJob(apiserver.Clientset, name, apiserver.Namespace)
}
