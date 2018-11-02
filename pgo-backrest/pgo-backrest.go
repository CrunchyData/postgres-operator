package main

import (
	"bytes"
	"flag"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"strings"
)

var Clientset *kubernetes.Clientset

const sourceCommand = `pgbackrest stanza-create --no-online && `
const backrestCommand = "pgbackrest"

const backrestBackupCommand = `backup`
const backrestInfoCommand = `info`
const backrestRestoreCommand = `restore`
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
	cmd := make([]string, 0)

	switch COMMAND {
	case crv1.PgtaskBackrestInfo:
		log.Info("backrest info command requested")
		cmd = append(cmd, sourceCommand)
		cmd = append(cmd, backrestCommand)
		cmd = append(cmd, backrestBackupCommand)
		cmd = append(cmd, COMMAND_OPTS)
	case crv1.PgtaskBackrestBackup:
		log.Info("backrest backup command requested")
		cmd = append(cmd, sourceCommand)
		cmd = append(cmd, backrestCommand)
		cmd = append(cmd, backrestBackupCommand)
		cmd = append(cmd, COMMAND_OPTS)
	case crv1.PgtaskBackrestRestore:
		err := os.Mkdir(os.Getenv("PGBACKREST_DB_PATH"), 0770)
		if err != nil {
			log.Error(err)
			os.Exit(2)
		}
		log.Info("backrest Restore command requested")
		cmd = append(cmd, sourceCommand)
		cmd = append(cmd, backrestCommand)
		cmd = append(cmd, backrestRestoreCommand)
		cmd = append(cmd, COMMAND_OPTS)
	default:
		log.Error("unsupported backup command specified " + COMMAND)
		os.Exit(2)
	}

	if COMMAND == crv1.PgtaskBackrestRestore {
		cmd := exec.Command(backrestCommand, backrestRestoreCommand, COMMAND_OPTS)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("output=[%s]", out.String())
	} else {
		log.Infof("command is %s ", strings.Join(cmd, " "))
		reader := strings.NewReader(strings.Join(cmd, " "))
		output, stderr, err := kubeapi.ExecToPodThroughAPI(config, Clientset, bashcmd, containername, PODNAME, Namespace, reader)
		if err != nil {
			log.Error(err)
		}
		log.Info("output=[" + output + "]")
		log.Info("stderr=[" + stderr + "]")
	}

	log.Info("pgo-backrest ends")

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
