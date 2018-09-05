package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
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

//type FailoverMapAverage FailoverMap

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

// GlobalFailoverMap holds the failover events for all clusters
// NotReady status events from the cluster will be logged in this map
// and processed by failover state machines as part of automated failover
var GlobalFailoverMap *FailoverMap

func main() {
	//initialize a global failover map
	GlobalFailoverMap = &FailoverMap{
		events: make(map[string][]FailoverEvent),
	}

	time.Sleep(time.Millisecond)

	//simulate contention on the GlobalFailoverMap with multiple goroutine
	go GlobalFailoverMap.AddEvent("test", "someevent")
	go GlobalFailoverMap.AddEvent("test", FAILOVER_EVENT_NOT_READY)
	go GlobalFailoverMap.AddEvent("test2", "2someevent")

	time.Sleep(time.Millisecond)

	//simulate getting a lock on the GlobalFailoverMap
	GlobalFailoverMap.Lock()
	log.Printf("%#v\n", GlobalFailoverMap.events)
	GlobalFailoverMap.Unlock()

	GlobalFailoverMap.print()

	//create a state machine to track the failovers for test cluster
	sm1 := StateMachine{
		SleepSeconds: 9,
		ClusterName:  "test",
	}

	go sm1.Run()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-signals:
			log.Printf("received signal %#v, exiting...\n", s)
			os.Exit(0)
		}
	}

}
