// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strconv"
	"time"
)

// StateMachine holds a state machine that is created when
// a cluster has received a NotReady event, this is the start
// The StateMachine is executed in a separate goroutine for
// any cluster founds to be NotReady
type StateMachine struct {
	Clientset    *kubernetes.Clientset
	RESTClient   *rest.RESTClient
	Namespace    string
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

type AutoFailoverTask struct {
}

//at operator startup, check for autofail enabled pods in Not Ready status
//in each namespace the operator is watching, trigger a failover if found
func InitializeAutoFailover(clientset *kubernetes.Clientset, restclient *rest.RESTClient, nsList []string) error {
	var err error
	aftask := AutoFailoverTask{}
	selector := config.LABEL_AUTOFAIL + "=true"

	for i := 0; i < len(nsList); i++ {
		ns := nsList[i]

		log.Infof("autofailover Initialize ns=%s ", ns)

		clusterList := crv1.PgclusterList{}

		err = kubeapi.GetpgclustersBySelector(restclient, &clusterList, selector, ns)
		if err != nil {
			log.Error(err)
			log.Error("could not InitializeAutoFailover")
			return err
		}
		log.Debugf("InitializeAutoFailover ns %s  pgclusters %d", ns, len(clusterList.Items))

		for i := 0; i < len(clusterList.Items); i++ {
			cl := clusterList.Items[i]
			clusterName := cl.Name
			selector := "service-name=" + clusterName
			pods, err := kubeapi.GetPods(clientset, selector, ns)
			if err != nil {
				log.Error(err)
				log.Error("could not InitializeAutoFailover")
				return err
			}
			if len(pods.Items) == 0 {
				log.Errorf("could not InitializeAutoFailover: zero primary pods were found for cluster %s", cl.Name)
				return err
			}

			p := pods.Items[0]
			for _, c := range p.Status.ContainerStatuses {
				if c.Name == "database" {
					if c.Ready {
						aftask.AddEvent(restclient, clusterName, FAILOVER_EVENT_READY, ns)
					} else {
						aftask.AddEvent(restclient, clusterName, FAILOVER_EVENT_NOT_READY, ns)
						secs, _ := strconv.Atoi(operator.Pgo.Pgo.AutofailSleepSeconds)
						log.Debugf("InitializeAutoFailover: started state machine for cluster %s", clusterName)
						sm := StateMachine{
							Clientset:    clientset,
							RESTClient:   restclient,
							Namespace:    ns,
							SleepSeconds: secs,
							ClusterName:  clusterName,
						}

						go sm.Run()

					}
				}
			}
		}

		aftask.Print(restclient, ns)
	}
	return err
}

func (s *StateMachine) Print() {

	log.Debugf("StateMachine: %s sleeping %d\n", s.ClusterName, s.SleepSeconds)
}

// Evaluate returns true if the autofail status is NotReady
func (s *StateMachine) Evaluate(status string, events map[string]string) bool {

	if status == FAILOVER_EVENT_NOT_READY {
		log.Debugf("Failover scenario caught: NotReady for %s\n", s.ClusterName)
		return true
	}

	return false
}

// Run is the heart of the failover state machine, started when a NotReady
// event is caught by the cluster watcher process, this statemachine
// runs until
func (s *StateMachine) Run() {

	log.Debugf("autofail Run called %s", s.ClusterName)

	//	aftask := AutoFailoverTask{}

	time.Sleep(time.Second * time.Duration(s.SleepSeconds))
	s.Print()

	//	status, events := aftask.GetEvents(s.RESTClient, s.ClusterName, s.Namespace)
	//	if len(events) == 0 {
	//		log.Debugf("no events for statemachine, exiting")
	//		return
	//	}

	//failoverRequired := s.Evaluate(status, events)
	failoverRequired := true
	if failoverRequired {
		log.Infof("failoverRequired is true, trigger failover on %s\n", s.ClusterName)
		s.triggerFailover()
		//clean up to not reprocess the failover event
		//		aftask.Clear(s.RESTClient, s.ClusterName, s.Namespace)
		//recreate a new autofail task to start anew (3.5.1)
		//aftask.AddEvent(s.RESTClient, s.ClusterName, FAILOVER_EVENT_NOT_READY, s.Namespace)
		//		aftask.AddEvent(s.RESTClient, s.ClusterName, FAILOVER_EVENT_READY, s.Namespace)
	} else {
		log.Infof("failoverRequired is false, no need to trigger failover\n")
	}

	//right now, there is no need for looping with this
	//simple failover check algorithm, later this loop
	//will be necessary potentially if the logic evaluates
	//failures over a span of time
	return

}

// AutofailBase ...
func AutofailBase(clientset *kubernetes.Clientset, restclient *rest.RESTClient, ready bool, clusterName, namespace string) {
	log.Infof("AutofailBase ready=%v cluster=%s namespace=%s\n", ready, clusterName, namespace)

	log.Debugf("autofail base with sleep secs = %d", operator.Pgo.Pgo.AutofailSleepSecondsValue)
	//aftask := AutoFailoverTask{}

	//exists := aftask.Exists(restclient, clusterName, namespace)
	//if exists {
	if !ready {
		log.Debugf("AutofailBase called with a not ready status which means we start a state machine to track it")
		//add notready event, start a state machine
		//		aftask.AddEvent(restclient, clusterName, FAILOVER_EVENT_NOT_READY, namespace)
		//create a state machine to track the failovers for test cluster
		sm := StateMachine{
			Clientset:    clientset,
			RESTClient:   restclient,
			Namespace:    namespace,
			SleepSeconds: operator.Pgo.Pgo.AutofailSleepSecondsValue,
			ClusterName:  clusterName,
		}

		go sm.Run()

	} else {
		log.Debugf("AutofailBase called with a ready status which means we don't do anything")
	}

	//} else {
	//we only register the autofail target once it
	//goes into a Ready status for the first time
	//if ready {
	//add new map entry to keep an eye on it
	//log.Infof("adding ready failover event for %s\n", clusterName)
	//aftask.AddEvent(restclient, clusterName, FAILOVER_EVENT_READY, namespace)
	//}
	//}

}

func (*AutoFailoverTask) Exists(restclient *rest.RESTClient, clusterName, namespace string) bool {
	task := crv1.Pgtask{}
	taskName := clusterName + "-" + config.LABEL_AUTOFAIL
	found, _ := kubeapi.Getpgtask(restclient, &task, taskName, namespace)
	return found
}

func (*AutoFailoverTask) AddEvent(restclient *rest.RESTClient, clusterName, eventType, namespace string) {
	var err error
	var found bool

	taskName := clusterName + "-" + config.LABEL_AUTOFAIL
	task := crv1.Pgtask{}
	found, err = kubeapi.Getpgtask(restclient, &task, taskName, namespace)
	if !found {
		task.Namespace = namespace
		task.Name = taskName
		task.Spec.Status = eventType
		task.Spec.Name = clusterName
		task.ObjectMeta.Labels = make(map[string]string)
		task.ObjectMeta.Labels[config.LABEL_AUTOFAIL] = "true"
		task.Spec.TaskType = crv1.PgtaskAutoFailover
		task.Spec.Parameters = make(map[string]string)
		task.Spec.Parameters[time.Now().String()] = eventType
		err = kubeapi.Createpgtask(restclient, &task, namespace)
		return
	}

	task.Spec.Status = eventType
	task.Spec.Parameters[time.Now().String()] = eventType
	err = kubeapi.Updatepgtask(restclient, &task, taskName, namespace)
	if err != nil {
		log.Error(err)
	}

}

func (*AutoFailoverTask) Print(restclient *rest.RESTClient, namespace string) {

	log.Infoln("AutoFail pgtask List....")

	tasklist := crv1.PgtaskList{}

	err := kubeapi.GetpgtasksBySelector(restclient, &tasklist, config.LABEL_AUTOFAIL, namespace)
	if err != nil {
		log.Error(err)
		return

	}
	for k, v := range tasklist.Items {
		log.Infof("k=%d v=%v tasktype=%s", k, v.Name, v.Spec.TaskType)
		for x, y := range v.Spec.Parameters {
			log.Infof("parameter %s %s\n", x, y)
		}
	}

}

func (*AutoFailoverTask) Clear(restclient *rest.RESTClient, clusterName, namespace string) {

	taskName := clusterName + "-" + config.LABEL_AUTOFAIL
	err := kubeapi.Deletepgtask(restclient, taskName, namespace)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("autofail Clear autofail pgtask %s" + taskName)
}

func (*AutoFailoverTask) GetEvents(restclient *rest.RESTClient, clusterName, namespace string) (string, map[string]string) {
	task := crv1.Pgtask{}
	taskName := clusterName + "-" + config.LABEL_AUTOFAIL
	found, _ := kubeapi.Getpgtask(restclient, &task, taskName, namespace)
	if found {
		log.Debugf("autofail GetEvents status %s on task %s events %d", task.Spec.Status, taskName, len(task.Spec.Parameters))
		return task.Spec.Status, task.Spec.Parameters
	}
	return "", make(map[string]string)
}

func getTargetDeployment(restclient *rest.RESTClient, clientset *kubernetes.Clientset, clusterName, ns string) (string, error) {

	selector := config.LABEL_SERVICE_NAME + "=" + clusterName + "-replica" + "," + config.LABEL_PG_CLUSTER + "=" + clusterName

	deployments, err := kubeapi.GetDeployments(clientset, selector, ns)
	if kerrors.IsNotFound(err) {
		log.Debug("autofail no replicas found ")
		return "", err
	} else if err != nil {
		log.Error("error getting deployments " + err.Error())
		return "", err
	}

	//return a deployment target that has a Ready database
	log.Debugf("autofail deps len %d\n", len(deployments.Items))
	found := false
	readyDeps := make([]v1.Deployment, 0)
	for _, dep := range deployments.Items {
		ready := getPodStatus(clientset, dep.Name, ns)
		if ready {
			log.Debug("autofail: found ready deployment " + dep.Name)
			found = true
			readyDeps = append(readyDeps, dep)
		} else {
			log.Debug("autofail: found not ready deployment " + dep.Name)
		}
	}

	if !found {
		log.Error("could not find a ready pod in autofailover for cluster " + clusterName)
		return "", errors.New("could not find a ready pod to failover to")
	}

	//at this point readyDeps should hold all the Ready deployments
	//we look for the most up to date and return that name

	//next get the ready deployments replication status
	readyTargets := make([]util.ReplicationInfo, 0)
	for _, d := range readyDeps {
		target := util.ReplicationInfo{}
		target.ReceiveLocation, target.ReplayLocation, target.Node, err = util.GetRepStatus(restclient, clientset, &d, ns, operator.Pgo.Cluster.Port)
		if err != nil {
			return "", err
		}
		target.DeploymentName = d.Name
		readyTargets = append(readyTargets, target)
		log.Debugf("autofail receive=%d replay=%d dep=%s\n", target.ReceiveLocation, target.ReplayLocation, d.Name)
	}

	//next see which one is the most up to date, this is the case
	//when pgo.yaml PreferredFailoverNode is not specified or no nodes
	//are found that satisfy that selector
	var value uint64
	value = 0
	var selectedDeploymentName string
	for _, t := range readyTargets {
		if t.ReceiveLocation > value {
			value = t.ReceiveLocation
			selectedDeploymentName = t.DeploymentName
		}
	}

	//this is the case when PreferredFailoverNode is specified
	//we use that selector to match targets to preferred nodes if
	//a match is found and all other target selection criteria is met
	//that being (pod is ready) and (rep status is the greatest value)
	var nodes []string
	if operator.Pgo.Pgo.PreferredFailoverNode != "" {
		log.Debug("autofail PreferredFailoverNode is set to %s", operator.Pgo.Pgo.PreferredFailoverNode)
		nodes, err = util.GetPreferredNodes(clientset, operator.Pgo.Pgo.PreferredFailoverNode, ns)
		if err != nil {
			log.Errorf("autofail error: ", err.Error())
			return "", err
		}
		log.Debugf("autofail nodes len is %d", len(nodes))
		if len(nodes) == 0 {
			log.Debugf("autofail no nodes were found to match the PreferredFailoverNode selector so using default target selection")
		} else {
			//get a list of targets that equal the greatest value
			//this is the case when multiple replicas are at the
			//same replication status
			equalTargets := make([]util.ReplicationInfo, 0)
			for _, t := range readyTargets {
				if t.ReceiveLocation == value {
					equalTargets = append(equalTargets, t)
				}
			}
			for _, e := range equalTargets {
				for _, n := range nodes {
					if n == e.Node {
						selectedDeploymentName = e.DeploymentName
						log.Debugf("autofail %s deployment on node %s matched the preferred node label %s", e.DeploymentName, e.Node, operator.Pgo.Pgo.PreferredFailoverNode)
					}
				}

			}
		}
	}

	log.Debugf("autofail logic selected deployment is %s receive=%d\n", selectedDeploymentName, value)
	return selectedDeploymentName, err

}

func getPodStatus(clientset *kubernetes.Clientset, depname, ns string) bool {
	//TODO verify this selector is optimal
	//get pods with replica-name=deployName
	pods, err := kubeapi.GetPods(clientset, config.LABEL_REPLICA_NAME+"="+depname, ns)
	if err != nil {
		return false
	}

	p := pods.Items[0]
	for _, c := range p.Status.ContainerStatuses {
		if c.Name == "database" {
			if c.Ready {
				return true
			} else {
				return false
			}
		}
	}

	return false

}

func (s *StateMachine) triggerFailover() {
	log.Debugf("triggerFailover called ")
	targetDeploy, err := getTargetDeployment(s.RESTClient, s.Clientset, s.ClusterName, s.Namespace)
	if targetDeploy == "" || err != nil {
		log.Errorf("could not autofailover with no replicas found for %s\n", s.ClusterName)
		return
	}

	spec := crv1.PgtaskSpec{}
	spec.Namespace = s.Namespace
	spec.Name = s.ClusterName + "-" + config.LABEL_FAILOVER

	//see if task is already present (e.g. a prior failover)
	priorTask := crv1.Pgtask{}
	var found bool
	found, err = kubeapi.Getpgtask(s.RESTClient, &priorTask, spec.Name, s.Namespace)
	if found {
		log.Debugf("deleting pgtask %s", spec.Name)
		err = kubeapi.Deletepgtask(s.RESTClient, spec.Name, s.Namespace)
		if err != nil {
			log.Errorf("error removing pgtask %s failover is not able to proceed", spec.Name)
			return
		}
		//give time to delete the pgtask before recreating it
		time.Sleep(time.Second * time.Duration(1))
	} else if !found {
		log.Debugf("pgtask %s not found, no need to delete it", spec.Name)
	} else if err != nil {
		log.Error(err)
		log.Error("problem removing pgtask in failover logic, %s", spec.Name)
		return
	}

	spec.TaskType = crv1.PgtaskFailover
	spec.Parameters = make(map[string]string)
	spec.Parameters[s.ClusterName] = s.ClusterName
	labels := make(map[string]string)
	labels[config.LABEL_TARGET] = targetDeploy
	labels[config.LABEL_PG_CLUSTER] = s.ClusterName
	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	err = kubeapi.Createpgtask(s.RESTClient,
		newInstance, s.Namespace)
	if err != nil {
		log.Error(err)
		log.Error("could not create pgtask for autofailover failover task")
	} else {
		log.Infof("created pgtask failover by autofailover %s\n", s.ClusterName)
	}

}
