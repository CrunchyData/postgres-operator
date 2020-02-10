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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var ClusterReplicaCount int
var ManagedUser bool
var AllNamespaces bool
var ContainerResources string
var ReplicaStorageConfig, StorageConfig string
var CustomConfig string
var ArchiveFlag, DisableAutofailFlag, EnableAutofailFlag, PgbouncerFlag, MetricsFlag, BadgerFlag bool
var BackrestRestoreFrom string
var PgbouncerSecret string
var CCPImage string
var CCPImageTag string
var Password string
var PgBouncerPassword string
var PgBouncerUser string
var SecretFrom string
var PoliciesFlag, PolicyFile, PolicyURL string
var UserLabels string
var TablespaceMounts string
var ServiceType string
var Schedule string
var ScheduleOptions string
var ScheduleType string
var SchedulePolicy string
var ScheduleDatabase string
var ScheduleSecret string
var PGBackRestType string
var Secret string
var PgouserPassword, PgouserRoles, PgouserNamespaces string
var Permissions string
var PodAntiAffinity string
var SyncReplication bool
var BackrestS3Key string
var BackrestS3KeySecret string
var BackrestS3Bucket string
var BackrestS3Endpoint string
var BackrestS3Region string

var Series int

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Postgres Operator resource",
	Long: `CREATE allows you to create a new Operator resource. For example:
    pgo create cluster
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

		// handle pgbouncer username and pass fields if --pgbouncer flag not specified.
		if !PgbouncerFlag && ((len(PgBouncerUser) > 0) || (len(PgBouncerPassword) > 0)) {
			fmt.Println("Error: The --pgbouncer flag is required when specifying --pgbouncer-user or --pgbouncer-pass.")
			return
		}

		if len(args) != 1 {
			fmt.Println(`Error: A single cluster name is required for this command.`)
		} else {
			createCluster(args, Namespace, cmd)
		}
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
		} else {
			createPgbouncer(args, Namespace)
		}
	},
}

// createScheduleCmd ...
var createScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Create a cron-like scheduled task",
	Long: `Schedule creates a cron-like scheduled task.  For example:

    pgo create schedule --schedule="* * * * *" --schedule-type=pgbackrest --pgbackrest-backup-type=full mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create schedule called ")
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: The --selector flag or a cluster name is required to create a schedule.")
			return
		}
		createSchedule(args, Namespace)
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
		log.Debug("create user called ")
		if Selector == "" && AllFlag == false && len(args) == 0 {
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
	CreateCmd.AddCommand(createPgbouncerCmd)
	CreateCmd.AddCommand(createPgouserCmd)
	CreateCmd.AddCommand(createPgoroleCmd)
	CreateCmd.AddCommand(createScheduleCmd)
	CreateCmd.AddCommand(createUserCmd)
	CreateCmd.AddCommand(createNamespaceCmd)

	createPgouserCmd.Flags().StringVarP(&PgouserPassword, "pgouser-password", "", "", "specify a password for a pgouser")
	createPgouserCmd.Flags().StringVarP(&PgouserRoles, "pgouser-roles", "", "", "specify a comma separated list of Roles for a pgouser")
	createPgouserCmd.Flags().StringVarP(&PgouserNamespaces, "pgouser-namespaces", "", "", "specify a comma separated list of Namespaces for a pgouser")
	createPgouserCmd.Flags().BoolVarP(&AllNamespaces, "all-namespaces", "", false, "specifies this user will have access to all namespaces.")
	createPgoroleCmd.Flags().StringVarP(&Permissions, "permissions", "", "", "specify a comma separated list of permissions for a pgorole")
	createClusterCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use with pgBackRest. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
	createClusterCmd.Flags().BoolVarP(&BadgerFlag, "pgbadger", "", false, "Adds the crunchy-pgbadger container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&PgbouncerFlag, "pgbouncer", "", false, "Adds a crunchy-pgbouncer deployment to the cluster.")
	//	createClusterCmd.Flags().BoolVarP(&ArchiveFlag, "archive", "", false, "Enables archive logging for the database cluster.")
	createClusterCmd.Flags().BoolVarP(&MetricsFlag, "metrics", "", false, "Adds the crunchy-collect container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&DisableAutofailFlag, "disable-autofail", "", false, "Disables autofail capabitilies in the cluster following cluster initialization.")
	createClusterCmd.Flags().StringVarP(&CustomConfig, "custom-config", "", "", "The name of a configMap that holds custom PostgreSQL configuration files used to override defaults.")
	createClusterCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster storage.")
	createClusterCmd.Flags().StringVarP(&ReplicaStorageConfig, "replica-storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster replica storage.")
	createClusterCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key=value) to use in placing the primary database. If not set, any node is used.")
	createClusterCmd.Flags().StringVarP(&ServiceType, "service-type", "", "", "The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.")
	createClusterCmd.Flags().StringVarP(&Password, "password", "w", "", "The password to use for initial database users.")
	//	createClusterCmd.Flags().StringVarP(&PgBouncerUser, "pgbouncer-user", "", "", "Username for the crunchy-pgboucer deployment, default is 'pgbouncer'.")
	createClusterCmd.Flags().StringVarP(&PgBouncerPassword, "pgbouncer-pass", "", "", "Password for the pgbouncer user of the crunchy-pgboucer deployment.")
	createClusterCmd.Flags().StringVarP(&SecretFrom, "secret-from", "s", "", "The cluster name to use when restoring secrets.")
	createClusterCmd.Flags().StringVarP(&UserLabels, "labels", "l", "", "The labels to apply to this cluster.")
	createClusterCmd.Flags().StringVarP(&TablespaceMounts, "tablespaces", "", "", "A list of extra mounts to create for tablespaces. The format is a comma seperated list of <mountName>=<storageConfig>")
	createClusterCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply when creating a cluster, comma separated.")
	createClusterCmd.Flags().StringVarP(&CCPImage, "ccp-image", "", "", "The CCPImage name to use for cluster creation. If specified, overrides the value crunchy-postgres.")
	createClusterCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createClusterCmd.Flags().IntVarP(&Series, "series", "e", 1, "The number of clusters to create in a series.")
	createClusterCmd.Flags().IntVarP(&ClusterReplicaCount, "replica-count", "", 0, "The number of replicas to create as part of the cluster.")
	createClusterCmd.Flags().StringVarP(&ContainerResources, "resources-config", "r", "", "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits.")
	createClusterCmd.Flags().StringVarP(&PodAntiAffinity, "pod-anti-affinity", "", "",
		"Specifies the type of anti-affinity that should be utilized when applying  "+
			"default pod anti-affinity rules to PG clusters (default \"preferred\")")
	createClusterCmd.Flags().BoolVarP(&SyncReplication, "sync-replication", "", false,
		"Enables synchronous replication for the cluster.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Key, "pgbackrest-s3-key", "", "",
		"The AWS S3 key that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3KeySecret, "pgbackrest-s3-key-secret", "", "",
		"The AWS S3 key secret that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Bucket, "pgbackrest-s3-bucket", "", "",
		"The AWS S3 bucket that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Endpoint, "pgbackrest-s3-endpoint", "", "",
		"The AWS S3 endpoint that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")
	createClusterCmd.Flags().StringVarP(&BackrestS3Region, "pgbackrest-s3-region", "", "",
		"The AWS S3 region that should be utilized for the cluster when the \"s3\" "+
			"storage type is enabled for pgBackRest.")

	createPolicyCmd.Flags().StringVarP(&PolicyURL, "url", "u", "", "The url to use for adding a policy.")
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for adding a policy.")

	createScheduleCmd.Flags().StringVarP(&Schedule, "schedule", "", "", "The schedule assigned to the cron task.")
	createScheduleCmd.Flags().StringVarP(&ScheduleType, "schedule-type", "", "", "The type of schedule to be created (pgbackrest or policy).")
	createScheduleCmd.Flags().StringVarP(&PGBackRestType, "pgbackrest-backup-type", "", "", "The type of pgBackRest backup to schedule (full, diff or incr).")
	createScheduleCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use when scheduling pgBackRest backups. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
	createScheduleCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createScheduleCmd.Flags().StringVarP(&ScheduleOptions, "schedule-opts", "", "", "The custom options passed to the create schedule API.")
	createScheduleCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createScheduleCmd.Flags().StringVarP(&SchedulePolicy, "policy", "", "", "The policy to use for SQL schedules.")
	createScheduleCmd.Flags().StringVarP(&ScheduleDatabase, "database", "", "", "The database to run the SQL policy against.")
	createScheduleCmd.Flags().StringVarP(&ScheduleSecret, "secret", "", "", "The secret name for the username and password of the PostgreSQL role for SQL schedules.")

	createUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createUserCmd.Flags().StringVarP(&Password, "password", "", "", "The password to use for creating a new user which overrides a generated password.")
	createUserCmd.Flags().StringVarP(&Username, "username", "", "", "The username to use for creating a new user")
	createUserCmd.Flags().BoolVarP(&ManagedUser, "managed", "", false, "Creates a user with secrets that can be managed by the Operator.")
	//	createUserCmd.Flags().StringVarP(&UserDBAccess, "db", "", "", "Grants the user access to a database.")
	createUserCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "", 30, "Sets passwords for new users to X days.")
	createUserCmd.Flags().IntVarP(&PasswordLength, "password-length", "", 0, "If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.")

	// createPgbouncerCmd.Flags().StringVarP(&PgBouncerUser, "pgbouncer-user", "", "", "Username for the crunchy-pgboucer deployment, default is 'pgbouncer'.")
	createPgbouncerCmd.Flags().StringVarP(&PgBouncerPassword, "pgbouncer-pass", "", "", "Password for the pgbouncer user of the crunchy-pgboucer deployment.")
	createPgbouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

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
