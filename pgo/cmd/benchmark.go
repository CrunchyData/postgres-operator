// Package cmd provides the command line functions of the crunchy CLI
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
	"os"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	BenchmarkClients      int
	BenchmarkJobs         int
	BenchmarkScale        int
	BenchmarkTransactions int
	BenchmarkDatabase     string
	BenchmarkInitOpts     string
	BenchmarkOpts         string
	BenchmarkPolicy       string
	BenchmarkUser         string
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Perform a pgBench benchmark against clusters",
	Long: `Benchmark run pgBench against PostgreSQL clusters, for example:

  pgo benchmark mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("benchmark called")
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster or a selector flag to benchmark.`)
			os.Exit(1)
		}
		createBenchmark(args, Namespace)
	},
}

func init() {
	RootCmd.AddCommand(benchmarkCmd)
	benchmarkCmd.Flags().StringVarP(&BenchmarkDatabase, "database", "d", "postgres", "The database where the benchmark should be run.")
	benchmarkCmd.Flags().StringVarP(&BenchmarkInitOpts, "init-opts", "i", "", "The extra flags passed to pgBench during the initialization of the benchmark.")
	benchmarkCmd.Flags().StringVarP(&BenchmarkOpts, "benchmark-opts", "b", "", "The extra flags passed to pgBench during the benchmark.")
	benchmarkCmd.Flags().StringVarP(&BenchmarkPolicy, "policy", "p", "", "The name of the policy specifying custom transaction SQL for advanced benchmarking.")
	benchmarkCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	benchmarkCmd.Flags().IntVarP(&BenchmarkClients, "clients", "c", 1, "The number of clients to be used in the benchmark.")
	benchmarkCmd.Flags().IntVarP(&BenchmarkJobs, "jobs", "j", 1, "The number of worker threads to use for the benchmark.")
	benchmarkCmd.Flags().IntVarP(&BenchmarkScale, "scale", "", 1, "The number to scale the amount of rows generated for the benchmark.")
	benchmarkCmd.Flags().IntVarP(&BenchmarkTransactions, "transactions", "t", 1, "The number of transaction each client should run in the benchmark.")
}

// showBenchmark ....
func showBenchmark(args []string, ns string) {
	log.Debugf("showBenchmark called %v", args)

	request := &msgs.ShowBenchmarkRequest{
		Args:      args,
		Namespace: ns,
		Selector:  Selector,
	}

	if len(args) > 0 {
		request.ClusterName = args[0]
	}

	if err := request.Validate(); err != nil {
		fmt.Printf("Show benchmark request invalid: %s\n", err)
		os.Exit(2)
	}

	response, err := api.ShowBenchmark(httpclient, &SessionCredentials, request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}
}

// deleteBenchmark ....
func deleteBenchmark(args []string, ns string) {
	log.Debugf("deleteBenchmark called %v", args)

	request := &msgs.DeleteBenchmarkRequest{
		Args:      args,
		Namespace: ns,
		Selector:  Selector,
	}

	if len(args) > 0 {
		request.ClusterName = args[0]
	}

	if err := request.Validate(); err != nil {
		fmt.Printf("Delete benchmark request invalid: %s\n", err)
		os.Exit(2)
	}

	response, err := api.DeleteBenchmark(httpclient, &SessionCredentials, request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}
}

// createBenchmark ....
func createBenchmark(args []string, ns string) {
	log.Debugf("createBenchmark called %v", args)

	request := &msgs.CreateBenchmarkRequest{
		Args:          args,
		BenchmarkOpts: BenchmarkOpts,
		Clients:       BenchmarkClients,
		Database:      BenchmarkDatabase,
		InitOpts:      BenchmarkInitOpts,
		Jobs:          BenchmarkJobs,
		Namespace:     ns,
		Policy:        BenchmarkPolicy,
		Scale:         BenchmarkScale,
		Selector:      Selector,
		Transactions:  BenchmarkTransactions,
		User:          BenchmarkUser,
	}

	if len(args) > 0 {
		request.ClusterName = args[0]
	}

	if err := request.Validate(); err != nil {
		fmt.Printf("Create benchmark request invalid: %s\n", err)
		os.Exit(2)
	}

	response, err := api.CreateBenchmark(httpclient, &SessionCredentials, request)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}
}
