package pgcluster

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"

	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds the connections for the controller
type Controller struct {
	PgclusterClient    *rest.RESTClient
	PgclusterClientset *kubernetes.Clientset
	Queue              workqueue.RateLimitingInterface
	Informer           informers.PgclusterInformer
}

// onAdd is called when a pgcluster is added
func (c *Controller) onAdd(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	log.Debugf("[pgcluster Controller] ns %s onAdd %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle the case when the operator restarts and don't
	//process already processed pgclusters
	if cluster.Status.State == crv1.PgclusterStateProcessed {
		log.Debug("pgcluster " + cluster.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("cluster putting key in queue %s", key)
		c.Queue.Add(key)
	}

}

func (c *Controller) RunWorker() {

	//process the 'add' work queue forever
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("cluster add queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.Queue.Done(key)

	// Invoke the method containing the business logic
	// in this case, the de-dupe logic is to test whether a cluster
	// deployment exists , if so, then we don't create another
	_, found, err := kubeapi.GetDeployment(c.PgclusterClientset, keyResourceName, keyNamespace)

	if found {
		log.Debugf("cluster add - dep already found, not creating again")
		return true
	}

	//get the pgcluster
	cluster := crv1.Pgcluster{}
	found, err = kubeapi.Getpgcluster(c.PgclusterClient, &cluster, keyResourceName, keyNamespace)
	if !found {
		log.Debugf("cluster add - pgcluster not found, this is invalid")
		return false
	}

	addIdentifier(&cluster)

	state := crv1.PgclusterStateProcessed
	message := "Successfully processed Pgcluster by controller"
	err = kubeapi.PatchpgclusterStatus(c.PgclusterClient, state, message, &cluster, keyNamespace)
	if err != nil {
		log.Errorf("ERROR updating pgcluster status on add: %s", err.Error())
		return false
	}

	log.Debugf("pgcluster added: %s", cluster.ObjectMeta.Name)

	clusteroperator.AddClusterBase(c.PgclusterClientset, c.PgclusterClient, &cluster, cluster.ObjectMeta.Namespace)

	return true
}

// onUpdate is called when a pgcluster is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	oldcluster := oldObj.(*crv1.Pgcluster)
	newcluster := newObj.(*crv1.Pgcluster)
	//	log.Debugf("pgcluster ns=%s %s onUpdate", newcluster.ObjectMeta.Namespace, newcluster.ObjectMeta.Name)

	// if the 'shutdown' parameter in the pgcluster update shows that the cluster should be either
	// shutdown or started but its current status does not properly reflect that it is, then
	// proceed with the logic needed to either shutdown or start the cluster
	if newcluster.Spec.Shutdown && newcluster.Status.State != crv1.PgclusterStateShutdown {
		clusteroperator.ShutdownCluster(c.PgclusterClientset, c.PgclusterClient, *newcluster)
	} else if !newcluster.Spec.Shutdown &&
		newcluster.Status.State != crv1.PgclusterStateInitialized {
		clusteroperator.StartupCluster(c.PgclusterClientset, *newcluster)
	}

	// check to see if the "autofail" label on the pgcluster CR has been changed from either true to false, or from
	// false to true.  If it has been changed to false, autofail will then be disabled in the pg cluster.  If has
	// been changed to true, autofail will then be enabled in the pg cluster
	if newcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] != "" {
		autofailEnabledOld, err := strconv.ParseBool(oldcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
		if err != nil {
			log.Error(err)
			return
		}
		autofailEnabledNew, err := strconv.ParseBool(newcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
		if err != nil {
			log.Error(err)
			return
		}
		if autofailEnabledNew != autofailEnabledOld {
			util.ToggleAutoFailover(c.PgclusterClientset, autofailEnabledNew,
				newcluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE],
				newcluster.ObjectMeta.Namespace)
		}

	}

	// handle standby being enabled and disabled for the cluster
	if oldcluster.Spec.Standby && !newcluster.Spec.Standby {
		if err := clusteroperator.DisableStandby(c.PgclusterClientset, *newcluster); err != nil {
			log.Error(err)
			return
		}
	} else if !oldcluster.Spec.Standby && newcluster.Spec.Standby {
		if err := clusteroperator.EnableStandby(c.PgclusterClientset, *newcluster); err != nil {
			log.Error(err)
			return
		}
	}

	// if we are not in a standby state, check to see if the tablespaces have
	// differed, and if so, add the additional volumes to the primary and replicas
	if !reflect.DeepEqual(oldcluster.Spec.TablespaceMounts, newcluster.Spec.TablespaceMounts) {
		if err := updateTablespaces(c, oldcluster, newcluster); err != nil {
			log.Error(err)
			return
		}
	}
}

// onDelete is called when a pgcluster is deleted
func (c *Controller) onDelete(obj interface{}) {
	//cluster := obj.(*crv1.Pgcluster)
	//	log.Debugf("[Controller] ns=%s onDelete %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle pgcluster cleanup
	//	clusteroperator.DeleteClusterBase(c.PgclusterClientset, c.PgclusterClient, cluster, cluster.ObjectMeta.Namespace)
}

func GetPrimaryPodStatus(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, ns string) (error, bool) {
	var ready bool
	var err error

	selector := config.LABEL_SERVICE_NAME + "=" + cluster.Name
	pods, err := kubeapi.GetPods(clientset, selector, ns)
	if err != nil {
		return err, ready
	}
	if len(pods.Items) == 0 {
		log.Error("GetPrimaryPodStatus found no primary pod for %s using %s", cluster.Name, selector)
		return err, ready
	}
	if len(pods.Items) > 1 {
		log.Error("GetPrimaryPodStatus found more than 1 primary pod for %s using %s", cluster.Name, selector)
		return err, ready
	}

	pod := pods.Items[0]
	var readyStatus string
	readyStatus, ready = getReadyStatus(&pod)
	log.Debugf("readyStatus found to be %s", readyStatus)
	return err, ready

}

//this code is taken from apiserver/cluster/clusterimpl.go, need
//to refactor into a higher level package to share the code
func getReadyStatus(pod *v1.Pod) (string, bool) {
	equal := false
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	if readyCount == containerCount {
		equal = true
	}
	return fmt.Sprintf("%d/%d", readyCount, containerCount), equal

}

// AddPGClusterEventHandler adds the pgcluster event handler to the pgcluster informer
func (c *Controller) AddPGClusterEventHandler() {

	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("pgcluster Controller: added event handler to informer")
}

func addIdentifier(clusterCopy *crv1.Pgcluster) {
	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
	}

	clusterCopy.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = string(u[:len(u)-1])
}

// updateTablespaces updates the PostgreSQL instance Deployments to reflect the
// new PostgreSQL tablespaces that should be added
func updateTablespaces(c *Controller, oldCluster *crv1.Pgcluster, newCluster *crv1.Pgcluster) error {
	// first, get a list of all of the available deployments so we can properly
	// mount the tablespace PVCs after we create them
	// NOTE: this will also get the pgBackRest deployments, but we will filter
	// these out later
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_VENDOR, config.LABEL_CRUNCHY,
		config.LABEL_PG_CLUSTER, newCluster.Name)

	deployments, err := kubeapi.GetDeployments(c.PgclusterClientset, selector, newCluster.Namespace)

	if err != nil {
		return err
	}

	// now get the instance names, which will make it easier to create all the
	// PVCs
	instanceNames := []string{}

	for _, deployment := range deployments.Items {
		labels := deployment.ObjectMeta.GetLabels()

		// get the name of the PostgreSQL instance. If the "deployment-name"
		// label is not present, then we know it's not a PostgreSQL cluster.
		// Otherwise, the "deployment-name" label doubles as the name of the
		// instance
		if instanceName, ok := labels[config.LABEL_DEPLOYMENT_NAME]; ok {
			log.Debugf("instance found [%s]", instanceName)

			instanceNames = append(instanceNames, instanceName)
		}
	}

	// iterate through the the tablespace mount map that is present in the new
	// cluster. Any entry that is not in the old cluster, create PVCs
	newTablespaces := map[string]crv1.PgStorageSpec{}

	for tablespaceName, storageSpec := range newCluster.Spec.TablespaceMounts {
		// if the tablespace does not exist in the old version of the cluster,
		// then add it in!
		if _, ok := oldCluster.Spec.TablespaceMounts[tablespaceName]; !ok {
			log.Debugf("new tablespace found: [%s]", tablespaceName)

			newTablespaces[tablespaceName] = storageSpec
		}
	}

	// now we can start creating the new tablespaces! First, create the new
	// PVCs. The PVCs are created for each **instance** in the cluster, as every
	// instance needs to have a distinct PVC for each tablespace
	for tablespaceName, storageSpec := range newTablespaces {
		for _, instanceName := range instanceNames {
			// get the name of the tablespace PVC for that instance
			tablespacePVCName := operator.GetTablespacePVCName(instanceName, tablespaceName)

			log.Debugf("creating tablespace PVC [%s] for [%s]", tablespacePVCName, instanceName)

			// and now create it! If it errors, we just need to return, which
			// potentially leaves things in an inconsistent state, but at this point
			// only PVC objects have been created
			if err := clusteroperator.CreateTablespacePVC(c.PgclusterClientset, newCluster.Namespace, newCluster.Name,
				tablespacePVCName, &storageSpec); err != nil {
				return err
			}
		}
	}

	// now the fun step: update each deployment with the new volumes
	for _, deployment := range deployments.Items {
		labels := deployment.ObjectMeta.GetLabels()

		// same deal as before: if this is not a PostgreSQL instance, skip it
		instanceName, ok := labels[config.LABEL_DEPLOYMENT_NAME]
		if !ok {
			continue
		}

		log.Debugf("attach tablespace volumes to [%s]", instanceName)

		// iterate through each table space and prepare the Volume and
		// VolumeMount clause for each instance
		for tablespaceName, _ := range newTablespaces {
			// this is the volume to be added for the tablespace
			volume := v1.Volume{
				Name: operator.GetTablespaceVolumeName(tablespaceName),
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: operator.GetTablespacePVCName(instanceName, tablespaceName),
					},
				},
			}

			// add the volume to the list of volumes
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

			// now add the volume mount point to that of the database container
			volumeMount := v1.VolumeMount{
				MountPath: fmt.Sprintf("%s%s", config.VOLUME_TABLESPACE_PATH_PREFIX, tablespaceName),
				Name:      operator.GetTablespaceVolumeName(tablespaceName),
			}

			// we can do this as we always know that the "database" contianer is the
			// first container in the list
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
		}

		// find the "PGHA_TABLESPACES" value and update it with the new tablespace
		// name list
		for i, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
			// yup, it's an old fashioned linear time lookup
			if envVar.Name == "PGHA_TABLESPACES" {
				deployment.Spec.Template.Spec.Containers[0].Env[i].Value = operator.GetTablespaceNames(
					newCluster.Spec.TablespaceMounts)
			}
		}

		// finally, update the Deployment. Potential to put things into an
		// inconsistent state if any of these updates fail
		if err := kubeapi.UpdateDeployment(c.PgclusterClientset, &deployment); err != nil {
			return err
		}
	}

	return nil
}
