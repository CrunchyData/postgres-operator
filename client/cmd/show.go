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
	"k8s.io/client-go/pkg/api/v1"
)

// ShowCmd represents the show command
var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if KubeconfigPath == "" {
			fmt.Println("kubeconfig is empty")
		} else {
			fmt.Println("kubeconfig is " + KubeconfigPath)
		}
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to show.  Valid resource types include:
			        * database
				* cluster`)
		} else {
			switch args[0] {
			case "database":
			case "cluster":
				break
			default:
				fmt.Println(`You must specify the type of resource to show.  Valid resource types include:
			        * database
				* cluster`)
			}
		}

	},
}

func init() {
	fmt.Println("show init called")
	RootCmd.AddCommand(ShowCmd)
	ShowCmd.AddCommand(ShowDatabaseCmd)
	ShowCmd.AddCommand(ShowClusterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ShowCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ShowCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// showDatbaseCmd represents the show database command
var ShowDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Show database information",
	Long: `Show a crunchy database. For example:

				crunchy show database mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		showDatabase()
	},
}

// ShowClusterCmd represents the show cluster command
var ShowClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show cluster information",
	Long: `Show a crunchy cluster. For example:

				crunchy show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		showCluster()
	},
}

func showDatabase() {
	//ConnectToKube()
	//fmt.Println("showDatabase called")
	// Fetch a list of our database TPRs
	databaseList := tpr.CrunchyDatabaseList{}
	err := Tprclient.Get().Resource("crunchydatabases").Do().Into(&databaseList)
	if err != nil {
		panic(err)
	}
	for _, database := range databaseList.Items {
		fmt.Println("database LIST: " + database.Spec.Name)
	}
}

func showCluster() {
	//ConnectToKube()
	// Fetch a list of our cluster TPRs
	clusterList := tpr.CrunchyClusterList{}
	err := Tprclient.Get().Resource("crunchyclusters").Do().Into(&clusterList)
	if err != nil {
		panic(err)
	}
	for _, cluster := range clusterList.Items {
		fmt.Println("cluster LIST: " + cluster.Spec.Name)
	}

}

func ListPods() {
	//ConnectToKube()

	lo := v1.ListOptions{LabelSelector: "k8s-app=kube-dns"}
	fmt.Println("label selector is " + lo.LabelSelector)
	pods, err := Clientset.Core().Pods("").List(lo)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	for _, pod := range pods.Items {
		fmt.Println("pod Name " + pod.ObjectMeta.Name)
		fmt.Println("pod phase is " + pod.Status.Phase)
	}

}
