package pgouserservice

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

const MAP_KEY_USERNAME = "username"
const MAP_KEY_PASSWORD = "password"
const MAP_KEY_ROLES = "roles"
const MAP_KEY_NAMESPACES = "namespaces"

// CreatePgouser ...
func CreatePgouser(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreatePgouserRequest) msgs.CreatePgouserResponse {

	log.Debugf("CreatePgouser %v", request)
	resp := msgs.CreatePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	err := validRoles(clientset, request.PgouserRoles)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}
	err = validNamespaces(request.PgouserNamespaces, request.AllNamespaces)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	err = createSecret(clientset, createdBy, request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if request.AllNamespaces && request.PgouserNamespaces != "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "--all-namespaces and --pgouser-namespaces are mutually exclusive."
		return resp
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGOCreateUserFormat{
		EventHeader: events.EventHeader{
			Namespace: apiserver.PgoNamespace,
			Username:  createdBy,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPGOCreateUser,
		},
		CreatedUsername: request.PgouserName,
	}

	err = events.Publish(f)
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

	selector := config.LABEL_PGO_PGOUSER + "=true"
	if request.AllFlag {
		secrets, err := kubeapi.GetSecrets(clientset, selector, apiserver.PgoNamespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		for _, s := range secrets.Items {
			info := msgs.PgouserInfo{}
			info.Username = s.ObjectMeta.Labels[config.LABEL_USERNAME]
			info.Role = make([]string, 0)
			info.Role = append(info.Role, string(s.Data[MAP_KEY_ROLES]))
			info.Namespace = make([]string, 0)
			info.Namespace = append(info.Namespace, string(s.Data[MAP_KEY_NAMESPACES]))

			resp.UserInfo = append(resp.UserInfo, info)
		}
	} else {
		for _, v := range request.PgouserName {
			secretName := "pgouser-" + v

			info := msgs.PgouserInfo{}
			info.Username = v
			info.Role = make([]string, 0)
			info.Namespace = make([]string, 0)

			s, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
			if !found || err != nil {
				info.Username = v + " was not found"
			} else {
				info.Username = v
				info.Role = append(info.Role, string(s.Data[MAP_KEY_ROLES]))
				info.Namespace = append(info.Namespace, string(s.Data[MAP_KEY_NAMESPACES]))
			}
			resp.UserInfo = append(resp.UserInfo, info)
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
				resp.Results = append(resp.Results, "deleted pgouser "+v)
				//publish event
				topics := make([]string, 1)
				topics[0] = events.EventTopicPGOUser

				f := events.EventPGODeleteUserFormat{
					EventHeader: events.EventHeader{
						Namespace: apiserver.PgoNamespace,
						Username:  deletedBy,
						Topic:     topics,
						Timestamp: time.Now(),
						EventType: events.EventPGODeleteUser,
					},
					DeletedUsername: v,
				}

				err = events.Publish(f)
				if err != nil {
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}

			}

		}
	}

	return resp

}

// UpdatePgouser - update the pgouser secret
func UpdatePgouser(clientset *kubernetes.Clientset, updatedBy string, request *msgs.UpdatePgouserRequest) msgs.UpdatePgouserResponse {

	resp := msgs.UpdatePgouserResponse{}
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	secretName := "pgouser-" + request.PgouserName

	secret, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	secret.ObjectMeta.Labels[config.LABEL_PGO_UPDATED_BY] = updatedBy
	secret.Data[MAP_KEY_USERNAME] = []byte(request.PgouserName)

	if request.PgouserPassword != "" {
		secret.Data[MAP_KEY_PASSWORD] = []byte(request.PgouserPassword)
	}
	if request.PgouserRoles != "" {
		err = validRoles(clientset, request.PgouserRoles)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		secret.Data[MAP_KEY_ROLES] = []byte(request.PgouserRoles)
	}
	if request.PgouserNamespaces != "" {
		err = validNamespaces(request.PgouserNamespaces, request.AllNamespaces)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		secret.Data[MAP_KEY_NAMESPACES] = []byte(request.PgouserNamespaces)
	} else if request.AllNamespaces {
		secret.Data[MAP_KEY_NAMESPACES] = []byte("")
	}

	log.Info("Updating secret for: ", request.PgouserName)
	err = kubeapi.UpdateSecret(clientset, secret, apiserver.PgoNamespace)
	if err != nil {
		log.Debug("Error updating pgouser secret: ", err.Error)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGOUpdateUserFormat{
		EventHeader: events.EventHeader{
			Namespace: apiserver.PgoNamespace,
			Username:  updatedBy,
			Topic:     topics,
			EventType: events.EventPGOUpdateUser,
		},
		UpdatedUsername: request.PgouserName,
	}

	err = events.Publish(f)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

func createSecret(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreatePgouserRequest) error {

	secretName := "pgouser-" + request.PgouserName

	_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
	if found {
		return err
	}

	secret := v1.Secret{}
	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels[config.LABEL_PGO_CREATED_BY] = createdBy
	secret.ObjectMeta.Labels[config.LABEL_USERNAME] = request.PgouserName
	secret.ObjectMeta.Labels[config.LABEL_PGO_PGOUSER] = "true"
	secret.ObjectMeta.Labels[config.LABEL_VENDOR] = "crunchydata"
	secret.Data = make(map[string][]byte)
	secret.Data[MAP_KEY_USERNAME] = []byte(request.PgouserName)
	secret.Data[MAP_KEY_ROLES] = []byte(request.PgouserRoles)
	secret.Data[MAP_KEY_NAMESPACES] = []byte(request.PgouserNamespaces)
	secret.Data[MAP_KEY_PASSWORD] = []byte(request.PgouserPassword)

	err = kubeapi.CreateSecret(clientset, &secret, apiserver.PgoNamespace)

	return err

}

func validRoles(clientset *kubernetes.Clientset, roles string) error {
	var err error
	fields := strings.Split(roles, ",")
	for _, v := range fields {
		r := strings.TrimSpace(v)
		secretName := "pgorole-" + r
		_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
		if !found || err != nil {
			return errors.New(v + " pgorole was not found")
		}

	}

	return err
}

func validNamespaces(namespaces string, allnamespaces bool) error {

	var err error

	if allnamespaces {
		return err
	}

	watchedNamespaces := ns.GetNamespaces(apiserver.Clientset, apiserver.InstallationName)

	fields := strings.Split(namespaces, ",")
	for _, v := range fields {
		ns := strings.TrimSpace(v)

		found := false
		for i := 0; i < len(watchedNamespaces); i++ {
			if watchedNamespaces[i] == ns {
				found = true
				break
			}
		}
		if !found {
			return errors.New(fmt.Sprintf("%s was not found in the watched namespaces %v", ns, watchedNamespaces))
		}

	}
	return err
}
