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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// Getpgpolicies gets a list of pgpolicies
func Getpgpolicies(client *rest.RESTClient, policyList *crv1.PgpolicyList, namespace string) error {

	err := client.Get().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(namespace).
		Do().Into(policyList)
	if err != nil {
		log.Error("error getting list of policies" + err.Error())
		return err
	}

	return err
}

// Getpgpolicy gets a pgpolicies by name
func Getpgpolicy(client *rest.RESTClient, policy *crv1.Pgpolicy, name, namespace string) (bool, error) {

	err := client.Get().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(policy)
	if kerrors.IsNotFound(err) {
		return false, err
	}

	if err != nil {
		log.Error("error getting policy" + err.Error())
		return false, err
	}

	return true, err
}

// Deletepgpolicy deletes pgpolicy by name
func Deletepgpolicy(client *rest.RESTClient, name, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().
		Error()
	if err != nil {
		log.Error("error deleting pgpolicy " + err.Error())
		return err
	}

	return err
}

// Createpgpolicy creates a pgpolicy
func Createpgpolicy(client *rest.RESTClient, policy *crv1.Pgpolicy, namespace string) error {

	result := crv1.Pgpolicy{}

	err := client.Post().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(namespace).
		Body(policy).
		Do().Into(&result)
	if err != nil {
		log.Error("error creating pgpolicy " + err.Error())
		return err
	}

	log.Debugf("created pgpolicy %s", policy.Name)
	return err
}

// Updatepgpolicy
func Updatepgpolicy(client *rest.RESTClient, task *crv1.Pgpolicy, name, namespace string) error {

	err := client.Put().
		Name(name).
		Namespace(namespace).
		Resource(crv1.PgpolicyResourcePlural).
		Body(task).
		Do().
		Error()
	if err != nil {
		log.Error("error updating pgpolicy " + err.Error())
		return err
	}

	log.Debugf("updated pgpolicy %s", task.Name)
	return err
}

func PatchpgpolicyStatus(restclient *rest.RESTClient, state crv1.PgpolicyState, message string, oldCrd *crv1.Pgpolicy, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Status = crv1.PgpolicyStatus{
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
		Resource(crv1.PgpolicyResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}
