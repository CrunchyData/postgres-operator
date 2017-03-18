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

// Package cmd provides the command line functions of the crunchy CLI
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "not implemented yet",
	Long:  `not implemented yet.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		fmt.Println("backup called")
	},
}

func init() {
	RootCmd.AddCommand(backupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// backupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// backupCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func showBackup(args []string) {
	fmt.Printf("showBackup called %v\n", args)
}

func createBackup(args []string) {
	fmt.Printf("createBackup called %v\n", args)
}

func deleteBackup(args []string) {
	fmt.Printf("createBackup called %v\n", args)
}
