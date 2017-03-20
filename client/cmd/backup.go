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
	"github.com/crunchydata/postgres-operator/tpr"
	//"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
)

func showBackup(args []string) {
	fmt.Printf("showBackup called %v\n", args)
	var err error

	var pod *v1.Pod

	//show pod information for job
	for _, arg := range args {
		fmt.Println("show backup called for " + arg)
		//pg-database=basic or
		//pgbackup=true
		if arg == "all" {
			lo := v1.ListOptions{LabelSelector: "pgbackup=true"}
			fmt.Println("label selector is " + lo.LabelSelector)
			pods, err2 := Clientset.Core().Pods(api.NamespaceDefault).List(lo)
			if err2 != nil {
				panic(err2.Error())
			}
			fmt.Printf("There are %d backup job pods in the cluster\n", len(pods.Items))
			for _, pod := range pods.Items {
				fmt.Println("backup pod Name " + pod.ObjectMeta.Name)
				fmt.Println("backup pod phase is " + pod.Status.Phase)
			}

		} else {
			pod, err = Clientset.Core().Pods(api.NamespaceDefault).Get(arg)
			if err != nil {
				fmt.Println("error in getting backup job pod " + arg)
				fmt.Println(err.Error())
			} else {
				fmt.Println(TREE_BRANCH + "pod " + pod.Name)
			}

		}

	}

}

func createBackup(args []string) {
	fmt.Printf("createBackup called %v\n", args)
	var err error

	for _, arg := range args {
		fmt.Println("create backup called for " + arg)
		result := tpr.PgBackup{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("pgbackups").
			Namespace(api.NamespaceDefault).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("pgbackup " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			fmt.Println("pgbackup " + arg + " not found so we will create it")
		} else {
			fmt.Println("error getting pgbackup " + arg)
			fmt.Println(err.Error())
			break
		}
		// Create an instance of our TPR
		newInstance := getBackupParams(arg)

		err = Tprclient.Post().
			Resource("pgbackups").
			Namespace(api.NamespaceDefault).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			fmt.Println("error in creating PgBackup TPR instance")
			fmt.Println(err.Error())
		}
		fmt.Println("created PgBackup " + arg)

	}

}

func deleteBackup(args []string) {
	fmt.Printf("deleteBackup called %v\n", args)
	var err error
	backupList := tpr.PgBackupList{}
	err = Tprclient.Get().Resource("pgbackups").Do().Into(&backupList)
	if err != nil {
		fmt.Println("error getting backup list")
		fmt.Println(err.Error())
		return
	}
	// delete the pgbackup resource instance
	for _, arg := range args {
		for _, backup := range backupList.Items {
			if arg == "all" || backup.Spec.Name == arg {
				err = Tprclient.Delete().
					Resource("pgbackups").
					Namespace(api.NamespaceDefault).
					Name(backup.Spec.Name).
					Do().
					Error()
				if err != nil {
					fmt.Println("error deleting pgbackup " + arg)
					fmt.Println(err.Error())
				}
				fmt.Println("deleted pgbackup " + backup.Spec.Name)
			}

		}

	}

	//delete Job
}

func getBackupParams(name string) *tpr.PgBackup {

	//TODO see if name is a database or cluster

	spec := tpr.PgBackupSpec{}
	spec.Name = name
	spec.PVC_NAME = "crunchy-pvc"
	spec.CCP_IMAGE_TAG = "centos7-9.5-1.2.8"
	spec.BACKUP_HOST = "basic"
	spec.BACKUP_USER = "master"
	spec.BACKUP_PASS = "password"
	spec.BACKUP_PORT = "5432"

	newInstance := &tpr.PgBackup{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance
}
