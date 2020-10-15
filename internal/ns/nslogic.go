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
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/pkg/events"

	log "github.com/sirupsen/logrus"
	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

// PgoServiceAccount is used to populate the following ServiceAccount templates:
// pgo-default-sa.json
// pgo-target-sa.json
// pgo-backrest-sa.json
// pgo-pg-sa.json
type PgoServiceAccount struct {
	TargetNamespace string
}

// PgoRole is used to populate the following Role templates:
// pgo-target-role.json
// pgo-backrest-role.json
// pgo-pg-role.json
type PgoRole struct {
	TargetNamespace string
}

// PgoRoleBinding is used to populate the following RoleBinding templates:
// pgo-target-role-binding.json
// pgo-backrest-role-binding.json
// pgo-pg-role-binding.json
type PgoRoleBinding struct {
	TargetNamespace   string
	OperatorNamespace string
}

// NamespaceOperatingMode defines the different namespace operating modes for the Operator
type NamespaceOperatingMode string

const (
	// NamespaceOperatingModeDynamic enables full dynamic namespace capabilities, in which the
	// Operator can create, delete and update any namespaces within the Kubernetes cluster.
	// Additionally, while in can listen for namespace events (e.g. namespace additions, updates
	// and deletions), and then create or remove controllers for various namespaces as those
	// namespaces are added or removed from the Kubernetes cluster.
	NamespaceOperatingModeDynamic NamespaceOperatingMode = "dynamic"
	// NamespaceOperatingModeReadOnly allows the Operator to listen for namespace events within the
	// Kubernetetes cluster, and then create and run and/or remove controllers as namespaces are
	// added and deleted.
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
		"namespaces": {"create", "update", "delete"},
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

// CreateFakeNamespaceClient creates a fake namespace client for use with the "disabled" namespace
// operating mode
func CreateFakeNamespaceClient(installationName string) (kubernetes.Interface, error) {

	var namespaces []runtime.Object
	for _, namespace := range getNamespacesFromEnv() {
		namespaces = append(namespaces, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					config.LABEL_VENDOR:                config.LABEL_CRUNCHY,
					config.LABEL_PGO_INSTALLATION_NAME: installationName,
				},
			},
		})
	}

	fakeClient := fake.NewSimpleClientset(namespaces...)

	return fakeClient, nil
}

// CreateNamespace creates a new namespace that is owned by the Operator.
func CreateNamespace(clientset kubernetes.Interface, installationName, pgoNamespace,
	createdBy, newNs string) error {

	log.Debugf("CreateNamespace %s %s %s", pgoNamespace, createdBy, newNs)

	//define the new namespace
	n := v1.Namespace{}
	n.ObjectMeta.Labels = make(map[string]string)
	n.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	n.ObjectMeta.Labels[config.LABEL_PGO_CREATED_BY] = createdBy
	n.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = installationName

	n.Name = newNs

	if _, err := clientset.CoreV1().Namespaces().Create(&n); err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("CreateNamespace %s created by %s", newNs, createdBy)

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
func DeleteNamespace(clientset kubernetes.Interface, installationName, pgoNamespace, deletedBy, ns string) error {

	err := clientset.CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
	if err != nil {
		log.Error(err)
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

// CopySecret copies a secret from the Operator namespace to target namespace
func CopySecret(clientset kubernetes.Interface, secretName, operatorNamespace, targetNamespace string) error {
	secret, err := clientset.CoreV1().Secrets(operatorNamespace).Get(secretName, metav1.GetOptions{})

	if err == nil {
		secret.ObjectMeta = metav1.ObjectMeta{
			Annotations: secret.ObjectMeta.Annotations,
			Labels:      secret.ObjectMeta.Labels,
			Name:        secret.ObjectMeta.Name,
		}

		if _, err = clientset.CoreV1().Secrets(targetNamespace).Create(secret); kerrors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().Secrets(targetNamespace).Update(secret)
		}
	}

	if !kubeapi.IsNotFound(err) {
		return err
	}

	return nil
}

// ReconcileRole reconciles a Role required by the operator in a target namespace
func ReconcileRole(clientset kubernetes.Interface, role, targetNamespace string,
	roleTemplate *template.Template) error {

	var createRole bool

	currRole, err := clientset.RbacV1().Roles(targetNamespace).Get(
		role, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.Debugf("Role %s in namespace %s does not exist and will be created",
				role, targetNamespace)
			createRole = true
		} else {
			return err
		}
	}

	var buffer bytes.Buffer
	if err := roleTemplate.Execute(&buffer,
		PgoRole{TargetNamespace: targetNamespace}); err != nil {
		return err
	}

	templatedRole := rbacv1.Role{}
	if err := json.Unmarshal(buffer.Bytes(), &templatedRole); err != nil {
		return err
	}

	if createRole {
		if _, err := clientset.RbacV1().Roles(targetNamespace).Create(
			&templatedRole); err != nil {
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(currRole.Rules, templatedRole.Rules) {

		log.Debugf("Role %s in namespace %s is invalid and will now be reconciled",
			currRole.Name, targetNamespace)

		currRole.Rules = templatedRole.Rules

		if _, err := clientset.RbacV1().Roles(targetNamespace).Update(
			currRole); err != nil {
			return err
		}
	}

	return nil
}

// ReconcileRoleBinding reconciles a RoleBinding required by the operator in a target namespace
func ReconcileRoleBinding(clientset kubernetes.Interface, pgoNamespace,
	roleBinding, targetNamespace string, roleBindingTemplate *template.Template) error {

	var createRoleBinding bool

	currRoleBinding, err := clientset.RbacV1().RoleBindings(targetNamespace).Get(
		roleBinding, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.Debugf("RoleBinding %s in namespace %s does not exist and will be created",
				roleBinding, targetNamespace)
			createRoleBinding = true
		} else {
			return err
		}
	}

	var buffer bytes.Buffer
	if err := roleBindingTemplate.Execute(&buffer,
		PgoRoleBinding{
			TargetNamespace:   targetNamespace,
			OperatorNamespace: pgoNamespace,
		}); err != nil {
		return err
	}

	templatedRoleBinding := rbacv1.RoleBinding{}
	if err := json.Unmarshal(buffer.Bytes(), &templatedRoleBinding); err != nil {
		return err
	}

	if createRoleBinding {
		if _, err := clientset.RbacV1().RoleBindings(targetNamespace).Create(
			&templatedRoleBinding); err != nil {
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(currRoleBinding.Subjects,
		templatedRoleBinding.Subjects) ||
		!reflect.DeepEqual(currRoleBinding.RoleRef,
			templatedRoleBinding.RoleRef) {

		log.Debugf("RoleBinding %s in namespace %s is invalid and will now be reconciled",
			currRoleBinding.Name, targetNamespace)

		currRoleBinding.Subjects = templatedRoleBinding.Subjects
		currRoleBinding.RoleRef = templatedRoleBinding.RoleRef

		if _, err := clientset.RbacV1().RoleBindings(targetNamespace).Update(
			currRoleBinding); err != nil {
			return err
		}
	}

	return nil
}

// ReconcileServiceAccount reconciles a ServiceAccount required by the operator in a target
// namespace
func ReconcileServiceAccount(clientset kubernetes.Interface,
	serviceAccount, targetNamespace string, serviceAccountTemplate *template.Template,
	imagePullSecrets []v1.LocalObjectReference) (bool, error) {

	var createServiceAccount, createdOrUpdated bool

	currServiceAccount, err := clientset.CoreV1().ServiceAccounts(
		targetNamespace).Get(serviceAccount, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.Debugf("ServiceAccount %s in namespace %s does not exist and will be created",
				serviceAccount, targetNamespace)
			createServiceAccount = true
		} else {
			return createdOrUpdated, err
		}
	}

	var buffer bytes.Buffer
	if err := serviceAccountTemplate.Execute(&buffer,
		PgoServiceAccount{TargetNamespace: targetNamespace}); err != nil {
		return createdOrUpdated, err
	}

	templatedServiceAccount := v1.ServiceAccount{}
	if err := json.Unmarshal(buffer.Bytes(), &templatedServiceAccount); err != nil {
		return createdOrUpdated, err
	}

	if createServiceAccount {
		templatedServiceAccount.ImagePullSecrets = imagePullSecrets
		if _, err := clientset.CoreV1().ServiceAccounts(targetNamespace).Create(
			&templatedServiceAccount); err != nil {
			return createdOrUpdated, err
		}
		createdOrUpdated = true
		return createdOrUpdated, nil
	}

	if !reflect.DeepEqual(currServiceAccount.ImagePullSecrets, imagePullSecrets) {

		log.Debugf("ServiceAccout %s in namespace %s is invalid and will now be reconciled",
			currServiceAccount.Name, targetNamespace)

		currServiceAccount.ImagePullSecrets = imagePullSecrets

		if _, err := clientset.CoreV1().ServiceAccounts(targetNamespace).Update(
			currServiceAccount); err != nil {
			return createdOrUpdated, err
		}
		createdOrUpdated = true
	}

	return createdOrUpdated, nil
}

// UpdateNamespace updates a new namespace to be owned by the Operator.
func UpdateNamespace(clientset kubernetes.Interface, installationName, pgoNamespace,
	updatedBy, ns string) error {

	log.Debugf("UpdateNamespace %s %s %s %s", installationName, pgoNamespace, updatedBy, ns)

	theNs, err := clientset.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if theNs.ObjectMeta.Labels == nil {
		theNs.ObjectMeta.Labels = make(map[string]string)
	}
	theNs.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	theNs.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] = installationName

	if _, err := clientset.CoreV1().Namespaces().Update(theNs); err != nil {
		log.Error(err)
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

// ConfigureInstallNamespaces is responsible for properly configuring up any namespaces provided for
// the installation of the Operator.  This includes creating or updating those namespaces so they can
// be utilized by the Operator to deploy PG clusters.
func ConfigureInstallNamespaces(clientset kubernetes.Interface, installationName, pgoNamespace string,
	namespaceNames []string, namespaceOperatingMode NamespaceOperatingMode) error {

	// now loop through all namespaces and either create or update them
	for _, namespaceName := range namespaceNames {

		nameSpaceExists := true
		// if we can get namespaces, make sure this one isn't part of another install
		if namespaceOperatingMode != NamespaceOperatingModeDisabled {

			namespace, err := clientset.CoreV1().Namespaces().Get(namespaceName,
				metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					nameSpaceExists = false
				} else {
					return err
				}
			}

			if nameSpaceExists {
				// continue if already owned by this install, or if owned by another install
				labels := namespace.ObjectMeta.Labels
				if labels != nil && labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY &&
					labels[config.LABEL_PGO_INSTALLATION_NAME] != installationName {
					log.Errorf("Configure install namespaces: namespace %s owned by another "+
						"installation, will not update it", namespaceName)
					continue
				}
			}
		}

		// if using the "dynamic" namespace mode we can now update the namespace to ensure it is
		// properly owned by this install
		if namespaceOperatingMode == NamespaceOperatingModeDynamic {
			if nameSpaceExists {
				// if not part of this or another install, then update the namespace to be owned by this
				// Operator install
				log.Infof("Configure install namespaces: namespace %s will be updated to be owned by this "+
					"installation", namespaceName)
				if err := UpdateNamespace(clientset, installationName, pgoNamespace,
					"operator-bootstrap", namespaceName); err != nil {
					return err
				}
			} else {
				log.Infof("Configure install namespaces: namespace %s will be created for this "+
					"installation", namespaceName)
				if err := CreateNamespace(clientset, installationName, pgoNamespace,
					"operator-bootstrap", namespaceName); err != nil {
					return err
				}
			}
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
func GetCurrentNamespaceList(clientset kubernetes.Interface,
	installationName string, namespaceOperatingMode NamespaceOperatingMode) ([]string, error) {

	if namespaceOperatingMode == NamespaceOperatingModeDisabled {
		return getNamespacesFromEnv(), nil
	}

	ns := make([]string, 0)

	nsList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
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
func ValidateNamespacesWatched(clientset kubernetes.Interface,
	namespaceOperatingMode NamespaceOperatingMode,
	installationName string, namespaces ...string) error {

	var err error
	var currNSList []string
	if namespaceOperatingMode != NamespaceOperatingModeDisabled {
		currNSList, err = GetCurrentNamespaceList(clientset, installationName,
			namespaceOperatingMode)
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

// GetNamespaceOperatingMode is responsible for returning the proper namespace operating mode for
// the current Operator install.  This is done by submitting a SubjectAccessReview in the local
// Kubernetes cluster to determine whether or not certain cluster-level privileges have been
// assigned to the Operator Service Account.  Based on the privileges identified, one of the
// a the proper NamespaceOperatingMode will be returned as applicable for those privileges
// (please see the various NamespaceOperatingMode types for a detailed explanation of each
// operating mode).
func GetNamespaceOperatingMode(clientset kubernetes.Interface) (NamespaceOperatingMode, error) {

	// first check to see if dynamic namespace capabilities can be enabled
	isDynamic, err := CheckAccessPrivs(clientset, namespacePrivsCoreDynamic, "", "")
	if err != nil {
		return "", err
	}

	// next check to see if readonly namespace capabilities can be enabled
	isReadOnly, err := CheckAccessPrivs(clientset, namespacePrivsCoreReadOnly, "", "")
	if err != nil {
		return "", err
	}

	// return the proper namespace operating mode based on the access privs identified
	switch {
	case isDynamic && isReadOnly:
		return NamespaceOperatingModeDynamic, nil
	case !isDynamic && isReadOnly:
		return NamespaceOperatingModeReadOnly, nil
	default:
		return NamespaceOperatingModeDisabled, nil
	}
}

// CheckAccessPrivs checks to see if the ServiceAccount currently running the operator has
// the permissions defined for various resources as specified in the provided permissions
// map. If an empty namespace is provided then it is assumed the resource is cluster-scoped.
// If the ServiceAccount has all of the permissions defined in the permissions map, then "true"
// is returned.  Otherwise, if the Service Account is missing any of the permissions specified,
// or if an error is encountered while attempting to deterine the permissions for the service
// account, then "false" is returned (along with the error in the event an error is encountered).
func CheckAccessPrivs(clientset kubernetes.Interface,
	privs map[string][]string, apiGroup, namespace string) (bool, error) {

	for resource, verbs := range privs {
		for _, verb := range verbs {
			sar, err := clientset.
				AuthorizationV1().SelfSubjectAccessReviews().
				Create(&authv1.SelfSubjectAccessReview{
					Spec: authv1.SelfSubjectAccessReviewSpec{
						ResourceAttributes: &authv1.ResourceAttributes{
							Namespace: namespace,
							Group:     apiGroup,
							Resource:  resource,
							Verb:      verb,
						},
					},
				})
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
func GetInitialNamespaceList(clientset kubernetes.Interface,
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
		namespaceListCluster, err = GetCurrentNamespaceList(clientset, installationName,
			namespaceOperatingMode)
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
