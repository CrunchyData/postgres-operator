package backupservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
)

// ShowBackup ...
func ShowBackup(namespace, name string) msgs.ShowBackupResponse {
	response := msgs.ShowBackupResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all backups
		err := apiserver.RESTClient.Get().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(namespace).
			Do().Into(&response.BackupList)
		if err != nil {
			log.Error("error getting list of backups" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debug("backups found len is %d\n", len(response.BackupList.Items))
	} else {
		backup := crv1.Pgbackup{}
		err := apiserver.RESTClient.Get().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(namespace).
			Name(name).
			Do().Into(&backup)
		if err != nil {
			log.Error("error getting backup" + err.Error())
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
func DeleteBackup(namespace, backupName string) msgs.DeleteBackupResponse {
	resp := msgs.DeleteBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var err error

	if backupName == "all" {
		err = apiserver.RESTClient.Delete().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(namespace).
			Do().
			Error()
		resp.Results = append(resp.Results, "all")
	} else {
		err = apiserver.RESTClient.Delete().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(namespace).
			Name(backupName).
			Do().
			Error()
		resp.Results = append(resp.Results, backupName)
	}

	if err != nil {
		log.Error("error deleting pgbackup ")
		log.Error(err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	return resp

}
