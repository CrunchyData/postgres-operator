package namespaceservice

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
	"bytes"
	"encoding/json"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	//"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const PGO_ROLE = "pgo-role"
const PGO_ROLE_BINDING = "pgo-role-binding"
const PGO_BACKREST_ROLE = "pgo-backrest-role"
const PGO_BACKREST_SERVICE_ACCOUNT = "pgo-backrest"
const PGO_BACKREST_ROLE_BINDING = "pgo-backrest-role-binding"

//pgo-backrest-sa.json
type PgoBackrestServiceAccount struct {
	TargetNamespace string
}

//pgo-role-binding.json
type PgoRoleBinding struct {
	TargetNamespace      string
	PgoOperatorNamespace string
}

//pgo-backrest-role.json
type PgoBackrestRole struct {
	TargetNamespace string
}

//pgo-backrest-role-binding.json
type PgoBackrestRoleBinding struct {
	TargetNamespace string
}

//pgo-role.json
type PgoRole struct {
	TargetNamespace string
}

func ShowNamespace(clientset *kubernetes.Clientset, username string, request *msgs.ShowNamespaceRequest) msgs.ShowNamespaceResponse {
	log.Debug("ShowNamespace called")
	resp := msgs.ShowNamespaceResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	resp.Username = username
	resp.Results = make([]msgs.NamespaceResult, 0)

	//namespaceList := util.GetNamespaces()

	nsList := make([]string, 0)

	if request.AllFlag {
		namespaceList, err := kubeapi.GetNamespaces(clientset)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		for _, v := range namespaceList.Items {
			nsList = append(nsList, v.Name)
		}
	} else {
		if len(request.Args) == 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace names or --all flag is required for this command"
			return resp
		}

		for i := 0; i < len(request.Args); i++ {
			_, found, _ := kubeapi.GetNamespace(clientset, request.Args[i])
			if found == false {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "namespace " + request.Args[i] + " not found"

				return resp
			} else {
				nsList = append(nsList, request.Args[i])
			}
		}
	}

	for i := 0; i < len(nsList); i++ {
		iaccess, uaccess := apiserver.UserIsPermittedInNamespace(username, nsList[i])
		r := msgs.NamespaceResult{
			Namespace:          nsList[i],
			InstallationAccess: iaccess,
			UserAccess:         uaccess,
		}
		resp.Results = append(resp.Results, r)
	}

	return resp
}

// CreateNamespace ...
func CreateNamespace(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreateNamespaceRequest) msgs.CreateNamespaceResponse {

	log.Debugf("CreateNamespace %v", request)
	resp := msgs.CreateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	//iterate thru all the args (namespace names)
	for _, ns := range request.Args {
		//validate the list of args (namespaces)
		errs := validation.IsDNS1035Label(ns)
		if len(errs) > 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "invalid namespace name format " + errs[0] + " namespace name " + ns
			return resp
		}

		_, found, _ := kubeapi.GetNamespace(clientset, ns)
		if found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " already exists on this Kube cluster"
			return resp
		}

		//define the new namespace
		newns := v1.Namespace{}
		newns.ObjectMeta.Labels = make(map[string]string)
		newns.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
		newns.ObjectMeta.Labels[config.LABEL_PGO_CREATED_BY] = createdBy
		newns.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = apiserver.Pgo.Pgo.InstallationName

		newns.Name = ns

		err := kubeapi.CreateNamespace(clientset, &newns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace create error " + ns + err.Error()
			return resp
		}

		log.Debugf("CreateNamespace %s created by %s", ns)

		//apply targeted rbac rules here
		err = installTargetRBAC(apiserver.Clientset, apiserver.PgoNamespace, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace RBAC create error " + ns + err.Error()
			return resp
		}

		resp.Results = append(resp.Results, "created namespace "+ns)
		//publish event
		topics := make([]string, 1)
		topics[0] = events.EventTopicPGO

		f := events.EventPGOCreateNamespaceFormat{
			EventHeader: events.EventHeader{
				Namespace: apiserver.PgoNamespace,
				Username:  createdBy,
				Topic:     topics,
				EventType: events.EventPGOCreateNamespace,
			},
			CreatedNamespace: ns,
		}

		err = events.Publish(f)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	return resp

}

// DeleteNamespace ...
func DeleteNamespace(clientset *kubernetes.Clientset, deletedBy string, request *msgs.DeleteNamespaceRequest) msgs.DeleteNamespaceResponse {
	resp := msgs.DeleteNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, ns := range request.Args {

		theNs, found, _ := kubeapi.GetNamespace(clientset, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " not found"
			return resp
		}

		if theNs.ObjectMeta.Labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || theNs.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] != apiserver.Pgo.Pgo.InstallationName {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " not owned by crunchy data or not part of Operator installation " + apiserver.Pgo.Pgo.InstallationName
			return resp
		}

		err := kubeapi.DeleteNamespace(clientset, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		log.Debugf("DeleteNamespace %s deleted by %s", ns)
		resp.Results = append(resp.Results, "deleted namespace "+ns)

		//publish the namespace delete event
		topics := make([]string, 1)
		topics[0] = events.EventTopicPGO

		f := events.EventPGODeleteNamespaceFormat{
			EventHeader: events.EventHeader{
				Namespace: apiserver.PgoNamespace,
				Username:  deletedBy,
				Topic:     topics,
				EventType: events.EventPGODeleteNamespace,
			},
			DeletedNamespace: ns,
		}

		err = events.Publish(f)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

	}

	return resp

}

func installTargetRBAC(clientset *kubernetes.Clientset, operatorNamespace, targetNamespace string) error {

	err := CreatePGOBackrestServiceAccount(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}
	err = CreatePGORole(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = CreatePGORoleBinding(clientset, targetNamespace, operatorNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = CreatePGOBackrestRole(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}
	err = CreatePGOBackrestRoleBinding(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil

}

func CreatePGORoleBinding(clientset *kubernetes.Clientset, targetNamespace, operatorNamespace string) error {
	//check for rolebinding existing
	_, found, _ := kubeapi.GetRoleBinding(clientset, PGO_ROLE_BINDING, targetNamespace)
	if found {
		log.Infof("rolebinding %s already exists, will delete and re-create", PGO_ROLE_BINDING)
		err := kubeapi.DeleteRoleBinding(clientset, PGO_ROLE_BINDING, targetNamespace)
		if err != nil {
			log.Errorf("error deleting rolebinding %s %s", PGO_ROLE_BINDING, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoRoleBindingTemplate.Execute(&buffer,
		PgoRoleBinding{
			TargetNamespace:      targetNamespace,
			PgoOperatorNamespace: operatorNamespace,
		})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info(buffer.String())

	rb := rbacv1.RoleBinding{}
	err = json.Unmarshal(buffer.Bytes(), &rb)
	if err != nil {
		log.Error("error unmarshalling " + config.PGORoleBindingPath + " json RoleBinding " + err.Error())
		return err
	}

	err = kubeapi.CreateRoleBinding(clientset, &rb, targetNamespace)
	if err != nil {
		return err
	}

	return err

}

func CreatePGOBackrestRole(clientset *kubernetes.Clientset, targetNamespace string) error {
	//check for role existing
	_, found, _ := kubeapi.GetRole(clientset, PGO_BACKREST_ROLE, targetNamespace)
	if found {
		log.Infof("role %s already exists, will delete and re-create", PGO_BACKREST_ROLE)
		err := kubeapi.DeleteRole(clientset, PGO_BACKREST_ROLE, targetNamespace)
		if err != nil {
			log.Errorf("error deleting role %s %s", PGO_BACKREST_ROLE, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoBackrestRoleTemplate.Execute(&buffer,
		PgoBackrestRole{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info(buffer.String())
	r := rbacv1.Role{}
	err = json.Unmarshal(buffer.Bytes(), &r)
	if err != nil {
		log.Error("error unmarshalling " + config.PGOBackrestRolePath + " json Role " + err.Error())
		return err
	}

	err = kubeapi.CreateRole(clientset, &r, targetNamespace)
	if err != nil {
		return err
	}

	return err
}

func CreatePGORole(clientset *kubernetes.Clientset, targetNamespace string) error {
	//check for role existing
	_, found, _ := kubeapi.GetRole(clientset, PGO_ROLE, targetNamespace)
	if found {
		log.Infof("role %s already exists, will delete and re-create", PGO_ROLE)
		err := kubeapi.DeleteRole(clientset, PGO_ROLE, targetNamespace)
		if err != nil {
			log.Errorf("error deleting role %s %s", PGO_ROLE, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoRoleTemplate.Execute(&buffer,
		PgoRole{
			TargetNamespace: targetNamespace,
		})

	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info(buffer.String())
	r := rbacv1.Role{}
	err = json.Unmarshal(buffer.Bytes(), &r)
	if err != nil {
		log.Error("error unmarshalling " + config.PGORolePath + " json Role " + err.Error())
		return err
	}

	err = kubeapi.CreateRole(clientset, &r, targetNamespace)
	if err != nil {
		return err
	}
	return err
}

func CreatePGOBackrestRoleBinding(clientset *kubernetes.Clientset, targetNamespace string) error {

	//check for rolebinding existing
	_, found, _ := kubeapi.GetRoleBinding(clientset, PGO_BACKREST_ROLE_BINDING, targetNamespace)
	if found {
		log.Infof("rolebinding %s already exists, will delete and re-create", PGO_BACKREST_ROLE_BINDING)
		err := kubeapi.DeleteRoleBinding(clientset, PGO_BACKREST_ROLE_BINDING, targetNamespace)
		if err != nil {
			log.Errorf("error deleting rolebinding %s %s", PGO_BACKREST_ROLE_BINDING, err.Error())
			return err
		}
	}
	var buffer bytes.Buffer
	err := config.PgoBackrestRoleBindingTemplate.Execute(&buffer,
		PgoBackrestRoleBinding{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error() + " on " + config.PGOBackrestRoleBindingPath)
		return err
	}
	log.Info(buffer.String())

	rb := rbacv1.RoleBinding{}
	err = json.Unmarshal(buffer.Bytes(), &rb)
	if err != nil {
		log.Error("error unmarshalling " + config.PGOBackrestRoleBindingPath + " json RoleBinding " + err.Error())
		return err
	}

	err = kubeapi.CreateRoleBinding(clientset, &rb, targetNamespace)
	if err != nil {
		return err
	}
	return err
}

// UpdateNamespace ...
func UpdateNamespace(clientset *kubernetes.Clientset, updatedBy string, request *msgs.UpdateNamespaceRequest) msgs.UpdateNamespaceResponse {

	log.Debugf("UpdateNamespace %v", request)
	resp := msgs.UpdateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	//iterate thru all the args (namespace names)
	for _, ns := range request.Args {

		theNs, found, _ := kubeapi.GetNamespace(clientset, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " doesn't exist"
			return resp
		}

		if theNs.ObjectMeta.Labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || theNs.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] != apiserver.Pgo.Pgo.InstallationName {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " not owned by crunchy data or not part of Operator installation " + apiserver.Pgo.Pgo.InstallationName
			return resp
		}

		//apply targeted rbac rules here
		err := installTargetRBAC(apiserver.Clientset, apiserver.PgoNamespace, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace RBAC create error " + ns + err.Error()
			return resp
		}

		resp.Results = append(resp.Results, "updated namespace "+ns)

		//publish event
		topics := make([]string, 1)
		topics[0] = events.EventTopicPGO

		f := events.EventPGOCreateNamespaceFormat{
			EventHeader: events.EventHeader{
				Namespace: apiserver.PgoNamespace,
				Username:  updatedBy,
				Topic:     topics,
				EventType: events.EventPGOCreateNamespace,
			},
			CreatedNamespace: ns,
		}

		err = events.Publish(f)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	return resp

}

func CreatePGOBackrestServiceAccount(clientset *kubernetes.Clientset, targetNamespace string) error {

	//check for serviceaccount existing
	_, found, _ := kubeapi.GetServiceAccount(clientset, PGO_BACKREST_SERVICE_ACCOUNT, targetNamespace)
	if found {
		log.Infof("serviceaccount %s already exists, will delete and re-create", PGO_BACKREST_SERVICE_ACCOUNT)
		err := kubeapi.DeleteServiceAccount(clientset, PGO_BACKREST_SERVICE_ACCOUNT, targetNamespace)
		if err != nil {
			log.Errorf("error deleting serviceaccount %s %s", PGO_BACKREST_SERVICE_ACCOUNT, err.Error())
			return err
		}
	}
	var buffer bytes.Buffer
	err := config.PgoBackrestServiceAccountTemplate.Execute(&buffer,
		PgoBackrestServiceAccount{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error() + " on " + config.PGOBackrestServiceAccountPath)
		return err
	}
	log.Info(buffer.String())

	rb := v1.ServiceAccount{}
	err = json.Unmarshal(buffer.Bytes(), &rb)
	if err != nil {
		log.Error("error unmarshalling " + config.PGOBackrestServiceAccountPath + " json ServiceAccount " + err.Error())
		return err
	}

	err = kubeapi.CreateServiceAccount(clientset, &rb, targetNamespace)
	if err != nil {
		return err
	}
	return err
}
