// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/testing/events"
)

func TestNewRunner(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	assert.NilError(t, err)

	public := pem.EncodeToMemory(&pem.Block{Bytes: der})
	assert.Assert(t, len(public) != 0)

	t.Run("Disabled", func(t *testing.T) {
		runner, err := NewRunner("", "", nil)
		assert.NilError(t, err)
		assert.Assert(t, runner != nil)
		assert.Assert(t, !runner.enabled)
	})

	t.Run("ConfiguredCorrectly", func(t *testing.T) {
		runner, err := NewRunner(string(public), "any", nil)
		assert.NilError(t, err)
		assert.Assert(t, runner != nil)
		assert.Assert(t, runner.enabled)

		t.Run("ExtraLines", func(t *testing.T) {
			input := "\n\n" + strings.ReplaceAll(string(public), "\n", "\n\n") + "\n\n"

			runner, err := NewRunner(input, "any", nil)
			assert.NilError(t, err)
			assert.Assert(t, runner != nil)
			assert.Assert(t, runner.enabled)
		})

		t.Run("WithoutPEMBoundaries", func(t *testing.T) {
			lines := strings.Split(strings.TrimSpace(string(public)), "\n")
			lines = lines[1 : len(lines)-1]

			for _, input := range []string{
				strings.Join(lines, ""),                       // single line
				strings.Join(lines, "\n"),                     // multi-line
				"\n\n" + strings.Join(lines, "\n\n") + "\n\n", // extra lines
			} {
				runner, err := NewRunner(input, "any", nil)
				assert.NilError(t, err)
				assert.Assert(t, runner != nil)
				assert.Assert(t, runner.enabled)
			}
		})
	})

	t.Run("ConfiguredIncorrectly", func(t *testing.T) {
		for _, tt := range []struct {
			key, path, msg string
		}{
			{msg: "public key", key: "", path: "any"},
			{msg: "token path", key: "bad", path: ""},
			{msg: "invalid key", key: "bad", path: "any"},
			{msg: "token path", key: string(public), path: ""},
		} {
			_, err := NewRunner(tt.key, tt.path, nil)
			assert.ErrorContains(t, err, tt.msg, "(key=%q, path=%q)", tt.key, tt.path)
		}
	})
}

func TestRunnerCheckToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	t.Run("SafeToCallDisabled", func(t *testing.T) {
		r := Runner{enabled: false}
		assert.NilError(t, r.CheckToken())
	})

	t.Run("FileMissing", func(t *testing.T) {
		r := Runner{enabled: true, tokenPath: filepath.Join(dir, "nope")}
		assert.NilError(t, r.CheckToken())
	})

	t.Run("FileUnreadable", func(t *testing.T) {
		r := Runner{enabled: true, tokenPath: filepath.Join(dir, "nope")}
		assert.NilError(t, os.WriteFile(r.tokenPath, nil, 0o200)) // Writeable

		assert.ErrorContains(t, r.CheckToken(), "permission")
		assert.Assert(t, r.token.ExpiresAt == nil)
	})

	t.Run("FileEmpty", func(t *testing.T) {
		r := Runner{enabled: true, tokenPath: filepath.Join(dir, "empty")}
		assert.NilError(t, os.WriteFile(r.tokenPath, nil, 0o400)) // Readable

		assert.ErrorContains(t, r.CheckToken(), "malformed")
		assert.Assert(t, r.token.ExpiresAt == nil)
	})

	t.Run("WrongAlgorithm", func(t *testing.T) {
		r := Runner{
			enabled:   true,
			publicKey: &key.PublicKey,
			tokenPath: filepath.Join(dir, "hs256"),
		}

		// Maliciously treating an RSA public key as an HMAC secret.
		// - https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
		public, err := x509.MarshalPKIXPublicKey(r.publicKey)
		assert.NilError(t, err)
		data, err := jwt.New(jwt.SigningMethodHS256).SignedString(public)
		assert.NilError(t, err)
		assert.NilError(t, os.WriteFile(r.tokenPath, []byte(data), 0o400)) // Readable

		assert.Assert(t, r.CheckToken() != nil, "HMAC algorithm should be rejected")
		assert.Assert(t, r.token.ExpiresAt == nil)
	})

	t.Run("MissingExpiration", func(t *testing.T) {
		r := Runner{
			enabled:   true,
			publicKey: &key.PublicKey,
			tokenPath: filepath.Join(dir, "no-claims"),
		}

		data, err := jwt.New(jwt.SigningMethodRS256).SignedString(key)
		assert.NilError(t, err)
		assert.NilError(t, os.WriteFile(r.tokenPath, []byte(data), 0o400)) // Readable

		err = r.CheckToken()
		assert.ErrorContains(t, err, "exp claim is required")
		assert.Assert(t, r.token.ExpiresAt == nil)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		r := Runner{
			enabled:   true,
			publicKey: &key.PublicKey,
			tokenPath: filepath.Join(dir, "expired"),
		}

		data, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"exp": jwt.NewNumericDate(time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)),
		}).SignedString(key)
		assert.NilError(t, err)
		assert.NilError(t, os.WriteFile(r.tokenPath, []byte(data), 0o400)) // Readable

		err = r.CheckToken()
		assert.ErrorContains(t, err, "is expired")
		assert.Assert(t, r.token.ExpiresAt == nil)
	})

	t.Run("ValidToken", func(t *testing.T) {
		r := Runner{
			enabled:   true,
			publicKey: &key.PublicKey,
			tokenPath: filepath.Join(dir, "valid"),
		}

		data, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}).SignedString(key)
		assert.NilError(t, err)
		assert.NilError(t, os.WriteFile(r.tokenPath, []byte(data), 0o400)) // Readable

		assert.NilError(t, r.CheckToken())
		assert.Assert(t, r.token.ExpiresAt != nil)
	})
}

func TestRunnerLeaderElectionRunnable(t *testing.T) {
	var runner manager.LeaderElectionRunnable = &Runner{}

	assert.Assert(t, runner.NeedLeaderElection())
}

func TestRunnerRequiredConditions(t *testing.T) {
	t.Parallel()

	t.Run("RegistrationDisabled", func(t *testing.T) {
		r := Runner{enabled: false}

		for _, tt := range []struct {
			before, after []metav1.Condition
		}{
			{
				before: []metav1.Condition{},
				after:  []metav1.Condition{},
			},
			{
				before: []metav1.Condition{{Type: "ExistingOther"}},
				after:  []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
				after:  []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{
					{Type: "Registered"},
					{Type: "ExistingOther"},
					{Type: "RegistrationRequired"},
				},
				after: []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{{Type: "TokenRequired"}},
				after:  []metav1.Condition{},
			},
		} {
			for _, exists := range []bool{false, true} {
				for _, expires := range []time.Time{
					time.Now().Add(time.Hour),
					time.Now().Add(-time.Hour),
				} {
					r.token.Exists = exists
					r.token.ExpiresAt = jwt.NewNumericDate(expires)

					conditions := append([]metav1.Condition{}, tt.before...)
					discard := new(events.Recorder)
					object := &corev1.ConfigMap{}

					result := r.Required(discard, object, &conditions)

					assert.Equal(t, result, false, "expected registration not required")
					assert.DeepEqual(t, conditions, tt.after)
				}
			}
		}
	})

	t.Run("RegistrationRequired", func(t *testing.T) {
		r := Runner{enabled: true}

		for _, tt := range []struct {
			exists  bool
			expires time.Time
			before  []metav1.Condition
		}{
			{
				exists: false, expires: time.Now().Add(time.Hour),
				before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
			},
			{
				exists: false, expires: time.Now().Add(-time.Hour),
				before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
			},
			{
				exists: true, expires: time.Now().Add(-time.Hour),
				before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
			},
		} {
			r.token.Exists = tt.exists
			r.token.ExpiresAt = jwt.NewNumericDate(tt.expires)

			conditions := append([]metav1.Condition{}, tt.before...)
			discard := new(events.Recorder)
			object := &corev1.ConfigMap{}

			result := r.Required(discard, object, &conditions)

			assert.Equal(t, result, true, "expected registration required")
			assert.DeepEqual(t, conditions, tt.before)
		}
	})

	t.Run("Registered", func(t *testing.T) {
		r := Runner{}
		r.token.Exists = true
		r.token.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))

		for _, tt := range []struct {
			before, after []metav1.Condition
		}{
			{
				before: []metav1.Condition{},
				after:  []metav1.Condition{},
			},
			{
				before: []metav1.Condition{{Type: "ExistingOther"}},
				after:  []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
				after:  []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{
					{Type: "Registered"},
					{Type: "ExistingOther"},
					{Type: "RegistrationRequired"},
				},
				after: []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{{Type: "TokenRequired"}},
				after:  []metav1.Condition{},
			},
		} {
			for _, enabled := range []bool{false, true} {
				r.enabled = enabled

				conditions := append([]metav1.Condition{}, tt.before...)
				discard := new(events.Recorder)
				object := &corev1.ConfigMap{}

				result := r.Required(discard, object, &conditions)

				assert.Equal(t, result, false, "expected registration not required")
				assert.DeepEqual(t, conditions, tt.after)
			}
		}
	})
}

func TestRunnerRequiredEvents(t *testing.T) {
	t.Parallel()

	t.Run("RegistrationDisabled", func(t *testing.T) {
		r := Runner{enabled: false}

		for _, tt := range []struct {
			before []metav1.Condition
		}{
			{
				before: []metav1.Condition{},
			},
			{
				before: []metav1.Condition{{Type: "ExistingOther"}},
			},
			{
				before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
			},
		} {
			for _, exists := range []bool{false, true} {
				for _, expires := range []time.Time{
					time.Now().Add(time.Hour),
					time.Now().Add(-time.Hour),
				} {
					r.token.Exists = exists
					r.token.ExpiresAt = jwt.NewNumericDate(expires)

					conditions := append([]metav1.Condition{}, tt.before...)
					object := &corev1.ConfigMap{}
					recorder := events.NewRecorder(t, scheme.Scheme)

					result := r.Required(recorder, object, &conditions)

					assert.Equal(t, result, false, "expected registration not required")
					assert.Equal(t, len(recorder.Events), 0, "expected no events")
				}
			}
		}
	})

	t.Run("RegistrationRequired", func(t *testing.T) {
		r := Runner{enabled: true}

		t.Run("MissingToken", func(t *testing.T) {
			r.token.Exists = false

			for _, tt := range []struct {
				before []metav1.Condition
			}{
				{
					before: []metav1.Condition{},
				},
				{
					before: []metav1.Condition{{Type: "ExistingOther"}},
				},
				{
					before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
				},
			} {
				conditions := append([]metav1.Condition{}, tt.before...)
				object := &corev1.ConfigMap{}
				recorder := events.NewRecorder(t, scheme.Scheme)

				result := r.Required(recorder, object, &conditions)

				assert.Equal(t, result, true, "expected registration required")
				assert.Equal(t, len(recorder.Events), 0, "expected no events")
			}
		})

		t.Run("InvalidToken", func(t *testing.T) {
			r.token.Exists = true
			r.token.ExpiresAt = nil

			for _, tt := range []struct {
				before []metav1.Condition
			}{
				{
					before: []metav1.Condition{},
				},
				{
					before: []metav1.Condition{{Type: "ExistingOther"}},
				},
				{
					before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
				},
			} {
				conditions := append([]metav1.Condition{}, tt.before...)
				object := &corev1.ConfigMap{}
				recorder := events.NewRecorder(t, scheme.Scheme)

				result := r.Required(recorder, object, &conditions)

				assert.Equal(t, result, true, "expected registration required")
				assert.Equal(t, len(recorder.Events), 1, "expected one event")
				assert.Equal(t, recorder.Events[0].Type, "Warning")
				assert.Equal(t, recorder.Events[0].Reason, "Token Authentication Failed")
			}
		})
	})

	t.Run("Registered", func(t *testing.T) {
		r := Runner{}
		r.token.Exists = true
		r.token.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))

		t.Run("AlwaysRegistered", func(t *testing.T) {
			// No prior registration conditions
			for _, tt := range []struct {
				before []metav1.Condition
			}{
				{
					before: []metav1.Condition{},
				},
				{
					before: []metav1.Condition{{Type: "ExistingOther"}},
				},
			} {
				for _, enabled := range []bool{false, true} {
					r.enabled = enabled

					conditions := append([]metav1.Condition{}, tt.before...)
					object := &corev1.ConfigMap{}
					recorder := events.NewRecorder(t, scheme.Scheme)

					result := r.Required(recorder, object, &conditions)

					assert.Equal(t, result, false, "expected registration not required")
					assert.Equal(t, len(recorder.Events), 0, "expected no events")
				}
			}
		})

		t.Run("PreviouslyUnregistered", func(t *testing.T) {
			r.enabled = true

			// One or more prior registration conditions
			for _, tt := range []struct {
				before []metav1.Condition
			}{
				{
					before: []metav1.Condition{{Type: "Registered"}, {Type: "ExistingOther"}},
				},
				{
					before: []metav1.Condition{
						{Type: "Registered"},
						{Type: "ExistingOther"},
						{Type: "RegistrationRequired"},
					},
				},
				{
					before: []metav1.Condition{{Type: "TokenRequired"}},
				},
			} {
				conditions := append([]metav1.Condition{}, tt.before...)
				object := &corev1.ConfigMap{}
				recorder := events.NewRecorder(t, scheme.Scheme)

				result := r.Required(recorder, object, &conditions)

				assert.Equal(t, result, false, "expected registration not required")
				assert.Equal(t, len(recorder.Events), 1, "expected one event")
				assert.Equal(t, recorder.Events[0].Type, "Normal")
				assert.Equal(t, recorder.Events[0].Reason, "Token Verified")
			}
		})
	})
}

func TestRunnerStart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(key)
	assert.NilError(t, err)

	t.Run("DisabledDoesNothing", func(t *testing.T) {
		runner := &Runner{
			enabled: false,
			refresh: time.Nanosecond,
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		assert.ErrorIs(t, runner.Start(ctx), context.DeadlineExceeded,
			"expected it to block until context is canceled")
	})

	t.Run("WithCallback", func(t *testing.T) {
		called := false
		runner := &Runner{
			changed:   func() { called = true },
			enabled:   true,
			publicKey: &key.PublicKey,
			refresh:   time.Second,
			tokenPath: filepath.Join(dir, "token"),
		}

		// Begin with an invalid token.
		assert.NilError(t, os.WriteFile(runner.tokenPath, nil, 0o600))
		assert.Assert(t, runner.CheckToken() != nil)

		// Replace it with a valid token.
		assert.NilError(t, os.WriteFile(runner.tokenPath, []byte(token), 0o600))

		// Run with a timeout that exceeds the refresh interval.
		ctx, cancel := context.WithTimeout(context.Background(), runner.refresh*3/2)
		defer cancel()

		assert.ErrorIs(t, runner.Start(ctx), context.DeadlineExceeded)
		assert.Assert(t, called, "expected a call back")
	})
}
