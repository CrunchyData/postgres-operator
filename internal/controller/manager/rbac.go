package manager

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"text/template"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/ns"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ErrReconcileRBAC defines the error string that is displayed when RBAC reconciliation is
	// enabled for the current PostgreSQL Operator installation but the Operator is unable to
	// to properly/fully reconcile its own RBAC in a target namespace.
	ErrReconcileRBAC = "operator is unable to reconcile RBAC resource"
)

// reconcileRBAC is responsible for reconciling the RBAC resources (ServiceAccounts, Roles and
// RoleBindings) required by the PostgreSQL Operator in a target namespace
func (c *ControllerManager) reconcileRBAC(targetNamespace string) {

	log.Debugf("Controller Manager: Now reconciling RBAC in namespace %s", targetNamespace)

	// Use the image pull secrets of the operator service account in the new namespace.
	operator, err := c.controllers[targetNamespace].kubeClientset.CoreV1().
		ServiceAccounts(c.pgoNamespace).Get(ns.OPERATOR_SERVICE_ACCOUNT, metav1.GetOptions{})
	if err != nil {
		// just log an error and continue so that we can attempt to reconcile other RBAC resources
		// that are not dependent on the Operator ServiceAccount, e.g. Roles and RoleBindings
		log.Errorf("%s: %v", ErrReconcileRBAC, err)
	}

	saCreatedOrUpdated := c.reconcileServiceAccounts(targetNamespace,
		operator.ImagePullSecrets)
	c.reconcileRoles(targetNamespace)
	c.reconcileRoleBindings(targetNamespace)

	// If a SA was created or updated, or if it doesnt exist, ensure the image pull secrets
	// are up to date
	for _, reference := range operator.ImagePullSecrets {

		var doesNotExist bool

		if _, err := c.controllers[targetNamespace].kubeClientset.CoreV1().
			Secrets(targetNamespace).Get(
			reference.Name, metav1.GetOptions{}); err != nil {
			if kerrors.IsNotFound(err) {
				doesNotExist = true
			} else {
				log.Errorf("%s: %v", ErrReconcileRBAC, err)
				continue
			}
		}

		if doesNotExist || saCreatedOrUpdated {
			if err := ns.CopySecret(c.controllers[targetNamespace].kubeClientset, reference.Name,
				c.pgoNamespace, targetNamespace); err != nil {
				log.Errorf("%s: %v", ErrReconcileRBAC, err)
			}
		}
	}
}

// reconcileRoles reconciles the Roles required by the operator in a target namespace
func (c *ControllerManager) reconcileRoles(targetNamespace string) {

	reconcileRoles := map[string]*template.Template{
		ns.PGO_TARGET_ROLE:   config.PgoTargetRoleTemplate,
		ns.PGO_BACKREST_ROLE: config.PgoBackrestRoleTemplate,
		ns.PGO_PG_ROLE:       config.PgoPgRoleTemplate,
	}

	for role, template := range reconcileRoles {
		if err := ns.ReconcileRole(c.controllers[targetNamespace].kubeClientset, role,
			targetNamespace, template); err != nil {
			log.Errorf("%s: %v", ErrReconcileRBAC, err)
		}
	}
}

// reconcileRoleBindings reconciles the RoleBindings required by the operator in a
// target namespace
func (c *ControllerManager) reconcileRoleBindings(targetNamespace string) {

	reconcileRoleBindings := map[string]*template.Template{
		ns.PGO_TARGET_ROLE_BINDING:   config.PgoTargetRoleBindingTemplate,
		ns.PGO_BACKREST_ROLE_BINDING: config.PgoBackrestRoleBindingTemplate,
		ns.PGO_PG_ROLE_BINDING:       config.PgoPgRoleBindingTemplate,
	}

	for roleBinding, template := range reconcileRoleBindings {
		if err := ns.ReconcileRoleBinding(c.controllers[targetNamespace].kubeClientset,
			c.pgoNamespace, roleBinding, targetNamespace, template); err != nil {
			log.Errorf("%s: %v", ErrReconcileRBAC, err)
		}
	}
}

// reconcileServiceAccounts reconciles the ServiceAccounts required by the operator in a
// target namespace
func (c *ControllerManager) reconcileServiceAccounts(targetNamespace string,
	imagePullSecrets []v1.LocalObjectReference) (saCreatedOrUpdated bool) {

	reconcileServiceAccounts := map[string]*template.Template{
		ns.PGO_DEFAULT_SERVICE_ACCOUNT:  config.PgoDefaultServiceAccountTemplate,
		ns.PGO_TARGET_SERVICE_ACCOUNT:   config.PgoTargetServiceAccountTemplate,
		ns.PGO_BACKREST_SERVICE_ACCOUNT: config.PgoBackrestServiceAccountTemplate,
		ns.PGO_PG_SERVICE_ACCOUNT:       config.PgoPgServiceAccountTemplate,
	}

	for serviceAccount, template := range reconcileServiceAccounts {
		createdOrUpdated, err := ns.ReconcileServiceAccount(c.controllers[targetNamespace].kubeClientset,
			serviceAccount, targetNamespace, template, imagePullSecrets)
		if err != nil {
			log.Errorf("%s: %v", ErrReconcileRBAC, err)
			continue
		}
		if !saCreatedOrUpdated && createdOrUpdated {
			saCreatedOrUpdated = true
		}
	}
	return
}
