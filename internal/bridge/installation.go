// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
	"github.com/crunchydata/postgres-operator/internal/logging"
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
		Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error
	}
	Writer interface {
		Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error
	}

	// Refresh is the frequency at which AuthObjects should be renewed.
	Refresh time.Duration

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
		Refresh:   2 * time.Hour,
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
		WatchesRawSource(
			runtime.NewTickerImmediate(time.Hour, event.GenericEvent{},
				handler.EnqueueRequestsFromMapFunc(
					func(context.Context, client.Object) []reconcile.Request {
						return []reconcile.Request{{NamespacedName: reconciler.SecretRef}}
					},
				),
			),
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

		result.RequeueAfter, err = r.reconcile(ctx, secret)
	}

	// Nothing can be written to a deleted namespace.
	if err != nil && apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
		return runtime.ErrorWithoutBackoff(err)
	}

	// Write conflicts are returned as errors; log and retry with backoff.
	if err != nil && apierrors.IsConflict(err) {
		logging.FromContext(ctx).Info("Requeue", "reason", err)
		return runtime.RequeueWithBackoff(), nil
	}

	return result, err
}

// reconcile looks for an Installation in read and stores it or another in
// the [self] singleton after a successful response from the Bridge API.
func (r *InstallationReconciler) reconcile(
	ctx context.Context, read *corev1.Secret) (next time.Duration, err error,
) {
	write, err := corev1apply.ExtractSecret(read, string(r.Owner))
	if err != nil {
		return 0, err
	}

	// We GET-extract-PATCH the Secret and do not build it up from scratch.
	// Send the ResourceVersion from the GET in the body of every PATCH.
	if len(read.ResourceVersion) != 0 {
		write.WithResourceVersion(read.ResourceVersion)
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
			return 0, r.register(ctx, write)
		}

		data := map[string][]byte{}
		data[KeyBridgeToken], _ = json.Marshal(self.Installation) //nolint:errchkjson

		return 0, r.persist(ctx, write.WithData(data))
	}

	// Read the timestamp from the Secret, if any.
	var touched time.Time
	if yaml.Unmarshal(read.Data[KeyBridgeLocalTime], &touched) != nil {
		touched = time.Time{}
	}

	// Refresh the AuthObject when there is no Installation in memory,
	// there is no timestamp, or the timestamp is far away. This writes to
	// the Secret which triggers another reconcile.
	if len(self.ID) == 0 || time.Since(touched) > r.Refresh || time.Until(touched) > r.Refresh {
		return 0, r.refresh(ctx, installation, write)
	}

	// Trigger another reconcile one interval after the stored timestamp.
	return wait.Jitter(time.Until(touched.Add(r.Refresh)), 0.1), nil
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

// refresh calls the Bridge API to refresh the AuthObject of installation. It
// combines the result with installation and stores that in the [self] singleton
// and the write object in Kubernetes. The Name and Namespace fields of the
// latter cannot be nil.
func (r *InstallationReconciler) refresh(
	ctx context.Context, installation Installation,
	write *corev1apply.SecretApplyConfiguration,
) error {
	result, err := r.NewClient().CreateAuthObject(ctx, installation.AuthObject)

	// An authentication error means the installation is irrecoverably expired.
	// Remove it from the singleton and move it to a dated entry in the Secret.
	if err != nil && errors.Is(err, errAuthentication) {
		self.Lock()
		self.Installation = Installation{}
		self.Unlock()

		keyExpiration := KeyBridgeToken +
			installation.AuthObject.ExpiresAt.UTC().Format("--2006-01-02")

		data := make(map[string][]byte, 2)
		data[KeyBridgeToken] = nil
		data[keyExpiration], _ = json.Marshal(installation) //nolint:errchkjson

		return r.persist(ctx, write.WithData(data))
	}

	if err == nil {
		installation.AuthObject = result

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
