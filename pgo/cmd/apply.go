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
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply a Policy",
	Long: `APPLY allows you to apply a Policy to a set of clusters
For example:

pgo apply mypolicy1 --selector=name=mycluster
pgo apply mypolicy1 --selector=someotherpolicy
pgo apply mypolicy1 --selector=someotherpolicy --dry-run
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("apply called")
		if Selector == "" {
			log.Error("selector is required to apply a policy")
			return
		}
		if len(args) == 0 {
			log.Error(`You must specify the name of a policy to apply.`)
		} else {
			applyPolicy(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	applyCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "--dry-run shows clusters that policy would be applied to but does not actually apply them")

}
