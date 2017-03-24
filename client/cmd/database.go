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
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
)

func showDatabase(args []string) {
	//get a list of all databases
	databaseList := tpr.PgDatabaseList{}
	err := Tprclient.Get().Resource("pgdatabases").Do().Into(&databaseList)
	if err != nil {
		log.Error("error getting list of databases" + err.Error())
		return
	}

	//each arg represents a database name or the special 'all' value
	var pod *v1.Pod
	var service *v1.Service
	for _, arg := range args {
		log.Debug("show database " + arg)
		for _, database := range databaseList.Items {
			if arg == "all" || database.Spec.Name == arg {
				fmt.Println("database : " + database.Spec.Name)
				pod, err = Clientset.Core().Pods(api.NamespaceDefault).Get(database.Spec.Name)
				if err != nil {
					log.Error("error in getting database pod " + database.Spec.Name + err.Error())
				} else {
					fmt.Println(TREE_BRANCH + "pod : " + pod.Name + " (" + string(pod.Status.Phase) + ")")
				}

				service, err = Clientset.Core().Services(api.NamespaceDefault).Get(database.Spec.Name)
				if err != nil {
					log.Error("error in getting database service " + database.Spec.Name + err.Error())
				} else {
					fmt.Println(TREE_TRUNK + "service : " + service.Name + " (" + service.Spec.ClusterIP + ")")

				}
			}
		}
	}
}

func createDatabase(args []string) {

	var err error

	for _, arg := range args {
		log.Debug("create database called for " + arg)
		result := tpr.PgDatabase{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("pgdatabases").
			Namespace(api.NamespaceDefault).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("pgdatabase " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			log.Debug("pgdatabase " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgdatabase " + arg + err.Error())
			break
		}

		// Create an instance of our TPR
		newInstance := getDatabaseParams(arg)

		err = Tprclient.Post().
			Resource("pgdatabases").
			Namespace(api.NamespaceDefault).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error("error in creating PgDatabase TPR instance" + err.Error())
		}
		fmt.Println("created PgDatabase " + arg)

	}
}

func getDatabaseParams(name string) *tpr.PgDatabase {

	//set to internal defaults
	spec := tpr.PgDatabaseSpec{
		Name:                name,
		PVC_NAME:            "",
		PVC_SIZE:            "100M",
		PVC_ACCESS_MODE:     "ReadWriteMany",
		Port:                "5432",
		CCP_IMAGE_TAG:       "centos7-9.5-1.2.8",
		PG_MASTER_USER:      "master",
		PG_MASTER_PASSWORD:  "password",
		PG_USER:             "testuser",
		PG_PASSWORD:         "password",
		PG_DATABASE:         "userdb",
		PG_ROOT_PASSWORD:    "password",
		BACKUP_PVC_NAME:     "",
		BACKUP_PATH:         "",
		FS_GROUP:            "",
		SUPPLEMENTAL_GROUPS: "",
	}

	//override any values from config file
	str := viper.GetString("db.CCP_IMAGE_TAG")
	if str != "" {
		spec.CCP_IMAGE_TAG = str
	}
	str = viper.GetString("db.Port")
	if str != "" {
		spec.Port = str
	}
	str = viper.GetString("db.PVC_NAME")
	if str != "" {
		spec.PVC_NAME = str
	}
	str = viper.GetString("db.PVC_SIZE")
	if str != "" {
		spec.PVC_SIZE = str
	}
	str = viper.GetString("db.PVC_ACCESS_MODE")
	if str != "" {
		spec.PVC_ACCESS_MODE = str
	}
	str = viper.GetString("db.PG_MASTER_USER")
	if str != "" {
		spec.PG_MASTER_USER = str
	}
	str = viper.GetString("db.PG_MASTER_PASSWORD")
	if str != "" {
		spec.PG_MASTER_PASSWORD = str
	}
	str = viper.GetString("db.PG_USER")
	if str != "" {
		spec.PG_USER = str
	}
	str = viper.GetString("db.PG_PASSWORD")
	if str != "" {
		spec.PG_PASSWORD = str
	}
	str = viper.GetString("db.PG_DATABASE")
	if str != "" {
		spec.PG_DATABASE = str
	}
	str = viper.GetString("db.PG_ROOT_PASSWORD")
	if str != "" {
		spec.PG_ROOT_PASSWORD = str
	}
	str = viper.GetString("db.fsGroup")
	if str != "" {
		spec.FS_GROUP = str
	}
	str = viper.GetString("db.supplementalGroups")
	if str != "" {
		spec.SUPPLEMENTAL_GROUPS = str
	}

	//pass along command line flags for a restore
	spec.BACKUP_PATH = BackupPath
	spec.BACKUP_PVC_NAME = BackupPVC

	//override from command line

	newInstance := &tpr.PgDatabase{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance
}

func deleteDatabase(args []string) {
	var err error
	//result := tpr.PgDatabaseList{}
	databaseList := tpr.PgDatabaseList{}
	err = Tprclient.Get().Resource("pgdatabases").Do().Into(&databaseList)
	if err != nil {
		log.Error("error getting database list" + err.Error())
		return
	}
	// delete the pgdatabase resource instance
	for _, arg := range args {
		for _, database := range databaseList.Items {
			if arg == "all" || database.Spec.Name == arg {
				log.Debug("deleting pgdatabase " + arg)
				err = Tprclient.Delete().
					Resource("pgdatabases").
					Namespace(api.NamespaceDefault).
					Name(database.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgdatabase " + arg + err.Error())
				}
				fmt.Println("deleted pgdatabase " + database.Spec.Name)
			}

		}

	}
}
