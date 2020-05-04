package ns

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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"

	log "github.com/sirupsen/logrus"
	authorizationapi "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const OPERATOR_SERVICE_ACCOUNT = "postgres-operator"
const PGO_DEFAULT_SERVICE_ACCOUNT = "pgo-default"

const PGO_TARGET_ROLE = "pgo-target-role"
const PGO_TARGET_ROLE_BINDING = "pgo-target-role-binding"
const PGO_TARGET_SERVICE_ACCOUNT = "pgo-target"

const PGO_BACKREST_ROLE = "pgo-backrest-role"
const PGO_BACKREST_SERVICE_ACCOUNT = "pgo-backrest"
const PGO_BACKREST_ROLE_BINDING = "pgo-backrest-role-binding"

const PGO_PG_ROLE = "pgo-pg-role"
const PGO_PG_ROLE_BINDING = "pgo-pg-role-binding"
const PGO_PG_SERVICE_ACCOUNT = "pgo-pg"

//pgo-default-sa.json
type PgoDefaultServiceAccount struct {
	TargetNamespace string
}

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

//pgo-pg-sa.json
type PgoPgServiceAccount struct {
	TargetNamespace string
}

//pgo-pg-role.json
type PgoPgRole struct {
	TargetNamespace string
}

//pgo-pg-role-binding.json
type PgoPgRoleBinding struct {
	TargetNamespace string
}

// NamespaceOperatingMode defines the different namespace operating modes for the Operator
type NamespaceOperatingMode string

const (
	// NamespaceOperatingModeDynamic enables full dynamic namespace capabilities, in which the
	// Operator can create, delete and update any namespaces within the Kubernetes cluster, while
	// then also having the ability to create the roles, role bindings and service accounts within
	// those namespaces as required for the Operator to create PG clusters.  Additionally, while in
	// this mode the Operator can listen for namespace events (e.g. namespace additions, updates
	// and deletions), and then create or remove controllers for various namespaces as those
	// namespaces are added or removed from the Kubernetes cluster.
	NamespaceOperatingModeDynamic NamespaceOperatingMode = "dynamic"
	// NamespaceOperatingModeReadOnly allows the Operator to listen for namespace events within the
	// Kubernetetes cluster, and then create and run and/or remove controllers as namespaces are
	// added and deleted.  However, while in this mode the Operator is unable to create, delete or
	// update namespaces, nor can it create the RBAC it requires in any of those namespaces to
	// create PG clusters.  Therefore,  while in a "readonly" mode namespaces must be
	// pre-configured with the proper RBAC, since the Operator cannot create the RBAC itself.
	NamespaceOperatingModeReadOnly NamespaceOperatingMode = "readonly"
	// NamespaceOperatingModeDisabled causes namespace capabilities to be disabled altogether.  In
	// this mode the Operator will simply attempt to work with the target namespaces specified
	// during installation.  If no target namespaces are specified, then it will be configured to
	// work within the namespace in which the Operator is deployed.
	NamespaceOperatingModeDisabled NamespaceOperatingMode = "disabled"

	// DNS-1123 formatting and error message for validating namespace names
	dns1123Fmt    string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123ErrMsg string = "A namespace name must consist of lower case" +
		"alphanumeric characters or '-', and must start and end with an alphanumeric character"
)

var (
	// namespacePrivsCoreDynamic defines the privileges in the Core API group required for the
	// Operator to run using the NamespaceOperatingModeDynamic namespace operating mode
	namespacePrivsCoreDynamic = map[string][]string{
		"namespaces":      {"get", "list", "watch", "create", "update", "delete"},
		"serviceaccounts": {"get", "create", "delete"},
	}
	// namespacePrivsDynamic defines the privileges in the rbac.authorization.k8s.io API group
	// required for the Operator to run using the NamespaceOperatingModeDynamic namespace
	// operating mode
	namespacePrivsRBACDynamic = map[string][]string{
		"roles":        {"get", "create", "delete", "bind", "escalate"},
		"rolebindings": {"get", "create", "delete"},
	}
	// namespacePrivsReadOnly defines the privileges in the Core API group required for the
	// Operator to run using the NamespaceOperatingModeReadOnly namespace operating mode
	namespacePrivsCoreReadOnly = map[string][]string{
		"namespaces": {"get", "list", "watch"},
	}

	// ErrInvalidNamespaceName defines the error that is thrown when a namespace does not meet the
	// requirements for naming set by Kubernetes
	ErrInvalidNamespaceName = errors.New(validation.RegexError(dns1123ErrMsg, dns1123Fmt,
		"my-name", "123-abc"))
	// ErrNamespaceNotWatched defines the error that is thrown when a namespace does not meet the
	// requirements for naming set by Kubernetes
	ErrNamespaceNotWatched = errors.New("The namespaces are not watched by the " +
		"current PostgreSQL Operator installation")
)

// CreateNamespaceAndRBAC creates a new namespace that is owned by the Operator, while then
// installing the required RBAC within that namespace as required to be utilized with the
// current Operator install.
func CreateNamespaceAndRBAC(clientset *kubernetes.Clientset, installationName, pgoNamespace,
	createdBy, newNs string) error {

	log.Debugf("CreateNamespace %s %s %s", pgoNamespace, createdBy, newNs)

	//define the new namespace
	n := v1.Namespace{}
	n.ObjectMeta.Labels = make(map[string]string)
	n.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	n.ObjectMeta.Labels[config.LABEL_PGO_CREATED_BY] = createdBy
	n.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = installationName

	n.Name = newNs

	if err := kubeapi.CreateNamespace(clientset, &n); err != nil {
		return err
	}

	log.Debugf("CreateNamespace %s created by %s", newNs, createdBy)

	//apply targeted rbac rules here
	if err := installTargetRBAC(clientset, pgoNamespace, newNs); err != nil {
		return err
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOCreateNamespaceFormat{
		EventHeader: events.EventHeader{
			Namespace: pgoNamespace,
			Username:  createdBy,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPGOCreateNamespace,
		},
		CreatedNamespace: newNs,
	}

	return events.Publish(f)
}

// DeleteNamespace deletes the namespace specified.
func DeleteNamespace(clientset *kubernetes.Clientset, installationName, pgoNamespace, deletedBy, ns string) error {

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
			Timestamp: time.Now(),
			EventType: events.EventPGODeleteNamespace,
		},
		DeletedNamespace: ns,
	}

	return events.Publish(f)
}

func copySecret(clientset *kubernetes.Clientset, secretName, operatorNamespace, targetNamespace string) error {
	secret, err := kubeapi.GetSecret(clientset, secretName, operatorNamespace)

	if err == nil {
		secret.ObjectMeta = metav1.ObjectMeta{
			Annotations: secret.ObjectMeta.Annotations,
			Labels:      secret.ObjectMeta.Labels,
			Name:        secret.ObjectMeta.Name,
		}
		if err = kubeapi.CreateSecret(clientset, secret, targetNamespace); kerrors.IsAlreadyExists(err) {
			err = kubeapi.UpdateSecret(clientset, secret, targetNamespace)
		}
	}

	if !kubeapi.IsNotFound(err) {
		return err
	}

	return nil
}

func installTargetRBAC(clientset *kubernetes.Clientset, operatorNamespace, targetNamespace string) error {

	// Use the image pull secrets of the operator service account in the new namespace.
	operator, exists, err := kubeapi.GetServiceAccount(clientset, OPERATOR_SERVICE_ACCOUNT, operatorNamespace)
	if !exists {
		log.Errorf("expected the operator account to exist: %v", err)
		return err
	}

	err = createPGODefaultServiceAccount(clientset, targetNamespace, operator.ImagePullSecrets)
	if err != nil {
		log.Error(err)
		return err
	}

	err = createPGOTargetServiceAccount(clientset, targetNamespace, operator.ImagePullSecrets)
	if err != nil {
		log.Error(err)
		return err
	}

	err = createPGOBackrestServiceAccount(clientset, targetNamespace, operator.ImagePullSecrets)
	if err != nil {
		log.Error(err)
		return err
	}

	err = createPGOTargetRole(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = createPGOTargetRoleBinding(clientset, targetNamespace, operatorNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = createPGOBackrestRole(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}
	err = createPGOBackrestRoleBinding(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	err = createPGOPgServiceAccount(clientset, targetNamespace, operator.ImagePullSecrets)
	if err != nil {
		log.Error(err)
		return err
	}
	err = createPGOPgRole(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}
	err = createPGOPgRoleBinding(clientset, targetNamespace)
	if err != nil {
		log.Error(err)
		return err
	}

	// Now that the operator has permission in the new namespace, copy any existing
	// image pull secrets to the new namespace.
	for _, reference := range operator.ImagePullSecrets {
		if err = copySecret(clientset, reference.Name, operatorNamespace, targetNamespace); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil

}

func createPGOTargetRoleBinding(clientset *kubernetes.Clientset, targetNamespace, operatorNamespace string) error {
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

	return kubeapi.CreateRoleBinding(clientset, &rb, targetNamespace)
}

func createPGOBackrestRole(clientset *kubernetes.Clientset, targetNamespace string) error {
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

	return kubeapi.CreateRole(clientset, &r, targetNamespace)
}

func createPGOTargetRole(clientset *kubernetes.Clientset, targetNamespace string) error {
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

	return kubeapi.CreateRole(clientset, &r, targetNamespace)
}

func createPGOBackrestRoleBinding(clientset *kubernetes.Clientset, targetNamespace string) error {

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

	return kubeapi.CreateRoleBinding(clientset, &rb, targetNamespace)
}

// UpdateNamespaceAndRBAC updates a new namespace to be owned by the Operator, while then
// installing (or re-installing) the required RBAC within that namespace as required to be
// utilized with the current Operator install.
func UpdateNamespaceAndRBAC(clientset *kubernetes.Clientset, installationName, pgoNamespace,
	updatedBy, ns string) error {

	log.Debugf("UpdateNamespace %s %s %s %s", installationName, pgoNamespace, updatedBy, ns)

	theNs, _, err := kubeapi.GetNamespace(clientset, ns)
	if err != nil {
		return err
	}

	if theNs.ObjectMeta.Labels == nil {
		theNs.ObjectMeta.Labels = make(map[string]string)
	}
	theNs.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	theNs.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = installationName
	if err := kubeapi.UpdateNamespace(clientset, theNs); err != nil {
		return err
	}

	//apply targeted rbac rules here
	if err := installTargetRBAC(clientset, pgoNamespace, ns); err != nil {
		return err
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOCreateNamespaceFormat{
		EventHeader: events.EventHeader{
			Namespace: pgoNamespace,
			Username:  updatedBy,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPGOCreateNamespace,
		},
		CreatedNamespace: ns,
	}

	return events.Publish(f)
}

func createPGOBackrestServiceAccount(clientset *kubernetes.Clientset, targetNamespace string, imagePullSecrets []v1.LocalObjectReference) error {

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

	var sa v1.ServiceAccount
	if err = json.Unmarshal(buffer.Bytes(), &sa); err != nil {
		log.Error("error unmarshalling " + config.PGOBackrestServiceAccountPath + " json ServiceAccount " + err.Error())
		return err
	}
	sa.ImagePullSecrets = imagePullSecrets

	return kubeapi.CreateServiceAccount(clientset, &sa, targetNamespace)
}

// ConfigureInstallNamespaces is responsible for properly configuring up any namespaces provided for
// the installation of the Operator.  This includes creating or updating those namespaces so they can
// be utilized by the Operator to deploy PG clusters.
func ConfigureInstallNamespaces(clientset *kubernetes.Clientset, installationName, pgoNamespace string,
	namespaceNames []string) error {

	// now loop through all namespaces and either create or update them
	for _, namespaceName := range namespaceNames {

		// First try to create the namespace. If the namespace already exists, then proceed with
		// updating it.  Otherwise if a new namespace was successfully created, simply move on to
		// the next namespace.
		if err := CreateNamespaceAndRBAC(clientset, installationName, pgoNamespace,
			"operator-bootstrap", namespaceName); err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		} else if err == nil {
			continue
		}

		namespace, _, err := kubeapi.GetNamespace(clientset, namespaceName)
		if err != nil {
			return err
		}

		// continue if already owned by this install, or if owned by another install
		labels := namespace.ObjectMeta.Labels
		if labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY {
			switch {
			case labels[config.LABEL_PGO_INSTALLATION_NAME] == installationName:
				log.Errorf("Configure install namespaces: namespace %s already owned by this "+
					"installation, will not update it", namespaceName)
				continue
			case labels[config.LABEL_PGO_INSTALLATION_NAME] != installationName:
				log.Errorf("Configure install namespaces: namespace %s owned by another "+
					"installation, will not update it", namespaceName)
				continue
			}
		}

		// if not part of this or another install, then update the namespace to be owned by this
		// Operator install
		log.Infof("Configure install namespaces: namespace %s will be updated to be owned by this "+
			"installation", namespaceName)
		if err := UpdateNamespaceAndRBAC(clientset, installationName, pgoNamespace,
			"operator-bootstrap", namespaceName); err != nil {
			return err
		}
	}

	return nil
}

// GetCurrentNamespaceList returns the current list namespaces being managed by the current
// Operateor installation.  When the current namespace mode is "dynamic" or "readOnly", this
// involves querying the Kube cluster for an namespaces with the "vendor" and
// "pgo-installation-name" labels corresponding to the current Operator install.  When the
// namespace mode is "disabled", a list of namespaces specified using the NAMESPACE env var during
// installation is returned (with the list defaulting to the Operator's own namespace in the event
// that NAMESPACE is empty).
func GetCurrentNamespaceList(clientset *kubernetes.Clientset,
	installationName string) ([]string, error) {

	ns := make([]string, 0)

	nsList, err := kubeapi.GetNamespaces(clientset)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	for _, v := range nsList.Items {
		labels := v.ObjectMeta.Labels
		if labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY &&
			labels[config.LABEL_PGO_INSTALLATION_NAME] == installationName {
			ns = append(ns, v.Name)
		}
	}

	return ns, nil
}

// ValidateNamespacesWatched validates whether or not the namespaces provided are being watched by
// the current Operator installation.  When the current namespace mode is "dynamic" or "readOnly",
// this involves ensuring the namespace specified has the proper "vendor" and
// "pgo-installation-name" labels corresponding to the current Operator install.  When the
// namespace mode is "disabled", this means ensuring the namespace is in the list of those
// specifiedusing the NAMESPACE env var during installation (with the list defaulting to the
// Operator's own namespace in the event that NAMESPACE is empty).  If any namespaces are found to
// be invalid, an ErrNamespaceNotWatched error is returned containing an error message listing
// the invalid namespaces.
func ValidateNamespacesWatched(clientset *kubernetes.Clientset,
	namespaceOperatingMode NamespaceOperatingMode,
	installationName string, namespaces ...string) error {

	var err error
	var currNSList []string
	if namespaceOperatingMode != NamespaceOperatingModeDisabled {
		currNSList, err = GetCurrentNamespaceList(clientset, installationName)
		if err != nil {
			return err
		}
	} else {
		currNSList = getNamespacesFromEnv()
	}

	var invalidNamespaces []string
	for _, ns := range namespaces {
		var validNS bool
		for _, currNS := range currNSList {
			if ns == currNS {
				validNS = true
				break
			}
		}
		if !validNS {
			invalidNamespaces = append(invalidNamespaces, ns)
		}
	}

	if len(invalidNamespaces) > 0 {
		return fmt.Errorf("The following namespaces are invalid: %v. %w", invalidNamespaces,
			ErrNamespaceNotWatched)
	}

	return nil
}

// getNamespacesFromEnv returns a slice containing the namespaces strored the NAMESPACE env var in
// csv format.  If NAMESPACE is empty, then the Operator namespace as specified in env var
// PGO_OPERATOR_NAMESPACE is returned.
func getNamespacesFromEnv() []string {
	namespaceEnvVar := os.Getenv("NAMESPACE")
	if namespaceEnvVar == "" {
		defaultNs := os.Getenv("PGO_OPERATOR_NAMESPACE")
		return []string{defaultNs}
	}
	return strings.Split(namespaceEnvVar, ",")
}

// ValidateNamespaceNames validates one or more namespace names to ensure they are valid per Kubernetes
// naming requirements.
func ValidateNamespaceNames(namespace ...string) error {
	var invalidNamespaces []string
	for _, ns := range namespace {
		if validation.IsDNS1123Label(ns) != nil {
			invalidNamespaces = append(invalidNamespaces, ns)
		}
	}

	if len(invalidNamespaces) > 0 {
		return fmt.Errorf("The following namespaces are invalid %v. %w", invalidNamespaces,
			ErrInvalidNamespaceName)
	}

	return nil
}

// createPGODefaultServiceAccount creates the default SA.
func createPGODefaultServiceAccount(clientset *kubernetes.Clientset, targetNamespace string, imagePullSecrets []v1.LocalObjectReference) error {

	_, found, _ := kubeapi.GetServiceAccount(clientset, PGO_DEFAULT_SERVICE_ACCOUNT, targetNamespace)
	if found {
		log.Infof("serviceaccount %s already exists, will delete and re-create", PGO_DEFAULT_SERVICE_ACCOUNT)
		err := kubeapi.DeleteServiceAccount(clientset, PGO_DEFAULT_SERVICE_ACCOUNT, targetNamespace)
		if err != nil {
			log.Errorf("error deleting serviceaccount %s %s", PGO_DEFAULT_SERVICE_ACCOUNT, err.Error())
			return err
		}
	}
	var buffer bytes.Buffer
	err := config.PgoDefaultServiceAccountTemplate.Execute(&buffer,
		PgoDefaultServiceAccount{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error() + " on " + config.PGODefaultServiceAccountPath)
		return err
	}
	log.Info(buffer.String())

	var sa v1.ServiceAccount
	if err = json.Unmarshal(buffer.Bytes(), &sa); err != nil {
		log.Error("error unmarshalling " + config.PGODefaultServiceAccountPath + " json ServiceAccount " + err.Error())
		return err
	}
	sa.ImagePullSecrets = imagePullSecrets

	return kubeapi.CreateServiceAccount(clientset, &sa, targetNamespace)
}

func createPGOTargetServiceAccount(clientset *kubernetes.Clientset, targetNamespace string, imagePullSecrets []v1.LocalObjectReference) error {

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

	var sa v1.ServiceAccount
	if err = json.Unmarshal(buffer.Bytes(), &sa); err != nil {
		log.Error("error unmarshalling " + config.PGOTargetServiceAccountPath + " json ServiceAccount " + err.Error())
		return err
	}
	sa.ImagePullSecrets = imagePullSecrets

	return kubeapi.CreateServiceAccount(clientset, &sa, targetNamespace)
}

// createPGOPgServiceAccount creates the SA for use by PG pods
func createPGOPgServiceAccount(clientset *kubernetes.Clientset, targetNamespace string, imagePullSecrets []v1.LocalObjectReference) error {

	//check for serviceaccount existing
	_, found, _ := kubeapi.GetServiceAccount(clientset, PGO_PG_SERVICE_ACCOUNT, targetNamespace)
	if found {
		log.Infof("serviceaccount %s already exists, will delete and re-create", PGO_PG_SERVICE_ACCOUNT)
		err := kubeapi.DeleteServiceAccount(clientset, PGO_PG_SERVICE_ACCOUNT, targetNamespace)
		if err != nil {
			log.Errorf("error deleting serviceaccount %s %s", PGO_PG_SERVICE_ACCOUNT, err.Error())
			return err
		}
	}
	var buffer bytes.Buffer
	err := config.PgoPgServiceAccountTemplate.Execute(&buffer,
		PgoPgServiceAccount{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error() + " on " + config.PGOPgServiceAccountPath)
		return err
	}
	log.Info(buffer.String())

	var sa v1.ServiceAccount
	if err = json.Unmarshal(buffer.Bytes(), &sa); err != nil {
		log.Error("error unmarshalling " + config.PGOPgServiceAccountPath + " json ServiceAccount " + err.Error())
		return err
	}
	sa.ImagePullSecrets = imagePullSecrets

	return kubeapi.CreateServiceAccount(clientset, &sa, targetNamespace)
}

// createPGOPgRole creates the role used by the 'pgo-sa' SA
func createPGOPgRole(clientset *kubernetes.Clientset, targetNamespace string) error {
	//check for role existing
	_, found, _ := kubeapi.GetRole(clientset, PGO_PG_ROLE, targetNamespace)
	if found {
		log.Infof("role %s already exists, will delete and re-create", PGO_PG_ROLE)
		err := kubeapi.DeleteRole(clientset, PGO_PG_ROLE, targetNamespace)
		if err != nil {
			log.Errorf("error deleting role %s %s", PGO_PG_ROLE, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoPgRoleTemplate.Execute(&buffer,
		PgoPgRole{
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
		log.Error("error unmarshalling " + config.PGOPgRolePath + " json Role " + err.Error())
		return err
	}

	return kubeapi.CreateRole(clientset, &r, targetNamespace)
}

// createPGOPgRoleBinding binds the 'pgo-pg-role' role to the 'pgo-sa' SA
func createPGOPgRoleBinding(clientset *kubernetes.Clientset, targetNamespace string) error {
	//check for rolebinding existing
	_, found, _ := kubeapi.GetRoleBinding(clientset, PGO_PG_ROLE_BINDING, targetNamespace)
	if found {
		log.Infof("rolebinding %s already exists, will delete and re-create", PGO_PG_ROLE_BINDING)
		err := kubeapi.DeleteRoleBinding(clientset, PGO_PG_ROLE_BINDING, targetNamespace)
		if err != nil {
			log.Errorf("error deleting rolebinding %s %s", PGO_PG_ROLE_BINDING, err.Error())
			return err
		}
	}

	var buffer bytes.Buffer
	err := config.PgoPgRoleBindingTemplate.Execute(&buffer,
		PgoPgRoleBinding{
			TargetNamespace: targetNamespace,
		})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info(buffer.String())

	rb := rbacv1.RoleBinding{}
	err = json.Unmarshal(buffer.Bytes(), &rb)
	if err != nil {
		log.Error("error unmarshalling " + config.PGOPgRoleBindingPath + " json RoleBinding " + err.Error())
		return err
	}

	return kubeapi.CreateRoleBinding(clientset, &rb, targetNamespace)
}

// GetNamespaceOperatingMode is responsible for returning the proper namespace operating mode for
// the current Operator install.  This is done by submitting a SubjectAccessReview in the local
// Kubernetes cluster to determine whether or not certain cluster-level privileges have been
// assigned to the Operator Service Account.  Based on the privileges identified, one of the
// a the proper NamespaceOperatingMode will be returned as applicable for those privileges
// (please see the various NamespaceOperatingMode types for a detailed explanation of each
// operating mode).
func GetNamespaceOperatingMode(clientset *kubernetes.Clientset) (NamespaceOperatingMode, error) {

	// first check to see if dynamic namespace capabilities can be enabled
	isDynamicCore, err := checkClusterPrivs(clientset, namespacePrivsCoreDynamic, "")
	if err != nil {
		return "", err
	}

	isDynamicRBAC, err := checkClusterPrivs(clientset, namespacePrivsRBACDynamic,
		"rbac.authorization.k8s.io")
	if err != nil {
		return "", err
	}

	if isDynamicCore && isDynamicRBAC {
		return NamespaceOperatingModeDynamic, nil
	}

	// now check if read-only namespace capabilities can be enabled
	isReadOnly, err := checkClusterPrivs(clientset, namespacePrivsCoreReadOnly, "")
	if err != nil {
		return "", err
	}
	if isReadOnly {
		return NamespaceOperatingModeReadOnly, nil
	}

	// if not dynamic or read-only, then disable namespace capabilities
	return NamespaceOperatingModeDisabled, nil
}

// checkClusterPrivs checks to see if the service account currently running the operator has
// the permissions defined for various resources as specified in the provided permissions
// map. This function assumes the resource being checked is cluster-scoped.  If the Service
// Account has all of the permissions defined in the permissions map, then "true" is returned.
// Otherwise, if the Service Account is missing any of the permissions specified, or if an error
// is encountered while attempting to deterine the permissions for the service account, then
// "false" is retured (along with the error in the event an error is encountered).
func checkClusterPrivs(clientset *kubernetes.Clientset,
	privs map[string][]string, apiGroup string) (bool, error) {

	for resource, verbs := range privs {
		for _, verb := range verbs {
			resAttrs := &authorizationapi.ResourceAttributes{
				Resource: resource,
				Verb:     verb,
				Group:    apiGroup,
			}
			sar, err := kubeapi.CreateSelfSubjectAccessReview(clientset, resAttrs)
			if err != nil {
				return false, err
			}
			if !sar.Status.Allowed {
				return false, nil
			}
		}
	}

	return true, nil
}

// GetInitialNamespaceList returns an initial list of namespaces for the current Operator install.
// This includes first obtaining any namespaces from the NAMESPACE env var, and then if the
// namespace operating mode permits, also querying the Kube cluster in order to obtain any other
// namespaces that are part of the install, but not included in the env var.  If no namespaces are
// identified via either of these methods, then the the PGO namespaces is returned as the default
// namespace.
func GetInitialNamespaceList(clientset *kubernetes.Clientset,
	namespaceOperatingMode NamespaceOperatingMode,
	installationName, pgoNamespace string) ([]string, error) {

	// next grab the namespaces provided using the NAMESPACE env var
	namespaceList := getNamespacesFromEnv()

	// make sure the namespaces obtained from the NAMESPACE env var are valid
	if err := ValidateNamespaceNames(namespaceList...); err != nil {
		return nil, err
	}

	nsEnvMap := make(map[string]struct{})
	for _, namespace := range namespaceList {
		nsEnvMap[namespace] = struct{}{}
	}

	// If the Operator is in a dynamic or readOnly mode, then refresh the namespace list by
	// querying the Kube cluster.  This allows us to account for all namespaces owned by the
	// Operator, including those not explicitly specified during the Operator install.
	var namespaceListCluster []string
	var err error
	if namespaceOperatingMode == NamespaceOperatingModeDynamic ||
		namespaceOperatingMode == NamespaceOperatingModeReadOnly {
		namespaceListCluster, err = GetCurrentNamespaceList(clientset, installationName)
		if err != nil {
			return nil, err
		}
	}

	for _, namespace := range namespaceListCluster {
		if _, ok := nsEnvMap[namespace]; !ok {
			namespaceList = append(namespaceList, namespace)
		}
	}

	return namespaceList, nil
}
