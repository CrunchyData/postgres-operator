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
	"context"
	"errors"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/apiserver/pgouserservice"
	"github.com/crunchydata/postgres-operator/internal/config"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreatePgorole ...
func CreatePgorole(clientset kubernetes.Interface, createdBy string, request *msgs.CreatePgoroleRequest) msgs.CreatePgoroleResponse {
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

	// publish event
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
func ShowPgorole(clientset kubernetes.Interface, request *msgs.ShowPgoroleRequest) msgs.ShowPgoroleResponse {
	ctx := context.TODO()
	resp := msgs.ShowPgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.RoleInfo = make([]msgs.PgoroleInfo, 0)

	selector := config.LABEL_PGO_PGOROLE + "=true"
	if request.AllFlag {
		secrets, err := clientset.
			CoreV1().Secrets(apiserver.PgoNamespace).
			List(ctx, metav1.ListOptions{LabelSelector: selector})
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
			s, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{})

			if err != nil {
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
func DeletePgorole(clientset kubernetes.Interface, deletedBy string, request *msgs.DeletePgoroleRequest) msgs.DeletePgoroleResponse {
	ctx := context.TODO()
	resp := msgs.DeletePgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, v := range request.PgoroleName {
		secretName := "pgorole-" + v
		log.Debugf("DeletePgorole %s deleted by %s", secretName, deletedBy)

		// try to see if a secret exists for this pgorole. If it does not, continue
		// on
		if _, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{}); err != nil {
			resp.Results = append(resp.Results, secretName+" not found")
			continue
		}

		// attempt to delete the pgorole secret. if it cannot be deleted, move on
		if err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil {
			resp.Results = append(resp.Results, "error deleting secret "+secretName)
			continue
		}

		// this was successful
		resp.Results = append(resp.Results, "deleted role "+v)

		// ensure the pgorole is deleted from the various users that may have this
		// role. Though it may be odd to return at this point, this is part of the
		// legacy of this function and is kept in for those purposes
		if err := deleteRoleFromUsers(clientset, v); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	return resp
}

func UpdatePgorole(clientset kubernetes.Interface, updatedBy string, request *msgs.UpdatePgoroleRequest) msgs.UpdatePgoroleResponse {
	ctx := context.TODO()
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

	secret, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	secret.ObjectMeta.Labels[config.LABEL_PGO_UPDATED_BY] = updatedBy
	secret.Data["rolename"] = []byte(request.PgoroleName)
	secret.Data["permissions"] = []byte(request.PgorolePermissions)

	_, err = clientset.CoreV1().Secrets(apiserver.PgoNamespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// publish event
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

func createSecret(clientset kubernetes.Interface, createdBy, pgorolename, permissions string) error {
	ctx := context.TODO()

	enRolename := pgorolename

	secretName := "pgorole-" + pgorolename

	// if this secret is found (i.e. no errors returned) return here
	if _, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{}); err == nil {
		return nil
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

	_, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Create(ctx, &secret, metav1.CreateOptions{})
	return err
}

func validPermissions(perms string) error {
	var err error
	fields := strings.Split(perms, ",")

	for _, v := range fields {
		if apiserver.PermMap[strings.TrimSpace(v)] == "" && strings.TrimSpace(v) != "*" {
			return errors.New(v + " not a valid Permission")
		}
	}

	return err
}

func deleteRoleFromUsers(clientset kubernetes.Interface, roleName string) error {
	ctx := context.TODO()

	// get pgouser Secrets

	selector := config.LABEL_PGO_PGOUSER + "=true"
	pgouserSecrets, err := clientset.
		CoreV1().Secrets(apiserver.PgoNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error("could not get pgouser Secrets")
		return err
	}

	for i := range pgouserSecrets.Items {
		s := &pgouserSecrets.Items[i]
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

		// update the pgouser Secret removing any roles as necessary
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
			if _, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Update(ctx, s, metav1.UpdateOptions{}); err != nil {
				return err
			}

		}
	}
	return err
}
