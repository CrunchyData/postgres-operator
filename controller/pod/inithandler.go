package pod

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
	"strconv"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	taskoperator "github.com/crunchydata/postgres-operator/operator/task"
	"github.com/crunchydata/postgres-operator/util"
	apiv1 "k8s.io/api/core/v1"

	log "github.com/sirupsen/logrus"
)

// handleClusterInit is responsible for proceeding with initialization of the PG cluster once the
// primary PG pod for a new or restored PG cluster reaches a ready status
func (c *Controller) handleClusterInit(newPod *apiv1.Pod, cluster *crv1.Pgcluster) error {

	clusterName := cluster.Name

	// handle common tasks for initializing a cluster, whether due to bootstap or reinitialization
	// following a restore, or if a regular or standby cluster
	if err := c.handleCommonInit(cluster); err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("Pod Controller: completed common init for pod %s in cluster %s", newPod.Name,
		clusterName)

	// call the appropriate initialization logic depending on the current state of the PG cluster,
	// e.g. whether or not is is initializing for the first time or reinitializing as the result of
	// a restore, and/or depending on certain properties for the cluster, e.g. whether or not it is
	// a standby clusteer
	switch {
	case cluster.Status.State == crv1.PgclusterStateRestore:
		log.Debugf("Pod Controller: restore detected during cluster %s init, calling restore "+
			"handler", clusterName)
		return c.handleRestoreInit(cluster)
	case cluster.Spec.Standby:
		log.Debugf("Pod Controller: standby cluster detected during cluster %s init, calling "+
			"standby handler", clusterName)
		return c.handleStandbyInit(cluster)
	default:
		log.Debugf("Pod Controller: calling bootstrap init for cluster %s", clusterName)
		return c.handleBootstrapInit(newPod, cluster)
	}
}

// handleCommonInit is resposible for handling common initilization tasks for a PG cluster
// regardless of the specific type of cluster (e.g. regualar or standby) or the reason the
// cluster is being initialized (initial bootstrap or restore)
func (c *Controller) handleCommonInit(cluster *crv1.Pgcluster) error {

	// Disable autofailover in the cluster that is now "Ready" if the autofail label is set
	// to "false" on the pgcluster (i.e. label "autofail=true")
	autofailEnabled, err := strconv.ParseBool(cluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
	if err != nil {
		log.Error(err)
		return err
	} else if !autofailEnabled {
		util.ToggleAutoFailover(c.PodClientset, false,
			cluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE], cluster.Namespace)
	}

	operator.UpdatePGHAConfigInitFlag(c.PodClientset, false, cluster.Name,
		cluster.Namespace)

	return nil
}

// handleRestoreInit is resposible for handling cluster initilization for a restored PG cluster
func (c *Controller) handleRestoreInit(cluster *crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	//look up the backrest-repo pod name
	selector := fmt.Sprintf("%s=%s,pgo-backrest-repo=true",
		config.LABEL_PG_CLUSTER, clusterName)
	pods, err := kubeapi.GetPods(c.PodClientset, selector,
		namespace)
	if len(pods.Items) != 1 {
		return fmt.Errorf("pods len != 1 for cluster %s", clusterName)
	}
	if err != nil {
		log.Error(err)
		return err
	}
	err = backrest.CleanBackupResources(c.PodClient, c.PodClientset,
		namespace, clusterName)
	if err != nil {
		log.Error(err)
		return err
	}

	backrestoperator.CreateInitialBackup(c.PodClient, namespace,
		clusterName, pods.Items[0].Name)

	return nil
}

// handleBootstrapInit is resposible for handling cluster initilization (e.g. initiating pgBackRest
// stanza creation) when a the database container within the primary PG Pod for a new PG cluster
// enters a ready status
func (c *Controller) handleBootstrapInit(newPod *apiv1.Pod, cluster *crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("%s went to Ready from Not Ready, apply policies...", clusterName)
	taskoperator.ApplyPolicies(clusterName, c.PodClientset, c.PodClient, c.PodConfig, namespace)

	taskoperator.CompleteCreateClusterWorkflow(clusterName, c.PodClientset, c.PodClient, namespace)

	//publish event for cluster complete
	publishClusterComplete(clusterName, namespace, cluster)
	//

	// create the pgBackRest stanza
	backrestoperator.StanzaCreate(newPod.ObjectMeta.Namespace, clusterName,
		c.PodClientset, c.PodClient)

	// if this is a pgbouncer enabled cluster, add a pgbouncer
	// Note: we only warn if we cannot create the pgBouncer, so eecution can
	// continue
	if cluster.Spec.PgBouncer.Enabled {
		if err := clusteroperator.AddPgbouncer(c.PodClientset, c.PodClient, c.PodConfig, cluster); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// handleStandbyInit is resposible for handling standby cluster initilization when the database
// container within the primary PG Pod for a new standby cluster enters a ready status
func (c *Controller) handleStandbyInit(cluster *crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	taskoperator.CompleteCreateClusterWorkflow(clusterName, c.PodClientset, c.PodClient, namespace)

	//publish event for cluster complete
	publishClusterComplete(clusterName, namespace, cluster)
	//

	// now scale any replicas deployments to 1
	clusteroperator.ScaleClusterDeployments(c.PodClientset, *cluster, 1, false, true, false)

	// Proceed with stanza-creation of this is not a standby cluster, or if its
	// a standby cluster that does not have "s3" storage only enabled.
	// If this is a standby cluster and the pgBackRest storage type is set
	// to "s3" for S3 storage only, set the cluster to an initialized status.
	if cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE] != "s3" {
		backrestoperator.StanzaCreate(namespace, clusterName,
			c.PodClientset, c.PodClient)
	} else {
		controller.SetClusterInitializedStatus(c.PodClient, clusterName,
			namespace)
	}

	// If a standby cluster initialize the creation of any replicas.  Replicas
	// can be initialized right away, i.e. there is no dependency on
	// stanza-creation and/or the creation of any backups, since the replicas
	// will be generated from the pgBackRest repository of an external PostgreSQL
	// database (which should already exist).
	controller.InitializeReplicaCreation(c.PodClient, clusterName, namespace)

	// if this is a pgbouncer enabled cluster, add a pgbouncer
	// Note: we only warn if we cannot create the pgBouncer, so eecution can
	// continue
	if cluster.Spec.PgBouncer.Enabled {
		if err := clusteroperator.AddPgbouncer(c.PodClientset, c.PodClient, c.PodConfig, cluster); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// labelPostgresPodAndDeployment
// see if this is a primary or replica being created
// update service-name label on the pod for each case
// to match the correct Service selector for the PG cluster
func (c *Controller) labelPostgresPodAndDeployment(newpod *apiv1.Pod) {

	depName := newpod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]
	ns := newpod.Namespace

	pgreplica := crv1.Pgreplica{}
	replica, _ := kubeapi.Getpgreplica(c.PodClient, &pgreplica, depName, ns)
	log.Debugf("checkPostgresPods --- dep %s replica %t", depName, replica)

	dep, _, err := kubeapi.GetDeployment(c.PodClientset, depName, ns)
	if err != nil {
		log.Errorf("could not get Deployment on pod Add %s", newpod.Name)
		return
	}

	serviceName := ""

	if dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] != "" {
		log.Debug("this means the deployment was already labeled")
		log.Debug("which means its pod was restarted for some reason")
		log.Debug("we will use the service name on the deployment")
		serviceName = dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME]
	} else if replica == false {
		log.Debugf("primary pod ADDED %s service-name=%s", newpod.Name, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER])
		//add label onto pod "service-name=clustername"
		serviceName = newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	} else if replica == true {
		log.Debugf("replica pod ADDED %s service-name=%s", newpod.Name, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]+"-replica")
		//add label onto pod "service-name=clustername-replica"
		serviceName = newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] + "-replica"
	}

	err = kubeapi.AddLabelToPod(c.PodClientset, newpod, config.LABEL_SERVICE_NAME, serviceName, ns)
	if err != nil {
		log.Error(err)
		log.Errorf(" could not add pod label for pod %s and label %s ...", newpod.Name, serviceName)
		return
	}

	//add the service name label to the Deployment
	err = kubeapi.AddLabelToDeployment(c.PodClientset, dep, config.LABEL_SERVICE_NAME, serviceName, ns)

	if err != nil {
		log.Error("could not add label to deployment on pod add")
		return
	}

}
