package pgreplica

/*
Copyright 2017 - 2023 Crunchy Data Solutions, Inc.
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
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds the connections for the controller
type Controller struct {
	Client               *kubeapi.Client
	Queue                workqueue.RateLimitingInterface
	Informer             informers.PgreplicaInformer
	PgreplicaWorkerCount int
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) RunWorker(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	go c.waitForShutdown(stopCh)

	for c.processNextItem() {
	}

	log.Debug("pgreplica Contoller: worker queue has been shutdown, writing to the done channel")
	doneCh <- struct{}{}
}

// waitForShutdown waits for a message on the stop channel and then shuts down the work queue
func (c *Controller) waitForShutdown(stopCh <-chan struct{}) {
	<-stopCh
	c.Queue.ShutDown()
	log.Debug("pgreplica Contoller: received stop signal, worker queue told to shutdown")
}

func (c *Controller) processNextItem() bool {
	ctx := context.TODO()

	// Wait until there is a new item in the working queue
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("pgreplica queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.Queue.Done(key)
	// Invoke the method containing the business logic
	// in this case, the de-dupe logic is to test whether a replica
	// deployment exists already , if so, then we don't create another
	// backup job
	_, err := c.Client.
		AppsV1().Deployments(keyNamespace).
		Get(ctx, keyResourceName, metav1.GetOptions{})

	depRunning := err == nil

	if depRunning {
		log.Debugf("working...found replica already, would do nothing")
	} else {
		log.Debugf("working...no replica found, means we process")

		// handle the case of when a pgreplica is added which is
		// scaling up a cluster
		replica, err := c.Client.CrunchydataV1().Pgreplicas(keyNamespace).Get(ctx, keyResourceName, metav1.GetOptions{})
		if err != nil {
			log.Error(err)
			c.Queue.Forget(key) // NB(cbandy): This should probably be a retry.
			return true
		}

		// get the pgcluster resource for the cluster the replica is a part of
		cluster, err := c.Client.CrunchydataV1().Pgclusters(keyNamespace).Get(ctx, replica.Spec.ClusterName, metav1.GetOptions{})
		if err != nil {
			log.Error(err)
			c.Queue.Forget(key) // NB(cbandy): This should probably be a retry.
			return true
		}

		// only process pgreplica if cluster has been initialized
		if cluster.Status.State == crv1.PgclusterStateInitialized {
			clusteroperator.ScaleBase(c.Client, replica, replica.ObjectMeta.Namespace)

			patch, err := json.Marshal(map[string]interface{}{
				"status": crv1.PgreplicaStatus{
					State:   crv1.PgreplicaStateProcessed,
					Message: "Successfully processed Pgreplica by controller",
				},
			})
			if err == nil {
				_, err = c.Client.CrunchydataV1().Pgreplicas(replica.Namespace).
					Patch(ctx, replica.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			}
			if err != nil {
				log.Errorf("ERROR updating pgreplica status: %s", err.Error())
			}
		} else {
			patch, err := json.Marshal(map[string]interface{}{
				"status": crv1.PgreplicaStatus{
					State:   crv1.PgreplicaStatePendingInit,
					Message: "Pgreplica processing pending the creation of the initial backup",
				},
			})
			if err == nil {
				_, err = c.Client.CrunchydataV1().Pgreplicas(replica.Namespace).
					Patch(ctx, replica.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			}
			if err != nil {
				log.Errorf("ERROR updating pgreplica status: %s", err.Error())
			}
		}
	}

	c.Queue.Forget(key)
	return true
}

// onAdd is called when a pgreplica is added
func (c *Controller) onAdd(obj interface{}) {
	replica := obj.(*crv1.Pgreplica)

	// handle the case of pgreplicas being processed already and
	// when the operator restarts
	if replica.Status.State == crv1.PgreplicaStateProcessed {
		log.Debug("pgreplica " + replica.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("onAdd putting key in queue %s", key)
		c.Queue.Add(key)
	}
}

// onUpdate is called when a pgreplica is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	ctx := context.TODO()

	oldPgreplica := oldObj.(*crv1.Pgreplica)
	newPgreplica := newObj.(*crv1.Pgreplica)

	log.Debugf("[pgreplica Controller] onUpdate ns=%s %s", newPgreplica.ObjectMeta.Namespace,
		newPgreplica.ObjectMeta.SelfLink)

	// get the pgcluster resource for the cluster the replica is a part of
	cluster, err := c.Client.
		CrunchydataV1().Pgclusters(newPgreplica.Namespace).
		Get(ctx, newPgreplica.Spec.ClusterName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return
	}

	// only process pgreplica if cluster has been initialized
	if cluster.Status.State == crv1.PgclusterStateInitialized && newPgreplica.Spec.Status != "complete" {
		clusteroperator.ScaleBase(c.Client, newPgreplica,
			newPgreplica.ObjectMeta.Namespace)

		patch, err := json.Marshal(map[string]interface{}{
			"status": crv1.PgreplicaStatus{
				State:   crv1.PgreplicaStateProcessed,
				Message: "Successfully processed Pgreplica by controller",
			},
		})
		if err == nil {
			_, err = c.Client.CrunchydataV1().Pgreplicas(newPgreplica.Namespace).
				Patch(ctx, newPgreplica.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		}
		if err != nil {
			log.Errorf("ERROR updating pgreplica status: %s", err.Error())
		}
	}

	// if the service type changed, updated on the instance
	// if there is an error, log but continue
	if oldPgreplica.Spec.ServiceType != newPgreplica.Spec.ServiceType {
		if err := clusteroperator.UpdateReplicaService(c.Client, cluster, newPgreplica); err != nil {
			log.Error(err)
		}
	}

	// if the tolerations array changed, updated the tolerations on the instance
	if !reflect.DeepEqual(oldPgreplica.Spec.Tolerations, newPgreplica.Spec.Tolerations) {
		// get the Deployment object associated with this instance
		deployment, err := c.Client.AppsV1().Deployments(newPgreplica.Namespace).Get(ctx,
			newPgreplica.Name, metav1.GetOptions{})

		if err != nil {
			log.Errorf("could not find instance for pgreplica: %q", err.Error())
			return
		}

		// determine the current Pod -- this is required to stop the instance
		pods, err := c.Client.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
			LabelSelector: fields.OneTermEqualSelector(config.LABEL_DEPLOYMENT_NAME, deployment.Name).String(),
		})

		// Even if there are errors with the Pods, we will continue on updating the
		// Deployment
		if err != nil {
			log.Warn(err)
		} else if len(pods.Items) == 0 {
			log.Infof("not shutting down PostgreSQL instance [%s] as the Pod cannot be found", deployment.Name)
		} else {
			// get the first pod off the items list
			pod := pods.Items[0]

			// we want to stop PostgreSQL on this instance to ensure all transactions
			// are safely flushed before we restart
			if err := util.StopPostgreSQLInstance(c.Client, c.Client.Config, &pod, deployment.Name); err != nil {
				log.Warn(err)
			}
		}

		// apply the tolerations and update the Deployment
		deployment.Spec.Template.Spec.Tolerations = newPgreplica.Spec.Tolerations

		if _, err := c.Client.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			log.Errorf("could not update deployment for pgreplica update: %q", err.Error())
		}
	}

	// handle PVC resizing, if needed
	if oldPgreplica.Spec.ReplicaStorage.Size != newPgreplica.Spec.ReplicaStorage.Size {
		// first check to see if the resize should occur
		annotations := newPgreplica.ObjectMeta.GetAnnotations()
		if annotations != nil {
			if _, ok := annotations[config.ANNOTATION_CLUSTER_DO_NOT_RESIZE]; ok {
				delete(newPgreplica.ObjectMeta.Annotations, config.ANNOTATION_CLUSTER_DO_NOT_RESIZE)
				if _, err := c.Client.CrunchydataV1().Pgreplicas(newPgreplica.Namespace).Update(ctx,
					newPgreplica, metav1.UpdateOptions{}); err != nil {
					log.Warnf("could not remove resize annotation from pgreplica: %s", err.Error())
				}
				return
			}
		}

		// if we get to this point, then we should resize the PVC
		// validate that this resize can occur
		if err := util.ValidatePVCResize(oldPgreplica.Spec.ReplicaStorage.Size, newPgreplica.Spec.ReplicaStorage.Size); err != nil {
			log.Error(err)
			return
		}

		// find the deployment
		deployment, err := c.Client.AppsV1().Deployments(newPgreplica.Namespace).Get(ctx,
			newPgreplica.Name, metav1.GetOptions{})
		if err != nil {
			log.Error(err)
			return
		}

		// get the size -- this will not error at this point as we just validated it
		size, _ := resource.ParseQuantity(newPgreplica.Spec.ReplicaStorage.Size)

		// OK, let's now perform the resize. In this case, we need to update the value
		// on the PVC.
		pvc, err := c.Client.CoreV1().PersistentVolumeClaims(newPgreplica.Namespace).Get(ctx,
			deployment.GetName(), metav1.GetOptions{})

		// if we can't locate the PVC, we can't resize, and we really need to return
		if err != nil {
			log.Error(err)
			return
		}

		// alright, update the PVC size
		pvc.Spec.Resources.Requests[v1.ResourceStorage] = size

		// and update!
		if _, err := c.Client.CoreV1().PersistentVolumeClaims(newPgreplica.Namespace).Update(ctx,
			pvc, metav1.UpdateOptions{}); err != nil {
			log.Error(err)
			return
		}

		// update the PostgreSQL instance
		// first, ensure it is stopped. if this errors, just warn
		if err := clusteroperator.StopPostgreSQLInstance(c.Client, c.Client.Config, *deployment); err != nil {
			log.Warn(err)
		}

		// scale down the deployment
		replicas := new(int32)
		if err := operator.ScaleDeployment(c.Client, deployment, replicas); err != nil {
			log.Error(err)
			return
		}

		// scale the deployment back up
		*replicas = 1
		if err := operator.ScaleDeployment(c.Client, deployment, replicas); err != nil {
			log.Error(err)
		}
	}
}

// onDelete is called when a pgreplica is deleted
func (c *Controller) onDelete(obj interface{}) {
	ctx := context.TODO()
	replica := obj.(*crv1.Pgreplica)
	log.Debugf("[pgreplica Controller] OnDelete ns=%s %s", replica.ObjectMeta.Namespace, replica.ObjectMeta.SelfLink)

	// make sure we are not removing a replica deployment
	// that is now the primary after a failover
	dep, err := c.Client.
		AppsV1().Deployments(replica.ObjectMeta.Namespace).
		Get(ctx, replica.Spec.Name, metav1.GetOptions{})
	if err == nil {
		if dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == dep.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] {
			// the replica was made a primary at some point
			// we will not scale down the deployment
			log.Debugf("[pgreplica Controller] OnDelete not scaling down the replica since it is acting as a primary")
		} else {
			clusteroperator.ScaleDownBase(c.Client, replica, replica.ObjectMeta.Namespace)
		}
	}
}

// AddPGReplicaEventHandler adds the pgreplica event handler to the pgreplica informer
func (c *Controller) AddPGReplicaEventHandler() {
	// Your custom resource event handlers.
	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("pgreplica Controller: added event handler to informer")
}

// WorkerCount returns the worker count for the controller
func (c *Controller) WorkerCount() int {
	return c.PgreplicaWorkerCount
}
