package ns

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
	"errors"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
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

// CreateNamespace ...
func CreateNamespace(clientset *kubernetes.Clientset, installationName, pgoNamespace, createdBy, newNs string) error {

	log.Debugf("CreateNamespace %s %s %s", pgoNamespace, createdBy, newNs)

	//validate the list of args (namespaces)
	errs := validation.IsDNS1035Label(newNs)
	if len(errs) > 0 {
		return errors.New("invalid namespace name format " + errs[0] + " namespace name " + newNs)
	}

	_, found, _ := kubeapi.GetNamespace(clientset, newNs)
	if found {
		return errors.New("namespace " + newNs + " already exists on this Kube cluster")
	}

	//define the new namespace
	n := v1.Namespace{}
	n.ObjectMeta.Labels = make(map[string]string)
	n.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	n.ObjectMeta.Labels[config.LABEL_PGO_CREATED_BY] = createdBy
	n.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = installationName

	n.Name = newNs

	err := kubeapi.CreateNamespace(clientset, &n)
	if err != nil {
		return errors.New("namespace create error " + newNs + err.Error())
	}

	log.Debugf("CreateNamespace %s created by %s", newNs, createdBy)

	//apply targeted rbac rules here
	err = installTargetRBAC(clientset, pgoNamespace, newNs)
	if err != nil {
		return errors.New("namespace RBAC create error " + newNs + err.Error())
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOCreateNamespaceFormat{
		EventHeader: events.EventHeader{
			Namespace: pgoNamespace,
			Username:  createdBy,
			Topic:     topics,
			EventType: events.EventPGOCreateNamespace,
		},
		CreatedNamespace: newNs,
	}

	err = events.Publish(f)
	if err != nil {
		return err
	}

	return nil

}

// DeleteNamespace ...
func DeleteNamespace(clientset *kubernetes.Clientset, installationName, pgoNamespace, deletedBy, ns string) error {

	theNs, found, _ := kubeapi.GetNamespace(clientset, ns)
	if !found {
		return errors.New("namespace " + ns + " not found")
	}

	if theNs.ObjectMeta.Labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || theNs.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] != installationName {
		errors.New("namespace " + ns + " not owned by crunchy data or not part of Operator installation " + installationName)
	}

	err := kubeapi.DeleteNamespace(clientset, ns)
	if err != nil {
		return err
	}

	log.Debugf("DeleteNamespace %s deleted by %s", ns, deletedBy)

	//publish the namespace delete event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGODeleteNamespaceFormat{
		EventHeader: events.EventHeader{
			Namespace: pgoNamespace,
			Username:  deletedBy,
			Topic:     topics,
			EventType: events.EventPGODeleteNamespace,
		},
		DeletedNamespace: ns,
	}

	err = events.Publish(f)
	if err != nil {
		return err
	}

	return nil

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
func UpdateNamespace(clientset *kubernetes.Clientset, installationName, pgoNamespace, updatedBy, ns string) error {

	log.Debugf("UpdateNamespace %s %s %s %s", installationName, pgoNamespace, updatedBy, ns)

	theNs, found, _ := kubeapi.GetNamespace(clientset, ns)
	if !found {
		return errors.New("namespace " + ns + " doesn't exist")
	}

	if theNs.ObjectMeta.Labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || theNs.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] != installationName {
		return errors.New("namespace " + ns + " not owned by crunchy data or not part of Operator installation " + installationName)
	}

	//apply targeted rbac rules here
	err := installTargetRBAC(clientset, pgoNamespace, ns)
	if err != nil {
		return errors.New("namespace RBAC create error " + ns + err.Error())
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOCreateNamespaceFormat{
		EventHeader: events.EventHeader{
			Namespace: pgoNamespace,
			Username:  updatedBy,
			Topic:     topics,
			EventType: events.EventPGOCreateNamespace,
		},
		CreatedNamespace: ns,
	}

	err = events.Publish(f)
	if err != nil {
		return err
	}

	return nil

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
