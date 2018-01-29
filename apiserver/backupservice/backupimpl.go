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
	"github.com/crunchydata/postgres-operator/apiserver/util"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/viper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ShowBackup ...
func ShowBackup(name string) msgs.ShowBackupResponse {
	response := msgs.ShowBackupResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all backups
		err := apiserver.RESTClient.Get().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(apiserver.Namespace).
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
			Namespace(apiserver.Namespace).
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
func DeleteBackup(backupName string) msgs.DeleteBackupResponse {
	resp := msgs.DeleteBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var err error

	if backupName == "all" {
		err = apiserver.RESTClient.Delete().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(apiserver.Namespace).
			Do().
			Error()
		resp.Results = append(resp.Results, "all")
	} else {
		err = apiserver.RESTClient.Delete().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(apiserver.Namespace).
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

//  CreateBackup ...
// pgo backup mycluster
// pgo backup all
// pgo backup --selector=name=mycluster
func CreateBackup(request *msgs.CreateBackupRequest) msgs.CreateBackupResponse {
	var err error
	resp := msgs.CreateBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var newInstance *crv1.Pgbackup

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		myselector, err := labels.Parse(request.Selector)
		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		log.Debug("myselector is " + myselector.String())

		//get the clusters list
		clusterList := crv1.PgclusterList{}
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(apiserver.Namespace).
			Param("labelSelector", myselector.String()).
			//LabelsSelectorParam(myselector).
			Do().
			Into(&clusterList)
		if err != nil {
			log.Error("error getting cluster list" + err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
			resp.Status.Msg = "no clusters found"
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
		result := crv1.Pgbackup{}

		// error if it already exists
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(apiserver.Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("pgbackup " + arg + " was found so we recreate it")
			dels := make([]string, 1)
			dels[0] = arg
			DeleteBackup(arg)
		} else if kerrors.IsNotFound(err) {
			msg := "pgbackup " + arg + " not found so we will create it"
			resp.Results = append(resp.Results, "pgbackup "+msg)
		} else {
			log.Error("error getting pgbackup " + arg)
			log.Error(err.Error())
			resp.Results = append(resp.Results, "error getting pgbackup for "+arg)
			break
		}
		// Create an instance of our CRD
		newInstance, err = getBackupParams(arg)
		if err != nil {
			msg := "error creating backup for " + arg
			log.Error(err)
			resp.Results = append(resp.Results, msg)
			break
		}
		err = apiserver.RESTClient.Post().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(apiserver.Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error("error in creating Pgbackup CRD instance")
			log.Error(err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Pgbackup "+arg)

	}

	return resp
}

func getBackupParams(name string) (*crv1.Pgbackup, error) {
	var err error
	var newInstance *crv1.Pgbackup

	storageSpec := crv1.PgStorageSpec{}
	spec := crv1.PgbackupSpec{}
	spec.Name = name
	spec.StorageSpec = storageSpec
	spec.StorageSpec.Name = viper.GetString("BackupStorage.Name")
	spec.StorageSpec.AccessMode = viper.GetString("BackupStorage.AccessMode")
	spec.StorageSpec.Size = viper.GetString("BackupStorage.Size")
	spec.StorageSpec.StorageClass = viper.GetString("BackupStorage.StorageClass")
	spec.StorageSpec.StorageType = viper.GetString("BackupStorage.StorageType")
	log.Debug("JEFF in backup setting storagetype to " + spec.StorageSpec.StorageType)
	spec.StorageSpec.SupplementalGroups = viper.GetString("BackupStorage.SupplementalGroups")
	spec.StorageSpec.Fsgroup = viper.GetString("BackupStorage.Fsgroup")
	spec.CCPImageTag = viper.GetString("Cluster.CCPImageTag")
	spec.BackupStatus = "initial"
	spec.BackupHost = "basic"
	spec.BackupUser = "primaryuser"
	spec.BackupPass = "password"
	spec.BackupPort = "5432"

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(apiserver.Namespace).
		Name(name).
		Do().
		Into(&cluster)
	if err == nil {
		spec.BackupHost = cluster.Spec.Name
		spec.BackupPass, err = util.GetSecretPassword(cluster.Spec.Name, crv1.PrimarySecretSuffix, apiserver.Namespace)
		if err != nil {
			return newInstance, err
		}
		spec.BackupPort = cluster.Spec.Port
	} else if kerrors.IsNotFound(err) {
		log.Debug(name + " is not a cluster")
		return newInstance, err
	} else {
		log.Error("error getting pgcluster " + name)
		log.Error(err.Error())
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
