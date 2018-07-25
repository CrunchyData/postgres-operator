package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var Clientset *kubernetes.Clientset
var COMMAND, PODNAME, Namespace string

const backrestCommand = "pgbackrest"

//pgbackrest --stanza=db backup
const backrestStanza = "--stanza=db"
const backrestBackupCommand = "backup"
const backrestInfoCommand = "info"
const containername = "database"

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

	Namespace = os.Getenv("NAMESPACE")
	log.Debug("setting NAMESPACE to " + Namespace)
	if Namespace == "" {
		log.Error("NAMESPACE env var not set")
		os.Exit(2)
	}

	COMMAND = os.Getenv("COMMAND")
	log.Debug("setting COMMAND to " + COMMAND)
	if COMMAND == "" {
		log.Error("COMMAND env var not set")
		os.Exit(2)
	}
	PODNAME = os.Getenv("PODNAME")
	log.Debug("setting PODNAME to " + PODNAME)
	if PODNAME == "" {
		log.Error("PODNAME env var not set")
		os.Exit(2)
	}

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating Clientset")
		panic(err.Error())
	}

	cmd := make([]string, 0)

	switch COMMAND {
	case crv1.PgtaskBackrestInfo:
		log.Info("backrest info command requested")
		cmd = append(cmd, "ls")
		cmd = append(cmd, "/pgdata")
		//pgbackrest --stanza=db info
		//cmd = append(cmd, backrestCommand)
		//cmd = append(cmd, backrestStanza)
		//cmd = append(cmd, backrestBackupCommand)
	case crv1.PgtaskBackrestBackup:
		log.Info("backrest backup command requested")
		cmd = append(cmd, "ls")
		cmd = append(cmd, "/pgdata")
	//pgbackrest --stanza=db backup
	//cmd = append(cmd, backrestCommand)
	//cmd = append(cmd, backrestStanza)
	//cmd = append(cmd, backrestBackupCommand)
	default:
		log.Error("unsupported backup command specified " + COMMAND)
		os.Exit(2)
	}

	err = util.Exec(config, Namespace, PODNAME, containername, cmd)
	if err != nil {
		log.Error(err)
	}
	log.Info("pgo-backrest ends")

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
