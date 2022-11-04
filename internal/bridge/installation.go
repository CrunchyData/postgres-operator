/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package bridge

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/naming"
)

// self is a singleton Installation. See [InstallationReconciler].
var self = new(struct {
	Installation
	sync.RWMutex
})

type AuthObject struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
	Secret    string    `json:"secret"`
}

type Installation struct {
	ID         string     `json:"id"`
	AuthObject AuthObject `json:"auth_object"`
}

type InstallationReconciler struct {
	Owner  client.FieldOwner
	Reader interface {
		Get(context.Context, client.ObjectKey, client.Object) error
	}
	Writer interface {
		Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error
	}

	// SecretRef is the name of the corev1.Secret in which to store Bridge tokens.
	SecretRef client.ObjectKey

	// NewClient is called each time a new Client is needed.
	NewClient func() *Client
}

// ManagedInstallationReconciler creates an [InstallationReconciler] and adds it to m.
func ManagedInstallationReconciler(m manager.Manager, newClient func() *Client) error {
	kubernetes := m.GetClient()
	reconciler := &InstallationReconciler{
		Owner:     naming.ControllerBridge,
		Reader:    kubernetes,
		Writer:    kubernetes,
		SecretRef: naming.AsObjectKey(naming.OperatorConfigurationSecret()),
		NewClient: newClient,
	}

	// NOTE: This name was selected to show something interesting in the logs.
	// The default is "secret".
	// TODO: Pick this name considering metrics and other controllers.
	return builder.ControllerManagedBy(m).Named("installation").
		//
		// Reconcile the one Secret that holds Bridge tokens.
		For(&corev1.Secret{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(func(secret client.Object) bool {
				return client.ObjectKeyFromObject(secret) == reconciler.SecretRef
			}),
		)).
		//
		// Wake periodically even when that Secret does not exist.
		Watches(
			runtime.NewTickerImmediate(time.Hour, event.GenericEvent{}),
			handler.EnqueueRequestsFromMapFunc(func(client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: reconciler.SecretRef}}
			}),
		).
		//
		Complete(reconciler)
}

func (r *InstallationReconciler) Reconcile(
	ctx context.Context, request reconcile.Request) (reconcile.Result, error,
) {
	result := reconcile.Result{}
	secret := &corev1.Secret{}
	err := client.IgnoreNotFound(r.Reader.Get(ctx, request.NamespacedName, secret))

	if err == nil {
		// It is easier later to treat a missing Secret the same as one that exists
		// and is empty. Fill in the metadata with information from the request to
		// make it so.
		secret.Namespace, secret.Name = request.Namespace, request.Name

		err = r.reconcile(ctx, secret)
	}

	// TODO: Check for corev1.NamespaceTerminatingCause after
	// k8s.io/apimachinery@v0.25; see https://issue.k8s.io/108528.

	return result, err
}

func (r *InstallationReconciler) reconcile(ctx context.Context, read *corev1.Secret) error {
	write, err := corev1apply.ExtractSecret(read, string(r.Owner))
	if err != nil {
		return err
	}

	// Read the Installation from the Secret, if any.
	var installation Installation
	if yaml.Unmarshal(read.Data[KeyBridgeToken], &installation) != nil {
		installation = Installation{}
	}

	// When the Secret lacks an Installation, write the one we have in memory
	// or register with the API for a new one. In both cases, we write to the
	// Secret which triggers another reconcile.
	if len(installation.ID) == 0 {
		if len(self.ID) == 0 {
			return r.register(ctx, write)
		}

		data := map[string][]byte{}
		data[KeyBridgeToken], _ = json.Marshal(self.Installation) //nolint:errchkjson

		return r.persist(ctx, write.WithData(data))
	}

	// When the Secret has an Installation, store it in memory.
	// TODO: Validate it first; perhaps refresh the AuthObject.
	if len(self.ID) == 0 {
		self.Lock()
		self.Installation = installation
		self.Unlock()
	}

	return nil
}

// persist uses Server-Side Apply to write config to Kubernetes. The Name and
// Namespace fields cannot be nil.
func (r *InstallationReconciler) persist(
	ctx context.Context, config *corev1apply.SecretApplyConfiguration,
) error {
	data, err := json.Marshal(config)
	apply := client.RawPatch(client.Apply.Type(), data)

	// [client.Client] decides where to write by looking at the underlying type,
	// namespace, and name of its [client.Object] argument. That is also where
	// it stores the API response.
	target := corev1.Secret{}
	target.Namespace, target.Name = *config.Namespace, *config.Name

	if err == nil {
		err = r.Writer.Patch(ctx, &target, apply, r.Owner, client.ForceOwnership)
	}

	return err
}

// register calls the Bridge API to register a new Installation. It stores the
// result in the [self] singleton and the write object in Kubernetes. The Name
// and Namespace fields of the latter cannot be nil.
func (r *InstallationReconciler) register(
	ctx context.Context, write *corev1apply.SecretApplyConfiguration,
) error {
	installation, err := r.NewClient().CreateInstallation(ctx)

	if err == nil {
		// Store the new value in the singleton.
		self.Lock()
		self.Installation = installation
		self.Unlock()

		// Store the new value in the Secret along with the current time.
		data := make(map[string][]byte, 2)
		data[KeyBridgeLocalTime], _ = metav1.Now().MarshalJSON()
		data[KeyBridgeToken], _ = json.Marshal(installation) //nolint:errchkjson

		err = r.persist(ctx, write.WithData(data))
	}

	return err
}
