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
	//print the pgupgrades TPR
	result := tpr.PgUpgrade{}
	err := Tprclient.Get().
		Resource("pgupgrades").
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&result)
	if err == nil {
		fmt.Printf("\npgupgrade %s\n", name+" was found NEW_PVC_NAME is "+result.Spec.Name)
	} else if errors.IsNotFound(err) {
		fmt.Printf("\npgupgrade %s\n", name+" was not found ")
	} else {
		log.Errorf("\npgupgrade %s\n", name+" lookup error ")
		log.Error(err.Error())
		return
	}

	//print the upgrade jobs if any exists
	lo := v1.ListOptions{
		LabelSelector: "pg-database=" + name + ",pgupgrade=true",
	}
	log.Debug("label selector is " + lo.LabelSelector)
	pods, err2 := Clientset.Core().Pods(Namespace).List(lo)
	if err2 != nil {
		log.Error(err2.Error())
	}
	fmt.Printf("\nupgrade job pods for database %s\n", name+"...")
	for _, p := range pods.Items {
		fmt.Printf("%s%s\n", TREE_TRUNK, p.Name)
	}

	//print the upgrade pod if it exists
	lo = v1.ListOptions{LabelSelector: "name=" + name}
	log.Debug("label selector is " + lo.LabelSelector)
	dbpods, err := Clientset.Core().Pods(Namespace).List(lo)
	if err != nil || len(dbpods.Items) == 0 {
		fmt.Printf("\nupgrade pod %s\n", name+" is not found")
		fmt.Println(err.Error())
	} else {
		fmt.Printf("\nupgrade pod %s\n", name+" is found")
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
		newInstance = getUpgradeParams(arg)

		err = Tprclient.Post().
			Resource("pgupgrades").
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error("error in creating PgUpgrade TPR instance", err.Error())
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

func getUpgradeParams(name string) *tpr.PgUpgrade {

	spec := tpr.PgUpgradeSpec{
		Name:              name,
		RESOURCE_TYPE:     "database",
		PVC_ACCESS_MODE:   viper.GetString("DB.PVC_ACCESS_MODE"),
		PVC_SIZE:          viper.GetString("DB.PVC_SIZE"),
		CCP_IMAGE_TAG:     viper.GetString("DB.CCP_IMAGE_TAG"),
		OLD_DATABASE_NAME: "basic",
		NEW_DATABASE_NAME: "master",
		OLD_VERSION:       "9.5",
		NEW_VERSION:       "9.6",
		OLD_PVC_NAME:      viper.GetString("PVC_NAME"),
		NEW_PVC_NAME:      viper.GetString("PVC_NAME"),
	}

	db := tpr.PgDatabase{}
	err := Tprclient.Get().
		Resource("pgdatabases").
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&db)
	if err == nil {
		fmt.Println(name + " is a database")
		spec.RESOURCE_TYPE = "database"
		spec.OLD_DATABASE_NAME = db.Spec.Name
		spec.OLD_PVC_NAME = db.Spec.PVC_NAME
		spec.NEW_PVC_NAME = db.Spec.PVC_NAME + "-upgrade"
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
			spec.RESOURCE_TYPE = "cluster"
			spec.OLD_DATABASE_NAME = cluster.Spec.Name
			spec.NEW_DATABASE_NAME = cluster.Spec.Name + "-upgrade"
		} else if errors.IsNotFound(err) {
			log.Debug(name + " is not a cluster")
			return nil
		} else {
			log.Error("error getting pgcluster " + name)
			log.Error(err.Error())
			return nil
		}
	} else {
		log.Error("error getting pgdatabase " + name)
		log.Error(err.Error())
		return nil
	}

	newInstance := &tpr.PgUpgrade{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance
}
