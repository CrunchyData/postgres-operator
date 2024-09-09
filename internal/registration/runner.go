// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"context"
	"crypto/rsa"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// Runner implements [Registration] by loading and validating the token at a
// fixed path. Its methods are safe to call concurrently.
type Runner struct {
	changed   func()
	enabled   bool
	publicKey *rsa.PublicKey
	refresh   time.Duration
	tokenPath string

	token struct {
		sync.RWMutex
		Exists bool `json:"-"`

		jwt.RegisteredClaims
		Iteration int `json:"itr"`
	}
}

// Runner implements [Registration] and [manager.Runnable].
var _ Registration = (*Runner)(nil)
var _ manager.Runnable = (*Runner)(nil)

// NewRunner creates a [Runner] that periodically checks the validity of the
// token at tokenPath. It calls changed when the validity of the token changes.
func NewRunner(publicKey, tokenPath string, changed func()) (*Runner, error) {
	runner := &Runner{
		changed:   changed,
		refresh:   time.Minute,
		tokenPath: tokenPath,
	}

	var err error
	switch {
	case publicKey != "" && tokenPath != "":
		if !strings.HasPrefix(strings.TrimSpace(publicKey), "-") {
			publicKey = "-----BEGIN -----\n" + publicKey + "\n-----END -----"
		}

		runner.enabled = true
		runner.publicKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))

	case publicKey == "" && tokenPath != "":
		err = errors.New("registration: missing public key")

	case publicKey != "" && tokenPath == "":
		err = errors.New("registration: missing token path")
	}

	return runner, err
}

// CheckToken loads and verifies the configured token, returning an error when
// the file exists but cannot be verified.
func (r *Runner) CheckToken() error {
	data, errFile := os.ReadFile(r.tokenPath)
	key := func(*jwt.Token) (any, error) { return r.publicKey, nil }

	// Assume [jwt] and [os] functions could do something unexpected; use defer
	// to safely write to the token.
	r.token.Lock()
	defer r.token.Unlock()

	_, errToken := jwt.ParseWithClaims(string(data), &r.token, key,
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256"}),
	)

	// The error from [os.ReadFile] indicates whether a token file exists.
	r.token.Exists = !os.IsNotExist(errFile)

	// Reset most claims if there is any problem loading, parsing, validating, or
	// verifying the token file.
	if errFile != nil || errToken != nil {
		r.token.RegisteredClaims = jwt.RegisteredClaims{}
	}

	switch {
	case !r.enabled || !r.token.Exists:
		return nil
	case errFile != nil:
		return errFile
	default:
		return errToken
	}
}

func (r *Runner) state() (failed, required bool) {
	// Assume [time] functions could do something unexpected; use defer to safely
	// read the token.
	r.token.RLock()
	defer r.token.RUnlock()

	failed = r.token.Exists && r.token.ExpiresAt == nil
	required = r.enabled &&
		(!r.token.Exists || failed || r.token.ExpiresAt.Before(time.Now()))
	return
}

// Required returns true when registration is required but the token is missing or invalid.
func (r *Runner) Required(
	recorder record.EventRecorder, object client.Object, conditions *[]metav1.Condition,
) bool {
	failed, required := r.state()

	if r.enabled && failed {
		emitFailedWarning(recorder, object)
	}

	if !required && conditions != nil {
		before := len(*conditions)
		meta.RemoveStatusCondition(conditions, v1beta1.Registered)
		meta.RemoveStatusCondition(conditions, "RegistrationRequired")
		meta.RemoveStatusCondition(conditions, "TokenRequired")
		found := len(*conditions) != before

		if r.enabled && found {
			emitVerifiedEvent(recorder, object)
		}
	}

	return required
}

// NeedLeaderElection returns true so that r runs only on the single
// [manager.Manager] that is elected leader in the Kubernetes namespace.
func (r *Runner) NeedLeaderElection() bool { return true }

// Start watches for a mounted registration token when enabled. It blocks
// until ctx is cancelled.
func (r *Runner) Start(ctx context.Context) error {
	var ticks <-chan time.Time

	if r.enabled {
		ticker := time.NewTicker(r.refresh)
		defer ticker.Stop()
		ticks = ticker.C
	}

	log := logging.FromContext(ctx).WithValues("controller", "registration")

	for {
		select {
		case <-ticks:
			_, before := r.state()
			if err := r.CheckToken(); err != nil {
				log.Error(err, "Unable to validate token")
			}
			if _, after := r.state(); before != after && r.changed != nil {
				r.changed()
			}
		case <-ctx.Done():
			// https://github.com/kubernetes-sigs/controller-runtime/issues/1927
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		}
	}
}
