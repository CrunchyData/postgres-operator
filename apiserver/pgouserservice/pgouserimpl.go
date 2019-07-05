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
	"k8s.io/client-go/rest"
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
func ShowPgouser(RESTClient *rest.RESTClient, request *msgs.ShowPgouserRequest) msgs.ShowPgouserResponse {
	resp := msgs.ShowPgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	return resp

}

// DeletePgouser ...
func DeletePgouser(RESTClient *rest.RESTClient, createdBy string, request *msgs.DeletePgouserRequest) msgs.DeletePgouserResponse {
	resp := msgs.DeletePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	return resp

}

func UpdatePgouser(createdBy string, request *msgs.UpdatePgouserRequest) msgs.UpdatePgouserResponse {

	resp := msgs.UpdatePgouserResponse{}
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

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
