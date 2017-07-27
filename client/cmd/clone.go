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
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/cobra"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"time"
)

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "perform a clone",
	Long: `CLONE creates a copy of an exising cluster, for example:
			pgo clone mycluster --name=newcluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("clone called")
		if len(args) == 0 {
			fmt.Println("You must specify the cluster to clone.")
		} else if len(args) > 1 {
			fmt.Println("Only a single cluster to clone can be specified.")
		} else if CloneName == "" {
			fmt.Println("--name is required for the clone")
		} else {
			createClone(args[0])
		}

	},
}

var CloneName string

func init() {
	RootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().StringVarP(&CloneName, "name", "n", "", "the name given to the new clone")
}

func createClone(clusterName string) {
	log.Debugf("createClone called %v\n", clusterName)

	var err error

	log.Debug("create clone called for " + clusterName)
	var newInstance *tpr.PgClone

	result := tpr.PgClone{}

	// error if it already exists
	err = Tprclient.Get().
		Resource(tpr.CLONE_RESOURCE).
		Namespace(Namespace).
		Name(clusterName).
		Do().
		Into(&result)
	if err == nil {
		fmt.Println("pgclone " + clusterName + " was found so we recreate it")
		dels := make([]string, 1)
		dels[0] = clusterName
		deleteClone(dels)
		//TODO replace sleep with proper wait
		time.Sleep(6000 * time.Millisecond)
	} else if errors.IsNotFound(err) {
		log.Debug("pgclone " + clusterName + " not found so we will create it")
	} else {
		log.Error("error getting pgclone " + clusterName)
		log.Error(err.Error())
		return
	}

	spec := tpr.PgCloneSpec{}

	spec.Name = CloneName
	spec.ClusterName = clusterName
	spec.Status = ""

	newInstance = &tpr.PgClone{
		Metadata: api.ObjectMeta{
			Name: CloneName,
		},
		Spec: spec,
	}

	err = Tprclient.Post().
		Resource(tpr.CLONE_RESOURCE).
		Namespace(Namespace).
		Body(newInstance).
		Do().Into(&result)
	if err != nil {
		log.Error("error in creating PgClone TPR instance")
		log.Error(err.Error())
		return
	}
	fmt.Println("created PgClone " + clusterName)

}

func deleteClone(args []string) {
	log.Debugf("deleteClone called %v\n", args)
	var err error
	cloneList := tpr.PgCloneList{}
	err = Tprclient.Get().Resource(tpr.CLONE_RESOURCE).Do().Into(&cloneList)
	if err != nil {
		log.Error("error getting clone list")
		log.Error(err.Error())
		return
	}
	// delete the pgclone resource instance
	for _, arg := range args {
		cloneFound := false
		for _, clone := range cloneList.Items {
			if arg == "all" || clone.Spec.Name == arg {
				cloneFound = true
				err = Tprclient.Delete().
					Resource(tpr.CLONE_RESOURCE).
					Namespace(Namespace).
					Name(clone.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgclone " + arg)
					log.Error(err.Error())
				}
				fmt.Println("deleted pgclone " + clone.Spec.Name)
			}

		}
		if !cloneFound {
			fmt.Println("clone " + arg + " not found")
		}

	}
}
