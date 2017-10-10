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
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/labels"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

const MAJOR_UPGRADE = "major"
const MINOR_UPGRADE = "minor"
const SEP = "-"

var UpgradeType string

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "perform an upgrade",
	Long: `UPGRADE performs an upgrade, for example:
		pgo upgrade mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("upgrade called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`You must specify the cluster to upgrade or a selector value.`)
		} else {
			err := validateCreateUpdate(args)
			if err != nil {
				log.Error(err.Error())
			} else {

				createUpgrade(args)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")

	upgradeCmd.Flags().StringVarP(&UpgradeType, "upgrade-type", "t", "minor", "The upgrade type to perform either minor or major, default is minor ")
	upgradeCmd.Flags().StringVarP(&CCP_IMAGE_TAG, "ccp-image-tag", "c", "", "The CCP_IMAGE_TAG to use for the upgrade target")

}

func validateCreateUpdate(args []string) error {
	var err error

	if UpgradeType == MAJOR_UPGRADE || UpgradeType == MINOR_UPGRADE {
	} else {
		return errors.New("upgrade-type requires either a value of major or minor, if not specified, minor is the default value")
	}
	return err
}

func showUpgrade(args []string) {
	var err error
	log.Debugf("showUpgrade called %v\n", args)

	//show pod information for job
	for _, arg := range args {
		log.Debug("show upgrade called for " + arg)
		if arg == "all" {
			crv1s := crv1.PgupgradeList{}
			err = RestClient.Get().
				Resource(crv1.PgupgradeResourcePlural).
				Namespace(Namespace).
				Do().Into(&crv1s)
			if err != nil {
				log.Error("error getting list of pgupgrades " + err.Error())
				return
			}
			for _, u := range crv1s.Items {
				showUpgradeItem(&u)
			}

		} else {
			var upgrade crv1.Pgupgrade

			err = RestClient.Get().
				Resource(crv1.PgupgradeResourcePlural).
				Namespace(Namespace).
				Name(arg).
				Do().Into(&upgrade)
			if kerrors.IsNotFound(err) {
				fmt.Println("pgupgrade " + arg + " not found ")
			} else {
				showUpgradeItem(&upgrade)
			}

		}

	}

}

func showUpgradeItem(upgrade *crv1.Pgupgrade) {

	//print the CRD
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgupgrade : "+upgrade.Spec.Name)
	fmt.Printf("%s%s\n", TREE_BRANCH, "upgrade_status : "+upgrade.Spec.UPGRADE_STATUS)
	fmt.Printf("%s%s\n", TREE_BRANCH, "resource_type : "+upgrade.Spec.RESOURCE_TYPE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "upgrade_type : "+upgrade.Spec.UPGRADE_TYPE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "pvc_access_mode : "+upgrade.Spec.StorageSpec.PvcAccessMode)
	fmt.Printf("%s%s\n", TREE_BRANCH, "pvc_size : "+upgrade.Spec.StorageSpec.PvcSize)
	fmt.Printf("%s%s\n", TREE_BRANCH, "ccp_image_tag : "+upgrade.Spec.CCP_IMAGE_TAG)
	fmt.Printf("%s%s\n", TREE_BRANCH, "old_database_name : "+upgrade.Spec.OLD_DATABASE_NAME)
	fmt.Printf("%s%s\n", TREE_BRANCH, "new_database_name : "+upgrade.Spec.NEW_DATABASE_NAME)
	fmt.Printf("%s%s\n", TREE_BRANCH, "old_version : "+upgrade.Spec.OLD_VERSION)
	fmt.Printf("%s%s\n", TREE_BRANCH, "new_version : "+upgrade.Spec.NEW_VERSION)
	fmt.Printf("%s%s\n", TREE_BRANCH, "old_pvc_name : "+upgrade.Spec.OLD_PVC_NAME)
	fmt.Printf("%s%s\n", TREE_TRUNK, "new_pvc_name : "+upgrade.Spec.NEW_PVC_NAME)

	//print the upgrade jobs if any exists
	lo := meta_v1.ListOptions{
		LabelSelector: "pg-database=" + upgrade.Spec.Name + ",pgupgrade=true",
	}
	log.Debug("label selector is " + lo.LabelSelector)
	pods, err2 := Clientset.CoreV1().Pods(Namespace).List(lo)
	if err2 != nil {
		log.Error(err2.Error())
	}

	if len(pods.Items) == 0 {
		fmt.Printf("\nno upgrade job pods for %s\n", upgrade.Spec.Name+" were found")
	} else {
		fmt.Printf("\nupgrade job pods for %s\n", upgrade.Spec.Name+"...")
		for _, p := range pods.Items {
			fmt.Printf("%s pod : %s (%s)\n", TREE_TRUNK, p.Name, p.Status.Phase)
		}
	}

	fmt.Println("")

}

func createUpgrade(args []string) {
	log.Debugf("createUpgrade called %v\n", args)

	var err error
	var newInstance *crv1.Pgupgrade

	if Selector != "" {
		//use the selector instead of an argument list to filter on

		myselector, err := labels.Parse(Selector)
		if err != nil {
			log.Error("could not parse selector flag")
			return
		}

		//get the clusters list
		clusterList := crv1.PgclusterList{}
		err = RestClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(Namespace).
			LabelsSelectorParam(myselector).
			Do().
			Into(&clusterList)
		if err != nil {
			log.Error("error getting cluster list" + err.Error())
			return
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			args = newargs
		}

	}

	for _, arg := range args {
		log.Debug("create upgrade called for " + arg)
		result := crv1.Pgupgrade{}

		// error if it already exists
		err = RestClient.Get().
			Resource(crv1.PgupgradeResourcePlural).
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			log.Warn("previous pgupgrade " + arg + " was found so we will remove it.")
			forDeletion := make([]string, 1)
			forDeletion[0] = arg
			deleteUpgrade(forDeletion)
		} else if kerrors.IsNotFound(err) {
			log.Debug("pgupgrade " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgupgrade " + arg)
			log.Error(err.Error())
			break
		}

		cl := crv1.Pgcluster{}

		err = RestClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&cl)
		if kerrors.IsNotFound(err) {
			log.Error("error getting pgupgrade " + arg)
			break
		}

		if cl.Spec.MasterStorage.StorageType == "emptydir" {
			fmt.Println("cluster " + arg + " uses emptydir storage and can not be upgraded")
			break
		}

		// Create an instance of our CRD
		newInstance, err = getUpgradeParams(arg)
		if err == nil {
			err = RestClient.Post().
				Resource(crv1.PgupgradeResourcePlural).
				Namespace(Namespace).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error("error in creating Pgupgrade CRD instance", err.Error())
			} else {
				fmt.Println("created Pgupgrade " + arg)
			}
		}

	}

}

func deleteUpgrade(args []string) {
	log.Debugf("deleteUpgrade called %v\n", args)
	var err error
	upgradeList := crv1.PgupgradeList{}
	err = RestClient.Get().Resource(crv1.PgupgradeResourcePlural).Do().Into(&upgradeList)
	if err != nil {
		log.Error("error getting upgrade list")
		log.Error(err.Error())
		return
	}
	// delete the pgupgrade resource instance
	// which will cause the operator to remove the related Job
	for _, arg := range args {
		upgradeFound := false
		for _, upgrade := range upgradeList.Items {
			if arg == "all" || upgrade.Spec.Name == arg {
				upgradeFound = true
				err = RestClient.Delete().
					Resource(crv1.PgupgradeResourcePlural).
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
		if !upgradeFound {
			fmt.Println("upgrade " + arg + " not found")
		}

	}

}

func getUpgradeParams(name string) (*crv1.Pgupgrade, error) {

	var err error
	var existingImage string
	var existingMajorVersion float64

	spec := crv1.PgupgradeSpec{
		Name:              name,
		RESOURCE_TYPE:     "cluster",
		UPGRADE_TYPE:      UpgradeType,
		CCP_IMAGE_TAG:     viper.GetString("CLUSTER.CCP_IMAGE_TAG"),
		StorageSpec:       crv1.PgStorageSpec{},
		OLD_DATABASE_NAME: "basic",
		NEW_DATABASE_NAME: "master",
		OLD_VERSION:       "9.5",
		NEW_VERSION:       "9.6",
		OLD_PVC_NAME:      viper.GetString("MASTER_STORAGE.PVC_NAME"),
		NEW_PVC_NAME:      viper.GetString("MASTER_STORAGE.PVC_NAME"),
	}

	spec.StorageSpec.PvcAccessMode = viper.GetString("MASTER_STORAGE.PVC_ACCESS_MODE")
	spec.StorageSpec.PvcSize = viper.GetString("MASTER_STORAGE.PVC_SIZE")

	if CCP_IMAGE_TAG != "" {
		log.Debug("using CCP_IMAGE_TAG from command line " + CCP_IMAGE_TAG)
		spec.CCP_IMAGE_TAG = CCP_IMAGE_TAG
	}

	cluster := crv1.Pgcluster{}
	err = RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&cluster)
	if err == nil {
		spec.RESOURCE_TYPE = "cluster"
		spec.OLD_DATABASE_NAME = cluster.Spec.Name
		spec.NEW_DATABASE_NAME = cluster.Spec.Name + "-upgrade"
		spec.OLD_PVC_NAME = cluster.Spec.MasterStorage.PvcName
		spec.NEW_PVC_NAME = cluster.Spec.MasterStorage.PvcName + "-upgrade"
		spec.BACKUP_PVC_NAME = cluster.Spec.BACKUP_PVC_NAME
		existingImage = cluster.Spec.CCP_IMAGE_TAG
		existingMajorVersion = parseMajorVersion(cluster.Spec.CCP_IMAGE_TAG)
	} else if kerrors.IsNotFound(err) {
		log.Debug(name + " is not a cluster")
		return nil, err
	} else {
		log.Error("error getting pgcluster " + name)
		log.Error(err.Error())
		return nil, err
	}

	var requestedMajorVersion float64

	if CCP_IMAGE_TAG != "" {
		if CCP_IMAGE_TAG == existingImage {
			log.Error("CCP_IMAGE_TAG is the same as the cluster")
			log.Error("can't upgrade to the same image version")

			return nil, errors.New("invalid image tag")
		}
		requestedMajorVersion = parseMajorVersion(CCP_IMAGE_TAG)
	} else if viper.GetString("CLUSTER.CCP_IMAGE_TAG") == existingImage {
		log.Error("CCP_IMAGE_TAG is the same as the cluster")
		log.Error("can't upgrade to the same image version")

		return nil, errors.New("invalid image tag")
	} else {
		requestedMajorVersion = parseMajorVersion(viper.GetString("CLUSTER.CCP_IMAGE_TAG"))
	}

	if UpgradeType == MAJOR_UPGRADE {
		if requestedMajorVersion == existingMajorVersion {
			log.Error("can't upgrade to the same major version")
			return nil, errors.New("requested upgrade major version can not equal existing upgrade major version")
		} else if requestedMajorVersion < existingMajorVersion {
			log.Error("can't upgrade to a previous major version")
			return nil, errors.New("requested upgrade major version can not be older than existing upgrade major version")
		}
	} else {
		//minor upgrade
		if requestedMajorVersion > existingMajorVersion {
			log.Error("can't do minor upgrade to a newer major version")
			return nil, errors.New("requested minor upgrade to major version is not allowed")
		}
	}

	newInstance := &crv1.Pgupgrade{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, err
}

func parseMajorVersion(st string) float64 {
	parts := strings.Split(st, SEP)
	//OS = parts[0]
	//PGVERSION = parts[1]
	//CVERSION = parts[2]

	f, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		fmt.Println(err.Error())
	}
	return f

}
