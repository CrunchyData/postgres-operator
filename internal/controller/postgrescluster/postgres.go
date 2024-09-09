// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgaudit"
	"github.com/crunchydata/postgres-operator/internal/postgis"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// generatePostgresUserSecret returns a Secret containing a password and
// connection details for the first database in spec. When existing is nil or
// lacks a password or verifier, a new password and verifier are generated.
func (r *Reconciler) generatePostgresUserSecret(
	cluster *v1beta1.PostgresCluster, spec *v1beta1.PostgresUserSpec, existing *corev1.Secret,
) (*corev1.Secret, error) {
	username := string(spec.Name)
	intent := &corev1.Secret{ObjectMeta: naming.PostgresUserSecret(cluster, username)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	initialize.ByteMap(&intent.Data)

	// Populate the Secret with libpq keywords for connecting through
	// the primary Service.
	// - https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS
	primary := naming.ClusterPrimaryService(cluster)
	hostname := primary.Name + "." + primary.Namespace + ".svc"
	port := fmt.Sprint(*cluster.Spec.Port)

	intent.Data["host"] = []byte(hostname)
	intent.Data["port"] = []byte(port)
	intent.Data["user"] = []byte(username)

	// Use the existing password and verifier.
	if existing != nil {
		intent.Data["password"] = existing.Data["password"]
		intent.Data["verifier"] = existing.Data["verifier"]
	}

	// When password is unset, generate a new one according to the specified policy.
	if len(intent.Data["password"]) == 0 {
		// NOTE: The tests around ASCII passwords are lacking. When changing
		// this, make sure that ASCII is the default.
		generate := util.GenerateASCIIPassword
		if spec.Password != nil {
			switch spec.Password.Type {
			case v1beta1.PostgresPasswordTypeAlphaNumeric:
				generate = util.GenerateAlphaNumericPassword
			}
		}

		password, err := generate(util.DefaultGeneratedPasswordLength)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		intent.Data["password"] = []byte(password)
		intent.Data["verifier"] = nil
	}

	// When a password has been generated or the verifier is empty,
	// generate a verifier based on the current password.
	// NOTE(cbandy): We don't have a function to compare a plaintext
	// password to a SCRAM verifier.
	if len(intent.Data["verifier"]) == 0 {
		verifier, err := pgpassword.NewSCRAMPassword(string(intent.Data["password"])).Build()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		intent.Data["verifier"] = []byte(verifier)
	}

	// When a database has been specified, include it and a connection URI.
	// - https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
	if len(spec.Databases) > 0 {
		database := string(spec.Databases[0])

		intent.Data["dbname"] = []byte(database)
		intent.Data["uri"] = []byte((&url.URL{
			Scheme: "postgresql",
			User:   url.UserPassword(username, string(intent.Data["password"])),
			Host:   net.JoinHostPort(hostname, port),
			Path:   database,
		}).String())

		// The JDBC driver requires a different URI scheme and query component.
		// - https://jdbc.postgresql.org/documentation/use/#connection-parameters
		query := url.Values{}
		query.Set("user", username)
		query.Set("password", string(intent.Data["password"]))
		intent.Data["jdbc-uri"] = []byte((&url.URL{
			Scheme:   "jdbc:postgresql",
			Host:     net.JoinHostPort(hostname, port),
			Path:     database,
			RawQuery: query.Encode(),
		}).String())
	}

	// When PgBouncer is enabled, include values for connecting through it.
	if cluster.Spec.Proxy != nil && cluster.Spec.Proxy.PGBouncer != nil {
		pgBouncer := naming.ClusterPGBouncer(cluster)
		hostname := pgBouncer.Name + "." + pgBouncer.Namespace + ".svc"
		port := fmt.Sprint(*cluster.Spec.Proxy.PGBouncer.Port)

		intent.Data["pgbouncer-host"] = []byte(hostname)
		intent.Data["pgbouncer-port"] = []byte(port)

		if len(spec.Databases) > 0 {
			database := string(spec.Databases[0])

			intent.Data["pgbouncer-uri"] = []byte((&url.URL{
				Scheme: "postgresql",
				User:   url.UserPassword(username, string(intent.Data["password"])),
				Host:   net.JoinHostPort(hostname, port),
				Path:   database,
			}).String())

			// The JDBC driver requires a different URI scheme and query component.
			// Disable prepared statements to be compatible with PgBouncer's
			// transaction pooling.
			// - https://jdbc.postgresql.org/documentation/use/#connection-parameters
			// - https://www.pgbouncer.org/faq.html#how-to-use-prepared-statements-with-transaction-pooling
			query := url.Values{}
			query.Set("user", username)
			query.Set("password", string(intent.Data["password"]))
			query.Set("prepareThreshold", "0")
			intent.Data["pgbouncer-jdbc-uri"] = []byte((&url.URL{
				Scheme:   "jdbc:postgresql",
				Host:     net.JoinHostPort(hostname, port),
				Path:     database,
				RawQuery: query.Encode(),
			}).String())
		}
	}

	intent.Annotations = cluster.Spec.Metadata.GetAnnotationsOrNil()
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:      cluster.Name,
			naming.LabelRole:         naming.RolePostgresUser,
			naming.LabelPostgresUser: username,
		})

	err := errors.WithStack(r.setControllerReference(cluster, intent))

	return intent, err
}

// reconcilePostgresDatabases creates databases inside of PostgreSQL.
func (r *Reconciler) reconcilePostgresDatabases(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
) error {
	const container = naming.ContainerDatabase
	var podExecutor postgres.Executor

	// Find the PostgreSQL instance that can execute SQL that writes system
	// catalogs. When there is none, return early.
	pod, _ := instances.writablePod(container)
	if pod == nil {
		return nil
	}

	ctx = logging.NewContext(ctx, logging.FromContext(ctx).WithValues("pod", pod.Name))
	podExecutor = func(
		ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return r.PodExec(ctx, pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
	}

	// Gather the list of database that should exist in PostgreSQL.

	databases := sets.Set[string]{}
	if cluster.Spec.Users == nil {
		// Users are unspecified; create one database matching the cluster name
		// if it is also a valid database name.
		// TODO(cbandy): Move this to a defaulting (mutating admission) webhook
		// to leverage regular validation.
		path := field.NewPath("spec", "users").Index(0).Child("databases").Index(0)

		// Database names cannot be too long. PostgresCluster.Name is a DNS
		// subdomain, so use len() to count characters.
		if n := len(cluster.Name); n > 63 {
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "InvalidDatabase",
				field.Invalid(path, cluster.Name,
					fmt.Sprintf("should be at most %d chars long", 63)).Error())
		} else {
			databases.Insert(cluster.Name)
		}
	} else {
		for _, user := range cluster.Spec.Users {
			for _, database := range user.Databases {
				databases.Insert(string(database))
			}
		}
	}

	var pgAuditOK, postgisInstallOK bool
	create := func(ctx context.Context, exec postgres.Executor) error {
		if pgAuditOK = pgaudit.EnableInPostgreSQL(ctx, exec) == nil; !pgAuditOK {
			// pgAudit can only be enabled after its shared library is loaded,
			// but early versions of PGO do not load it automatically. Assume
			// that an error here is because the cluster started during one of
			// those versions and has not been restarted.
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "pgAuditDisabled",
				"Unable to install pgAudit")
		}

		// Enabling PostGIS extensions is a one-way operation
		// e.g., you can take a PostgresCluster and turn it into a PostGISCluster,
		// but you cannot reverse the process, as that would potentially remove an extension
		// that is being used by some database/tables
		if cluster.Spec.PostGISVersion == "" {
			postgisInstallOK = true
		} else if postgisInstallOK = postgis.EnableInPostgreSQL(ctx, exec) == nil; !postgisInstallOK {
			// TODO(benjaminjb): Investigate under what conditions postgis would fail install
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "PostGISDisabled",
				"Unable to install PostGIS")
		}

		return postgres.CreateDatabasesInPostgreSQL(ctx, exec, sets.List(databases))
	}

	// Calculate a hash of the SQL that should be executed in PostgreSQL.
	revision, err := safeHash32(func(hasher io.Writer) error {
		// Discard log messages about executing SQL.
		return create(logging.NewContext(ctx, logging.Discard()), func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			_, err := fmt.Fprint(hasher, command)
			if err == nil && stdin != nil {
				_, err = io.Copy(hasher, stdin)
			}
			return err
		})
	})

	if err == nil && revision == cluster.Status.DatabaseRevision {
		// The necessary SQL has already been applied; there's nothing more to do.

		// TODO(cbandy): Give the user a way to trigger execution regardless.
		// The value of an annotation could influence the hash, for example.
		return nil
	}

	// Apply the necessary SQL and record its hash in cluster.Status. Include
	// the hash in any log messages.

	if err == nil {
		log := logging.FromContext(ctx).WithValues("revision", revision)
		err = errors.WithStack(create(logging.NewContext(ctx, log), podExecutor))
	}
	if err == nil && pgAuditOK && postgisInstallOK {
		cluster.Status.DatabaseRevision = revision
	}

	return err
}

// reconcilePostgresUsers writes the objects necessary to manage users and their
// passwords in PostgreSQL.
func (r *Reconciler) reconcilePostgresUsers(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
) error {
	r.validatePostgresUsers(cluster)

	users, secrets, err := r.reconcilePostgresUserSecrets(ctx, cluster)
	if err == nil {
		err = r.reconcilePostgresUsersInPostgreSQL(ctx, cluster, instances, users, secrets)
	}
	if err == nil {
		// Copy PostgreSQL users and passwords into pgAdmin. This is here because
		// reconcilePostgresUserSecrets is building a (default) PostgresUserSpec
		// that is not in the PostgresClusterSpec. The freshly generated Secrets
		// are available here, too.
		err = r.reconcilePGAdminUsers(ctx, cluster, users, secrets)
	}
	return err
}

// validatePostgresUsers emits warnings when cluster.Spec.Users contains values
// that are no longer valid. NOTE(ratcheting) NOTE(validation)
// - https://docs.k8s.io/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-ratcheting
func (r *Reconciler) validatePostgresUsers(cluster *v1beta1.PostgresCluster) {
	if len(cluster.Spec.Users) == 0 {
		return
	}

	path := field.NewPath("spec", "users")
	reComments := regexp.MustCompile(`(?:--|/[*]|[*]/)`)
	rePassword := regexp.MustCompile(`(?i:PASSWORD)`)

	for i := range cluster.Spec.Users {
		errs := field.ErrorList{}
		spec := cluster.Spec.Users[i]

		if reComments.MatchString(spec.Options) {
			errs = append(errs,
				field.Invalid(path.Index(i).Child("options"), spec.Options,
					"cannot contain comments"))
		}
		if rePassword.MatchString(spec.Options) {
			errs = append(errs,
				field.Invalid(path.Index(i).Child("options"), spec.Options,
					"cannot assign password"))
		}

		if len(errs) > 0 {
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "InvalidUser",
				errs.ToAggregate().Error())
		}
	}
}

// +kubebuilder:rbac:groups="",resources="secrets",verbs={list}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,delete,patch}

// reconcilePostgresUserSecrets writes Secrets for the PostgreSQL users
// specified in cluster and deletes existing Secrets that are not specified.
// It returns the user specifications it acted on (because defaults) and the
// Secrets it wrote.
func (r *Reconciler) reconcilePostgresUserSecrets(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (
	[]v1beta1.PostgresUserSpec, map[string]*corev1.Secret, error,
) {
	// When users are unspecified, create one user matching the cluster name if
	// it is also a valid user name.
	// TODO(cbandy): Move this to a defaulting (mutating admission) webhook to
	// leverage regular validation.
	specUsers := cluster.Spec.Users
	if specUsers == nil {
		path := field.NewPath("spec", "users").Index(0).Child("name")
		reUser := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
		allErrors := field.ErrorList{}

		// User names cannot be too long. PostgresCluster.Name is a DNS
		// subdomain, so use len() to count characters.
		if n := len(cluster.Name); n > 63 {
			allErrors = append(allErrors,
				field.Invalid(path, cluster.Name,
					fmt.Sprintf("should be at most %d chars long", 63)))
		}
		// See v1beta1.PostgresRoleSpec validation markers.
		if !reUser.MatchString(cluster.Name) {
			allErrors = append(allErrors,
				field.Invalid(path, cluster.Name,
					fmt.Sprintf("should match '%s'", reUser)))
		}

		if len(allErrors) > 0 {
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "InvalidUser",
				allErrors.ToAggregate().Error())
		} else {
			identifier := v1beta1.PostgresIdentifier(cluster.Name)
			specUsers = []v1beta1.PostgresUserSpec{{
				Name:      identifier,
				Databases: []v1beta1.PostgresIdentifier{identifier},
			}}
		}
	}

	// Index user specifications by PostgreSQL user name.
	userSpecs := make(map[string]*v1beta1.PostgresUserSpec, len(specUsers))
	for i := range specUsers {
		userSpecs[string(specUsers[i].Name)] = &specUsers[i]
	}

	secrets := &corev1.SecretList{}
	selector, err := naming.AsSelector(naming.ClusterPostgresUsers(cluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, secrets,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	// Sorts the slice of secrets.Items based on secrets with identical labels
	// If one secret has "pguser" in its name and the other does not, the
	// one without "pguser" is moved to the front.
	// If both secrets have "pguser" in their names or neither has "pguser", they
	// are sorted by creation timestamp.
	// If two secrets have the same creation timestamp, they are further sorted by name.
	// The secret to be used by PGO is put at the end of the sorted slice.
	sort.Slice(secrets.Items, func(i, j int) bool {
		// Check if either secrets have "pguser" in their names
		isIPgUser := strings.Contains(secrets.Items[i].Name, "pguser")
		isJPgUser := strings.Contains(secrets.Items[j].Name, "pguser")

		// If one secret has "pguser" and the other does not,
		// move the one without "pguser" to the front
		if isIPgUser && !isJPgUser {
			return false
		} else if !isIPgUser && isJPgUser {
			return true
		}

		if secrets.Items[i].CreationTimestamp.Time.Equal(secrets.Items[j].CreationTimestamp.Time) {
			// If the creation timestamps are equal, sort by name
			return secrets.Items[i].Name < secrets.Items[j].Name
		}

		// If both secrets have "pguser" or neither have "pguser",
		// sort by creation timestamp
		return secrets.Items[i].CreationTimestamp.Time.After(secrets.Items[j].CreationTimestamp.Time)
	})

	// Index secrets by PostgreSQL user name and delete any that are not in the
	// cluster spec. Keep track of the deprecated default secret to migrate its
	// contents when the current secret doesn't exist.
	var (
		defaultSecret     *corev1.Secret
		defaultSecretName = naming.DeprecatedPostgresUserSecret(cluster).Name
		defaultUserName   string
		userSecrets       = make(map[string]*corev1.Secret, len(secrets.Items))
	)
	if err == nil {
		for i := range secrets.Items {
			secret := &secrets.Items[i]
			secretUserName := secret.Labels[naming.LabelPostgresUser]

			if _, specified := userSpecs[secretUserName]; specified {
				if secret.Name == defaultSecretName {
					defaultSecret = secret
					defaultUserName = secretUserName
				} else {
					userSecrets[secretUserName] = secret
				}
			} else if err == nil {
				err = errors.WithStack(r.deleteControlled(ctx, cluster, secret))
			}
		}
	}

	// Reconcile each PostgreSQL user in the cluster spec.
	for userName, user := range userSpecs {
		secret := userSecrets[userName]

		if secret == nil && userName == defaultUserName {
			// The current secret doesn't exist, so read from the deprecated
			// default secret, if any.
			secret = defaultSecret
		}

		if err == nil {
			userSecrets[userName], err = r.generatePostgresUserSecret(cluster, user, secret)
		}
		if err == nil {
			err = errors.WithStack(r.apply(ctx, userSecrets[userName]))
		}
	}

	return specUsers, userSecrets, err
}

// reconcilePostgresUsersInPostgreSQL creates users inside of PostgreSQL and
// sets their options and database access as specified.
func (r *Reconciler) reconcilePostgresUsersInPostgreSQL(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
	specUsers []v1beta1.PostgresUserSpec, userSecrets map[string]*corev1.Secret,
) error {
	const container = naming.ContainerDatabase
	var podExecutor postgres.Executor

	// Find the PostgreSQL instance that can execute SQL that writes system
	// catalogs. When there is none, return early.

	for _, instance := range instances.forCluster {
		if terminating, known := instance.IsTerminating(); terminating || !known {
			continue
		}
		if writable, known := instance.IsWritable(); !writable || !known {
			continue
		}
		running, known := instance.IsRunning(container)
		if running && known && len(instance.Pods) > 0 {
			pod := instance.Pods[0]
			ctx = logging.NewContext(ctx, logging.FromContext(ctx).WithValues("pod", pod.Name))

			podExecutor = func(
				ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
			) error {
				return r.PodExec(ctx, pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
			}
			break
		}
	}
	if podExecutor == nil {
		return nil
	}

	// Calculate a hash of the SQL that should be executed in PostgreSQL.

	verifiers := make(map[string]string, len(userSecrets))
	for userName := range userSecrets {
		verifiers[userName] = string(userSecrets[userName].Data["verifier"])
	}

	write := func(ctx context.Context, exec postgres.Executor) error {
		return postgres.WriteUsersInPostgreSQL(ctx, cluster, exec, specUsers, verifiers)
	}

	revision, err := safeHash32(func(hasher io.Writer) error {
		// Discard log messages about executing SQL.
		return write(logging.NewContext(ctx, logging.Discard()), func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			_, err := fmt.Fprint(hasher, command)
			if err == nil && stdin != nil {
				_, err = io.Copy(hasher, stdin)
			}
			return err
		})
	})

	if err == nil && revision == cluster.Status.UsersRevision {
		// The necessary SQL has already been applied; there's nothing more to do.

		// TODO(cbandy): Give the user a way to trigger execution regardless.
		// The value of an annotation could influence the hash, for example.
		return nil
	}

	// Apply the necessary SQL and record its hash in cluster.Status. Include
	// the hash in any log messages.

	if err == nil {
		log := logging.FromContext(ctx).WithValues("revision", revision)
		err = errors.WithStack(write(logging.NewContext(ctx, log), podExecutor))
	}
	if err == nil {
		cluster.Status.UsersRevision = revision
	}

	return err
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// reconcilePostgresDataVolume writes the PersistentVolumeClaim for instance's
// PostgreSQL data volume.
func (r *Reconciler) reconcilePostgresDataVolume(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instanceSpec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
	clusterVolumes []corev1.PersistentVolumeClaim, sourceCluster *v1beta1.PostgresCluster,
) (*corev1.PersistentVolumeClaim, error) {

	labelMap := map[string]string{
		naming.LabelCluster:     cluster.Name,
		naming.LabelInstanceSet: instanceSpec.Name,
		naming.LabelInstance:    instance.Name,
		naming.LabelRole:        naming.RolePostgresData,
		naming.LabelData:        naming.DataPostgres,
	}

	var pvc *corev1.PersistentVolumeClaim
	existingPVCName, err := getPGPVCName(labelMap, clusterVolumes)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if existingPVCName != "" {
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.GetNamespace(),
			Name:      existingPVCName,
		}}
	} else {
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
	}

	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	err = errors.WithStack(r.setControllerReference(cluster, pvc))

	pvc.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		instanceSpec.Metadata.GetAnnotationsOrNil())

	pvc.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		instanceSpec.Metadata.GetLabelsOrNil(),
		labelMap,
	)

	pvc.Spec = instanceSpec.DataVolumeClaimSpec

	// If a source cluster was provided and VolumeSnapshots are turned on in the source cluster and
	// there is a VolumeSnapshot available for the source cluster that is ReadyToUse, use it as the
	// source for the PVC. If there is an error when retrieving VolumeSnapshots, or no ReadyToUse
	// snapshots were found, create a warning event, but continue creating PVC in the usual fashion.
	if sourceCluster != nil && sourceCluster.Spec.Backups.Snapshots != nil && feature.Enabled(ctx, feature.VolumeSnapshots) {
		snapshots, err := r.getSnapshotsForCluster(ctx, sourceCluster)
		if err == nil {
			snapshot := getLatestReadySnapshot(snapshots)
			if snapshot != nil {
				r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "BootstrappingWithSnapshot",
					"Snapshot found for %v; bootstrapping cluster with snapshot.", sourceCluster.Name)
				pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
					APIGroup: initialize.String("snapshot.storage.k8s.io"),
					Kind:     snapshot.Kind,
					Name:     snapshot.Name,
				}
			} else {
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "SnapshotNotFound",
					"No ReadyToUse snapshots were found for %v; proceeding with typical restore process.", sourceCluster.Name)
			}
		} else {
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "SnapshotNotFound",
				"Could not get snapshots for %v, proceeding with typical restore process.", sourceCluster.Name)
		}
	}

	r.setVolumeSize(ctx, cluster, pvc, instanceSpec.Name)

	// Clear any set limit before applying PVC. This is needed to allow the limit
	// value to change later.
	pvc.Spec.Resources.Limits = nil

	if err == nil {
		err = r.handlePersistentVolumeClaimError(cluster,
			errors.WithStack(r.apply(ctx, pvc)))
	}

	return pvc, err
}

// setVolumeSize compares the potential sizes from the instance spec, status
// and limit and sets the appropriate current value.
func (r *Reconciler) setVolumeSize(ctx context.Context, cluster *v1beta1.PostgresCluster,
	pvc *corev1.PersistentVolumeClaim, instanceSpecName string) {
	log := logging.FromContext(ctx)

	// Store the limit for this instance set. This value will not change below.
	volumeLimitFromSpec := pvc.Spec.Resources.Limits.Storage()

	// Capture the largest pgData volume size currently defined for a given instance set.
	// This value will capture our desired update.
	volumeRequestSize := pvc.Spec.Resources.Requests.Storage()

	// If the request value is greater than the set limit, use the limit and issue
	// a warning event. A limit of 0 is ignorned.
	if !volumeLimitFromSpec.IsZero() &&
		volumeRequestSize.Value() > volumeLimitFromSpec.Value() {
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "VolumeRequestOverLimit",
			"pgData volume request (%v) for %s/%s is greater than set limit (%v). Limit value will be used.",
			volumeRequestSize, cluster.Name, instanceSpecName, volumeLimitFromSpec)

		pvc.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: *resource.NewQuantity(volumeLimitFromSpec.Value(), resource.BinarySI),
		}
		// Otherwise, if the limit is not set or the feature gate is not enabled, do not autogrow.
	} else if !volumeLimitFromSpec.IsZero() && feature.Enabled(ctx, feature.AutoGrowVolumes) {
		for i := range cluster.Status.InstanceSets {
			if instanceSpecName == cluster.Status.InstanceSets[i].Name {
				for _, dpv := range cluster.Status.InstanceSets[i].DesiredPGDataVolume {
					if dpv != "" {
						desiredRequest, err := resource.ParseQuantity(dpv)
						if err == nil {
							if desiredRequest.Value() > volumeRequestSize.Value() {
								volumeRequestSize = &desiredRequest
							}
						} else {
							log.Error(err, "Unable to parse volume request: "+dpv)
						}
					}
				}
			}
		}

		// If the volume request size is greater than or equal to the limit and the
		// limit is not zero, update the request size to the limit value.
		// If the user manually requests a lower limit that is smaller than the current
		// or requested volume size, it will be ignored in favor of the limit value.
		if volumeRequestSize.Value() >= volumeLimitFromSpec.Value() {

			r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "VolumeLimitReached",
				"pgData volume(s) for %s/%s are at size limit (%v).", cluster.Name,
				instanceSpecName, volumeLimitFromSpec)

			// If the volume size request is greater than the limit, issue an
			// additional event warning.
			if volumeRequestSize.Value() > volumeLimitFromSpec.Value() {
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "DesiredVolumeAboveLimit",
					"The desired size (%v) for the %s/%s pgData volume(s) is greater than the size limit (%v).",
					volumeRequestSize, cluster.Name, instanceSpecName, volumeLimitFromSpec)
			}

			volumeRequestSize = volumeLimitFromSpec
		}
		pvc.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: *resource.NewQuantity(volumeRequestSize.Value(), resource.BinarySI),
		}
	}
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// reconcileTablespaceVolumes writes the PersistentVolumeClaims for instance's
// tablespace data volumes.
func (r *Reconciler) reconcileTablespaceVolumes(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instanceSpec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
	clusterVolumes []corev1.PersistentVolumeClaim,
) (tablespaceVolumes []*corev1.PersistentVolumeClaim, err error) {

	if !feature.Enabled(ctx, feature.TablespaceVolumes) {
		return
	}

	if instanceSpec.TablespaceVolumes == nil {
		return
	}

	for _, vol := range instanceSpec.TablespaceVolumes {
		labelMap := map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: instanceSpec.Name,
			naming.LabelInstance:    instance.Name,
			naming.LabelRole:        "tablespace",
			naming.LabelData:        vol.Name,
		}

		var pvc *corev1.PersistentVolumeClaim
		existingPVCName, err := getPGPVCName(labelMap, clusterVolumes)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if existingPVCName != "" {
			pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.GetNamespace(),
				Name:      existingPVCName,
			}}
		} else {
			pvc = &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstanceTablespaceDataVolume(instance, vol.Name)}
		}

		pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

		err = errors.WithStack(r.setControllerReference(cluster, pvc))

		pvc.Annotations = naming.Merge(
			cluster.Spec.Metadata.GetAnnotationsOrNil(),
			instanceSpec.Metadata.GetAnnotationsOrNil())

		pvc.Labels = naming.Merge(
			cluster.Spec.Metadata.GetLabelsOrNil(),
			instanceSpec.Metadata.GetLabelsOrNil(),
			labelMap,
		)

		pvc.Spec = vol.DataVolumeClaimSpec

		if err == nil {
			err = r.handlePersistentVolumeClaimError(cluster,
				errors.WithStack(r.apply(ctx, pvc)))
		}

		if err != nil {
			return nil, err
		}

		tablespaceVolumes = append(tablespaceVolumes, pvc)
	}

	return
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={get}
// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,delete,patch}

// reconcilePostgresWALVolume writes the PersistentVolumeClaim for instance's
// PostgreSQL WAL volume.
func (r *Reconciler) reconcilePostgresWALVolume(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instanceSpec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
	observed *Instance, clusterVolumes []corev1.PersistentVolumeClaim,
) (*corev1.PersistentVolumeClaim, error) {

	labelMap := map[string]string{
		naming.LabelCluster:     cluster.Name,
		naming.LabelInstanceSet: instanceSpec.Name,
		naming.LabelInstance:    instance.Name,
		naming.LabelRole:        naming.RolePostgresWAL,
		naming.LabelData:        naming.DataPostgres,
	}

	var pvc *corev1.PersistentVolumeClaim
	existingPVCName, err := getPGPVCName(labelMap, clusterVolumes)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if existingPVCName != "" {
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.GetNamespace(),
			Name:      existingPVCName,
		}}
	} else {
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresWALVolume(instance)}
	}

	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	if instanceSpec.WALVolumeClaimSpec == nil {
		// No WAL volume is specified; delete the PVC safely if it exists. Check
		// the client cache first using Get.
		key := client.ObjectKeyFromObject(pvc)
		err := errors.WithStack(r.Client.Get(ctx, key, pvc))
		if err != nil {
			return nil, client.IgnoreNotFound(err)
		}

		// The "StorageObjectInUseProtection" admission controller adds a
		// finalizer to every PVC so that the "pvc-protection" controller can
		// remove it safely. Return early when it is already scheduled for deletion.
		// - https://docs.k8s.io/reference/access-authn-authz/admission-controllers/
		if pvc.DeletionTimestamp != nil {
			return nil, nil
		}

		// The WAL PVC exists and should be removed. Delete it only when WAL
		// files are safely on their intended volume. The PVC will continue to
		// exist until all Pods using it are also deleted.
		// - https://docs.k8s.io/concepts/storage/persistent-volumes/#storage-object-in-use-protection
		var walDirectory string
		if observed != nil && len(observed.Pods) == 1 {
			if running, known := observed.IsRunning(naming.ContainerDatabase); running && known {
				// NOTE(cbandy): Despite the guard above, calling PodExec may still fail
				// due to a missing or stopped container.

				// This assumes that $PGDATA matches the configured PostgreSQL "data_directory".
				var stdout bytes.Buffer
				err = errors.WithStack(r.PodExec(
					ctx, observed.Pods[0].Namespace, observed.Pods[0].Name, naming.ContainerDatabase,
					nil, &stdout, nil, "bash", "-ceu", "--", `exec realpath "${PGDATA}/pg_wal"`))

				walDirectory = strings.TrimRight(stdout.String(), "\n")
			}
		}
		if err == nil && walDirectory == postgres.WALDirectory(cluster, instanceSpec) {
			return nil, errors.WithStack(
				client.IgnoreNotFound(r.deleteControlled(ctx, cluster, pvc)))
		}

		// The WAL PVC exists and might contain WAL files. There is no spec to
		// reconcile toward so return early.
		return pvc, err
	}

	err = errors.WithStack(r.setControllerReference(cluster, pvc))

	pvc.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		instanceSpec.Metadata.GetAnnotationsOrNil())

	pvc.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		instanceSpec.Metadata.GetLabelsOrNil(),
		labelMap,
	)

	pvc.Spec = *instanceSpec.WALVolumeClaimSpec

	if err == nil {
		err = r.handlePersistentVolumeClaimError(cluster,
			errors.WithStack(r.apply(ctx, pvc)))
	}

	return pvc, err
}

// reconcileDatabaseInitSQL runs custom SQL files in the database. When
// DatabaseInitSQL is defined, the function will find the primary pod and run
// SQL from the defined ConfigMap
func (r *Reconciler) reconcileDatabaseInitSQL(ctx context.Context,
	cluster *v1beta1.PostgresCluster, instances *observedInstances) error {
	log := logging.FromContext(ctx)

	// Spec is not defined, unset status and return
	if cluster.Spec.DatabaseInitSQL == nil {
		// If database init sql is not requested, we will always expect the
		// status to be nil
		cluster.Status.DatabaseInitSQL = nil
		return nil
	}

	// Spec is defined but status is already set, return
	if cluster.Status.DatabaseInitSQL != nil {
		return nil
	}

	// Based on the previous checks, the user wants to run sql in the database.
	// Check the provided ConfigMap name and key to ensure the a string
	// exists in the ConfigMap data
	var (
		err  error
		data string
	)

	getDataFromConfigMap := func() (string, error) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Spec.DatabaseInitSQL.Name,
				Namespace: cluster.Namespace,
			},
		}
		err := r.Client.Get(ctx, client.ObjectKeyFromObject(cm), cm)
		if err != nil {
			return "", err
		}

		key := cluster.Spec.DatabaseInitSQL.Key
		if _, ok := cm.Data[key]; !ok {
			err := errors.Errorf("ConfigMap did not contain expected key: %s", key)
			return "", err
		}

		return cm.Data[key], nil
	}

	if data, err = getDataFromConfigMap(); err != nil {
		log.Error(err, "Could not get data from ConfigMap",
			"ConfigMap", cluster.Spec.DatabaseInitSQL.Name,
			"Key", cluster.Spec.DatabaseInitSQL.Key)
		return err
	}

	// Now that we have the data provided by the user. We can check for a
	// writable pod and get the podExecutor for the pod's database container
	var podExecutor postgres.Executor
	pod, _ := instances.writablePod(naming.ContainerDatabase)
	if pod == nil {
		log.V(1).Info("Could not find a pod with a writable database container.")
		return nil
	}

	podExecutor = func(
		ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return r.PodExec(ctx, pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
	}

	// A writable pod executor has been found and we have the sql provided by
	// the user. Setup a write function to execute the sql using the podExecutor
	write := func(ctx context.Context, exec postgres.Executor) error {
		stdout, stderr, err := exec.Exec(ctx, strings.NewReader(data), map[string]string{})
		log.V(1).Info("applied init SQL", "stdout", stdout, "stderr", stderr)
		return err
	}

	// Update the logger to include fields from the user provided ResourceRef
	log = log.WithValues(
		"name", cluster.Spec.DatabaseInitSQL.Name,
		"key", cluster.Spec.DatabaseInitSQL.Key,
	)

	// Write SQL to database using the podExecutor
	err = errors.WithStack(write(logging.NewContext(ctx, log), podExecutor))

	// If the podExec returns with exit code 0 the write is considered a
	// success, keep track of the ConfigMap using a status. This helps to
	// ensure SQL doesn't get run again. SQL can be run again if the
	// status is lost and the DatabaseInitSQL field exists in the spec.
	if err == nil {
		status := cluster.Spec.DatabaseInitSQL.Name
		cluster.Status.DatabaseInitSQL = &status
	}

	return err
}
