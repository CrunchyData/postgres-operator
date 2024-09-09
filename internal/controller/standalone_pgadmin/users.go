// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type Executor func(
	ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// pgAdminUserForJson is used for user data that is put in the users.json file in the
// pgAdmin secret. IsAdmin and Username come from the user spec, whereas Password is
// generated when the user is created.
type pgAdminUserForJson struct {
	// Whether the user has admin privileges or not.
	IsAdmin bool `json:"isAdmin"`

	// The user's password
	Password string `json:"password"`

	// The username for User in pgAdmin.
	// Must be unique in the pgAdmin's users list.
	Username string `json:"username"`
}

// reconcilePGAdminUsers reconciles the users listed in the pgAdmin spec, adding them
// to the pgAdmin secret, and creating/updating them in pgAdmin when appropriate.
func (r *PGAdminReconciler) reconcilePGAdminUsers(ctx context.Context, pgadmin *v1beta1.PGAdmin) error {
	const container = naming.ContainerPGAdmin
	var podExecutor Executor
	log := logging.FromContext(ctx)

	// Find the running pgAdmin container. When there is none, return early.
	pod := &corev1.Pod{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	pod.Name += "-0"

	err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(pod), pod))
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	var running bool
	var pgAdminImageSha string
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == container {
			running = status.State.Running != nil
			pgAdminImageSha = status.ImageID
		}
	}
	if terminating := pod.DeletionTimestamp != nil; running && !terminating {
		ctx = logging.NewContext(ctx, logging.FromContext(ctx).WithValues("pod", pod.Name))

		podExecutor = func(
			ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			return r.PodExec(ctx, pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
		}
	}
	if podExecutor == nil {
		return nil
	}

	// If the pgAdmin version is not in the status or the image SHA has changed, get
	// the pgAdmin version and store it in the status.
	var pgadminVersion int
	if pgadmin.Status.MajorVersion == 0 || pgadmin.Status.ImageSHA != pgAdminImageSha {
		pgadminVersion, err = r.reconcilePGAdminMajorVersion(ctx, podExecutor)
		if err != nil {
			return err
		}
		pgadmin.Status.MajorVersion = pgadminVersion
		pgadmin.Status.ImageSHA = pgAdminImageSha
	} else {
		pgadminVersion = pgadmin.Status.MajorVersion
	}

	// If the pgAdmin version is not v8 or higher, return early as user management is
	// only supported for pgAdmin v8 and higher.
	if pgadminVersion < 8 {
		// If pgAdmin version is less than v8 and user management is being attempted,
		// log a message clarifying that it is only supported for pgAdmin v8 and higher.
		if len(pgadmin.Spec.Users) > 0 {
			log.Info("User management is only supported for pgAdmin v8 and higher.",
				"pgadminVersion", pgadminVersion)
		}
		return err
	}

	return r.writePGAdminUsers(ctx, pgadmin, podExecutor)
}

// reconcilePGAdminMajorVersion execs into the pgAdmin pod and retrieves the pgAdmin major version
func (r *PGAdminReconciler) reconcilePGAdminMajorVersion(ctx context.Context, exec Executor) (int, error) {
	script := fmt.Sprintf(`
PGADMIN_DIR=%s
cd $PGADMIN_DIR && python3 -c "import config; print(config.APP_RELEASE)"
`, pgAdminDir)

	var stdin, stdout, stderr bytes.Buffer

	err := exec(ctx, &stdin, &stdout, &stderr,
		[]string{"bash", "-ceu", "--", script}...)

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(strings.TrimSpace(stdout.String()))
}

// writePGAdminUsers takes the users in the pgAdmin spec and writes (adds or updates) their data
// to both pgAdmin and the users.json file that is stored in the pgAdmin secret. If a user is
// removed from the spec, its data is removed from users.json, but it is not deleted from pgAdmin.
func (r *PGAdminReconciler) writePGAdminUsers(ctx context.Context, pgadmin *v1beta1.PGAdmin,
	exec Executor) error {
	log := logging.FromContext(ctx)

	existingUserSecret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	err := errors.WithStack(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existingUserSecret), existingUserSecret))
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	intentUserSecret := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	intentUserSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))

	intentUserSecret.Annotations = naming.Merge(
		pgadmin.Spec.Metadata.GetAnnotationsOrNil(),
	)
	intentUserSecret.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminLabels(pgadmin.Name))

	// Initialize secret data map, or copy existing data if not nil
	intentUserSecret.Data = make(map[string][]byte)

	setupScript := fmt.Sprintf(`
PGADMIN_DIR=%s
cd $PGADMIN_DIR
`, pgAdminDir)

	var existingUsersArr []pgAdminUserForJson
	if existingUserSecret.Data["users.json"] != nil {
		err := json.Unmarshal(existingUserSecret.Data["users.json"], &existingUsersArr)
		if err != nil {
			return err
		}
	}
	existingUsersMap := make(map[string]pgAdminUserForJson)
	for _, user := range existingUsersArr {
		existingUsersMap[user.Username] = user
	}
	intentUsers := []pgAdminUserForJson{}
	for _, user := range pgadmin.Spec.Users {
		var stdin, stdout, stderr bytes.Buffer
		typeFlag := "--nonadmin"
		isAdmin := false
		if user.Role == "Administrator" {
			typeFlag = "--admin"
			isAdmin = true
		}

		// Get password from secret
		userPasswordSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
			Namespace: pgadmin.Namespace,
			Name:      user.PasswordRef.LocalObjectReference.Name,
		}}
		err := errors.WithStack(
			r.Client.Get(ctx, client.ObjectKeyFromObject(userPasswordSecret), userPasswordSecret))
		if err != nil {
			log.Error(err, "Could not get user password secret")
			continue
		}

		// Make sure the password isn't nil or empty
		password := userPasswordSecret.Data[user.PasswordRef.Key]
		if password == nil {
			log.Error(nil, `Could not retrieve password from secret. Make sure secret name and key are correct.`)
			continue
		}
		if len(password) == 0 {
			log.Error(nil, `Password must not be empty.`)
			continue
		}

		// Assemble user that will be used in add/update command and in updating
		// the users.json file in the secret
		intentUser := pgAdminUserForJson{
			Username: user.Username,
			Password: string(password),
			IsAdmin:  isAdmin,
		}
		// If the user already exists in users.json and isAdmin or password has
		// changed, run the update-user command. If the user already exists in
		// users.json, but it hasn't changed, do nothing. If the user doesn't
		// exist in users.json, run the add-user command.
		if existingUser, present := existingUsersMap[user.Username]; present {
			// If Password or IsAdmin have changed, attempt update-user command
			if intentUser.IsAdmin != existingUser.IsAdmin || intentUser.Password != existingUser.Password {
				script := setupScript + fmt.Sprintf(`python3 setup.py update-user %s --password "%s" "%s"`,
					typeFlag, intentUser.Password, intentUser.Username) + "\n"
				err = exec(ctx, &stdin, &stdout, &stderr,
					[]string{"bash", "-ceu", "--", script}...)

				// If any errors occurred during update, we want to log a message,
				// add the existing user to users.json since the update was
				// unsuccessful, and continue reconciling users.
				if err != nil {
					log.Error(err, "PodExec failed: ")
					intentUsers = append(intentUsers, existingUser)
					continue
				} else if strings.TrimSpace(stderr.String()) != "" {
					log.Error(errors.New(stderr.String()), fmt.Sprintf("pgAdmin setup.py error for %s: ",
						intentUser.Username))
					intentUsers = append(intentUsers, existingUser)
					continue
				}
				// If update user fails due to user not found or password length:
				// https://github.com/pgadmin-org/pgadmin4/blob/REL-8_5/web/setup.py#L263
				// https://github.com/pgadmin-org/pgadmin4/blob/REL-8_5/web/setup.py#L246
				if strings.Contains(stdout.String(), "User not found") ||
					strings.Contains(stdout.String(), "Password must be") {

					log.Info("Failed to update pgAdmin user", "user", intentUser.Username, "error", stdout.String())
					r.Recorder.Event(pgadmin,
						corev1.EventTypeWarning, "InvalidUserWarning",
						fmt.Sprintf("Failed to update pgAdmin user %s: %s",
							intentUser.Username, stdout.String()))
					intentUsers = append(intentUsers, existingUser)
					continue
				}
			}
		} else {
			// New user, so attempt add-user command
			script := setupScript + fmt.Sprintf(`python3 setup.py add-user %s -- "%s" "%s"`,
				typeFlag, intentUser.Username, intentUser.Password) + "\n"
			err = exec(ctx, &stdin, &stdout, &stderr,
				[]string{"bash", "-ceu", "--", script}...)

			// If any errors occurred when attempting to add user, we want to log a message,
			// and continue reconciling users.
			if err != nil {
				log.Error(err, "PodExec failed: ")
				continue
			}
			if strings.TrimSpace(stderr.String()) != "" {
				log.Error(errors.New(stderr.String()), fmt.Sprintf("pgAdmin setup.py error for %s: ",
					intentUser.Username))
				continue
			}
			// If add user fails due to invalid username or password length:
			// https://github.com/pgadmin-org/pgadmin4/blob/REL-8_5/web/pgadmin/tools/user_management/__init__.py#L457
			// https://github.com/pgadmin-org/pgadmin4/blob/REL-8_5/web/setup.py#L374
			if strings.Contains(stdout.String(), "Invalid email address") ||
				strings.Contains(stdout.String(), "Password must be") {

				log.Info(fmt.Sprintf("Failed to create pgAdmin user %s: %s",
					intentUser.Username, stdout.String()))
				r.Recorder.Event(pgadmin,
					corev1.EventTypeWarning, "InvalidUserWarning",
					fmt.Sprintf("Failed to create pgAdmin user %s: %s",
						intentUser.Username, stdout.String()))
				continue
			}
		}
		// If we've gotten here, the user was successfully added or updated or nothing was done
		// to the user at all, so we want to add it to the slice of users that will be put in the
		// users.json file in the secret.
		intentUsers = append(intentUsers, intentUser)
	}

	// We've at least attempted to reconcile all users in the spec. If errors occurred when attempting
	// to add a user, that user will not be in intentUsers. If errors occurred when attempting to
	// update a user, the user will be in intentUsers as it existed before. We now want to marshal the
	// intentUsers to json and write the users.json file to the secret.
	usersJSON, err := json.Marshal(intentUsers)
	if err != nil {
		return err
	}
	intentUserSecret.Data["users.json"] = usersJSON

	err = errors.WithStack(r.setControllerReference(pgadmin, intentUserSecret))
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intentUserSecret))
	}

	return err
}
