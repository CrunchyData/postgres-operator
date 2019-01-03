// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
)

// FailoverBase ...
func FailoverBase(namespace string, clientset *kubernetes.Clientset, client *rest.RESTClient, task *crv1.Pgtask, restconfig *rest.Config) {
	var err error

	//look up the pgcluster for this task
	//in the case, the clustername is passed as a key in the
	//parameters map
	var clusterName string
	for k, _ := range task.Spec.Parameters {
		clusterName = k
	}

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

	//get initial count of replicas --selector=pg-cluster=clusterName
	replicaList := crv1.PgreplicaList{}
	selector := util.LABEL_PG_CLUSTER + "=" + clusterName
	err = kubeapi.GetpgreplicasBySelector(client, &replicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("replica count before failover is %d", len(replicaList.Items))

	strategy.Failover(clientset, client, clusterName, task, namespace, restconfig)
	//remove the pgreplica CRD for the promoted replica
	kubeapi.Deletepgreplica(client, task.ObjectMeta.Labels[util.LABEL_TARGET], namespace)

	//optionally, scale up the replicas to replace the failover target
	replaced := false
	userSelection := task.ObjectMeta.Labels[util.LABEL_AUTOFAIL_REPLACE_REPLICA]
	if userSelection == "true" {
		log.Debug("replacing replica based on user selection")
		replaceReplica(client, &cluster)
		replaced = true
	} else if userSelection == "false" {
		log.Debug("not replacing replica based on user selection")
	} else if operator.Pgo.Cluster.AutofailReplaceReplica {
		log.Debug("replacing replica based on pgo.yaml setting")
		replaceReplica(client, &cluster)
		replaced = true
	} else {
		log.Debug("not replacing replica")
	}

	//see if the replica service needs to be removed
	if !replaced {
		if len(replicaList.Items) == 1 {
			log.Debug("removing replica service since last replica was removed by the failover %s", clusterName)
			err = kubeapi.DeleteService(clientset, clusterName+"-replica", namespace)
			if err != nil {
				log.Error("could not delete replica service as part of failover")
				return
			}
		}
	}

	//if a backrest-repo exists, bounce it with the new
	//DB_PATH set to the new primary deployment name
	if cluster.ObjectMeta.Labels[util.LABEL_BACKREST] == "true" {
		updateDBPath(clientset, &cluster, task.ObjectMeta.Labels[util.LABEL_TARGET], namespace)
	}

}

func replaceReplica(client *rest.RESTClient, cluster *crv1.Pgcluster) {

	//generate new replica name
	uniqueName := cluster.Spec.Name + "-" + util.RandStringBytesRmndr(4)

	spec := crv1.PgreplicaSpec{}
	spec.Name = uniqueName
	spec.ClusterName = cluster.Spec.Name
	spec.ReplicaStorage = cluster.Spec.ReplicaStorage
	spec.ContainerResources = cluster.Spec.ContainerResources
	spec.UserLabels = make(map[string]string)

	for k, v := range cluster.Spec.UserLabels {
		spec.UserLabels[k] = v
	}
	spec.UserLabels[util.LABEL_PRIMARY] = "false"
	spec.UserLabels[util.LABEL_PG_CLUSTER] = cluster.Spec.Name

	newInstance := &crv1.Pgreplica{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: uniqueName,
		},
		Spec: spec,
		Status: crv1.PgreplicaStatus{
			State:   crv1.PgreplicaStateCreated,
			Message: "Created, not processed yet",
		},
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	for x, y := range cluster.ObjectMeta.Labels {
		newInstance.ObjectMeta.Labels[x] = y
	}
	newInstance.ObjectMeta.Labels[util.LABEL_PRIMARY] = "false"
	newInstance.ObjectMeta.Labels[util.LABEL_NAME] = uniqueName

	kubeapi.Createpgreplica(client, newInstance, operator.NAMESPACE)

}

//update the PGBACKREST_DB_PATH env var of the backrest-repo
//deployment for a given cluster, the deployment is bounced as
//part of this process
func updateDBPath(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, target, namespace string) error {
	var err error
	newPath := "/pgdata/" + target
	depName := cluster.Name + "-backrest-repo"

	var deployment *v1beta1.Deployment
	deployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error getting deployment in updateDBPath using name " + depName)
		return err
	}

	log.Debugf("replicas %d", *deployment.Spec.Replicas)

	//drain deployment to 0 pods
	*deployment.Spec.Replicas = 0

	containerIndex := -1
	envIndex := -1
	//update the env var Value
	//template->spec->containers->env["PGBACKREST_DB_PATH"]
	for kc, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == "database" {
			log.Debugf(" %s is the container name at %d", c.Name, kc)
			containerIndex = kc
			for ke, e := range c.Env {
				if e.Name == "PGBACKREST_DB_PATH" {
					log.Debugf("PGBACKREST_DB_PATH is %s", e.Value)
					envIndex = ke
				}
			}
		}
	}

	if containerIndex == -1 || envIndex == -1 {
		return errors.New("error in getting container with PGBACRKEST_DB_PATH for cluster " + cluster.Name)
	}

	deployment.Spec.Template.Spec.Containers[containerIndex].Env[envIndex].Value = newPath

	//update the deployment (drain and update the env var)
	err = kubeapi.UpdateDeployment(clientset, deployment, namespace)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	//wait till deployment goes to 0
	var zero bool
	for i := 0; i < 8; i++ {
		deployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
		if err != nil {
			log.Error("could not get deployment updateDBPath " + err.Error())
			return err
		}

		log.Debugf("status replicas %d\n", deployment.Status.Replicas)
		if deployment.Status.Replicas == 0 {
			log.Debugf("deployment %s replicas is now 0", deployment.Name)
			zero = true
			break
		} else {
			log.Debug("updateDBPath: sleeping till deployment goes to 0")
			time.Sleep(time.Second * time.Duration(2))
		}
	}
	if !zero {
		return errors.New("deployment replicas never went to 0")
	}

	//update the deployment back to replicas 1
	*deployment.Spec.Replicas = 1
	err = kubeapi.UpdateDeployment(clientset, deployment, namespace)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Debugf("updated PGBACKREST_DB_PATH to %s on deployment %s", newPath, cluster.Name)

	return err
}
