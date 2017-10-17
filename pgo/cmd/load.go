// Package cmd provides the command line functions of the crunchy CLI
package cmd

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

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	"os"
	"text/template"
)

type loadJobTemplateFields struct {
	Name            string
	COImageTag      string
	DbHost          string
	DbDatabase      string
	DbUser          string
	DbPass          string
	DbPort          string
	TableToLoad     string
	CSVFilePath     string
	PVCName         string
	SecurityContext string
}

// LoadConfig ...
var LoadConfig string

// LoadConfigTemplate ....
var LoadConfigTemplate loadJobTemplateFields

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "perform a data load",
	Long: `LOAD performs a load, for example:
			pgo load --load-config=./load.json --selector=project=xray`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("load called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`You must specify the cluster to load or a selector flag.`)
		} else {
			if LoadConfig == "" {
				fmt.Println("You must specify the load-config ")
				return
			}

			createLoad(args)
		}

	},
}

// CSVLoadTemplatePath ...
var CSVLoadTemplatePath string

// JobTemplate ...
var JobTemplate *template.Template

func init() {
	RootCmd.AddCommand(loadCmd)

	loadCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	loadCmd.Flags().StringVarP(&LoadConfig, "load-config", "l", "", "The load configuration to use that defines the load job")
	log.Debug(" csvload config is " + viper.GetString("Pgo.CsvloadTemplate"))
}

func createLoad(args []string) {
	CSVLoadTemplatePath = viper.GetString("Pgo.CSVLoadTemplate")
	if CSVLoadTemplatePath == "" {
		log.Error("Pgo.CSVLoadTemplate not defined in pgo config.")
		os.Exit(2)
	}
	getLoadConfigFile()
	fmt.Println("using csvload template from " + CSVLoadTemplatePath)
	getJobTemplate()
	log.Debugf("createLoad called %v\n", args)

	//var err error

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
		log.Debug("load called for " + arg)

		fmt.Println("created load for " + arg)
		createJob(Clientset, arg, Namespace)

	}

}

func getLoadConfigFile() {
	LoadConfigTemplate = loadJobTemplateFields{}
	viper.SetConfigFile(LoadConfig)
	err := viper.ReadInConfig()
	if err == nil {
		log.Debugf("Using load config file: %s\n", viper.ConfigFileUsed())
	} else {
		log.Error("load config file not found")
		os.Exit(2)
	}

	//LoadConfigTemplate.Name = viper.GetString("Name")
	LoadConfigTemplate.COImageTag = viper.GetString("COImageTag")
	//LoadConfigTemplate.DbHost = viper.GetString("DbHost")
	LoadConfigTemplate.DbDatabase = viper.GetString("DbDatabase")
	LoadConfigTemplate.DbUser = viper.GetString("DbUser")
	//LoadConfigTemplate.DbPass = viper.GetString("DbPass")
	LoadConfigTemplate.DbPort = viper.GetString("DbPort")
	LoadConfigTemplate.TableToLoad = viper.GetString("TableToLoad")
	LoadConfigTemplate.CSVFilePath = viper.GetString("CSVFilePath")
	LoadConfigTemplate.PVCName = viper.GetString("PVCName")
	LoadConfigTemplate.SecurityContext = viper.GetString("SecurityContext")

}

func getJobTemplate() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(CSVLoadTemplatePath)
	if err != nil {
		log.Error("error loading csvload job template..." + err.Error())
		os.Exit(2)
	}
	JobTemplate = template.Must(template.New("csvload job template").Parse(string(buf)))

}

func createJob(clientset *kubernetes.Clientset, clusterName string, namespace string) {
	var err error

	LoadConfigTemplate.Name = "csvload-" + clusterName
	LoadConfigTemplate.DbHost = clusterName
	LoadConfigTemplate.DbPass = GetSecretPassword(clusterName, crv1.RootSecretSuffix)

	var doc2 bytes.Buffer
	err = JobTemplate.Execute(&doc2, LoadConfigTemplate)
	if err != nil {
		log.Error(err.Error())
		return
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	resultJob, err := Clientset.Batch().Jobs(Namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return
	}
	log.Info("created load Job " + resultJob.Name)

}
