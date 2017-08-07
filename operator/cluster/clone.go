/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package cluster

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func ProcessClone(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgClone)

	source := cache.NewListWatchFromClient(client, tpr.CLONE_RESOURCE, namespace, fields.Everything())

	createAddHandler := func(obj interface{}) {
		clone := obj.(*tpr.PgClone)
		eventchan <- clone
		addClone(clientset, client, clone, namespace)
	}
	createDeleteHandler := func(obj interface{}) {
		clone := obj.(*tpr.PgClone)
		eventchan <- clone
		//deleteClone(clientset, client, clone, namespace)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		clone := obj.(*tpr.PgClone)
		eventchan <- clone
		//log.Info("updating PgUpgrade object")
		//log.Info("updated with Name=" + job.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgClone{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createAddHandler,
			UpdateFunc: updateHandler,
			DeleteFunc: createDeleteHandler,
		})

	go controller.Run(stopchan)

	for {
		select {
		case event := <-eventchan:
			//log.Infof("%#v\n", event)
			if event == nil {
				log.Info("event was null")
			}
		}
	}

}

func addClone(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, clone *tpr.PgClone, namespace string) {
	log.Debug("addClone called")

	log.Debug("clone.Spec.Name is " + clone.Spec.Name)
	log.Debug("clone.Spec.ClusterName is " + clone.Spec.ClusterName)

	//get PgCluster
	//lookup the cluster
	cl := tpr.PgCluster{}
	err := tprclient.Get().
		Resource(tpr.CLUSTER_RESOURCE).
		Namespace(namespace).
		Name(clone.Spec.ClusterName).
		Do().
		Into(&cl)
	if err == nil {
		log.Debug("got cluster in clone prep")
	} else if kerrors.IsNotFound(err) {
		log.Error("could not get cluster in clone prep using " + clone.Spec.ClusterName)
		return
	} else {
		log.Errorf("\npgcluster %s\n", clone.Spec.ClusterName+" lookup error "+err.Error())
		return
	}

	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for clone creation" + cl.Spec.STRATEGY)
		return
	}

	strategy.PrepareClone(clientset, tprclient, clone.Spec.Name, &cl, namespace)

}

func CompleteClone(config *rest.Config, clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	log.Debug("setting up clone Deployment watch")

	lo := meta_v1.ListOptions{LabelSelector: "clone"}
	fw, err := clientset.ExtensionsV1beta1().Deployments(namespace).Watch(lo)
	if err != nil {
		log.Error("error watching pg-cluster deployments" + err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infoln("got a deployment watch event")

		switch event.Type {
		case watch.Added:
			dep := event.Object.(*v1beta1.Deployment)
			log.Infof("clone pg-cluster Deployment added=%d\n", dep.Status.AvailableReplicas)
		case watch.Deleted:
			dep := event.Object.(*v1beta1.Deployment)
			log.Infof("clone pg-cluster Deployment deleted=%d\n", dep.Status.AvailableReplicas)
		case watch.Error:
			log.Infof("clone pg-cluster Deployment watch error event")
		case watch.Modified:
			dep := event.Object.(*v1beta1.Deployment)
			log.Infof("clone pg-cluster Deployment modified=%d\n", dep.Status.AvailableReplicas)
			if dep.Status.AvailableReplicas == 1 {
				log.Infoln("clone pg-cluster Deployment " + dep.Name + " succeeded")
				finishClone(config, clientset, client, dep, namespace)

			}
		default:
			log.Infoln("pg-cluster Deployment unknown watch event %v\n", event.Type)
		}

		return false, nil
	})

	if err4 != nil {
		log.Error("erro in clone complete " + err4.Error())
	}

}

func finishClone(config *rest.Config, clientset *kubernetes.Clientset, tprclient *rest.RESTClient, dep *v1beta1.Deployment, namespace string) {
	//trigger the failover of the clone replica to make it a master
	//cmd := "touch /tmp/pg-failover-trigger"
	//cmd := "ls"
	containername := "database"
	podname, err := getMasterPodName(clientset, dep.Name, namespace)
	cmd := []string{"touch", "/tmp/pg-failover-trigger"}
	err = util.Exec(config, namespace, podname, containername, cmd)

	clone := tpr.PgClone{}
	err = tprclient.Get().
		Resource(tpr.CLONE_RESOURCE).
		Namespace(namespace).
		Name(dep.Name).
		Do().
		Into(&clone)
	if kerrors.IsNotFound(err) {
		log.Error("pgclone TPR not found for " + dep.Name)
		return
	} else if err != nil {
		log.Error("pgclone " + dep.Name + " lookup error " + err.Error())
		return
	}

	cluster := tpr.PgCluster{}
	err = tprclient.Get().
		Resource(tpr.CLUSTER_RESOURCE).
		Namespace(namespace).
		Name(clone.Spec.ClusterName).
		Do().
		Into(&cluster)
	if kerrors.IsNotFound(err) {
		log.Error("pgcluster TPR not found for " + clone.Spec.ClusterName)
		return
	} else if err != nil {
		log.Error("pgcluster " + clone.Spec.ClusterName + " lookup error " + err.Error())
		return
	}

	//copy the secrets
	err = util.CopySecrets(clientset, namespace, clone.Spec.ClusterName, clone.Spec.Name)
	if err != nil {
		log.Error("error in copying secrets for clone " + err.Error())
		return
	}

	//override old name with new name
	newcluster := copyTPR(&cluster, &clone)

	//create the tpr for the clone, this will cause services and replica
	//deployment to be created

	result := tpr.PgCluster{}

	err = tprclient.Post().
		Resource(tpr.CLUSTER_RESOURCE).
		Namespace(namespace).
		Body(newcluster).
		Do().Into(&result)
	if err != nil {
		log.Error("error in finish clone " + err.Error())
	}

	//delete the pgclone after the cloning
	err = tprclient.Delete().
		Resource(tpr.CLONE_RESOURCE).
		Namespace(namespace).
		Name(dep.Name).
		Do().
		Error()

	if err != nil {
		log.Error("error deleting pgclone " + err.Error())
	} else {
		log.Info("deleted pgclone " + dep.Name)
	}

	log.Info("finished clone " + dep.Name)

}

func getMasterPodName(clientset *kubernetes.Clientset, clusterName, namespace string) (string, error) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + clusterName}
	pods, err := clientset.CoreV1().Pods(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of pods" + err.Error())
		return "", err
	}
	//assume the first pod since this is for getting the master pod
	for _, pod := range pods.Items {
		return pod.ObjectMeta.Name, err
	}

	return "", err

}

func copyTPR(cluster *tpr.PgCluster, clone *tpr.PgClone) *tpr.PgCluster {
	spec := tpr.PgClusterSpec{}

	spec.PGUSER_SECRET_NAME = strings.Replace(cluster.Spec.PGUSER_SECRET_NAME, cluster.Spec.ClusterName, clone.Spec.Name, 1)
	spec.PGMASTER_SECRET_NAME = strings.Replace(cluster.Spec.PGMASTER_SECRET_NAME, cluster.Spec.ClusterName, clone.Spec.Name, 1)
	spec.PGROOT_SECRET_NAME = strings.Replace(cluster.Spec.PGROOT_SECRET_NAME, cluster.Spec.ClusterName, clone.Spec.Name, 1)

	spec.Name = clone.Spec.Name
	spec.ClusterName = clone.Spec.Name

	spec.CCP_IMAGE_TAG = cluster.Spec.CCP_IMAGE_TAG
	spec.Port = cluster.Spec.Port

	spec.MasterStorage = cluster.Spec.MasterStorage
	spec.ReplicaStorage = cluster.Spec.ReplicaStorage

	spec.SECRET_FROM = cluster.Spec.SECRET_FROM
	spec.BACKUP_PATH = cluster.Spec.BACKUP_PATH
	spec.BACKUP_PVC_NAME = cluster.Spec.BACKUP_PVC_NAME
	spec.PG_MASTER_HOST = cluster.Spec.PG_MASTER_HOST
	spec.PG_MASTER_USER = cluster.Spec.PG_MASTER_USER
	spec.PG_MASTER_PASSWORD = cluster.Spec.PG_MASTER_PASSWORD
	spec.PG_USER = cluster.Spec.PG_USER
	spec.PG_PASSWORD = cluster.Spec.PG_PASSWORD
	spec.PG_DATABASE = cluster.Spec.PG_DATABASE
	spec.PG_ROOT_PASSWORD = cluster.Spec.PG_ROOT_PASSWORD
	spec.REPLICAS = cluster.Spec.REPLICAS
	spec.FS_GROUP = cluster.Spec.FS_GROUP
	spec.SUPPLEMENTAL_GROUPS = cluster.Spec.SUPPLEMENTAL_GROUPS
	spec.STRATEGY = cluster.Spec.STRATEGY
	spec.BACKUP_PATH = cluster.Spec.BACKUP_PATH

	newInstance := &tpr.PgCluster{
		Metadata: meta_v1.ObjectMeta{
			Name: clone.Spec.Name,
		},
		Spec: spec,
	}
	return newInstance

}
