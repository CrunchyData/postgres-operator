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
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
)

func showUpgrade(args []string) {
	log.Debugf("showUpgrade called %v\n", args)

	//show pod information for job
	for _, arg := range args {
		log.Debug("show upgrade called for " + arg)
		//pg-database=basic or
		//pgupgrade=true
		if arg == "all" {
			lo := v1.ListOptions{LabelSelector: "pgupgrade=true"}
			log.Debug("label selector is " + lo.LabelSelector)
			pods, err2 := Clientset.Core().Pods(Namespace).List(lo)
			if err2 != nil {
				log.Error(err2.Error())
				return
			}
			for _, pod := range pods.Items {
				showUpgradeItem(pod.Name)
			}

		} else {
			showUpgradeItem(arg)

		}

	}

}

func showUpgradeItem(name string) {
	var pvcName string
	//print the pgupgrades TPR
	result := tpr.PgUpgrade{}
	err := Tprclient.Get().
		Resource("pgupgrades").
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&result)
	if err == nil {
		if result.Spec.PVC_NAME == "" {
			pvcName = name + "-upgrade-pvc"
		} else {
			pvcName = result.Spec.PVC_NAME
		}
		fmt.Printf("\npgupgrade %s\n", name+" was found PVC_NAME is "+pvcName)
	} else if errors.IsNotFound(err) {
		configPVC := viper.GetString("DB.PVC_NAME")
		if configPVC == "" {
			pvcName = name + "-upgrade-pvc"
		} else {
			pvcName = configPVC
		}
		fmt.Printf("\npgupgrade %s\n", name+" was not found assuming PVC_NAME is "+pvcName)
	} else {
		log.Errorf("\npgupgrade %s\n", name+" lookup error ")
		log.Error(err.Error())
		return
	}

	//print the upgrade jobs if any exists
	lo := v1.ListOptions{LabelSelector: "pg-database=" + name}
	log.Debug("label selector is " + lo.LabelSelector)
	pods, err2 := Clientset.Core().Pods(Namespace).List(lo)
	if err2 != nil {
		log.Error(err2.Error())
	}
	fmt.Printf("\nupgrade job pods for database %s\n", name+"...")
	for _, p := range pods.Items {
		fmt.Printf("%s%s\n", TREE_TRUNK, p.Name)
	}

	//print the database pod if it exists
	lo = v1.ListOptions{LabelSelector: "name=" + name}
	log.Debug("label selector is " + lo.LabelSelector)
	dbpods, err := Clientset.Core().Pods(Namespace).List(lo)
	if err != nil || len(dbpods.Items) == 0 {
		fmt.Printf("\ndatabase pod %s\n", name+" is not found")
		fmt.Println(err.Error())
	} else {
		fmt.Printf("\ndatabase pod %s\n", name+" is found")
	}

	fmt.Println("")

}

func createUpgrade(args []string) {
	log.Debugf("createUpgrade called %v\n", args)

	var err error
	var newInstance *tpr.PgUpgrade

	for _, arg := range args {
		log.Debug("create upgrade called for " + arg)
		result := tpr.PgUpgrade{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("pgupgrades").
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("pgupgrade " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			fmt.Println("pgupgrade " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgupgrade " + arg)
			log.Error(err.Error())
			break
		}
		// Create an instance of our TPR
		newInstance, err = getUpgradeParams(arg)
		if err != nil {
			log.Error("error creating upgrade")
			break
		}

		err = Tprclient.Post().
			Resource("pgupgrades").
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error("error in creating PgUpgrade TPR instance")
			log.Error(err.Error())
		}
		fmt.Println("created PgUpgrade " + arg)

	}

}

func deleteUpgrade(args []string) {
	log.Debugf("deleteUpgrade called %v\n", args)
	var err error
	upgradeList := tpr.PgUpgradeList{}
	err = Tprclient.Get().Resource("pgupgrades").Do().Into(&upgradeList)
	if err != nil {
		log.Error("error getting upgrade list")
		log.Error(err.Error())
		return
	}
	// delete the pgupgrade resource instance
	// which will cause the operator to remove the related Job
	for _, arg := range args {
		for _, upgrade := range upgradeList.Items {
			if arg == "all" || upgrade.Spec.Name == arg {
				err = Tprclient.Delete().
					Resource("pgupgrades").
					Namespace(Namespace).
					Name(upgrade.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgupgrade " + arg)
					log.Error(err.Error())
				}
				fmt.Println("deleted pgupgrade " + upgrade.Spec.Name)
			}

		}

	}

}

func getUpgradeParams(name string) (*tpr.PgUpgrade, error) {
	var newInstance *tpr.PgUpgrade

	spec := tpr.PgUpgradeSpec{}
	spec.Name = name
	spec.PVC_NAME = viper.GetString("PVC_NAME")
	spec.PVC_ACCESS_MODE = viper.GetString("DB.PVC_ACCESS_MODE")
	spec.PVC_SIZE = viper.GetString("DB.PVC_SIZE")
	spec.CCP_IMAGE_TAG = viper.GetString("DB.CCP_IMAGE_TAG")
	spec.OLD_DATABASE_NAME = "basic"
	spec.NEW_DATABASE_NAME = "master"
	spec.OLD_VERSION = "9.5"
	spec.NEW_VERSION = "9.6"

	//TODO see if name is a database or cluster
	db := tpr.PgDatabase{}
	err := Tprclient.Get().
		Resource("pgdatabases").
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&db)
	if err == nil {
		fmt.Println(name + " is a database")
		spec.OLD_DATABASE_NAME = db.Spec.Name
		spec.NEW_DATABASE_NAME = db.Spec.Name + "-upgrade"
	} else if errors.IsNotFound(err) {
		log.Debug(name + " is not a database")
		cluster := tpr.PgCluster{}
		err = Tprclient.Get().
			Resource("pgclusters").
			Namespace(Namespace).
			Name(name).
			Do().
			Into(&cluster)
		if err == nil {
			fmt.Println(name + " is a cluster")
			spec.OLD_DATABASE_NAME = cluster.Spec.Name
			spec.NEW_DATABASE_NAME = cluster.Spec.Name + "-upgrade"
		} else if errors.IsNotFound(err) {
			log.Debug(name + " is not a cluster")
			return newInstance, err
		} else {
			log.Error("error getting pgcluster " + name)
			log.Error(err.Error())
			return newInstance, err
		}
	} else {
		log.Error("error getting pgdatabase " + name)
		log.Error(err.Error())
		return newInstance, err
	}

	newInstance = &tpr.PgUpgrade{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, nil
}
