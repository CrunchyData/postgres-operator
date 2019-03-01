package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	schedulerLabel = "crunchy-scheduler=true"
	namespaceEnv   = "NAMESPACE"
	timeoutEnv     = "TIMEOUT"
	inCluster      = true
)

var namespace string
var timeout time.Duration
var seconds int
var kubeClient *kubernetes.Clientset

func init() {
	var err error
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	namespace = os.Getenv(namespaceEnv)
	if namespace == "" {
		log.WithFields(log.Fields{}).Fatalf("Failed to get namespace environment: %s", namespaceEnv)
	}

	secondsEnv := os.Getenv(timeoutEnv)
	seconds = 300
	if secondsEnv == "" {
		log.WithFields(log.Fields{}).Info("No timeout set, defaulting to 300 seconds")
	} else {
		seconds, err = strconv.Atoi(secondsEnv)
		if err != nil {
			log.WithFields(log.Fields{}).Fatalf("Failed to convert timeout env to seconds: %s", err)
		}
	}

	log.WithFields(log.Fields{}).Infof("Setting timeout to: %d", seconds)
	timeout = time.Second * time.Duration(seconds)

	kubeClient, err = newKubeClient()
	if err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to connect to kubernetes: %s", err)
	}

	if err := scheduler.Init(); err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to open template: %s", err)
	}
}

func main() {
	log.Info("Starting Crunchy Scheduler")
	scheduler := scheduler.New(schedulerLabel, namespace, kubeClient)
	scheduler.CronClient.Start()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.WithFields(log.Fields{
			"signal": sig,
		}).Warning("Received signal")
		done <- true
	}()

	go func() {
		for {
			if err := scheduler.AddSchedules(); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed to add schedules")
			}
			time.Sleep(time.Second * 10)
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 10)
			if err := scheduler.DeleteSchedules(); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed to delete schedules")
			}
		}
	}()

	for {
		select {
		case <-done:
			log.Warning("Shutting down scheduler")
			scheduler.CronClient.Stop()
			os.Exit(0)
		default:
			time.Sleep(time.Second * 1)
		}
	}
}

func newKubeClient() (*kubernetes.Clientset, error) {
	var client *kubernetes.Clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		return client, err
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return client, err
	}
	return client, nil
}
