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
	var err error
	log.Debugf("showUpgrade called %v\n", args)

	//show pod information for job
	for _, arg := range args {
		log.Debug("show upgrade called for " + arg)
		if arg == "all" {
			tprs := tpr.PgUpgradeList{}
			err = Tprclient.Get().Resource("pgupgrades").Do().Into(&tprs)
			if err != nil {
				log.Error("error getting list of pgupgrades " + err.Error())
				return
			}
			for _, u := range tprs.Items {
				showUpgradeItem(&u)
			}

		} else {
			var upgrade tpr.PgUpgrade

			err = Tprclient.Get().
				Resource("pgupgrades").
				Namespace(Namespace).
				Name(arg).
				Do().Into(&upgrade)
			if errors.IsNotFound(err) {
				fmt.Println("pgupgrade " + arg + " not found ")
			} else {
				showUpgradeItem(&upgrade)
			}

		}

	}

}

func showUpgradeItem(upgrade *tpr.PgUpgrade) {

	//print the TPR
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgupgrade : "+upgrade.Spec.Name)
	fmt.Printf("%s%s\n", TREE_BRANCH, "upgrade_status : "+upgrade.Spec.UPGRADE_STATUS)
	fmt.Printf("%s%s\n", TREE_BRANCH, "resource_type : "+upgrade.Spec.RESOURCE_TYPE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "upgrade_type : "+upgrade.Spec.UPGRADE_TYPE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "pvc_access_mode : "+upgrade.Spec.PVC_ACCESS_MODE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "pvc_size : "+upgrade.Spec.PVC_SIZE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "ccp_image_tag : "+upgrade.Spec.CCP_IMAGE_TAG)
	fmt.Printf("%s%s\n", TREE_BRANCH, "old_database_name : "+upgrade.Spec.OLD_DATABASE_NAME)
	fmt.Printf("%s%s\n", TREE_BRANCH, "new_database_name : "+upgrade.Spec.NEW_DATABASE_NAME)
	fmt.Printf("%s%s\n", TREE_BRANCH, "old_version : "+upgrade.Spec.OLD_VERSION)
	fmt.Printf("%s%s\n", TREE_BRANCH, "new_version : "+upgrade.Spec.NEW_VERSION)
	fmt.Printf("%s%s\n", TREE_BRANCH, "old_pvc_name : "+upgrade.Spec.OLD_PVC_NAME)
	fmt.Printf("%s%s\n", TREE_TRUNK, "new_pvc_name : "+upgrade.Spec.NEW_PVC_NAME)

	//print the upgrade jobs if any exists
	lo := v1.ListOptions{
		LabelSelector: "pg-database=" + upgrade.Spec.Name + ",pgupgrade=true",
	}
	log.Debug("label selector is " + lo.LabelSelector)
	pods, err2 := Clientset.Core().Pods(Namespace).List(lo)
	if err2 != nil {
		log.Error(err2.Error())
	}

	if len(pods.Items) == 0 {
		fmt.Printf("\nno upgrade job pods for database %s\n", upgrade.Spec.Name+" were found")
	} else {
		fmt.Printf("\nupgrade job pods for database %s\n", upgrade.Spec.Name+"...")
		for _, p := range pods.Items {
			fmt.Printf("%s pod : %s (%s)\n", TREE_TRUNK, p.Name, p.Status.Phase)
		}
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
		UPGRADE_TYPE:      UpgradeType,
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
