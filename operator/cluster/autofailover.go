// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	//crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	//"os"
	//"os/signal"
	"sync"
	//"syscall"
	"time"
)

// StateMachine holds a state machine that is created when
// a cluster has received a NotReady event, this is the start
// The StateMachine is executed in a separate goroutine for
// any cluster founds to be NotReady
type StateMachine struct {
	ClusterName  string
	SleepSeconds int
}

// FailoverEvent holds a record of a NotReady or other event that
// is used by the failover algorithm, FailoverEvents can build up
// for a given cluster
type FailoverEvent struct {
	EventType string
	EventTime time.Time
}

const FAILOVER_EVENT_NOT_READY = "NotReady"
const FAILOVER_EVENT_READY = "Ready"

// FailoverMap holds failover events for each cluster that
// has an associated set of failover events like a 'Not Ready'
// the map is made consistent with the Mutex, required since
// multiple goroutines will be mutating the map
// the key is the cluster name, the map entry is cleared upon
// a failover being triggered
type FailoverMap struct {
	sync.Mutex
	events map[string][]FailoverEvent
}

// GlobalFailoverMap holds the failover events for all clusters
// NotReady status events from the cluster will be logged in this map
// and processed by failover state machines as part of automated failover
var GlobalFailoverMap *FailoverMap

func InitializeAutoFailover(clientset *kubernetes.Clientset, ns string) {
	log.Infoln("autofailover Initialize map")
	GlobalFailoverMap = &FailoverMap{
		events: make(map[string][]FailoverEvent),
	}

	pods, _ := kubeapi.GetPods(clientset, util.LABEL_AUTOFAIL, ns)
	log.Infof("%d autofail pods found\n", len(pods.Items))

	for _, p := range pods.Items {
		clusterName := p.ObjectMeta.Labels[util.LABEL_PG_CLUSTER]
		for _, c := range p.Status.ContainerStatuses {
			if c.Name == "database" {
				if c.Ready {
					GlobalFailoverMap.AddEvent(clusterName, FAILOVER_EVENT_READY)
				} else {
					GlobalFailoverMap.AddEvent(clusterName, FAILOVER_EVENT_NOT_READY)
					sm := StateMachine{
						SleepSeconds: 9,
						ClusterName:  clusterName,
					}

					go sm.Run()

				}
			}
		}
	}

	GlobalFailoverMap.print()
}

func (s *StateMachine) Print() {

	log.Debugf("StateMachine: %s sleeping %d\n", s.ClusterName, s.SleepSeconds)
}

// Evaluate returns true if the failover events indicate a failover
// scenario, the algorithm is currently simple, if one of the
// failover events is a NotReady then we return true
func (s *StateMachine) Evaluate(events []FailoverEvent) bool {
	for k, v := range events {
		log.Debugf("event %d: %s\n", k, v.EventType)
		if v.EventType == FAILOVER_EVENT_NOT_READY {
			log.Debugf("Failover scenario caught: NotReady for %s\n", s.ClusterName)
			//here is where you would call the failover

			//clean up to not reprocess the failover event
			GlobalFailoverMap.Clear(s.ClusterName)
			return true

		}
	}
	return false
}

// Run is the heart of the failover state machine, started when a NotReady
// event is caught by the cluster watcher process, this statemachine
// runs until
func (s *StateMachine) Run() {

	for {
		time.Sleep(time.Second * time.Duration(s.SleepSeconds))
		s.Print()

		events := GlobalFailoverMap.GetEvents(s.ClusterName)
		if len(events) == 0 {
			log.Debugf("no events for statemachine, exiting")
			return
		}

		failoverRequired := s.Evaluate(events)
		if failoverRequired {
			log.Infof("failoverRequired is true, trigger failover on %s\n", s.ClusterName)
		} else {
			log.Infof("failoverRequired is false, no need to trigger failover\n")
			return
		}

	}

}

func (s *FailoverMap) print() {
	s.Lock()
	defer s.Unlock()

	log.Infoln("GlobalFailoverMap....")
	for k, v := range s.events {
		log.Infof("k=%s v=%v\n", k, v)
	}
}

func (s *FailoverMap) Clear(key string) {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.events[key]; !exists {
		log.Errorf("%s FailoverMap key doesnt exist during Clear\n", key)
	} else {
		delete(s.events, key)
		log.Infof("cleared FailoverMap for %s\n", key)
	}
}

func (s *FailoverMap) Exists(key string) bool {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.events[key]; !exists {
		return false
	}
	return true
}

func (s *FailoverMap) GetEvents(key string) []FailoverEvent {
	s.Lock()
	defer s.Unlock()

	if c, exists := s.events[key]; !exists {
		log.Errorf("GetEvents could not find key %s\n", key)
		return make([]FailoverEvent, 0)
	} else {
		return c
	}
}
func (s *FailoverMap) AddEvent(key, e string) {
	s.Lock()
	defer s.Unlock()

	_, exists := s.events[key]
	if !exists {
		log.Infof("no key for this add event, creating FailoverMap for %s\n", key)
		s.events[key] = make([]FailoverEvent, 0)
	}

	s.events[key] = append(s.events[key], FailoverEvent{EventType: e, EventTime: time.Now()})
}

// AutofailBase ...
func AutofailBase(ready bool, clusterName, namespace string) {
	log.Infof("AutofailBase ready=%v cluster=%s namespace=%s\n", ready, clusterName, namespace)

	exists := GlobalFailoverMap.Exists(clusterName)
	if exists {
		if !ready {
			//add notready event, start a state machine
			GlobalFailoverMap.AddEvent(clusterName, FAILOVER_EVENT_NOT_READY)
			//create a state machine to track the failovers for test cluster
			sm := StateMachine{
				SleepSeconds: 9,
				ClusterName:  clusterName,
			}

			go sm.Run()

		}

	} else {
		if ready {
			//add new map entry to keep an eye on it
			log.Infof("adding ready failover event for %s\n", clusterName)
			GlobalFailoverMap.AddEvent(clusterName, FAILOVER_EVENT_READY)
		}
	}

	/**
	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(client, &cluster,
		clusterName, namespace)
	if err != nil {
		return
	}

	if cluster.Spec.Strategy == "" {
		cluster.Spec.Strategy = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cluster.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster failover " + cluster.Spec.Strategy)
		return
	}

	strategy.Failover(clientset, client, clusterName, task, namespace, restconfig)
	*/

}
