package kubeapi

/*
 Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	log "github.com/sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	jsonpatch "github.com/evanphx/json-patch"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// Getpgbackups gets a list of pgbackups
func Getpgbackups(client *rest.RESTClient, backupList *crv1.PgbackupList, namespace string) error {

	err := client.Get().
		Resource(crv1.PgbackupResourcePlural).
		Namespace(namespace).
		Do().Into(backupList)
	if err != nil {
		log.Error("error getting list of backups" + err.Error())
		return err
	}

	return err
}

// Getpgbackup gets a pgbackups by name
func Getpgbackup(client *rest.RESTClient, backup *crv1.Pgbackup, name, namespace string) (bool, error) {

	err := client.Get().
		Resource(crv1.PgbackupResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(backup)
	if kerrors.IsNotFound(err) {
		return false, err
	}

	if err != nil {
		log.Error("error getting backup" + err.Error())
		return false, err
	}

	return true, err
}

// Deletepgbackup deletes pgbackup by name
func Deletepgbackup(client *rest.RESTClient, name, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgbackupResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().
		Error()
	if err != nil {
		log.Error("error deleting pgbackup " + err.Error())
		return err
	}

	return err
}

// Deletepgbackups deletes all pgbackups
func DeleteAllpgbackup(client *rest.RESTClient, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgbackupResourcePlural).
		Namespace(namespace).
		Do().Error()
	if err != nil {
		log.Error("error deleting all pgbackup" + err.Error())
		return err
	}

	return err
}

// Createpgbackup creates a pgbackup
func Createpgbackup(client *rest.RESTClient, backup *crv1.Pgbackup, namespace string) error {

	result := crv1.Pgbackup{}

	err := client.Post().
		Resource(crv1.PgbackupResourcePlural).
		Namespace(namespace).
		Body(backup).
		Do().Into(&result)
	if err != nil {
		log.Error("error creating pgbackup " + err.Error())
		return err
	}

	return err
}

func PatchpgbackupStatus(restclient *rest.RESTClient, state crv1.PgbackupState, message string, oldCrd *crv1.Pgbackup, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Status = crv1.PgbackupStatus{
		State:   state,
		Message: message,
	}

	//create the patch
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCrd)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	//apply patch
	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgbackupResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

// PatchpgbacupBackupStatus - patch the backup status
func PatchpgbackupBackupStatus(restclient *rest.RESTClient, status string, oldCrd *crv1.Pgbackup, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Spec.BackupStatus = status

	//create the patch
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCrd)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	//apply patch
	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgbackupResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}