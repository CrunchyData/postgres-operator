// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"fmt"
	"hash/fnv"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// ContainerDatabase is the name of the container running PostgreSQL and
	// supporting tools: Patroni, pgBackRest, etc.
	ContainerDatabase = "database"

	// ContainerPGAdmin is the name of a container running pgAdmin.
	ContainerPGAdmin = "pgadmin"

	// ContainerPGAdminStartup is the name of the initialization container
	// that prepares the filesystem for pgAdmin.
	ContainerPGAdminStartup = "pgadmin-startup"

	// ContainerPGBackRestConfig is the name of a container supporting pgBackRest.
	ContainerPGBackRestConfig = "pgbackrest-config"

	// ContainerPGBouncer is the name of a container running PgBouncer.
	ContainerPGBouncer = "pgbouncer"
	// ContainerPGBouncerConfig is the name of a container supporting PgBouncer.
	ContainerPGBouncerConfig = "pgbouncer-config"

	// ContainerPostgresStartup is the name of the initialization container
	// that prepares the filesystem for PostgreSQL.
	ContainerPostgresStartup = "postgres-startup"

	// ContainerClientCertCopy is the name of the container that is responsible for copying and
	// setting proper permissions on the client certificate and key after initialization whenever
	// there is a change in the certificates or key
	ContainerClientCertCopy = "replication-cert-copy"
	// ContainerNSSWrapperInit is the name of the init container utilized to configure support
	// for the nss_wrapper
	ContainerNSSWrapperInit = "nss-wrapper-init"

	// ContainerPGBackRestLogDirInit is the name of the init container utilized to make
	// a pgBackRest log directory when using a dedicated repo host.
	ContainerPGBackRestLogDirInit = "pgbackrest-log-dir"

	// ContainerPGMonitorExporter is the name of a container running postgres_exporter
	ContainerPGMonitorExporter = "exporter"

	// ContainerJobMovePGDataDir is the name of the job container utilized to copy v4 Operator
	// pgData directories to the v5 default location
	ContainerJobMovePGDataDir = "pgdata-move-job"
	// ContainerJobMovePGWALDir is the name of the job container utilized to copy v4 Operator
	// pg_wal directories to the v5 default location
	ContainerJobMovePGWALDir = "pgwal-move-job"
	// ContainerJobMovePGBackRestRepoDir is the name of the job container utilized to copy v4
	// Operator pgBackRest repo directories to the v5 default location
	ContainerJobMovePGBackRestRepoDir = "repo-move-job"
)

const (
	// PortExporter is the named port for the "exporter" container
	PortExporter = "exporter"
	// PortPGAdmin is the name of a port that connects to pgAdmin.
	PortPGAdmin = "pgadmin"
	// PortPGBouncer is the name of a port that connects to PgBouncer.
	PortPGBouncer = "pgbouncer"
	// PortPostgreSQL is the name of a port that connects to PostgreSQL.
	PortPostgreSQL = "postgres"
)

const (
	// RootCertSecret is the default root certificate secret name
	RootCertSecret = "pgo-root-cacert" /* #nosec */
	// ClusterCertSecret is the default cluster leaf certificate secret name
	ClusterCertSecret = "%s-cluster-cert" /* #nosec */
)

const (
	// CertVolume is the name of the Certificate volume and volume mount in a
	// PostgreSQL instance Pod
	CertVolume = "cert-volume"

	// CertMountPath is the path for mounting the postgrescluster certificates
	// and key
	CertMountPath = "/pgconf/tls"

	// ReplicationDirectory is the directory at CertMountPath where the replication
	// certificates and key are mounted
	ReplicationDirectory = "/replication"

	// ReplicationTmp is the directory where the replication certificates and key can
	// have the proper permissions set due to:
	// https://github.com/kubernetes/kubernetes/issues/57923
	ReplicationTmp = "/tmp/replication"

	// ReplicationCert is the secret key to the postgrescluster's
	// replication/rewind user's client certificate
	ReplicationCert = "tls.crt"

	// ReplicationCertPath is the path to the postgrescluster's replication/rewind
	// user's client certificate
	ReplicationCertPath = "replication/tls.crt"

	// ReplicationPrivateKey is the secret key to the postgrescluster's
	// replication/rewind user's client private key
	ReplicationPrivateKey = "tls.key"

	// ReplicationPrivateKeyPath is the path to the postgrescluster's
	// replication/rewind user's client private key
	ReplicationPrivateKeyPath = "replication/tls.key"

	// ReplicationCACert is the key name of the postgrescluster's replication/rewind
	// user's client CA certificate
	// Note: when using auto-generated certificates, this will be identical to the
	// server CA cert
	ReplicationCACert = "ca.crt"

	// ReplicationCACertPath is the path to the postgrescluster's replication/rewind
	// user's client CA certificate
	ReplicationCACertPath = "replication/ca.crt"
)

const (
	// PGBackRestRepoContainerName is the name assigned to the container used to run pgBackRest
	PGBackRestRepoContainerName = "pgbackrest"

	// PGBackRestRestoreContainerName is the name assigned to the container used to run pgBackRest
	// restores
	PGBackRestRestoreContainerName = "pgbackrest-restore"

	// PGBackRestRepoName is the name used for a pgbackrest repository
	PGBackRestRepoName = "%s-pgbackrest-repo-%s"

	// PGBackRestPGDataLogPath is the pgBackRest default log path configuration used by the
	// PostgreSQL instance.
	PGBackRestPGDataLogPath = "/pgdata/pgbackrest/log"

	// PGBackRestRepoLogPath is the pgBackRest default log path configuration used by the
	// dedicated repo host, if configured.
	PGBackRestRepoLogPath = "/pgbackrest/%s/log"

	// suffix used with postgrescluster name for associated configmap.
	// for instance, if the cluster is named 'mycluster', the
	// configmap will be named 'mycluster-pgbackrest-config'
	cmNameSuffix = "%s-pgbackrest-config"

	// suffix used with postgrescluster name for associated configmap.
	// for instance, if the cluster is named 'mycluster', the
	// configmap will be named 'mycluster-ssh-config'
	// Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.
	// TODO(tjmoore4): Once we no longer need this for cleanup purposes, this should be removed.
	sshCMNameSuffix = "%s-ssh-config"

	// suffix used with postgrescluster name for associated secret.
	// for instance, if the cluster is named 'mycluster', the
	// secret will be named 'mycluster-ssh'
	// Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.
	// TODO(tjmoore4): Once we no longer need this for cleanup purposes, this should be removed.
	sshSecretNameSuffix = "%s-ssh"

	// RestoreConfigCopySuffix is the suffix used for ConfigMap or Secret configuration
	// resources needed when restoring from a PostgresCluster data source. If, for
	// example, a Secret is named 'mysecret' and is the first item in the configuration
	// slice, the copied Secret will be named 'mysecret-restorecopy-0'
	RestoreConfigCopySuffix = "%s-restorecopy-%d"
)

// AsObjectKey converts the ObjectMeta API type to a client.ObjectKey.
// When you have a client.Object, use client.ObjectKeyFromObject() instead.
func AsObjectKey(m metav1.ObjectMeta) client.ObjectKey {
	return client.ObjectKey{Namespace: m.Namespace, Name: m.Name}
}

// ClusterConfigMap returns the ObjectMeta necessary to lookup
// cluster's shared ConfigMap.
func ClusterConfigMap(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-config",
	}
}

// ClusterInstanceRBAC returns the ObjectMeta necessary to lookup the
// ServiceAccount, Role, and RoleBinding for cluster's PostgreSQL instances.
func ClusterInstanceRBAC(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-instance",
	}
}

// ClusterPGAdmin returns the ObjectMeta necessary to lookup the ConfigMap,
// Service, StatefulSet, or Volume for the cluster's pgAdmin user interface.
func ClusterPGAdmin(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pgadmin",
	}
}

// ClusterPGBouncer returns the ObjectMeta necessary to lookup the ConfigMap,
// Deployment, Secret, PodDisruptionBudget or Service that is cluster's
// PgBouncer proxy.
func ClusterPGBouncer(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pgbouncer",
	}
}

// ClusterPodService returns the ObjectMeta necessary to lookup the Service
// that is responsible for the network identity of Pods.
func ClusterPodService(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	// The hyphen below ensures that the DNS name will not be interpreted as a
	// top-level domain. Partially qualified requests for "{pod}.{cluster}-pods"
	// should not leave the Kubernetes cluster, and if they do they are less
	// likely to resolve.
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pods",
	}
}

// ClusterPrimaryService returns the ObjectMeta necessary to lookup the Service
// that exposes the PostgreSQL primary instance.
func ClusterPrimaryService(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-primary",
	}
}

// ClusterReplicaService returns the ObjectMeta necessary to lookup the Service
// that exposes PostgreSQL replica instances.
func ClusterReplicaService(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-replicas",
	}
}

// ClusterVolumeSnapshot returns the ObjectMeta, including a random name, for a
// new pgdata VolumeSnapshot.
func ClusterVolumeSnapshot(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pgdata-snapshot-" + rand.String(4),
	}
}

// GenerateInstance returns a random name for a member of cluster and set.
func GenerateInstance(
	cluster *v1beta1.PostgresCluster, set *v1beta1.PostgresInstanceSetSpec,
) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-" + set.Name + "-" + rand.String(4),
	}
}

// GenerateStartupInstance returns a stable name that's shaped like
// GenerateInstance above. The stable name is based on a four character
// hash of the cluster name and instance set name
func GenerateStartupInstance(
	cluster *v1beta1.PostgresCluster, set *v1beta1.PostgresInstanceSetSpec,
) metav1.ObjectMeta {
	// Calculate a stable name that's shaped like GenerateInstance above.
	// hash.Hash.Write never returns an error: https://pkg.go.dev/hash#Hash.
	hash := fnv.New32()
	_, _ = hash.Write([]byte(cluster.Name + set.Name))
	suffix := rand.SafeEncodeString(fmt.Sprint(hash.Sum32()))[:4]

	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-" + set.Name + "-" + suffix,
	}
}

// InstanceConfigMap returns the ObjectMeta necessary to lookup
// instance's shared ConfigMap.
func InstanceConfigMap(instance metav1.Object) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-config",
	}
}

// InstanceCertificates returns the ObjectMeta necessary to lookup the Secret
// containing instance's certificates.
func InstanceCertificates(instance metav1.Object) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-certs",
	}
}

// InstanceSet returns the ObjectMeta necessary to lookup the objects
// associated with a single instance set. Includes PodDisruptionBudgets
func InstanceSet(cluster *v1beta1.PostgresCluster,
	set *v1beta1.PostgresInstanceSetSpec) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      cluster.Name + "-set-" + set.Name,
		Namespace: cluster.Namespace,
	}
}

// InstancePostgresDataVolume returns the ObjectMeta for the PostgreSQL data
// volume for instance.
func InstancePostgresDataVolume(instance *appsv1.StatefulSet) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-pgdata",
	}
}

// InstanceTablespaceDataVolume returns the ObjectMeta for the tablespace data
// volume for instance.
func InstanceTablespaceDataVolume(instance *appsv1.StatefulSet, tablespaceName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: instance.GetNamespace(),
		Name: instance.GetName() +
			"-" + tablespaceName +
			"-tablespace",
	}
}

// InstancePostgresWALVolume returns the ObjectMeta for the PostgreSQL WAL
// volume for instance.
func InstancePostgresWALVolume(instance *appsv1.StatefulSet) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-pgwal",
	}
}

// MonitoringUserSecret returns ObjectMeta necessary to lookup the Secret
// containing authentication credentials for monitoring tools.
func MonitoringUserSecret(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-monitoring",
	}
}

// ExporterWebConfigMap returns ObjectMeta necessary to lookup and create the
// exporter web configmap. This configmap is used to configure the exporter
// web server.
func ExporterWebConfigMap(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-exporter-web-config",
	}
}

// ExporterQueriesConfigMap returns ObjectMeta necessary to lookup and create the
// exporter queries configmap. This configmap is used to pass the default queries
// to the exporter.
func ExporterQueriesConfigMap(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-exporter-queries-config",
	}
}

// OperatorConfigurationSecret returns the ObjectMeta necessary to lookup the
// Secret containing PGO configuration.
func OperatorConfigurationSecret() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: config.PGONamespace(),
		Name:      "pgo-config",
	}
}

// ReplicationClientCertSecret returns ObjectMeta necessary to lookup the Secret
// containing the Patroni client authentication certificate information.
func ReplicationClientCertSecret(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-replication-cert",
	}
}

// PatroniDistributedConfiguration returns the ObjectMeta necessary to lookup
// the DCS created by Patroni for cluster. This same name is used for both
// ConfigMap and Endpoints. See Patroni DCS "config_path".
func PatroniDistributedConfiguration(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      PatroniScope(cluster) + "-config",
	}
}

// PatroniLeaderConfigMap returns the ObjectMeta necessary to lookup the
// ConfigMap created by Patroni for the leader election of cluster.
// See Patroni DCS "leader_path".
func PatroniLeaderConfigMap(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      PatroniScope(cluster) + "-leader",
	}
}

// PatroniLeaderEndpoints returns the ObjectMeta necessary to lookup the
// Endpoints created by Patroni for the leader election of cluster.
// See Patroni DCS "leader_path".
func PatroniLeaderEndpoints(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      PatroniScope(cluster),
	}
}

// PatroniScope returns the "scope" Patroni uses for cluster.
func PatroniScope(cluster *v1beta1.PostgresCluster) string {
	return cluster.Name + "-ha"
}

// PatroniTrigger returns the ObjectMeta necessary to lookup the ConfigMap or
// Endpoints Patroni creates for cluster to initiate a controlled change of the
// leader. See Patroni DCS "failover_path".
func PatroniTrigger(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      PatroniScope(cluster) + "-failover",
	}
}

// PGBackRestConfig returns the ObjectMeta for a pgBackRest ConfigMap
func PGBackRestConfig(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.GetNamespace(),
		Name:      fmt.Sprintf(cmNameSuffix, cluster.GetName()),
	}
}

// PGBackRestBackupJob returns the ObjectMeta for the pgBackRest backup Job utilized
// to create replicas using pgBackRest
func PGBackRestBackupJob(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      cluster.GetName() + "-backup-" + rand.String(4),
		Namespace: cluster.GetNamespace(),
	}
}

// PGBackRestCronJob returns the ObjectMeta for a pgBackRest CronJob
func PGBackRestCronJob(cluster *v1beta1.PostgresCluster, backuptype, repoName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.Name + "-" + repoName + "-" + backuptype,
	}
}

// PGBackRestRestoreJob returns the ObjectMeta for a pgBackRest restore Job
func PGBackRestRestoreJob(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.Name + "-pgbackrest-restore",
	}
}

// PGBackRestRBAC returns the ObjectMeta necessary to lookup the ServiceAccount, Role, and
// RoleBinding for pgBackRest Jobs
func PGBackRestRBAC(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pgbackrest",
	}
}

// PGBackRestRepoVolume returns the ObjectMeta for a pgBackRest repository volume
func PGBackRestRepoVolume(cluster *v1beta1.PostgresCluster,
	repoName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-%s", cluster.GetName(), repoName),
		Namespace: cluster.GetNamespace(),
	}
}

// PGBackRestSSHConfig returns the ObjectMeta for a pgBackRest SSHD ConfigMap
// Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.
// TODO(tjmoore4): Once we no longer need this for cleanup purposes, this should be removed.
func PGBackRestSSHConfig(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      fmt.Sprintf(sshCMNameSuffix, cluster.GetName()),
		Namespace: cluster.GetNamespace(),
	}
}

// PGBackRestSSHSecret returns the ObjectMeta for a pgBackRest SSHD Secret
// Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.
// TODO(tjmoore4): Once we no longer need this for cleanup purposes, this should be removed.
func PGBackRestSSHSecret(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      fmt.Sprintf(sshSecretNameSuffix, cluster.GetName()),
		Namespace: cluster.GetNamespace(),
	}
}

// PGBackRestSecret returns the ObjectMeta for a pgBackRest Secret
func PGBackRestSecret(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      cluster.GetName() + "-pgbackrest",
		Namespace: cluster.GetNamespace(),
	}
}

// DeprecatedPostgresUserSecret returns the ObjectMeta necessary to lookup the
// old Secret containing the default Postgres user and connection information.
// Use PostgresUserSecret instead.
func DeprecatedPostgresUserSecret(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pguser",
	}
}

// PostgresUserSecret returns the ObjectMeta necessary to lookup a Secret
// containing a PostgreSQL user and its connection information.
func PostgresUserSecret(cluster *v1beta1.PostgresCluster, username string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-pguser-" + username,
	}
}

// PostgresTLSSecret returns the ObjectMeta necessary to lookup the Secret
// containing the default Postgres TLS certificates and key
func PostgresTLSSecret(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-cluster-cert",
	}
}

// MovePGDataDirJob returns the ObjectMeta for a pgData directory move Job
func MovePGDataDirJob(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.Name + "-move-pgdata-dir",
	}
}

// MovePGWALDirJob returns the ObjectMeta for a pg_wal directory move Job
func MovePGWALDirJob(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.Name + "-move-pgwal-dir",
	}
}

// MovePGBackRestRepoDirJob returns the ObjectMeta for a pgBackRest repo directory move Job
func MovePGBackRestRepoDirJob(cluster *v1beta1.PostgresCluster) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.Name + "-move-pgbackrest-repo-dir",
	}
}

// StandalonePGAdmin returns the ObjectMeta necessary to lookup the ConfigMap,
// Service, StatefulSet, or Volume for the cluster's pgAdmin user interface.
func StandalonePGAdmin(pgadmin *v1beta1.PGAdmin) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: pgadmin.Namespace,
		Name:      fmt.Sprintf("pgadmin-%s", pgadmin.UID),
	}
}

// UpgradeCheckConfigMap returns the ObjectMeta for the PGO ConfigMap
func UpgradeCheckConfigMap() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: config.PGONamespace(),
		Name:      "pgo-upgrade-check",
	}
}
