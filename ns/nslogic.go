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
	"os"
	"strings"
)

const PGO_TARGET_ROLE = "pgo-target-role"
const PGO_TARGET_ROLE_BINDING = "pgo-target-role-binding"
const PGO_TARGET_SERVICE_ACCOUNT = "pgo-target"

const PGO_BACKREST_ROLE = "pgo-backrest-role"
const PGO_BACKREST_SERVICE_ACCOUNT = "pgo-backrest"
const PGO_BACKREST_ROLE_BINDING = "pgo-backrest-role-binding"

//pgo-backrest-sa.json
type PgoBackrestServiceAccount struct {
	TargetNamespace string
}

//pgo-target-sa.json
type PgoTargetServiceAccount struct {
	TargetNamespace string
}

//pgo-target-role-binding.json
type PgoTargetRoleBinding struct {
	TargetNamespace   string
	OperatorNamespace string
}

//pgo-backrest-role.json
type PgoBackrestRole struct {
	TargetNamespace string
}

//pgo-backrest-role-binding.json
type PgoBackrestRoleBinding struct {
	TargetNamespace string
}

//pgo-target-role.json
type PgoTargetRole struct {
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

	var err error

	/**
	err := CreatePGOTargetServiceAccount(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}
	*/
	err = CreatePGOBackrestServiceAccount(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = CreatePGOTargetRole(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = CreatePGOTargetRoleBinding(clientset, targetNamespace, operatorNamespace)
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

func CreatePGOTargetRoleBinding(clientset *kubernetes.Clientset, targetNamespace, operatorNamespace string) error {
	//check for rolebinding existing
	_, found, _ := kubeapi.GetRoleBinding(clientset, PGO_TARGET_ROLE_BINDING, targetNamespace)
	if found {
		log.Infof("rolebinding %s already exists, will delete and re-create", PGO_TARGET_ROLE_BINDING)
		err := kubeapi.DeleteRoleBinding(clientset, PGO_TARGET_ROLE_BINDING, targetNamespace)
		if err != nil {
			log.Errorf("error deleting rolebinding %s %s", PGO_TARGET_ROLE_BINDING, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoTargetRoleBindingTemplate.Execute(&buffer,
		PgoTargetRoleBinding{
			TargetNamespace:   targetNamespace,
			OperatorNamespace: operatorNamespace,
		})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info(buffer.String())

	rb := rbacv1.RoleBinding{}
	err = json.Unmarshal(buffer.Bytes(), &rb)
	if err != nil {
		log.Error("error unmarshalling " + config.PGOTargetRoleBindingPath + " json RoleBinding " + err.Error())
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

func CreatePGOTargetRole(clientset *kubernetes.Clientset, targetNamespace string) error {
	//check for role existing
	_, found, _ := kubeapi.GetRole(clientset, PGO_TARGET_ROLE, targetNamespace)
	if found {
		log.Infof("role %s already exists, will delete and re-create", PGO_TARGET_ROLE)
		err := kubeapi.DeleteRole(clientset, PGO_TARGET_ROLE, targetNamespace)
		if err != nil {
			log.Errorf("error deleting role %s %s", PGO_TARGET_ROLE, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoTargetRoleTemplate.Execute(&buffer,
		PgoTargetRole{
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
		log.Error("error unmarshalling " + config.PGOTargetRolePath + " json Role " + err.Error())
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

func ValidateNamespaces(clientset *kubernetes.Clientset, installationName, pgoNamespace string) error {
	raw := os.Getenv("NAMESPACE")

	//the case of 'all' namespaces
	if raw == "" {
		return nil
	}

	allFound := false

	nsList := strings.Split(raw, ",")

	//check for the invalid case where a user has NAMESPACE=demo1,,demo2
	if len(nsList) > 1 {
		for i := 0; i < len(nsList); i++ {
			if nsList[i] == "" {
				allFound = true
			}
		}
	}

	if allFound && len(nsList) > 1 {
		return errors.New("'' (empty string), found within the NAMESPACE environment variable along with other namespaces, this is not an accepted format")
	}

	//check for the case of a non-existing namespace being used
	for i := 0; i < len(nsList); i++ {
		namespace, found, _ := kubeapi.GetNamespace(clientset, nsList[i])
		if !found {
			//return errors.New("NAMESPACE environment variable contains a namespace of " + nsList[i] + " but that is not found on this kube system")
			//create the namespace here
			err := CreateNamespace(clientset, installationName, pgoNamespace, "operator-bootstrap", nsList[i])
			if err != nil {
				return err
			}
		} else {
			//verify the namespace doesn't belong to another
			//installation, if not, update it to belong to this
			//installation
			if namespace.ObjectMeta.Labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY && namespace.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] != installationName {
				log.Errorf("%s namespace onwed by another installation, will not convert it to this installation", namespace.Name)
			} else if namespace.ObjectMeta.Labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY && namespace.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] == installationName {
				log.Infof("%s namespace already part of this installation", namespace.Name)
			} else {
				log.Infof("%s namespace will be updated to be owned by this installation", namespace.Name)
				namespace.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
				namespace.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = installationName
				err := kubeapi.UpdateNamespace(clientset, namespace)
				if err != nil {
					return err
				}
				err = UpdateNamespace(clientset, installationName, pgoNamespace, "operator-bootstrap", namespace.Name)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil

}

func GetNamespaces(clientset *kubernetes.Clientset, installationName string) []string {
	ns := make([]string, 0)

	nsList, err := kubeapi.GetNamespaces(clientset)
	if err != nil {
		log.Error(err.Error())
		return ns
	}

	for _, v := range nsList.Items {
		labels := v.ObjectMeta.Labels
		if labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY &&
			labels[config.LABEL_PGO_INSTALLATION_NAME] == installationName {
			ns = append(ns, v.Name)
		}
	}

	return ns

}

func WatchingNamespace(clientset *kubernetes.Clientset, requestedNS, installationName string) bool {

	log.Debugf("WatchingNamespace [%s]", requestedNS)

	nsList := GetNamespaces(clientset, installationName)

	//handle the case where we are watching all namespaces but
	//the user might enter an invalid namespace not on the kube
	if nsList[0] == "" {
		_, found, _ := kubeapi.GetNamespace(clientset, requestedNS)
		if !found {
			return false
		} else {
			return true
		}
	}

	for i := 0; i < len(nsList); i++ {
		if nsList[i] == requestedNS {
			return true
		}
	}

	return false
}

func CreatePGOTargetServiceAccount(clientset *kubernetes.Clientset, targetNamespace string) error {

	//check for serviceaccount existing
	_, found, _ := kubeapi.GetServiceAccount(clientset, PGO_TARGET_SERVICE_ACCOUNT, targetNamespace)
	if found {
		log.Infof("serviceaccount %s already exists, will delete and re-create", PGO_TARGET_SERVICE_ACCOUNT)
		err := kubeapi.DeleteServiceAccount(clientset, PGO_TARGET_SERVICE_ACCOUNT, targetNamespace)
		if err != nil {
			log.Errorf("error deleting serviceaccount %s %s", PGO_TARGET_SERVICE_ACCOUNT, err.Error())
			return err
		}
	}
	var buffer bytes.Buffer
	err := config.PgoTargetServiceAccountTemplate.Execute(&buffer,
		PgoTargetServiceAccount{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error() + " on " + config.PGOTargetServiceAccountPath)
		return err
	}
	log.Info(buffer.String())

	rb := v1.ServiceAccount{}
	err = json.Unmarshal(buffer.Bytes(), &rb)
	if err != nil {
		log.Error("error unmarshalling " + config.PGOTargetServiceAccountPath + " json ServiceAccount " + err.Error())
		return err
	}

	err = kubeapi.CreateServiceAccount(clientset, &rb, targetNamespace)
	if err != nil {
		return err
	}
	return err
}
