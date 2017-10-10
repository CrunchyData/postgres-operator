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
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/util"
	"github.com/spf13/cobra"
	"strconv"
)

var ReplicaCount int

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale a Cluster",
	Long: `scale allows you to adjust a Cluster's replica configuration
For example:

pgo scale mycluster --replica-count=1
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("scale called")
		if ReplicaCount < 0 {
			fmt.Println(`--replica-count command line flag is required`)
		} else if len(args) == 0 {
			fmt.Println(`You must specify the clusters to scale.`)
		} else {
			scaleCluster(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(scaleCmd)

	scaleCmd.Flags().IntVarP(&ReplicaCount, "replica-count", "r", -1, "The replica count to apply to the clusters")

}

func scaleCluster(args []string) {
	//get a list of all clusters
	clusterList := crv1.PgclusterList{}
	err := RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(Namespace).
		Do().Into(&clusterList)
	if err != nil {
		log.Error("error getting list of clusters" + err.Error())
		return
	}

	if len(clusterList.Items) == 0 {
		fmt.Println("no clusters found")
		return
	}

	itemFound := false

	for _, arg := range args {
		log.Debugf(" %s ReplicaCount is %d\n", arg, ReplicaCount)
		for _, cluster := range clusterList.Items {
			if arg == "all" || cluster.Spec.Name == arg {
				itemFound = true
				fmt.Printf("scaling %s to %d\n", arg, ReplicaCount)
				err = util.Patch(RestClient, "/spec/replicas", strconv.Itoa(ReplicaCount), crv1.PgclusterResourcePlural, arg, Namespace)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		if !itemFound {
			fmt.Println(arg + " was not found")
		}
		itemFound = false

	}
}
