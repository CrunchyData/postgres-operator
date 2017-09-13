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
	log "github.com/Sirupsen/logrus"
	//"github.com/crunchydata/postgres-operator/operator/util"
	//"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
	//"io/ioutil"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"os/user"
	//"strings"
)

var pswCmd = &cobra.Command{
	Use:   "psw",
	Short: "manage passwords",
	Long: `PSW allows you to manage passwords across a set of clusters
For example:

pgo psw --selector=name=mycluster --update
pgo psw --dry-run --selector=someotherpolicy
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("psw called")
		updatePasswords()
	},
}

func init() {
	RootCmd.AddCommand(pswCmd)

	pswCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	pswCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "--dry-run shows clusters and passwords that would be updated to but does not actually apply them")

}

func updatePasswords() {
	//build the selector based on the selector parameter
	//get the clusters list

	//get filtered list of Deployments
	var sel string
	if Selector != "" {
		sel = Selector + ",pg-cluster,!replica"
	} else {
		sel = "pg-cluster,!replica"
	}
	log.Infoln("selector string=[" + sel + "]")

	lo := meta_v1.ListOptions{LabelSelector: sel}
	deployments, err := Clientset.ExtensionsV1beta1().Deployments(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}

	if DryRun {
		fmt.Println("dry run only....")
	}

	for _, d := range deployments.Items {
		fmt.Println("deployment : " + d.ObjectMeta.Name)
		if !DryRun {
		}

	}

}
