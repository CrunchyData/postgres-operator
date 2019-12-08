package util

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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	replInfoQueryPre10 = `SELECT
			COALESCE(pg_xlog_location_diff(pg_last_xlog_receive_location(), '0/0'), 0)::bigint AS replication_position,
			COALESCE(pg_xlog_location_diff(pg_last_xlog_replay_location(), '0/0'), 0)::bigint AS replay_position`

	replInfoQueryPost10 = `SELECT
			COALESCE(pg_wal_lsn_diff(pg_last_wal_receive_lsn(), '0/0')::bigint, 0) AS replication_position,
			COALESCE(pg_wal_lsn_diff(pg_last_wal_replay_lsn(), '0/0'), 0)::bigint AS replay_position`
)

type ReplicationInfo struct {
	ReceiveLocation uint64
	ReplayLocation  uint64
	Node            string
	DeploymentName  string
}

// GetBestTarget
func GetBestTarget(clientset *kubernetes.Clientset, clusterName, namespace string) (*v1.Pod, *appsv1.Deployment, error) {

	var err error

	//get all the replica deployment pods for this cluster
	var pod v1.Pod
	var deployment appsv1.Deployment

	//get all the deployments that are replicas for this clustername

	var pods *v1.PodList

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_SERVICE_NAME + "=" + clusterName + "-replica"

	pods, err = kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		return &pod, &deployment, err
	}

	if len(pods.Items) == 0 {
		return &pod, &deployment, errors.New("no replica pods found for cluster " + clusterName)
	}

	for _, p := range pods.Items {
		pod = p
		log.Debugf("pod found for replica %s", pod.Name)
		if len(pods.Items) == 1 {
			log.Debug("only 1 pod found for failover best match..using it by default")
			return &pod, &deployment, err
		}

		for _, c := range pod.Spec.Containers {
			log.Debugf("container %s found in pod", c.Name)
		}

	}

	return &pod, &deployment, err
}

// GetPod determines the best target to fail to
func GetPod(clientset *kubernetes.Clientset, deploymentName, namespace string) (*v1.Pod, error) {

	var err error

	var pod *v1.Pod
	var pods *v1.PodList

	selector := config.LABEL_DEPLOYMENT_NAME + "=" + deploymentName + "," + config.LABEL_PGHA_ROLE + "=replica"
	pods, err = kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		return pod, err
	}
	if len(pods.Items) != 1 {
		return pod, errors.New("could not determine which pod to failover to")
	}

	for _, v := range pods.Items {
		pod = &v
	}

	found := false

	//make sure the pod has a database container it it
	for _, c := range pod.Spec.Containers {
		if c.Name == "database" {
			found = true
		}
	}

	if !found {
		return pod, errors.New("could not find a database container in the pod")
	}

	return pod, err
}

// GetRepStatus is responsible for retrieving and returning the replication status of the pg replica
// deployed in the pod specified.  Specifically, the function returns the location of the WAL file
// that was most recently synced to disk on the replica (i.e. pg_last_xlog_receive_location() or
// pg_last_wal_receive_lsn()), and the location of the WAL file that was most recently replayed on
// the replica(i.e. pg_last_xlog_replay_location() or pg_last_wal_replay_lsn()), both as unint64.
// Additionally, the  name of the Kubernetes node the replica is running on is returned.  This
// function therefore provides insight into the replication lag for a replica, as needed to support
// selection of replicas when performing manual failovers and scaling down the cluster.
func GetRepStatus(restclient *rest.RESTClient, clientset *kubernetes.Clientset, pod *v1.Pod, namespace, databasePort string) (uint64, uint64, string, error) {
	var err error

	var receiveLocation, replayLocation uint64

	var nodeName string

	//get the crd for this dep
	cluster := crv1.Pgcluster{}
	var clusterfound bool
	clusterfound, err = kubeapi.Getpgcluster(restclient, &cluster, pod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER], namespace)
	if err != nil || !clusterfound {
		log.Error("Getpgcluster error: " + err.Error())
		return receiveLocation, replayLocation, nodeName, err
	}

	//get the postgres secret for this dep
	var secretInfo []msgs.ShowUserSecret
	secretInfo, err = getSecrets(clientset, &cluster, namespace)
	var pgSecret msgs.ShowUserSecret
	var found bool
	for _, si := range secretInfo {
		if si.Username == "postgres" {
			pgSecret = si
			found = true
			log.Debug("postgres secret found")
		}
	}

	if !found {
		log.Error("postgres secret not found for " + pod.Labels[config.LABEL_DEPLOYMENT_NAME])
		return receiveLocation, replayLocation, nodeName, errors.New("postgres secret not found for " +
			pod.Labels[config.LABEL_DEPLOYMENT_NAME])
	}

	port := databasePort
	databaseName := "postgres"
	target := getSQLTarget(pod, pgSecret.Username, pgSecret.Password, port, databaseName)
	var repInfo *ReplicationInfo
	repInfo, err = GetReplicationInfo(target)
	if err != nil {
		log.Error(err)
		return receiveLocation, replayLocation, nodeName, err
	}

	receiveLocation = repInfo.ReceiveLocation
	replayLocation = repInfo.ReplayLocation

	nodeName = pod.Spec.NodeName

	return receiveLocation, replayLocation, nodeName, nil
}

func getSQLTarget(pod *v1.Pod, username, password, port, db string) string {
	target := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		username,
		password,
		pod.Status.PodIP,
		port,
		db,
	)
	return target

}
func GetReplicationInfo(target string) (*ReplicationInfo, error) {
	conn, err := sql.Open("postgres", target)

	if err != nil {
		log.Errorf("Could not connect to: %s", target)
		return nil, err
	}

	defer conn.Close()

	// Get PG version
	var version int

	rows, err := conn.Query("SELECT current_setting('server_version_num')")

	if err != nil {
		log.Errorf("Could not perform query for version: %s", target)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
	}

	// Get replication info
	replicationInfoQuery := replInfoQueryPost10
	if version < 100000 {
		replicationInfoQuery = replInfoQueryPre10
	}

	var recvLocation uint64
	var replayLocation uint64

	rows, err = conn.Query(replicationInfoQuery)

	if err != nil {
		log.Errorf("Could not perform replication info query: %s", target)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&recvLocation, &replayLocation); err != nil {
			return nil, err
		}
	}

	return &ReplicationInfo{
		ReceiveLocation: recvLocation,
		ReplayLocation:  replayLocation,
		Node:            "",
		DeploymentName:  "",
	}, nil
}

func getSecrets(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, namespace string) ([]msgs.ShowUserSecret, error) {

	output := make([]msgs.ShowUserSecret, 0)
	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	secrets, err := kubeapi.GetSecrets(clientset, selector, namespace)
	if err != nil {
		return output, err
	}

	log.Debugf("got %d secrets for %s", len(secrets.Items), cluster.Spec.Name)
	for _, s := range secrets.Items {
		d := msgs.ShowUserSecret{}
		d.Name = s.Name
		d.Username = string(s.Data["username"][:])
		d.Password = string(s.Data["password"][:])
		output = append(output, d)

	}

	return output, err
}

func GetPreferredNodes(clientset *kubernetes.Clientset, selector, namespace string) ([]string, error) {
	nodes := make([]string, 0)

	nodeList, err := kubeapi.GetNodes(clientset, selector, namespace)
	if err != nil {
		return nodes, err
	}

	log.Debugf("getPreferredNodes shows %d nodes", len(nodeList.Items))

	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}

	return nodes, err
}

// ToggleAutoFailover enables or disables autofailover for a cluster.  Disabling autofailover means "pausing"
// Patroni, which will result in Patroni stepping aside from managing the cluster.  This will effectively cause
// Patroni to stop responding to failures or other database activities, e.g. it will not attempt to start the
// database when stopped to perform maintenance
func ToggleAutoFailover(clientset *kubernetes.Clientset, enable bool, pghaScope, namespace string) error {

	// find the "config" configMap created by Patroni
	configMapName := pghaScope + "-config"
	log.Debugf("setting autofailover to %t for cluster with pgha scope %s", enable, pghaScope)

	configMap, found := kubeapi.GetConfigMap(clientset, configMapName, namespace)
	if !found {
		err := fmt.Errorf("Unable to find configMap %s when attempting disable autofailover", configMapName)
		log.Error(err)
		return err
	}

	configJSONStr := configMap.ObjectMeta.Annotations["config"]

	var configJSON map[string]interface{}
	json.Unmarshal([]byte(configJSONStr), &configJSON)

	if !enable {
		// disable autofail condition
		disableFailover(clientset, configMap, configJSON, namespace)
	} else {
		// enable autofail
		enableFailover(clientset, configMap, configJSON, namespace)
	}

	return nil
}

// If "pause" is present in the config and set to "true", then it needs to be removed to enable
// failover.  Otherwise, if "pause" isn't present in the config or if it has a value other than
// true, then assume autofail is enabled and do nothing (when Patroni see's an invalid value for
// "pause" it sets it to "true")
func enableFailover(clientset *kubernetes.Clientset, configMap *v1.ConfigMap, configJSON map[string]interface{},
	namespace string) error {
	if _, ok := configJSON["pause"]; ok && configJSON["pause"] == true {
		log.Debugf("updating pause key in configMap %s to enable autofailover", configMap.Name)
		//  disabled autofail by removing "pause" from the config
		delete(configJSON, "pause")
		configJSONFinalStr, err := json.Marshal(configJSON)
		if err != nil {
			return err
		}
		configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
		err = kubeapi.UpdateConfigMap(clientset, configMap, namespace)
		if err != nil {
			return err
		}
	} else {
		log.Debugf("autofailover already enabled according to the pause key (or lack thereof) in configMap %s",
			configMap.Name)
	}
	return nil
}

// If "pause" isn't present in the config then assume autofail is enabled and needs to be disabled
// by setting "pause" to true.  Or if it is present and set to something other than "true" (e.g.
// "false" or "null"), then it also needs to be disabled by setting "pause" to true.
func disableFailover(clientset *kubernetes.Clientset, configMap *v1.ConfigMap, configJSON map[string]interface{},
	namespace string) error {
	if _, ok := configJSON["pause"]; !ok || configJSON["pause"] != true {
		log.Debugf("updating pause key in configMap %s to disable autofailover", configMap.Name)
		// disable autofail by setting "pause" to true
		configJSON["pause"] = true
		configJSONFinalStr, err := json.Marshal(configJSON)
		if err != nil {
			return err
		}
		configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
		err = kubeapi.UpdateConfigMap(clientset, configMap, namespace)
		if err != nil {
			return err
		}
	} else {
		log.Debugf("autofailover already disabled according to the pause key in configMap %s",
			configMap.Name)
	}
	return nil
}
