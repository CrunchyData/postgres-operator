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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var KubeconfigPath string
var Labelselector string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pgo",
	Short: "The pgo command line interface.",
	Long: `The pgo command line interface lets you
create and manage both databases and clusters.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pgo.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	RootCmd.PersistentFlags().StringVar(&KubeconfigPath, "kubeconfig", "", "kube config file")
	RootCmd.PersistentFlags().StringVar(&Labelselector, "selector", "", "label selector string")

	//fmt.Println("root init called")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".pgo")     // name of config file (without extension)
	viper.AddConfigPath(".")        // adding home directory as first search path
	viper.AddConfigPath("$HOME")    // adding home directory as first search path
	viper.AddConfigPath("/etc/pgo") // adding home directory as first search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println("config file not found")
	}

	if KubeconfigPath == "" {
		KubeconfigPath = viper.GetString("kubeconfig")
	}
	if KubeconfigPath == "" {
		fmt.Println("--kubeconfig flag is not set and required")
		os.Exit(2)
	}

	fmt.Println("kubeconfig path is " + viper.GetString("kubeconfig"))
	//fmt.Println("viper CCP_IMAGE_TAG value is " + viper.GetString("cluster.CCP_IMAGE_TAG"))
	//fmt.Println(" root initConfig called")
	ConnectToKube()

}
