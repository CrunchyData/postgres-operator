package manager

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"fmt"
	"sync"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller"
	"github.com/crunchydata/postgres-operator/internal/controller/configmap"
	"github.com/crunchydata/postgres-operator/internal/controller/job"
	"github.com/crunchydata/postgres-operator/internal/controller/pgcluster"
	"github.com/crunchydata/postgres-operator/internal/controller/pgpolicy"
	"github.com/crunchydata/postgres-operator/internal/controller/pgreplica"
	"github.com/crunchydata/postgres-operator/internal/controller/pgtask"
	"github.com/crunchydata/postgres-operator/internal/controller/pod"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/ns"
	"github.com/crunchydata/postgres-operator/internal/operator/operatorupgrade"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// the following variables represent the resources the operator must has "list" access to in order
// to start an informer
var (
	listerResourcesCrunchy = []string{"pgtasks", "pgclusters", "pgreplicas", "pgpolicies"}
	listerResourcesCore    = []string{"pods", "configmaps"}
)

// ControllerManager manages a map of controller groups, each of which is comprised of the various
// controllers needed to handle events within a specific namespace.  Only one controllerGroup is
// allowed per namespace.
type ControllerManager struct {
	mgrMutex               sync.Mutex
	controllers            map[string]*controllerGroup
	installationName       string
	namespaceOperatingMode ns.NamespaceOperatingMode
	pgoConfig              config.PgoConfig
	pgoNamespace           string
	sem                    *semaphore.Weighted

	NewKubernetesClient func() (*kubeapi.Client, error)
}

// controllerGroup is a struct for managing the various controllers created to handle events
// in a specific namespace
type controllerGroup struct {
	stopCh                         chan struct{}
	doneCh                         chan struct{}
	started                        bool
	pgoInformerFactory             informers.SharedInformerFactory
	kubeInformerFactory            kubeinformers.SharedInformerFactory
	kubeInformerFactoryWithRefresh kubeinformers.SharedInformerFactory
	controllersWithWorkers         []controller.WorkerRunner
	informerSyncedFuncs            []cache.InformerSynced
	clientset                      kubeapi.Interface
}

// NewControllerManager returns a new ControllerManager comprised of controllerGroups for each
// namespace included in the 'namespaces' parameter.
func NewControllerManager(namespaces []string,
	pgoConfig config.PgoConfig, pgoNamespace, installationName string,
	namespaceOperatingMode ns.NamespaceOperatingMode) (*ControllerManager, error) {
	controllerManager := ControllerManager{
		controllers:            make(map[string]*controllerGroup),
		installationName:       installationName,
		namespaceOperatingMode: namespaceOperatingMode,
		pgoConfig:              pgoConfig,
		pgoNamespace:           pgoNamespace,
		sem:                    semaphore.NewWeighted(1),

		NewKubernetesClient: kubeapi.NewClient,
	}

	// create controller groups for each namespace provided
	for _, ns := range namespaces {
		if err := controllerManager.AddGroup(ns); err != nil {
			log.Error(err)
			return nil, err
		}
	}

	log.Debugf("Controller Manager: new controller manager created for namespaces %v",
		namespaces)

	return &controllerManager, nil
}

// AddGroup adds a new controller group for the namespace specified.  Each controller
// group is comprised of controllers for the following resources:
// - pods
// - jobs
// - pgclusters
// - pgpolicys
// - pgtasks
// Two SharedInformerFactory's are utilized (one for Kube resources and one for PosgreSQL Operator
// resources) to create and track the informers for each type of resource, while any controllers
// utilizing worker queues are also tracked (this allows all informers and worker queues to be
// easily started as needed). Each controller group also receives its own clients, which can then
// be utilized by the various controllers within that controller group.
func (c *ControllerManager) AddGroup(namespace string) error {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	// only return an error if not a group already exists error
	if err := c.addControllerGroup(namespace); err != nil &&
		!errors.Is(err, controller.ErrControllerGroupExists) {
		return err
	}

	return nil
}

// AddAndRunGroup is a convenience function that adds a controller group for the
// namespace specified, and then immediately runs the controllers in that group.
func (c *ControllerManager) AddAndRunGroup(namespace string) error {
	if c.controllers[namespace] != nil && !c.pgoConfig.Pgo.DisableReconcileRBAC {
		// first reconcile RBAC in the target namespace if RBAC reconciliation is enabled
		c.reconcileRBAC(namespace)
	}

	// now add and run the controller group
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	// only return an error if not a group already exists error
	if err := c.addControllerGroup(namespace); err != nil &&
		!errors.Is(err, controller.ErrControllerGroupExists) {
		return err
	}

	if err := c.runControllerGroup(namespace); err != nil {
		return err
	}

	return nil
}

// RemoveAll removes all controller groups managed by the controller manager, first stopping all
// controllers within each controller group managed by the controller manager.
func (c *ControllerManager) RemoveAll() {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	for ns := range c.controllers {
		c.removeControllerGroup(ns)
	}

	log.Debug("Controller Manager: all contollers groups have been removed")
}

// RemoveGroup removes the controller group for the namespace specified, first stopping all
// controllers within that group
func (c *ControllerManager) RemoveGroup(namespace string) {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	c.removeControllerGroup(namespace)
}

// RunAll runs all controllers across all controller groups managed by the controller manager.
func (c *ControllerManager) RunAll() error {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	for ns := range c.controllers {
		if err := c.runControllerGroup(ns); err != nil {
			return err
		}
	}

	log.Debug("Controller Manager: all contoller groups are now running")

	return nil
}

// RunGroup runs the controllers within the controller group for the namespace specified.
func (c *ControllerManager) RunGroup(namespace string) error {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: unable to run controller group for namespace %s because "+
			"a controller group for this namespace does not exist", namespace)
		return nil
	}

	if err := c.runControllerGroup(namespace); err != nil {
		return err
	}

	log.Debugf("Controller Manager: the controller group for ns %s is now running", namespace)

	return nil
}

// addControllerGroup adds a new controller group for the namespace specified
func (c *ControllerManager) addControllerGroup(namespace string) error {
	if _, ok := c.controllers[namespace]; ok {
		log.Debugf("Controller Manager: a controller for namespace %s already exists", namespace)
		return controller.ErrControllerGroupExists
	}

	// create a client for kube resources
	client, err := c.NewKubernetesClient()
	if err != nil {
		log.Error(err)
		return err
	}

	pgoInformerFactory := informers.NewSharedInformerFactoryWithOptions(client, 0,
		informers.WithNamespace(namespace))

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(client, 0,
		kubeinformers.WithNamespace(namespace))

	kubeInformerFactoryWithRefresh := kubeinformers.NewSharedInformerFactoryWithOptions(client,
		time.Duration(*c.pgoConfig.Pgo.ControllerGroupRefreshInterval)*time.Second,
		kubeinformers.WithNamespace(namespace))

	pgTaskcontroller := &pgtask.Controller{
		Client:            client,
		Queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Informer:          pgoInformerFactory.Crunchydata().V1().Pgtasks(),
		PgtaskWorkerCount: *c.pgoConfig.Pgo.PGTaskWorkerCount,
	}

	pgClustercontroller := &pgcluster.Controller{
		Client:               client,
		Queue:                workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Informer:             pgoInformerFactory.Crunchydata().V1().Pgclusters(),
		PgclusterWorkerCount: *c.pgoConfig.Pgo.PGClusterWorkerCount,
	}

	pgReplicacontroller := &pgreplica.Controller{
		Client:               client,
		Queue:                workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Informer:             pgoInformerFactory.Crunchydata().V1().Pgreplicas(),
		PgreplicaWorkerCount: *c.pgoConfig.Pgo.PGReplicaWorkerCount,
	}

	pgPolicycontroller := &pgpolicy.Controller{
		Clientset: client,
		Informer:  pgoInformerFactory.Crunchydata().V1().Pgpolicies(),
	}

	podcontroller := &pod.Controller{
		Client:   client,
		Informer: kubeInformerFactory.Core().V1().Pods(),
	}

	jobcontroller := &job.Controller{
		Client:   client,
		Informer: kubeInformerFactory.Batch().V1().Jobs(),
	}

	configMapController, err := configmap.NewConfigMapController(client.Config,
		client, kubeInformerFactoryWithRefresh.Core().V1().ConfigMaps(),
		pgoInformerFactory.Crunchydata().V1().Pgclusters(),
		*c.pgoConfig.Pgo.ConfigMapWorkerCount)
	if err != nil {
		log.Errorf("Unable to create ConfigMap controller: %v", err)
		return err
	}

	// add the proper event handler to the informer in each controller
	pgTaskcontroller.AddPGTaskEventHandler()
	pgClustercontroller.AddPGClusterEventHandler()
	pgReplicacontroller.AddPGReplicaEventHandler()
	pgPolicycontroller.AddPGPolicyEventHandler()
	podcontroller.AddPodEventHandler()
	jobcontroller.AddJobEventHandler()

	group := &controllerGroup{
		clientset:                      client,
		stopCh:                         make(chan struct{}),
		doneCh:                         make(chan struct{}),
		pgoInformerFactory:             pgoInformerFactory,
		kubeInformerFactory:            kubeInformerFactory,
		kubeInformerFactoryWithRefresh: kubeInformerFactoryWithRefresh,
		informerSyncedFuncs: []cache.InformerSynced{
			pgoInformerFactory.Crunchydata().V1().Pgtasks().Informer().HasSynced,
			pgoInformerFactory.Crunchydata().V1().Pgclusters().Informer().HasSynced,
			pgoInformerFactory.Crunchydata().V1().Pgreplicas().Informer().HasSynced,
			pgoInformerFactory.Crunchydata().V1().Pgpolicies().Informer().HasSynced,
			kubeInformerFactory.Core().V1().Pods().Informer().HasSynced,
			kubeInformerFactory.Batch().V1().Jobs().Informer().HasSynced,
			kubeInformerFactoryWithRefresh.Core().V1().ConfigMaps().Informer().HasSynced,
		},
	}

	// store the controllers containing worker queues so that the queues can also be started
	// when any informers in the controller are started
	group.controllersWithWorkers = append(group.controllersWithWorkers,
		pgTaskcontroller, pgClustercontroller, pgReplicacontroller, configMapController)

	c.controllers[namespace] = group

	log.Debugf("Controller Manager: added controller group for namespace %s", namespace)

	// now reconcile RBAC in the namespace if RBAC reconciliation is enabled
	if !c.pgoConfig.Pgo.DisableReconcileRBAC {
		c.reconcileRBAC(namespace)
	}

	return nil
}

// hasListerPrivs verifies the Operator has the privileges required to start the controllers
// for the namespace specified.
func (c *ControllerManager) hasListerPrivs(namespace string) bool {
	controllerGroup := c.controllers[namespace]

	var err error
	var hasCrunchyPrivs, hasCorePrivs, hasBatchPrivs bool

	for _, listerResource := range listerResourcesCrunchy {
		hasCrunchyPrivs, err = ns.CheckAccessPrivs(controllerGroup.clientset,
			map[string][]string{listerResource: {"list"}},
			crv1.GroupName, namespace)
		if err != nil {
			log.Errorf(err.Error())
		} else if !hasCrunchyPrivs {
			log.Errorf("Controller Manager: Controller Group for namespace %s does not have the "+
				"required list privileges for resource %s in the %s API",
				namespace, listerResource, crv1.GroupName)
		}
	}

	for _, listerResource := range listerResourcesCore {
		hasCorePrivs, err = ns.CheckAccessPrivs(controllerGroup.clientset,
			map[string][]string{listerResource: {"list"}},
			"", namespace)
		if err != nil {
			log.Errorf(err.Error())
		} else if !hasCorePrivs {
			log.Errorf("Controller Manager: Controller Group for namespace %s does not have the "+
				"required list privileges for resource %s in the Core API",
				namespace, listerResource)
		}
	}

	hasBatchPrivs, err = ns.CheckAccessPrivs(controllerGroup.clientset,
		map[string][]string{"jobs": {"list"}},
		"batch", namespace)
	if err != nil {
		log.Errorf(err.Error())
	} else if !hasBatchPrivs {
		log.Errorf("Controller Manager: Controller Group for namespace %s does not have the "+
			"required list privileges for resource %s in the Batch API",
			namespace, "jobs")
	}

	return (hasCrunchyPrivs && hasCorePrivs && hasBatchPrivs)
}

// runControllerGroup is responsible running the controllers for the controller group corresponding
// to the namespace provided
func (c *ControllerManager) runControllerGroup(namespace string) error {
	controllerGroup := c.controllers[namespace]
	hasListerPrivs := c.hasListerPrivs(namespace)

	switch {
	case controllerGroup.started && hasListerPrivs:
		log.Debugf("Controller Manager: controller group for namespace %s is already running",
			namespace)
		return nil
	case controllerGroup.started && !hasListerPrivs:
		c.removeControllerGroup(namespace)
		return fmt.Errorf("Controller Manager: removing the running controller group for "+
			"namespace %s because it no longer has the required privs, will attempt to "+
			"restart on the next ns refresh interval", namespace)
	case !hasListerPrivs:
		return fmt.Errorf("Controller Manager: cannot start controller group for namespace %s "+
			"because it does not have the required privs, will attempt to start on the next ns "+
			"refresh interval", namespace)
	}

	// before starting, first successfully check the versions of all pgcluster's in the namespace
	if err := operatorupgrade.CheckVersion(controllerGroup.clientset, namespace); err != nil {
		log.Errorf("Controller Manager: Unsuccessful pgcluster version check for namespace %s, "+
			"the controller group will not be started", namespace)
		return err
	}

	controllerGroup.kubeInformerFactory.Start(controllerGroup.stopCh)
	controllerGroup.pgoInformerFactory.Start(controllerGroup.stopCh)
	controllerGroup.kubeInformerFactoryWithRefresh.Start(controllerGroup.stopCh)

	if ok := cache.WaitForNamedCacheSync(namespace, controllerGroup.stopCh,
		controllerGroup.informerSyncedFuncs...); !ok {
		return fmt.Errorf("Controller Manager: failed waiting for caches to sync")
	}

	for _, worker := range controllerGroup.controllersWithWorkers {
		for i := 0; i < worker.WorkerCount(); i++ {
			go worker.RunWorker(controllerGroup.stopCh, controllerGroup.doneCh)
		}
	}

	controllerGroup.started = true

	log.Debugf("Controller Manager: controller group for namespace %s is now running", namespace)

	return nil
}

// removeControllerGroup removes the controller group for the namespace specified.  Any worker
// queues associated with the controllers inside of the controller group are first shutdown
// prior to removing the controller group.
func (c *ControllerManager) removeControllerGroup(namespace string) {
	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: no controller group to remove for ns %s", namespace)
		return
	}

	c.stopControllerGroup(namespace)
	delete(c.controllers, namespace)

	log.Debugf("Controller Manager: the controller group for ns %s has been removed", namespace)
}

// stopControllerGroup stops the controller group associated with the namespace specified.  This is
// done by calling the ShutdownWorker function associated with the controller.  If the controller
// does not have a ShutdownWorker function then no action is taken.
func (c *ControllerManager) stopControllerGroup(namespace string) {
	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: unable to stop controller group for namespace %s because "+
			"a controller group for this namespace does not exist", namespace)
		return
	}

	if !c.controllers[namespace].started {
		log.Debugf("Controller Manager: controller group for namespace %s was never started, "+
			"skipping worker shutdown", namespace)
		return
	}

	controllerGroup := c.controllers[namespace]

	// close the stop channel to stop all informers and instruct the workers queues to shutdown
	close(controllerGroup.stopCh)

	// wait for all worker queues to shutdown
	log.Debugf("Waiting for %d workers in the controller group for namespace %s to shutdown",
		len(controllerGroup.controllersWithWorkers), namespace)
	var numWorkers int
	for _, worker := range controllerGroup.controllersWithWorkers {
		for i := 0; i < worker.WorkerCount(); i++ {
			numWorkers++
		}
	}
	for i := 0; i < numWorkers; i++ {
		<-controllerGroup.doneCh
	}
	close(controllerGroup.doneCh)

	controllerGroup.started = false

	log.Debugf("Controller Manager: the controller group for ns %s has been stopped", namespace)
}
