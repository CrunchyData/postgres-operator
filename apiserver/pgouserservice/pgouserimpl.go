package pgouserservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/userservice"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// CreatePgouser ...
func CreatePgouser(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreatePgouserRequest) msgs.CreatePgouserResponse {

	log.Debugf("CreatePgouser %v", request)
	resp := msgs.CreatePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	err := userservice.ValidPassword(request.PgouserPassword)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	err = createSecret(clientset, createdBy, request.PgouserName, request.PgouserPassword)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

// ShowPgouser ...
func ShowPgouser(clientset *kubernetes.Clientset, request *msgs.ShowPgouserRequest) msgs.ShowPgouserResponse {
	resp := msgs.ShowPgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if request.AllFlag {
		secrets, err := kubeapi.GetSecrets(clientset, "pgo-auth=true", apiserver.PgoNamespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		for _, s := range secrets.Items {
			resp.PgouserName = append(resp.PgouserName, s.Name)
		}
	} else {
		for _, v := range request.PgouserName {
			secretName := "pgouser-" + v
			_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
			if !found || err != nil {
				resp.PgouserName = append(resp.PgouserName, v+" was not found")
			} else {
				resp.PgouserName = append(resp.PgouserName, v)
			}
		}
	}

	return resp

}

// DeletePgouser ...
func DeletePgouser(clientset *kubernetes.Clientset, deletedBy string, request *msgs.DeletePgouserRequest) msgs.DeletePgouserResponse {
	resp := msgs.DeletePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, v := range request.PgouserName {
		secretName := "pgouser-" + v
		log.Debugf("DeletePgouser %s deleted by %s", secretName, deletedBy)
		_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
		if !found {
			resp.Results = append(resp.Results, secretName+" not found")
		} else {
			err = kubeapi.DeleteSecret(clientset, secretName, apiserver.PgoNamespace)
			if err != nil {
				resp.Results = append(resp.Results, "error deleting secret "+secretName)
			} else {
				resp.Results = append(resp.Results, "deleted secret "+secretName)
			}

		}
	}

	return resp

}

func UpdatePgouser(clientset *kubernetes.Clientset, updatedBy string, request *msgs.UpdatePgouserRequest) msgs.UpdatePgouserResponse {

	resp := msgs.UpdatePgouserResponse{}
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	err := userservice.ValidPassword(request.PgouserPassword)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	secretName := "pgouser-" + request.PgouserName

	secret, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	secret.ObjectMeta.Labels[config.LABEL_PGOUSER] = updatedBy
	secret.Data["password"] = []byte(request.PgouserPassword)

	err = kubeapi.UpdateSecret(clientset, secret, apiserver.PgoNamespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

func createSecret(clientset *kubernetes.Clientset, createdBy, pgousername, password string) error {

	var enUsername = pgousername

	secretName := "pgouser-" + pgousername

	_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
	if found {
		return err
	}

	secret := v1.Secret{}
	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels[config.LABEL_PGOUSER] = createdBy
	secret.ObjectMeta.Labels[config.LABEL_PGO_AUTH] = "true"
	secret.Data = make(map[string][]byte)
	secret.Data["username"] = []byte(enUsername)
	secret.Data["password"] = []byte(password)

	err = kubeapi.CreateSecret(clientset, &secret, apiserver.PgoNamespace)

	return err

}
