package main

import "github.com/crunchydata/postgres-operator/events"
import log "github.com/sirupsen/logrus"
import "os"

//import "bytes"
//import "encoding/json"

func main() {
	log.SetLevel(log.DebugLevel)
	log.Info("starting")
	//somebytes := []byte("some message")
	namespace := "pgouserx"
	username := "usernamex"
	someheader := events.EventHeader{
		Namespace: namespace,
		Username:  username,
	}
	e := events.EventReloadClusterFormat{
		EventHeader: someheader,
		Clustername: "alpha",
	}
	err := events.NewEventReloadCluster(&e)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	err = events.Publish(e)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("published a message")
	}

	someheader = events.EventHeader{
		Namespace: namespace,
		Username:  username,
	}
	f := events.EventCreateClusterFormat{
		EventHeader: someheader,
		Clustername: "betavalue",
	}

	err = events.NewEventCreateCluster(&f)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("published a message")
	}
}
