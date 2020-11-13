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
	"context"
	"errors"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/ns"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const MAP_KEY_USERNAME = "username"
const MAP_KEY_PASSWORD = "password"
const MAP_KEY_ROLES = "roles"
const MAP_KEY_NAMESPACES = "namespaces"

// CreatePgouser ...
func CreatePgouser(clientset kubernetes.Interface, createdBy string, request *msgs.CreatePgouserRequest) msgs.CreatePgouserResponse {

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

	return resp
}

// ShowPgouser ...
func ShowPgouser(clientset kubernetes.Interface, request *msgs.ShowPgouserRequest) msgs.ShowPgouserResponse {
	ctx := context.TODO()
	resp := msgs.ShowPgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	selector := config.LABEL_PGO_PGOUSER + "=true"
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

			s, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{})

			if err != nil {
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
func DeletePgouser(clientset kubernetes.Interface, deletedBy string, request *msgs.DeletePgouserRequest) msgs.DeletePgouserResponse {
	ctx := context.TODO()
	resp := msgs.DeletePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, v := range request.PgouserName {
		secretName := "pgouser-" + v
		log.Debugf("DeletePgouser %s deleted by %s", secretName, deletedBy)

		if _, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{}); err != nil {
			resp.Results = append(resp.Results, secretName+" not found")
		} else {
			err = clientset.CoreV1().Secrets(apiserver.PgoNamespace).Delete(ctx, secretName, metav1.DeleteOptions{})
			if err != nil {
				resp.Results = append(resp.Results, "error deleting secret "+secretName)
			} else {
				resp.Results = append(resp.Results, "deleted pgouser "+v)
			}
		}
	}

	return resp

}

// UpdatePgouser - update the pgouser secret
func UpdatePgouser(clientset kubernetes.Interface, updatedBy string, request *msgs.UpdatePgouserRequest) msgs.UpdatePgouserResponse {
	ctx := context.TODO()
	resp := msgs.UpdatePgouserResponse{}
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	secretName := "pgouser-" + request.PgouserName

	secret, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
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
	_, err = clientset.CoreV1().Secrets(apiserver.PgoNamespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		log.Debug("Error updating pgouser secret: ", err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp
}

func createSecret(clientset kubernetes.Interface, createdBy string, request *msgs.CreatePgouserRequest) error {
	ctx := context.TODO()
	secretName := "pgouser-" + request.PgouserName

	// if this secret is found (no errors returned), returned here
	if _, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{}); err == nil {
		return nil
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

	_, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Create(ctx, &secret, metav1.CreateOptions{})
	return err
}

func validRoles(clientset kubernetes.Interface, roles string) error {
	ctx := context.TODO()

	var err error
	fields := strings.Split(roles, ",")
	for _, v := range fields {
		r := strings.TrimSpace(v)
		secretName := "pgorole-" + r

		if _, err := clientset.CoreV1().Secrets(apiserver.PgoNamespace).Get(ctx, secretName, metav1.GetOptions{}); err != nil {
			return errors.New(v + " pgorole was not found")
		}

	}

	return err
}

func validNamespaces(namespaces string, allnamespaces bool) error {

	if allnamespaces {
		return nil
	}

	nsSlice := strings.Split(namespaces, ",")
	for i := range nsSlice {
		nsSlice[i] = strings.TrimSpace(nsSlice[i])
	}

	err := ns.ValidateNamespacesWatched(apiserver.Clientset, apiserver.NamespaceOperatingMode(),
		apiserver.InstallationName, nsSlice...)
	if err != nil {
		return err
	}

	return nil
}
