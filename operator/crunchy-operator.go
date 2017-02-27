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

package main

import (
	"github.com/crunchydata/fishysmell/operatorcontroller"
	//	"k8s.io/client-go/kubernetes"
	//	"k8s.io/client-go/pkg/api"
	//	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/record"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var logger *log.Logger
var NAMESPACE string

func main() {
	logger = log.New(os.Stdout, "logger: ", log.Lshortfile|log.Ldate|log.Ltime)
	//set up signal catcher logic
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Println(sig)
		done <- true
		logger.Println("crunchy-operator caught signal, exiting...")
		os.Exit(0)
	}()

	logger.Println("crunchy-operator v.1.2.9 starting")

	getEnvVars()

	// Workaround for watching TPR resource.
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	operatorcontroller.MasterHost = restCfg.Host
	restcli, err := operatorcontroller.NewTPRClient()
	if err != nil {
		panic(err)
	}
	operatorcontroller.KubeHttpCli = restcli.Client

	logger.Printf("crunchy-operator: NAMESPACE %s\n", NAMESPACE)

	for true {
		time.Sleep(time.Duration(1) * time.Minute)
		process()
	}

}

func process() {
	logger.Println("processing...")
	operatorcontroller.DoSomething()

}

func getEnvVars() error {
	var err error
	var tempval = os.Getenv("NAMESPACE")
	if tempval != "" {
		NAMESPACE = tempval
	} else {
		logger.Println("error in NAMESPACE env var, not set")
		os.Exit(2)
	}

	return err

}
