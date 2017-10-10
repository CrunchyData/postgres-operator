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
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "perform a Backup",
	Long: `BACKUP performs a Backup, for example:
			pgo backup mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("backup called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`You must specify the cluster to backup or a selector flag.`)
		} else {
			createBackup(args)
		}

	},
}

func init() {
	RootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
}

func showBackup(args []string) {
	log.Debugf("showBackup called %v\n", args)

	//show pod information for job
	for _, arg := range args {
		log.Debug("show backup called for " + arg)
		//pg-database=basic or
		//pgbackup=true
		if arg == "all" {
			lo := meta_v1.ListOptions{LabelSelector: "pgbackup=true"}
			log.Debug("label selector is " + lo.LabelSelector)
			pods, err2 := Clientset.CoreV1().Pods(Namespace).List(lo)
			if err2 != nil {
				log.Error(err2.Error())
				return
			}
			for _, pod := range pods.Items {
				showBackupInfo(pod.ObjectMeta.Labels["pg-database"])
			}

		} else {
			showBackupInfo(arg)

		}

	}

}
func showBackupInfo(name string) {
	fmt.Println("\nbackup information for " + name + "...")
	//print the pgbackups TPR if it exists
	result := crv1.Pgbackup{}
	err := RestClient.Get().
		Resource(crv1.PgbackupResourcePlural).
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&result)
	if err == nil {
		printBackupTPR(&result)
	} else if errors.IsNotFound(err) {
		fmt.Println("\npgbackup TPR not found ")
	} else {
		log.Errorf("\npgbackup %s\n", name+" lookup error ")
		log.Error(err.Error())
		return
	}

	//print the backup jobs if any exists
	lo := meta_v1.ListOptions{LabelSelector: "pgbackup=true,pg-database=" + name}
	log.Debug("label selector is " + lo.LabelSelector)
	pods, err2 := Clientset.CoreV1().Pods(Namespace).List(lo)
	if err2 != nil {
		log.Error(err2.Error())
	}
	fmt.Printf("\nbackup job pods for database %s\n", name+"...")

	pvcMap := make(map[string]string)

	for _, p := range pods.Items {

		//get the pgdata volume info
		for _, v := range p.Spec.Volumes {
			if v.Name == "pgdata" {
				fmt.Printf("%s%s (pvc %s)\n\n", TREE_TRUNK, p.Name, v.VolumeSource.PersistentVolumeClaim.ClaimName)
				pvcMap[v.VolumeSource.PersistentVolumeClaim.ClaimName] = v.VolumeSource.PersistentVolumeClaim.ClaimName
			}
		}
		fmt.Println("")

	}

	log.Debugf("ShowPVC is %v\n", ShowPVC)

	if ShowPVC {
		//print pvc information for all jobs
		for key, _ := range pvcMap {
			PrintPVCListing(key)
		}
	}
}

func printBackupTPR(result *crv1.Pgbackup) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgbackup : "+result.Spec.Name)

	fmt.Printf("%s%s\n", TREE_BRANCH, "PVC Name:\t"+result.Spec.StorageSpec.PvcName)
	fmt.Printf("%s%s\n", TREE_BRANCH, "PVC Access Mode:\t"+result.Spec.StorageSpec.PvcAccessMode)
	fmt.Printf("%s%s\n", TREE_BRANCH, "PVC Size:\t\t"+result.Spec.StorageSpec.PvcSize)
	fmt.Printf("%s%s\n", TREE_BRANCH, "CCP_IMAGE_TAG:\t"+result.Spec.CCP_IMAGE_TAG)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup Status:\t"+result.Spec.BACKUP_STATUS)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup Host:\t"+result.Spec.BACKUP_HOST)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup User:\t"+result.Spec.BACKUP_USER)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup Pass:\t"+result.Spec.BACKUP_PASS)
	fmt.Printf("%s%s\n", TREE_TRUNK, "Backup Port:\t"+result.Spec.BACKUP_PORT)

}

func createBackup(args []string) {
	log.Debugf("createBackup called %v\n", args)

	var err error
	var newInstance *crv1.Pgbackup

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
		log.Debug("create backup called for " + arg)
		result := crv1.Pgbackup{}

		// error if it already exists
		err = RestClient.Get().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("pgbackup " + arg + " was found so we recreate it")
			dels := make([]string, 1)
			dels[0] = arg
			deleteBackup(dels)
			time.Sleep(2000 * time.Millisecond)
		} else if errors.IsNotFound(err) {
			log.Debug("pgbackup " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgbackup " + arg)
			log.Error(err.Error())
			break
		}
		// Create an instance of our TPR
		newInstance, err = getBackupParams(arg)
		if err != nil {
			log.Error("error creating backup")
			break
		}

		err = RestClient.Post().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error("error in creating Pgbackup TPR instance")
			log.Error(err.Error())
		}
		fmt.Println("created Pgbackup " + arg)

	}

}

func deleteBackup(args []string) {
	log.Debugf("deleteBackup called %v\n", args)
	var err error
	backupList := crv1.PgbackupList{}
	err = RestClient.Get().Resource(crv1.PgbackupResourcePlural).Do().Into(&backupList)
	if err != nil {
		log.Error("error getting backup list")
		log.Error(err.Error())
		return
	}
	// delete the pgbackup resource instance
	// which will cause the operator to remove the related Job
	for _, arg := range args {
		backupFound := false
		for _, backup := range backupList.Items {
			if arg == "all" || backup.Spec.Name == arg {
				backupFound = true
				err = RestClient.Delete().
					Resource(crv1.PgbackupResourcePlural).
					Namespace(Namespace).
					Name(backup.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgbackup " + arg)
					log.Error(err.Error())
				}
				fmt.Println("deleted pgbackup " + backup.Spec.Name)
			}

		}
		if !backupFound {
			fmt.Println("backup " + arg + " not found")
		}

	}

}

func getBackupParams(name string) (*crv1.Pgbackup, error) {
	var newInstance *crv1.Pgbackup

	storageSpec := crv1.PgStorageSpec{}
	spec := crv1.PgbackupSpec{}
	spec.Name = name
	spec.StorageSpec = storageSpec
	spec.StorageSpec.PvcName = viper.GetString("BACKUP_STORAGE.PVC_NAME")
	spec.StorageSpec.PvcAccessMode = viper.GetString("BACKUP_STORAGE.PVC_ACCESS_MODE")
	spec.StorageSpec.PvcSize = viper.GetString("BACKUP_STORAGE.PVC_SIZE")
	spec.StorageSpec.StorageClass = viper.GetString("BACKUP_STORAGE.STORAGE_CLASS")
	spec.StorageSpec.StorageType = viper.GetString("BACKUP_STORAGE.STORAGE_TYPE")
	spec.StorageSpec.SUPPLEMENTAL_GROUPS = viper.GetString("BACKUP_STORAGE.SUPPLEMENTAL_GROUPS")
	spec.StorageSpec.FSGROUP = viper.GetString("BACKUP_STORAGE.FSGROUP")
	spec.CCP_IMAGE_TAG = viper.GetString("CLUSTER.CCP_IMAGE_TAG")
	spec.BACKUP_STATUS = "initial"
	spec.BACKUP_HOST = "basic"
	spec.BACKUP_USER = "master"
	spec.BACKUP_PASS = "password"
	spec.BACKUP_PORT = "5432"

	cluster := crv1.Pgcluster{}
	err := RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&cluster)
	if err == nil {
		spec.BACKUP_HOST = cluster.Spec.Name
		//spec.BACKUP_USER = cluster.Spec.PG_MASTER_USER
		//spec.BACKUP_PASS = cluster.Spec.PG_MASTER_PASSWORD
		spec.BACKUP_PASS = GetSecretPassword(cluster.Spec.Name, crv1.PGMASTER_SECRET_SUFFIX)
		spec.BACKUP_PORT = cluster.Spec.Port
	} else if errors.IsNotFound(err) {
		log.Debug(name + " is not a cluster")
		return newInstance, err
	} else {
		log.Error("error getting pgcluster " + name)
		log.Error(err.Error())
		return newInstance, err
	}

	newInstance = &crv1.Pgbackup{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, nil
}

type PodTemplateFields struct {
	Name         string
	CO_IMAGE_TAG string
	BACKUP_ROOT  string
	PVC_NAME     string
}
