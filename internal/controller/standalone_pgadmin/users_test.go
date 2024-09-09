// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilePGAdminUsers(t *testing.T) {
	ctx := context.Background()

	pgadmin := &v1beta1.PGAdmin{}
	pgadmin.Namespace = "ns1"
	pgadmin.Name = "pgadmin1"
	pgadmin.UID = "123"
	pgadmin.Spec.Users = []v1beta1.PGAdminUser{
		{
			Username: "testuser",
			Role:     "Administrator",
		},
	}

	t.Run("NoPods", func(t *testing.T) {
		r := new(PGAdminReconciler)
		r.Client = fake.NewClientBuilder().Build()
		assert.NilError(t, r.reconcilePGAdminUsers(ctx, pgadmin))
	})

	// Pod in the namespace
	pod := corev1.Pod{}
	pod.Namespace = pgadmin.Namespace
	pod.Name = fmt.Sprintf("pgadmin-%s-0", pgadmin.UID)

	t.Run("ContainerNotRunning", func(t *testing.T) {
		pod := pod.DeepCopy()

		pod.DeletionTimestamp = nil
		pod.Status.ContainerStatuses = nil

		r := new(PGAdminReconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, pgadmin))
	})

	t.Run("PodTerminating", func(t *testing.T) {
		pod := pod.DeepCopy()

		// Must add finalizer when adding deletion timestamp otherwise fake client will panic:
		// https://github.com/kubernetes-sigs/controller-runtime/pull/2316
		pod.Finalizers = append(pod.Finalizers, "some-finalizer")

		pod.DeletionTimestamp = new(metav1.Time)
		*pod.DeletionTimestamp = metav1.Now()
		pod.Status.ContainerStatuses =
			[]corev1.ContainerStatus{{Name: naming.ContainerPGAdmin}}
		pod.Status.ContainerStatuses[0].State.Running =
			new(corev1.ContainerStateRunning)

		r := new(PGAdminReconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, pgadmin))
	})

	// We only test v7 because if we did v8 then the writePGAdminUsers would
	// be called and that method has its own tests later in this file
	t.Run("PodHealthyVersionNotSet", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()
		pod := pod.DeepCopy()

		pod.DeletionTimestamp = nil
		pod.Status.ContainerStatuses =
			[]corev1.ContainerStatus{{Name: naming.ContainerPGAdmin}}
		pod.Status.ContainerStatuses[0].State.Running =
			new(corev1.ContainerStateRunning)
		pod.Status.ContainerStatuses[0].ImageID = "fakeSHA"

		r := new(PGAdminReconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		calls := 0
		r.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			assert.Equal(t, pod, "pgadmin-123-0")
			assert.Equal(t, namespace, pgadmin.Namespace)
			assert.Equal(t, container, naming.ContainerPGAdmin)

			// Simulate a v7 version of pgAdmin by setting stdout to "7" for
			// podexec call in reconcilePGAdminMajorVersion
			stdout.Write([]byte("7"))
			return nil
		}

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, pgadmin))
		assert.Equal(t, calls, 1, "PodExec should be called once")
		assert.Equal(t, pgadmin.Status.MajorVersion, 7)
		assert.Equal(t, pgadmin.Status.ImageSHA, "fakeSHA")
	})

	t.Run("PodHealthyShaChanged", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()
		pgadmin.Status.MajorVersion = 7
		pgadmin.Status.ImageSHA = "fakeSHA"
		pod := pod.DeepCopy()

		pod.DeletionTimestamp = nil
		pod.Status.ContainerStatuses =
			[]corev1.ContainerStatus{{Name: naming.ContainerPGAdmin}}
		pod.Status.ContainerStatuses[0].State.Running =
			new(corev1.ContainerStateRunning)
		pod.Status.ContainerStatuses[0].ImageID = "newFakeSHA"

		r := new(PGAdminReconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		calls := 0
		r.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			// Simulate a v7 version of pgAdmin by setting stdout to "7" for
			// podexec call in reconcilePGAdminMajorVersion
			stdout.Write([]byte("7"))
			return nil
		}

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, pgadmin))
		assert.Equal(t, calls, 1, "PodExec should be called once")
		assert.Equal(t, pgadmin.Status.MajorVersion, 7)
		assert.Equal(t, pgadmin.Status.ImageSHA, "newFakeSHA")
	})
}

func TestReconcilePGAdminMajorVersion(t *testing.T) {
	ctx := context.Background()
	pod := corev1.Pod{}
	pod.Namespace = "test-namespace"
	pod.Name = "pgadmin-123-0"
	reconciler := &PGAdminReconciler{}

	podExecutor := func(
		ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return reconciler.PodExec(ctx, pod.Namespace, pod.Name, "pgadmin", stdin, stdout, stderr, command...)
	}

	t.Run("SuccessfulRetrieval", func(t *testing.T) {
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.Equal(t, pod, "pgadmin-123-0")
			assert.Equal(t, namespace, "test-namespace")
			assert.Equal(t, container, naming.ContainerPGAdmin)

			// Simulate a v7 version of pgAdmin by setting stdout to "7" for
			// podexec call in reconcilePGAdminMajorVersion
			stdout.Write([]byte("7"))
			return nil
		}

		version, err := reconciler.reconcilePGAdminMajorVersion(ctx, podExecutor)
		assert.NilError(t, err)
		assert.Equal(t, version, 7)
	})

	t.Run("FailedRetrieval", func(t *testing.T) {
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			// Simulate the python call giving bad data (not a version int)
			stdout.Write([]byte("asdfjkl;"))
			return nil
		}

		version, err := reconciler.reconcilePGAdminMajorVersion(ctx, podExecutor)
		assert.Check(t, err != nil)
		assert.Equal(t, version, 0)
	})

	t.Run("PodExecError", func(t *testing.T) {
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			return errors.New("PodExecError")
		}

		version, err := reconciler.reconcilePGAdminMajorVersion(ctx, podExecutor)
		assert.Check(t, err != nil)
		assert.Equal(t, version, 0)
	})
}

func TestWritePGAdminUsers(t *testing.T) {
	ctx := context.Background()
	cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	recorder := events.NewRecorder(t, runtime.Scheme)
	reconciler := &PGAdminReconciler{
		Client:   cc,
		Owner:    client.FieldOwner(t.Name()),
		Recorder: recorder,
	}

	ns := setupNamespace(t, cc)
	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Name = "test-standalone-pgadmin"
	pgadmin.Namespace = ns.Name
	assert.NilError(t, cc.Create(ctx, pgadmin))

	userPasswordSecret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-password-secret1",
			Namespace: ns.Name,
		},
		Data: map[string][]byte{
			"password": []byte(`asdf`),
		},
	}
	assert.NilError(t, cc.Create(ctx, userPasswordSecret1))

	userPasswordSecret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-password-secret2",
			Namespace: ns.Name,
		},
		Data: map[string][]byte{
			"password": []byte(`qwer`),
		},
	}
	assert.NilError(t, cc.Create(ctx, userPasswordSecret2))

	t.Cleanup(func() {
		assert.Check(t, cc.Delete(ctx, pgadmin))
		assert.Check(t, cc.Delete(ctx, userPasswordSecret1))
		assert.Check(t, cc.Delete(ctx, userPasswordSecret2))
	})

	pod := corev1.Pod{}
	pod.Namespace = pgadmin.Namespace
	pod.Name = fmt.Sprintf("pgadmin-%s-0", pgadmin.UID)

	podExecutor := func(
		ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return reconciler.PodExec(ctx, pod.Namespace, pod.Name, "pgadmin", stdin, stdout, stderr, command...)
	}

	t.Run("CreateOneUser", func(t *testing.T) {
		pgadmin.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser1",
				Role:     "Administrator",
			},
		}

		calls := 0
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			assert.Equal(t, pod, fmt.Sprintf("pgadmin-%s-0", pgadmin.UID))
			assert.Equal(t, namespace, pgadmin.Namespace)
			assert.Equal(t, container, naming.ContainerPGAdmin)
			assert.Equal(t, strings.Contains(strings.Join(command, " "),
				`python3 setup.py add-user --admin -- "testuser1" "asdf"`), true)

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 1, "PodExec should be called once")

		secret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, true)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}
	})

	t.Run("AddAnotherUserEditExistingUser", func(t *testing.T) {
		pgadmin.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser1",
				Role:     "User",
			},
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret2",
					},
					Key: "password",
				},
				Username: "testuser2",
				Role:     "Administrator",
			},
		}

		calls := 0
		addUserCalls := 0
		updateUserCalls := 0
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++
			if strings.Contains(strings.Join(command, " "), "python3 setup.py add-user") {
				addUserCalls++
			}
			if strings.Contains(strings.Join(command, " "), "python3 setup.py update-user") {
				updateUserCalls++
			}

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 2, "PodExec should be called twice")
		assert.Equal(t, addUserCalls, 1, "The add-user command should be executed once")
		assert.Equal(t, updateUserCalls, 1, "The update-user command should be executed once")

		secret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 2)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
			assert.Equal(t, usersArr[1].Username, "testuser2")
			assert.Equal(t, usersArr[1].IsAdmin, true)
			assert.Equal(t, usersArr[1].Password, "qwer")
		}
	})

	t.Run("AddOneEditOneLeaveOneAlone", func(t *testing.T) {
		pgadmin.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser1",
				Role:     "User",
			},
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser2",
				Role:     "User",
			},
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret2",
					},
					Key: "password",
				},
				Username: "testuser3",
				Role:     "Administrator",
			},
		}
		calls := 0
		addUserCalls := 0
		updateUserCalls := 0
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++
			if strings.Contains(strings.Join(command, " "), "python3 setup.py add-user") {
				addUserCalls++
			}
			if strings.Contains(strings.Join(command, " "), "python3 setup.py update-user") {
				updateUserCalls++
			}

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 2, "PodExec should be called twice")
		assert.Equal(t, addUserCalls, 1, "The add-user command should be executed once")
		assert.Equal(t, updateUserCalls, 1, "The update-user command should be executed once")

		secret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 3)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
			assert.Equal(t, usersArr[1].Username, "testuser2")
			assert.Equal(t, usersArr[1].IsAdmin, false)
			assert.Equal(t, usersArr[1].Password, "asdf")
			assert.Equal(t, usersArr[2].Username, "testuser3")
			assert.Equal(t, usersArr[2].IsAdmin, true)
			assert.Equal(t, usersArr[2].Password, "qwer")
		}
	})

	t.Run("DeleteUsers", func(t *testing.T) {
		pgadmin.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser1",
				Role:     "User",
			},
		}
		calls := 0
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 0, "PodExec should be called zero times")

		secret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}
	})

	t.Run("ErrorsWhenUpdating", func(t *testing.T) {
		pgadmin.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser1",
				Role:     "Administrator",
			},
		}

		// PodExec error
		calls := 0
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			return errors.New("podexec failure")
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 1, "PodExec should be called once")

		// User in users.json should be unchanged
		secret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}

		// setup.py error in stderr
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			stderr.Write([]byte("issue running setup.py update-user command"))

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 2, "PodExec should be called once more")

		// User in users.json should be unchanged
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}
	})

	t.Run("ErrorsWhenAdding", func(t *testing.T) {
		pgadmin.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret1",
					},
					Key: "password",
				},
				Username: "testuser1",
				Role:     "User",
			},
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "user-password-secret2",
					},
					Key: "password",
				},
				Username: "testuser2",
				Role:     "Administrator",
			},
		}

		// PodExec error
		calls := 0
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			return errors.New("podexec failure")
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 1, "PodExec should be called once")

		// User in users.json should be unchanged and attempt to add user should not
		// have succeeded
		secret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}

		// setup.py error in stderr
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			stderr.Write([]byte("issue running setup.py add-user command"))

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 2, "PodExec should be called once more")

		// User in users.json should be unchanged and attempt to add user should not
		// have succeeded
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}

		// setup.py error in stdout regarding email address
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			stdout.Write([]byte("Invalid email address"))

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 3, "PodExec should be called once more")

		// User in users.json should be unchanged and attempt to add user should not
		// have succeeded
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}
		assert.Equal(t, len(recorder.Events), 1)

		// setup.py error in stdout regarding password
		reconciler.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			stdout.Write([]byte("Password must be at least 6 characters long"))

			return nil
		}

		assert.NilError(t, reconciler.writePGAdminUsers(ctx, pgadmin, podExecutor))
		assert.Equal(t, calls, 4, "PodExec should be called once more")

		// User in users.json should be unchanged and attempt to add user should not
		// have succeeded
		assert.NilError(t, errors.WithStack(
			reconciler.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret)))
		if assert.Check(t, secret.Data["users.json"] != nil) {
			var usersArr []pgAdminUserForJson
			assert.NilError(t, json.Unmarshal(secret.Data["users.json"], &usersArr))
			assert.Equal(t, len(usersArr), 1)
			assert.Equal(t, usersArr[0].Username, "testuser1")
			assert.Equal(t, usersArr[0].IsAdmin, false)
			assert.Equal(t, usersArr[0].Password, "asdf")
		}
		assert.Equal(t, len(recorder.Events), 2)
	})
}
