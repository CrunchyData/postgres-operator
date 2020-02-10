package pgoroleservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/pgouserservice"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

// CreatePgorole ...
func CreatePgorole(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreatePgoroleRequest) msgs.CreatePgoroleResponse {

	log.Debugf("CreatePgorole %v", request)
	resp := msgs.CreatePgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	err := validPermissions(request.PgorolePermissions)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	err = createSecret(clientset, createdBy, request.PgoroleName, request.PgorolePermissions)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGOCreateRoleFormat{
		EventHeader: events.EventHeader{
			Namespace: apiserver.PgoNamespace,
			Username:  createdBy,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPGOCreateRole,
		},
		CreatedRolename: request.PgoroleName,
	}

	err = events.Publish(f)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

// ShowPgorole ...
func ShowPgorole(clientset *kubernetes.Clientset, request *msgs.ShowPgoroleRequest) msgs.ShowPgoroleResponse {
	resp := msgs.ShowPgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.RoleInfo = make([]msgs.PgoroleInfo, 0)

	selector := config.LABEL_PGO_PGOROLE + "=true"
	if request.AllFlag {
		secrets, err := kubeapi.GetSecrets(clientset, selector, apiserver.PgoNamespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		for _, s := range secrets.Items {
			info := msgs.PgoroleInfo{}
			info.Name = s.ObjectMeta.Labels[config.LABEL_ROLENAME]
			info.Permissions = string(s.Data["permissions"])
			resp.RoleInfo = append(resp.RoleInfo, info)
		}
	} else {
		for _, v := range request.PgoroleName {
			info := msgs.PgoroleInfo{}
			secretName := "pgorole-" + v
			s, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
			if !found || err != nil {
				info.Name = v + " was not found"
				info.Permissions = ""
			} else {
				info.Name = v
				info.Permissions = string(s.Data["permissions"])
			}
			resp.RoleInfo = append(resp.RoleInfo, info)
		}
	}

	return resp

}

// DeletePgorole ...
func DeletePgorole(clientset *kubernetes.Clientset, deletedBy string, request *msgs.DeletePgoroleRequest) msgs.DeletePgoroleResponse {
	resp := msgs.DeletePgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, v := range request.PgoroleName {
		secretName := "pgorole-" + v
		log.Debugf("DeletePgorole %s deleted by %s", secretName, deletedBy)
		_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
		if !found {
			resp.Results = append(resp.Results, secretName+" not found")
		} else {
			err = kubeapi.DeleteSecret(clientset, secretName, apiserver.PgoNamespace)
			if err != nil {
				resp.Results = append(resp.Results, "error deleting secret "+secretName)
			} else {
				resp.Results = append(resp.Results, "deleted role "+v)
				//publish event
				topics := make([]string, 1)
				topics[0] = events.EventTopicPGOUser

				f := events.EventPGODeleteRoleFormat{
					EventHeader: events.EventHeader{
						Namespace: apiserver.PgoNamespace,
						Username:  deletedBy,
						Topic:     topics,
						Timestamp: time.Now(),
						EventType: events.EventPGODeleteRole,
					},
					DeletedRolename: v,
				}

				err = events.Publish(f)
				if err != nil {
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}

				//delete this role from all pgousers
				err = deleteRoleFromUsers(clientset, v)
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

func UpdatePgorole(clientset *kubernetes.Clientset, updatedBy string, request *msgs.UpdatePgoroleRequest) msgs.UpdatePgoroleResponse {

	resp := msgs.UpdatePgoroleResponse{}
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	err := validPermissions(request.PgorolePermissions)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	secretName := "pgorole-" + request.PgoroleName

	secret, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	secret.ObjectMeta.Labels[config.LABEL_PGO_UPDATED_BY] = updatedBy
	secret.Data["rolename"] = []byte(request.PgoroleName)
	secret.Data["permissions"] = []byte(request.PgorolePermissions)

	err = kubeapi.UpdateSecret(clientset, secret, apiserver.PgoNamespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGOUpdateRoleFormat{
		EventHeader: events.EventHeader{
			Namespace: apiserver.PgoNamespace,
			Username:  updatedBy,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPGOUpdateRole,
		},
		UpdatedRolename: request.PgoroleName,
	}

	err = events.Publish(f)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

func createSecret(clientset *kubernetes.Clientset, createdBy, pgorolename, permissions string) error {

	var enRolename = pgorolename

	secretName := "pgorole-" + pgorolename

	_, found, err := kubeapi.GetSecret(clientset, secretName, apiserver.PgoNamespace)
	if found {
		return err
	}

	secret := v1.Secret{}
	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels[config.LABEL_PGO_CREATED_BY] = createdBy
	secret.ObjectMeta.Labels[config.LABEL_ROLENAME] = pgorolename
	secret.ObjectMeta.Labels[config.LABEL_PGO_PGOROLE] = "true"
	secret.ObjectMeta.Labels[config.LABEL_VENDOR] = "crunchydata"
	secret.Data = make(map[string][]byte)
	secret.Data["rolename"] = []byte(enRolename)
	secret.Data["permissions"] = []byte(permissions)

	err = kubeapi.CreateSecret(clientset, &secret, apiserver.PgoNamespace)

	return err

}

func validPermissions(perms string) error {
	var err error
	fields := strings.Split(perms, ",")

	for _, v := range fields {
		if apiserver.PermMap[strings.TrimSpace(v)] == "" {
			return errors.New(v + " not a valid Permission")
		}
	}

	return err
}

func deleteRoleFromUsers(clientset *kubernetes.Clientset, roleName string) error {

	//get pgouser Secrets

	selector := config.LABEL_PGO_PGOUSER + "=true"
	pgouserSecrets, err := kubeapi.GetSecrets(clientset, selector, apiserver.PgoNamespace)
	if err != nil {
		log.Error("could not get pgouser Secrets")
		return err
	}

	for _, s := range pgouserSecrets.Items {
		rolesString := string(s.Data[pgouserservice.MAP_KEY_ROLES])
		roles := strings.Split(rolesString, ",")
		resultRoles := make([]string, 0)

		var rolesUpdated bool
		for _, r := range roles {
			if r != roleName {
				resultRoles = append(resultRoles, r)
			} else {
				rolesUpdated = true
			}
		}

		//update the pgouser Secret removing any roles as necessary
		if rolesUpdated {
			var resultingRoleString string

			for i := 0; i < len(resultRoles); i++ {
				if i == len(resultRoles)-1 {
					resultingRoleString = resultingRoleString + resultRoles[i]
				} else {
					resultingRoleString = resultingRoleString + resultRoles[i] + ","
				}
			}

			s.Data[pgouserservice.MAP_KEY_ROLES] = []byte(resultingRoleString)
			err = kubeapi.UpdateSecret(clientset, &s, apiserver.PgoNamespace)
			if err != nil {
				return err
			}

		}
	}
	return err
}
