/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package cmd

import (
	"fmt"
	"github.com/crunchydata/operator/tpr"
	"github.com/spf13/cobra"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Database or Cluster",
	Long: `CREATE allows you to create a new Database or Cluster 
For example:

crunchy create database
crunchy create cluster
.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to create.  Valid resource types include:
	* database
	* cluster`)
		}
	},
}

// createDatbaseCmd represents the create database command
var createDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Create a new database",
	Long: `Create a crunchy database which consists of a Service and Pod
For example:

crunchy create database mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		createDatabase(args)
	},
}

// createClusterCmd represents the create database command
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a database cluster",
	Long: `Create a crunchy cluster which consist of a
master and a number of replica backends. For example:

crunchy create cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create cluster called")
		createCluster(args)
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createDatabaseCmd)
	createCmd.AddCommand(createClusterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func createDatabase(args []string) {

	var err error

	for _, arg := range args {
		fmt.Println("create database called for " + arg)
		result := tpr.CrunchyDatabase{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("crunchydatabases").
			Namespace(api.NamespaceDefault).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("crunchydatabase " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			fmt.Println("crunchydatabase " + arg + " not found so we will create it")
		} else {
			fmt.Println("error getting crunchydatabase " + arg)
			fmt.Println(err.Error())
			break
		}

		// Create an instance of our TPR
		newInstance := &tpr.CrunchyDatabase{
			Metadata: api.ObjectMeta{
				Name: arg,
			},
			Spec: tpr.CrunchyDatabaseSpec{
				Name:               arg,
				PVC_NAME:           "crunchy-pvc",
				Port:               "5432",
				CCP_IMAGE_TAG:      "centos7-9.5-1.2.8",
				PG_MASTER_USER:     "master",
				PG_MASTER_PASSWORD: "password",
				PG_USER:            "testuser",
				PG_PASSWORD:        "password",
				PG_DATABASE:        "userdb",
				PG_ROOT_PASSWORD:   "password",
			},
		}

		err = Tprclient.Post().
			Resource("crunchydatabases").
			Namespace(api.NamespaceDefault).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			fmt.Println("error in creating CrunchyDatabase TPR instance")
			fmt.Println(err.Error())
		}
		fmt.Println("created CrunchyDatabase " + arg)

	}
}

func createCluster(args []string) {
	var err error

	for _, arg := range args {
		fmt.Println("create cluster called for " + arg)
		result := tpr.CrunchyCluster{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("crunchyclusters").
			Namespace(api.NamespaceDefault).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("crunchycluster " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			fmt.Println("crunchycluster " + arg + " not found so we will create it")
		} else {
			fmt.Println("error getting crunchycluster " + arg)
			fmt.Println(err.Error())
			break
		}

		// Create an instance of our TPR
		newInstance := &tpr.CrunchyCluster{
			Metadata: api.ObjectMeta{
				Name: arg,
			},
			Spec: tpr.CrunchyClusterSpec{
				Name:               arg,
				ClusterName:        arg,
				CCP_IMAGE_TAG:      "centos7-9.5-1.2.8",
				Port:               "5432",
				PVC_NAME:           "crunchy-pvc",
				PG_MASTER_HOST:     arg,
				PG_MASTER_USER:     "master",
				PG_MASTER_PASSWORD: "password",
				PG_USER:            "testuser",
				PG_PASSWORD:        "password",
				PG_DATABASE:        "userdb",
				PG_ROOT_PASSWORD:   "password",
				REPLICAS:           "2",
			},
		}

		err = Tprclient.Post().
			Resource("crunchyclusters").
			Namespace(api.NamespaceDefault).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			fmt.Println("error in creating CrunchyCluster instance")
			fmt.Println(err.Error())
		}
		fmt.Println("created CrunchyCluster " + arg)

	}
}
