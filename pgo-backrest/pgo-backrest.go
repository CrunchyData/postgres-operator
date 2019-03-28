package main

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Clientset *kubernetes.Clientset

const backrestCommand = "pgbackrest"

const backrestBackupCommand = `backup`
const backrestInfoCommand = `info`
const backrestStanzaCreateCommand = `stanza-create`
const containername = "database"
const repoTypeFlagS3 = "--repo-type=s3"

func main() {
	log.Info("pgo-backrest starts")
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	Namespace := os.Getenv("NAMESPACE")
	log.Debugf("setting NAMESPACE to %s", Namespace)
	if Namespace == "" {
		log.Error("NAMESPACE env var not set")
		os.Exit(2)
	}

	COMMAND := os.Getenv("COMMAND")
	log.Debugf("setting COMMAND to %s", COMMAND)
	if COMMAND == "" {
		log.Error("COMMAND env var not set")
		os.Exit(2)
	}

	COMMAND_OPTS := os.Getenv("COMMAND_OPTS")
	log.Debugf("setting COMMAND_OPTS to %s", COMMAND_OPTS)

	PODNAME := os.Getenv("PODNAME")
	log.Debugf("setting PODNAME to %s", PODNAME)
	if PODNAME == "" {
		log.Error("PODNAME env var not set")
		os.Exit(2)
	}

	REPO_TYPE := os.Getenv("PGBACKREST_REPO_TYPE")
	log.Debugf("setting REPO_TYPE to %s", REPO_TYPE)

	BACKREST_LOCAL_AND_S3_STORAGE, err := strconv.ParseBool(os.Getenv("BACKREST_LOCAL_AND_S3_STORAGE"))
	if err != nil {
		panic(err)
	}
	log.Debugf("setting BACKREST_LOCAL_AND_S3_STORAGE to %s", BACKREST_LOCAL_AND_S3_STORAGE)

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating Clientset")
		panic(err.Error())
	}

	bashcmd := make([]string, 1)
	bashcmd[0] = "bash"
	cmdStrs := make([]string, 0)

	switch COMMAND {
	case crv1.PgtaskBackrestStanzaCreate:
		log.Info("backrest stanza-create command requested")
		time.Sleep(time.Second * time.Duration(30))
		log.Info("sleeping 30 seconds to avoid race with PG during startup")
		cmdStrs = append(cmdStrs, backrestCommand)
		cmdStrs = append(cmdStrs, backrestStanzaCreateCommand)
		cmdStrs = append(cmdStrs, COMMAND_OPTS)
	case crv1.PgtaskBackrestInfo:
		log.Info("backrest info command requested")
		cmdStrs = append(cmdStrs, backrestCommand)
		cmdStrs = append(cmdStrs, backrestInfoCommand)
		cmdStrs = append(cmdStrs, COMMAND_OPTS)
	case crv1.PgtaskBackrestBackup:
		log.Info("backrest backup command requested")
		cmdStrs = append(cmdStrs, backrestCommand)
		cmdStrs = append(cmdStrs, backrestBackupCommand)
		cmdStrs = append(cmdStrs, COMMAND_OPTS)
	default:
		log.Error("unsupported backup command specified " + COMMAND)
		os.Exit(2)
	}

	if BACKREST_LOCAL_AND_S3_STORAGE {
		firstCmd := cmdStrs
		cmdStrs = append(cmdStrs, "&&")
		cmdStrs = append(cmdStrs, strings.Join(firstCmd, " "))
		cmdStrs = append(cmdStrs, repoTypeFlagS3)
		log.Info("backrest command will be executed for both local and s3 storage")
	} else if REPO_TYPE == "s3" {
		cmdStrs = append(cmdStrs, repoTypeFlagS3)
		log.Info("s3 flag enabled for backrest command")
	}

	log.Infof("command to execute is [%s]", strings.Join(cmdStrs, " "))

	log.Infof("command is %s ", strings.Join(cmdStrs, " "))
	reader := strings.NewReader(strings.Join(cmdStrs, " "))
	output, stderr, err := kubeapi.ExecToPodThroughAPI(config, Clientset, bashcmd, containername, PODNAME, Namespace, reader)
	if err != nil {
		log.Info("output=[" + output + "]")
		log.Info("stderr=[" + stderr + "]")
		log.Error(err)
		os.Exit(2)
	}
	log.Info("output=[" + output + "]")
	log.Info("stderr=[" + stderr + "]")

	log.Info("pgo-backrest ends")

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
