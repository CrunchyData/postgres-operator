package main

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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
	"flag"
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"strings"
)

var Clientset *kubernetes.Clientset

const backrestCommand = "pgbackrest"

const backrestRestoreCommand = `restore`
const containername = "database"

func main() {
	log.Info("pgo-backrest-restore starts")
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

	COMMAND_OPTS := os.Getenv("COMMAND_OPTS")
	log.Debugf("setting COMMAND_OPTS to %s", COMMAND_OPTS)

	PITR_TARGET := os.Getenv("PITR_TARGET")
	log.Debugf("setting PITR_TARGET to %s", PITR_TARGET)

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating Clientset")
		panic(err.Error())
	}

	cmdStrs := make([]string, 0)

	err = os.Mkdir(os.Getenv("PGBACKREST_DB_PATH"), 0770)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}
	log.Info("backrest Restore command requested")
	cmdStrs = append(cmdStrs, backrestCommand)
	cmdStrs = append(cmdStrs, backrestRestoreCommand)
	cmdStrs = append(cmdStrs, COMMAND_OPTS)
	if PITR_TARGET != "" {
		//cmdStrs = append(cmdStrs, "--target='"+PITR_TARGET+"'")
		cmdStrs = append(cmdStrs, "--target="+PITR_TARGET)
	}

	log.Infof("command to execute is [%s]", strings.Join(cmdStrs, " "))

	var cmd *exec.Cmd
	if PITR_TARGET != "" {
		//PITR_OPTS := "--target='" + PITR_TARGET + "'"
		PITR_OPTS := "--target=" + PITR_TARGET
		cmd = exec.Command(backrestCommand, backrestRestoreCommand, COMMAND_OPTS, PITR_OPTS)
	} else {
		cmd = exec.Command(backrestCommand, backrestRestoreCommand, COMMAND_OPTS)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	log.Infof("stdout=[%s]", stdout.String())
	log.Infof("stderr=[%s]", stderr.String())
	if err != nil {
		log.Fatal(err)
	}

	log.Info("pgo-backrest-restore ends")

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
