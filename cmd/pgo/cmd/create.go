package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var ClusterReplicaCount int
var ManagedUser bool
var AllNamespaces bool
var BackrestStorageConfig, ReplicaStorageConfig, StorageConfig string
var CustomConfig string
var ArchiveFlag, DisableAutofailFlag, EnableAutofailFlag, PgbouncerFlag, MetricsFlag, BadgerFlag bool
var BackrestRestoreFrom string
var CCPImage string
var CCPImageTag string
var CCPImagePrefix string
var PGOImagePrefix string
var Database string
var Password string
var SecretFrom string
var PoliciesFlag, PolicyFile, PolicyURL string
var UserLabels string
var Tablespaces []string
var ServiceType string
var PGBackRestType string
var Secret string
var PgouserPassword, PgouserRoles, PgouserNamespaces string
var Permissions string
var PodAntiAffinity string
var PodAntiAffinityPgBackRest string
var PodAntiAffinityPgBouncer string
var SyncReplication bool
var BackrestConfig string
var BackrestS3Key string
var BackrestS3KeySecret string
var BackrestS3Bucket string
var BackrestS3Endpoint string
var BackrestS3Region string
var BackrestS3URIStyle string
var BackrestS3VerifyTLS bool
var PVCSize string
var BackrestPVCSize string
var WALStorageConfig string
var WALPVCSize string
var RestoreFrom string

// group the annotation requests
var (
	// Annotations contains the global annotations for a cluster
	Annotations []string

	// AnnotationsBackrest contains annotations specifc to pgBackRest
	AnnotationsBackrest []string

	// AnnotationsPgBouncer contains annotations specifc to pgBouncer
	AnnotationsPgBouncer []string

	// AnnotationsPostgres contains annotations specifc to PostgreSQL instances
	AnnotationsPostgres []string
)

// group the various container resource requests together, i.e. for CPU/Memory
var (
	// the resource requests / limits for PostgreSQL instances
	CPURequest, MemoryRequest string
	CPULimit, MemoryLimit     string
	// the resource requests / limits for the pgBackRest repository
	BackrestCPURequest, BackrestMemoryRequest string
	BackrestCPULimit, BackrestMemoryLimit     string
	// the resource requests / limits for pgBouncer instances
	PgBouncerCPURequest, PgBouncerMemoryRequest string
	PgBouncerCPULimit, PgBouncerMemoryLimit     string
	// the resource requests / limits for Crunchy Postgres Exporter the sidecar container
	ExporterCPURequest, ExporterMemoryRequest string
	ExporterCPULimit, ExporterMemoryLimit     string
)

// BackrestS3CASecretName, if provided, is the name of a secret to use that
// contains a CA certificate to use for the pgBackRest repo
var BackrestS3CASecretName string

// BackrestRepoPath allows the pgBackRest repo path to be defined instead of using the default
var BackrestRepoPath string

// Standby determines whether or not the cluster should be created as a standby cluster
var Standby bool

// PasswordType allows one to specify if the password should be MD5 or SCRAM
// we presently ensure it defaults to MD5
var PasswordType string

// PasswordSuperuser specifies the password for the cluster superuser
var PasswordSuperuser string

// PasswordReplication specifies the password for the cluster replication user
var PasswordReplication string

// variables used for setting up TLS-enabled PostgreSQL clusters
var (
	// TLSOnly indicates that only TLS connections will be accepted for a
	// PostgreSQL cluster
	TLSOnly bool
	// TLSSecret is the name of the secret that contains the TLS information for
	// enabling TLS in a PostgreSQL cluster
	TLSSecret string
	// ReplicationTLSSecret is the name of the secret that contains the TLS
	// information for enabling certificate-based authentication between instances
	// in a PostgreSQL cluster, particularly for replication
	ReplicationTLSSecret string
	// CASecret is the name of the secret that contains the CA information for
	// enabling TLS in a PostgreSQL cluster
	CASecret string
)

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Postgres Operator resource",
	Long: `CREATE allows you to create a new Operator resource. For example:
    pgo create cluster
    pgo create pgadmin
    pgo create pgbouncer
    pgo create pgouser
    pgo create pgorole
    pgo create policy
    pgo create namespace
    pgo create user`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to create.  Valid resource types include:
    * cluster
    * pgadmin
    * pgbouncer
    * pgouser
    * pgorole
    * policy
    * namespace
    * user`)
		} else {
			switch args[0] {
			case "cluster", "pgbouncer", "pgouser", "pgorole", "policy", "user", "namespace":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to create.  Valid resource types include:
    * cluster
    * pgadmin
    * pgbouncer
    * pgouser
    * pgorole
    * policy
    * namespace
    * user`)
			}
		}
	},
}

// createClusterCmd ...
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a PostgreSQL cluster",
	Long: `Create a PostgreSQL cluster consisting of a primary and a number of replica backends. For example:

    pgo create cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create cluster called")

		if len(args) != 1 {
			fmt.Println(`Error: A single cluster name is required for this command.`)
			os.Exit(1)
		}

		if PgbouncerFlag && PgBouncerReplicas < 0 {
			fmt.Println("Error: You must specify one or more replicas for pgBouncer.")
			os.Exit(1)
		}

		createCluster(args, Namespace, cmd)
	},
}

// createPolicyCmd ...
var createPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Create a SQL policy",
	Long: `Create a policy. For example:

    pgo create policy mypolicy --in-file=/tmp/mypolicy.sql`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create policy called ")
		if PolicyFile == "" && PolicyURL == "" {
			fmt.Println(`Error: The --in-file or --url flags are required to create a policy.`)
			return
		}

		if len(args) == 0 {
			fmt.Println(`Error: A policy name is required for this command.`)
		} else {
			createPolicy(args, Namespace)
		}
	},
}

// createPgAdminCmd ...
var createPgAdminCmd = &cobra.Command{
	Use:   "pgadmin",
	Short: "Create a pgAdmin instance ",
	Long: `Create a pgAdmin instance for mycluster. For example:

	pgo create pgadmin mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create pgadmin called ")

		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: A cluster name or selector is required for this command.`)
		} else {
			createPgAdmin(args, Namespace)
		}
	},
}

// createPgbouncerCmd ...
var createPgbouncerCmd = &cobra.Command{
	Use:   "pgbouncer",
	Short: "Create a pgbouncer ",
	Long: `Create a pgbouncer. For example:

	pgo create pgbouncer mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create pgbouncer called ")

		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: A cluster name or selector is required for this command.`)
			os.Exit(1)
		}

		if PgBouncerReplicas < 0 {
			fmt.Println("Error: You must specify one or more replicas.")
			os.Exit(1)
		}

		createPgbouncer(args, Namespace)
	},
}

// createUserCmd ...
var createUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Create a PostgreSQL user",
	Long: `Create a postgres user. For example:

    pgo create user --username=someuser --all --managed
    pgo create user --username=someuser  mycluster --managed
    pgo create user --username=someuser -selector=name=mycluster --managed
    pgo create user --username=user1 --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create user called")
		if Selector == "" && !AllFlag && len(args) == 0 {
			fmt.Println(`Error: a cluster name(s), --selector flag, or --all flag is required to create a user.`)
			return
		}

		createUser(args, Namespace)
	},
}

func init() {
	RootCmd.AddCommand(CreateCmd)
	CreateCmd.AddCommand(createClusterCmd)
	CreateCmd.AddCommand(createPolicyCmd)
	CreateCmd.AddCommand(createPgAdminCmd)
	CreateCmd.AddCommand(createPgbouncerCmd)
	CreateCmd.AddCommand(createPgouserCmd)
	CreateCmd.AddCommand(createPgoroleCmd)
	CreateCmd.AddCommand(createUserCmd)
	CreateCmd.AddCommand(createNamespaceCmd)

	// flags for "pgo create cluster"
	createClusterCmd.Flags().StringSliceVar(&Annotations, "annotation", []string{},
		"Add an Annotation to all of the managed deployments (PostgreSQL, pgBackRest, pgBouncer)\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"\n\n"+
			"For example, to add two annotations: \"--annotation=hippo=awesome,elephant=cool\"")
	createClusterCmd.Flags().StringSliceVar(&AnnotationsBackrest, "annotation-pgbackrest", []string{},
		"Add an Annotation specifically to pgBackRest deployments\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"")
	createClusterCmd.Flags().StringSliceVar(&AnnotationsPgBouncer, "annotation-pgbouncer", []string{},
		"Add an Annotation specifically to pgBouncer deployments\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"")
	createClusterCmd.Flags().StringSliceVar(&AnnotationsPostgres, "annotation-postgres", []string{},
		"Add an Annotation specifically to PostgreSQL deployments\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"")
	createClusterCmd.Flags().StringVarP(&CCPImage, "ccp-image", "", "", "The CCPImage name to use for cluster creation. If specified, overrides the value crunchy-postgres.")
	createClusterCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createClusterCmd.Flags().StringVarP(&CCPImagePrefix, "ccp-image-prefix", "", "", "The CCPImagePrefix to use for cluster creation. If specified, overrides the global configuration.")
	createClusterCmd.Flags().StringVarP(&PGOImagePrefix, "pgo-image-prefix", "", "", "The PGOImagePrefix to use for cluster creation. If specified, overrides the global configuration.")
	createClusterCmd.Flags().StringVar(&CPURequest, "cpu", "", "Set the number of millicores to request for the CPU, e.g. "+
		"\"100m\" or \"0.1\".")
	createClusterCmd.Flags().StringVar(&CPULimit, "cpu-limit", "", "Set the number of millicores to limit for the CPU, e.g. "+
		"\"100m\" or \"0.1\".")
	createClusterCmd.Flags().StringVarP(&CustomConfig, "custom-config", "", "", "The name of a configMap that holds custom PostgreSQL configuration files used to override defaults.")
	createClusterCmd.Flags().StringVarP(&Database, "database", "d", "", "If specified, sets the name of the initial database that is created for the user. Defaults to the value set in the PostgreSQL Operator configuration, or if that is not present, the name of the cluster")
	createClusterCmd.Flags().BoolVarP(&DisableAutofailFlag, "disable-autofail", "", false, "Disables autofail capabitilies in the cluster following cluster initialization.")
	createClusterCmd.Flags().StringVarP(&UserLabels, "labels", "l", "", "The labels to apply to this cluster.")
	createClusterCmd.Flags().StringVar(&MemoryRequest, "memory", "", "Set the amount of RAM to request, e.g. "+
		"1GiB. Overrides the default server value.")
	createClusterCmd.Flags().StringVar(&MemoryLimit, "memory-limit", "", "Set the amount of RAM to limit, e.g. "+
		"1GiB.")
	createClusterCmd.Flags().BoolVarP(&MetricsFlag, "metrics", "", false, "Adds the crunchy-postgres-exporter container to the database pod.")
	createClusterCmd.Flags().StringVar(&ExporterCPURequest, "exporter-cpu", "", "Set the number of millicores to request for CPU "+
		"for the Crunchy Postgres Exporter sidecar container, e.g. \"100m\" or \"0.1\". Defaults to being unset.")
	createClusterCmd.Flags().StringVar(&ExporterCPULimit, "exporter-cpu-limit", "", "Set the number of millicores to limit for CPU "+
		"for the Crunchy Postgres Exporter sidecar container, e.g. \"100m\" or \"0.1\". Defaults to being unset.")
	createClusterCmd.Flags().StringVar(&ExporterMemoryRequest, "exporter-memory", "", "Set the amount of memory to request for "+
		"the Crunchy Postgres Exporter sidecar container. Defaults to server value (24Mi).")
	createClusterCmd.Flags().StringVar(&ExporterMemoryLimit, "exporter-memory-limit", "", "Set the amount of memory to limit for "+
		"the Crunchy Postgres Exporter sidecar container.")
	createClusterCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key=value) to use in placing the primary database. If not set, any node is used.")
	createClusterCmd.Flags().StringVarP(&Password, "password", "", "", "The password to use for standard user account created during cluster initialization.")
	createClusterCmd.Flags().IntVarP(&PasswordLength, "password-length", "", 0, "If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.")
	createClusterCmd.Flags().StringVarP(&PasswordSuperuser, "password-superuser", "", "", "The password to use for the PostgreSQL superuser.")
	createClusterCmd.Flags().StringVarP(&PasswordReplication, "password-replication", "", "", "The password to use for the PostgreSQL replication user.")
	createClusterCmd.Flags().StringVar(&BackrestCPURequest, "pgbackrest-cpu", "", "Set the number of millicores to request for CPU "+
		"for the pgBackRest repository.")
	createClusterCmd.Flags().StringVar(&BackrestCPULimit, "pgbackrest-cpu-limit", "", "Set the number of millicores to limit for CPU "+
		"for the pgBackRest repository.")
	createClusterCmd.Flags().StringVar(&BackrestConfig, "pgbackrest-custom-config", "", "The name of a ConfigMap containing pgBackRest configuration files.")
	createClusterCmd.Flags().StringVar(&BackrestMemoryRequest, "pgbackrest-memory", "", "Set the amount of memory to request for "+
		"the pgBackRest repository. Defaults to server value (48Mi).")
	createClusterCmd.Flags().StringVar(&BackrestMemoryLimit, "pgbackrest-memory-limit", "", "Set the amount of memory to limit for "+
		"the pgBackRest repository.")
	createClusterCmd.Flags().StringVarP(&BackrestPVCSize, "pgbackrest-pvc-size", "", "",
		`The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "local" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"`)
	createClusterCmd.Flags().StringVarP(&BackrestRepoPath, "pgbackrest-repo-path", "", "",
		"The pgBackRest repository path that should be utilized instead of the default. Required "+
			"for standby\nclusters to define the location of an existing pgBackRest repository.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Key, "pgbackrest-s3-key", "", "",
		"The AWS S3 key that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Bucket, "pgbackrest-s3-bucket", "", "",
		"The AWS S3 bucket that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVar(&BackrestS3CASecretName, "pgbackrest-s3-ca-secret", "",
		"If used, specifies a Kubernetes secret that uses a different CA certificate for "+
			"S3 or a S3-like storage interface. Must contain a key with the value \"aws-s3-ca.crt\"")
	createClusterCmd.Flags().StringVarP(&BackrestS3Endpoint, "pgbackrest-s3-endpoint", "", "",
		"The AWS S3 endpoint that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3KeySecret, "pgbackrest-s3-key-secret", "", "",
		"The AWS S3 key secret that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Region, "pgbackrest-s3-region", "", "",
		"The AWS S3 region that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3URIStyle, "pgbackrest-s3-uri-style", "", "", "Specifies whether \"host\" or \"path\" style URIs will be used when connecting to S3.")
	createClusterCmd.Flags().BoolVarP(&BackrestS3VerifyTLS, "pgbackrest-s3-verify-tls", "", true, "This sets if pgBackRest should verify the TLS certificate when connecting to S3. To disable, use \"--pgbackrest-s3-verify-tls=false\".")
	createClusterCmd.Flags().StringVar(&BackrestStorageConfig, "pgbackrest-storage-config", "", "The name of the storage config in pgo.yaml to use for the pgBackRest local repository.")
	createClusterCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use with pgBackRest. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
	createClusterCmd.Flags().BoolVarP(&BadgerFlag, "pgbadger", "", false, "Adds the crunchy-pgbadger container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&PgbouncerFlag, "pgbouncer", "", false, "Adds a crunchy-pgbouncer deployment to the cluster.")
	createClusterCmd.Flags().StringVar(&PgBouncerCPURequest, "pgbouncer-cpu", "", "Set the number of millicores to request for CPU "+
		"for pgBouncer. Defaults to being unset.")
	createClusterCmd.Flags().StringVar(&PgBouncerCPULimit, "pgbouncer-cpu-limit", "", "Set the number of millicores to limit for CPU "+
		"for pgBouncer. Defaults to being unset.")
	createClusterCmd.Flags().StringVar(&PgBouncerMemoryRequest, "pgbouncer-memory", "", "Set the amount of memory to request for "+
		"pgBouncer. Defaults to server value (24Mi).")
	createClusterCmd.Flags().StringVar(&PgBouncerMemoryLimit, "pgbouncer-memory-limit", "", "Set the amount of memory to limit for "+
		"pgBouncer.")
	createClusterCmd.Flags().Int32Var(&PgBouncerReplicas, "pgbouncer-replicas", 0, "Set the total number of pgBouncer instances to deploy. If not set, defaults to 1.")
	createClusterCmd.Flags().StringVarP(&ReplicaStorageConfig, "replica-storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster replica storage.")
	createClusterCmd.Flags().StringVarP(&PodAntiAffinity, "pod-anti-affinity", "", "",
		"Specifies the type of anti-affinity that should be utilized when applying  "+
			"default pod anti-affinity rules to PG clusters (default \"preferred\")")
	createClusterCmd.Flags().StringVarP(&PodAntiAffinityPgBackRest, "pod-anti-affinity-pgbackrest", "", "",
		"Set the Pod anti-affinity rules specifically for the pgBackRest "+
			"repository. Defaults to the default cluster pod anti-affinity (i.e. \"preferred\"), "+
			"or the value set by --pod-anti-affinity")
	createClusterCmd.Flags().StringVarP(&PodAntiAffinityPgBouncer, "pod-anti-affinity-pgbouncer", "", "",
		"Set the Pod anti-affinity rules specifically for the pgBouncer "+
			"Pods. Defaults to the default cluster pod anti-affinity (i.e. \"preferred\"), "+
			"or the value set by --pod-anti-affinity")
	createClusterCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply when creating a cluster, comma separated.")
	createClusterCmd.Flags().StringVarP(&PVCSize, "pvc-size", "", "",
		`The size of the PVC capacity for primary and replica PostgreSQL instances. Overrides the value set in the storage class. Must follow the standard Kubernetes format, e.g. "10.1Gi"`)
	createClusterCmd.Flags().IntVarP(&ClusterReplicaCount, "replica-count", "", 0, "The number of replicas to create as part of the cluster.")
	createClusterCmd.Flags().StringVar(&ReplicationTLSSecret, "replication-tls-secret", "", "The name of the secret that contains "+
		"the TLS keypair to use for enabling certificate-based authentication between PostgreSQL instances, "+
		"particularly for the purpose of replication. Must be used with \"server-tls-secret\" and \"server-ca-secret\".")
	createClusterCmd.Flags().StringVarP(&RestoreFrom, "restore-from", "", "", "The name of cluster to restore from when bootstrapping a new cluster")
	createClusterCmd.Flags().StringVarP(&BackupOpts, "restore-opts", "", "",
		"The options to pass into pgbackrest where performing a restore to bootrap the cluster. "+
			"Only applicable when a \"restore-from\" value is specified")
	createClusterCmd.Flags().StringVarP(&SecretFrom, "secret-from", "s", "", "The cluster name to use when restoring secrets.")
	createClusterCmd.Flags().StringVar(&CASecret, "server-ca-secret", "", "The name of the secret that contains "+
		"the certficate authority (CA) to use for enabling the PostgreSQL cluster to accept TLS connections. "+
		"Must be used with \"server-tls-secret\".")
	createClusterCmd.Flags().StringVar(&TLSSecret, "server-tls-secret", "", "The name of the secret that contains "+
		"the TLS keypair to use for enabling the PostgreSQL cluster to accept TLS connections. "+
		"Must be used with \"server-ca-secret\"")
	createClusterCmd.Flags().StringVarP(&ServiceType, "service-type", "", "", "The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.")
	createClusterCmd.Flags().BoolVar(&ShowSystemAccounts, "show-system-accounts", false, "Include the system accounts in the results.")
	createClusterCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster storage.")
	createClusterCmd.Flags().BoolVarP(&SyncReplication, "sync-replication", "", false,
		"Enables synchronous replication for the cluster.")
	createClusterCmd.Flags().BoolVar(&TLSOnly, "tls-only", false, "If true, forces all PostgreSQL connections to be over TLS. "+
		"Must also set \"server-tls-secret\" and \"server-ca-secret\"")
	createClusterCmd.Flags().BoolVarP(&Standby, "standby", "", false, "Creates a standby cluster "+
		"that replicates from a pgBackRest repository in AWS S3.")
	createClusterCmd.Flags().StringSliceVar(&Tablespaces, "tablespace", []string{},
		"Create a PostgreSQL tablespace on the cluster, e.g. \"name=ts1:storageconfig=nfsstorage\". The format is "+
			"a key/value map that is delimited by \"=\" and separated by \":\". The following parameters are available:\n\n"+
			"- name (required): the name of the PostgreSQL tablespace\n"+
			"- storageconfig (required): the storage configuration to use, as specified in the list available in the "+
			"\"pgo-config\" ConfigMap (aka \"pgo.yaml\")\n"+
			"- pvcsize: the size of the PVC capacity, which overrides the value set in the specified storageconfig. "+
			"Follows the Kubernetes quantity format.\n\n"+
			"For example, to create a tablespace with the NFS storage configuration with a PVC of size 10GiB:\n\n"+
			"--tablespace=name=ts1:storageconfig=nfsstorage:pvcsize=10Gi")
	createClusterCmd.Flags().StringVarP(&Username, "username", "u", "", "The username to use for creating the PostgreSQL user with standard permissions. Defaults to the value in the PostgreSQL Operator configuration.")
	createClusterCmd.Flags().StringVar(&WALStorageConfig, "wal-storage-config", "",
		`The name of a storage configuration in pgo.yaml to use for PostgreSQL's write-ahead log (WAL).`)
	createClusterCmd.Flags().StringVar(&WALPVCSize, "wal-storage-size", "",
		`The size of the capacity for WAL storage, which overrides any value in the storage configuration. Follows the Kubernetes quantity format.`)

	// pgo create pgadmin
	createPgAdminCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

	// pgo create pgbouncer
	createPgbouncerCmd.Flags().StringVar(&PgBouncerCPURequest, "cpu", "", "Set the number of millicores to request for CPU "+
		"for pgBouncer. Defaults to being unset.")
	createPgbouncerCmd.Flags().StringVar(&PgBouncerCPULimit, "cpu-limit", "", "Set the number of millicores to request for CPU "+
		"for pgBouncer.")
	createPgbouncerCmd.Flags().StringVar(&PgBouncerMemoryRequest, "memory", "", "Set the amount of memory to request for "+
		"pgBouncer. Defaults to server value (24Mi).")
	createPgbouncerCmd.Flags().StringVar(&PgBouncerMemoryLimit, "memory-limit", "", "Set the amount of memory to limit for "+
		"pgBouncer.")
	createPgbouncerCmd.Flags().Int32Var(&PgBouncerReplicas, "replicas", 0, "Set the total number of pgBouncer instances to deploy. If not set, defaults to 1.")
	createPgbouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

	// "pgo create pgouser" flags
	createPgouserCmd.Flags().BoolVarP(&AllNamespaces, "all-namespaces", "", false, "specifies this user will have access to all namespaces.")
	createPgoroleCmd.Flags().StringVarP(&Permissions, "permissions", "", "", "specify a comma separated list of permissions for a pgorole")
	createPgouserCmd.Flags().StringVarP(&PgouserPassword, "pgouser-password", "", "", "specify a password for a pgouser")
	createPgouserCmd.Flags().StringVarP(&PgouserRoles, "pgouser-roles", "", "", "specify a comma separated list of Roles for a pgouser")
	createPgouserCmd.Flags().StringVarP(&PgouserNamespaces, "pgouser-namespaces", "", "", "specify a comma separated list of Namespaces for a pgouser")

	// "pgo create policy" flags
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for adding a policy.")
	createPolicyCmd.Flags().StringVarP(&PolicyURL, "url", "u", "", "The url to use for adding a policy.")

	// "pgo create user" flags
	createUserCmd.Flags().BoolVar(&AllFlag, "all", false, "Create a user on every cluster.")
	createUserCmd.Flags().BoolVarP(&ManagedUser, "managed", "", false, "Creates a user with secrets that can be managed by the Operator.")
	createUserCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	createUserCmd.Flags().StringVarP(&Password, "password", "", "", "The password to use for creating a new user which overrides a generated password.")
	createUserCmd.Flags().IntVarP(&PasswordLength, "password-length", "", 0, "If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.")
	createUserCmd.Flags().StringVar(&PasswordType, "password-type", "md5", "The type of password hashing to use."+
		"Choices are: (md5, scram-sha-256).")
	createUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createUserCmd.Flags().StringVarP(&Username, "username", "", "", "The username to use for creating a new user")
	createUserCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "", 0, "Sets the number of days that a password is valid. Defaults to the server value.")
}

// createPgouserCmd ...
var createPgouserCmd = &cobra.Command{
	Use:   "pgouser",
	Short: "Create a pgouser",
	Long: `Create a pgouser. For example:

    pgo create pgouser someuser`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create pgouser called ")

		if len(args) == 0 {
			fmt.Println(`Error: A pgouser username is required for this command.`)
		} else {
			createPgouser(args, Namespace)
		}
	},
}

// createPgoroleCmd ...
var createPgoroleCmd = &cobra.Command{
	Use:   "pgorole",
	Short: "Create a pgorole",
	Long: `Create a pgorole. For example:

    pgo create pgorole somerole --permissions="Cat,Ls"`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create pgorole  called ")

		if len(args) == 0 {
			fmt.Println(`Error: A pgouser role name is required for this command.`)
		} else {
			createPgorole(args, Namespace)
		}
	},
}

// createNamespaceCmd ...
var createNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Create a namespace",
	Long: `Create a namespace. For example:

	pgo create namespace somenamespace

	Note: For Kubernetes versions prior to 1.12, this command will not function properly
    - use $PGOROOT/deploy/add_targted_namespace.sh scriptor or give the user cluster-admin privileges.
    For more details, see the Namespace Creation section under Installing Operator Using Bash in the documentation.`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create namespace called ")

		if len(args) == 0 {
			fmt.Println(`Error: A namespace name is required for this command.`)
		} else {
			createNamespace(args, Namespace)
		}
	},
}
