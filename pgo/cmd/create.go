package cmd

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
var ContainerResources string
var ReplicaStorageConfig, StorageConfig string
var CustomConfig string
var ArchiveFlag, AutofailFlag, PgpoolFlag, PgbouncerFlag, MetricsFlag, BadgerFlag bool
var BackrestFlag, BackrestRestoreFrom string
var PgpoolSecret string
var PgbouncerSecret string
var CCPImage string
var CCPImageTag string
var Password string
var PgBouncerPassword string
var PgBouncerUser string
var SecretFrom string
var PoliciesFlag, PolicyFile, PolicyURL string
var UserLabels string
var ServiceType string
var Schedule string
var ScheduleOptions string
var ScheduleType string
var SchedulePolicy string
var ScheduleDatabase string
var ScheduleSecret string
var PGBackRestType string
var Secret string

var Series int

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Cluster, PGBouncer, PGPool, Policy, Schedule, or User",
	Long: `CREATE allows you to create a new Cluster, PGBouncer, PGPool, Policy, Schedule or User. For example: 

    pgo create cluster
    pgo create pgbouncer
    pgo create pgpool
    pgo create policy
    pgo create user`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 || (args[0] != "cluster" && args[0] != "policy") && args[0] != "user" {
			fmt.Println(`Error: You must specify the type of resource to create.  Valid resource types include:
    * cluster
    * pgbouncer
    * pgpool
    * policy
    * user`)
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
			createCluster(args, Namespace)
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

// createPgpoolCmd ...
var createPgpoolCmd = &cobra.Command{
	Use:   "pgpool",
	Short: "Create a pgpool ",
	Long: `Create a pgpool. For example:

    pgo create pgpool mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create pgpool called ")

		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: A cluster name or selector is required for this command.`)
		} else {
			createPgpool(args, Namespace)
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

    pgo create user manageduser --managed --selector=name=mycluster
    pgo create user user1 --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("create user called ")
		if Selector == "" {
			fmt.Println(`Error: The --selector flag is required to create a user.`)
			return
		}

		if len(args) == 0 {
			fmt.Println(`Error: A user name is required for this command.`)
		} else {
			createUser(args, Namespace)
		}
	},
}

func init() {
	RootCmd.AddCommand(CreateCmd)
	CreateCmd.AddCommand(createClusterCmd)
	CreateCmd.AddCommand(createPolicyCmd)
	CreateCmd.AddCommand(createPgbouncerCmd)
	CreateCmd.AddCommand(createPgpoolCmd)
	CreateCmd.AddCommand(createScheduleCmd)
	CreateCmd.AddCommand(createUserCmd)

	createClusterCmd.Flags().StringVarP(&BackrestFlag, "pgbackrest", "", "", "Enables a pgBackRest volume for the database pod, \"true\" or \"false\". Default from pgo.yaml, command line overrides default.")
	createClusterCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use with pgBackRest. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
	createClusterCmd.Flags().BoolVarP(&BadgerFlag, "pgbadger", "", false, "Adds the crunchy-pgbadger container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&PgpoolFlag, "pgpool", "", false, "Adds the crunchy-pgpool container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&PgbouncerFlag, "pgbouncer", "", false, "Adds a crunchy-pgbouncer deployment to the cluster.")
	//	createClusterCmd.Flags().BoolVarP(&ArchiveFlag, "archive", "", false, "Enables archive logging for the database cluster.")
	createClusterCmd.Flags().StringVarP(&PgpoolSecret, "pgpool-secret", "", "", "The name of a pgpool secret to use for the pgpool configuration.")
	createClusterCmd.Flags().BoolVarP(&MetricsFlag, "metrics", "", false, "Adds the crunchy-collect container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&AutofailFlag, "autofail", "", false, "If set, will cause autofailover to be enabled on this cluster.")
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
	createClusterCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply when creating a cluster, comma separated.")
	createClusterCmd.Flags().StringVarP(&CCPImage, "ccp-image", "", "", "The CCPImage name to use for cluster creation. If specified, overrides the value crunchy-postgres.")
	createClusterCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createClusterCmd.Flags().IntVarP(&Series, "series", "e", 1, "The number of clusters to create in a series.")
	createClusterCmd.Flags().IntVarP(&ClusterReplicaCount, "replica-count", "", 0, "The number of replicas to create as part of the cluster.")
	createClusterCmd.Flags().StringVarP(&ContainerResources, "resources-config", "r", "", "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits.")

	createPolicyCmd.Flags().StringVarP(&PolicyURL, "url", "u", "", "The url to use for adding a policy.")
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for adding a policy.")

	createScheduleCmd.Flags().StringVarP(&Schedule, "schedule", "", "", "The schedule assigned to the cron task.")
	createScheduleCmd.Flags().StringVarP(&ScheduleType, "schedule-type", "", "", "The type of schedule to be created (pgbackrest, pgbasebackup or policy).")
	createScheduleCmd.Flags().StringVarP(&PGBackRestType, "pgbackrest-backup-type", "", "", "The type of pgBackRest backup to schedule (full or diff).")
	createScheduleCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use when scheduling pgBackRest backups. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
	createScheduleCmd.Flags().StringVarP(&PVCName, "pvc-name", "", "", "The name of the backup PVC to use (only used in pgbasebackup schedules).")
	createScheduleCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createScheduleCmd.Flags().StringVarP(&ScheduleOptions, "schedule-opts", "", "", "The custom options passed to the create schedule API.")
	createScheduleCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createScheduleCmd.Flags().StringVarP(&SchedulePolicy, "policy", "", "", "The policy to use for SQL schedules.")
	createScheduleCmd.Flags().StringVarP(&ScheduleDatabase, "database", "", "", "The database to run the SQL policy against.")
	createScheduleCmd.Flags().StringVarP(&ScheduleSecret, "secret", "", "", "The secret name for the username and password of the PostgreSQL role for SQL schedules.")

	createUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createUserCmd.Flags().StringVarP(&Password, "password", "", "", "The password to use for creating a new user which overrides a generated password.")
	createUserCmd.Flags().BoolVarP(&ManagedUser, "managed", "", false, "Creates a user with secrets that can be managed by the Operator.")
	createUserCmd.Flags().StringVarP(&UserDBAccess, "db", "", "", "Grants the user access to a database.")
	createUserCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "", 30, "Sets passwords for new users to X days.")
	createUserCmd.Flags().IntVarP(&PasswordLength, "password-length", "", 12, "If no password is supplied, this is the length of the auto generated password")
	createPgpoolCmd.Flags().StringVarP(&PgpoolSecret, "pgpool-secret", "", "", "The name of a pgpool secret to use for the pgpool configuration.")

	// createPgbouncerCmd.Flags().StringVarP(&PgBouncerUser, "pgbouncer-user", "", "", "Username for the crunchy-pgboucer deployment, default is 'pgbouncer'.")
	createPgbouncerCmd.Flags().StringVarP(&PgBouncerPassword, "pgbouncer-pass", "", "", "Password for the pgbouncer user of the crunchy-pgboucer deployment.")
	createPgbouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

}
