package util

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
	"database/sql"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	_ "github.com/lib/pq"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	replInfoQueryFormat = "SELECT %s(%s(), '0/0')::bigint, %s(%s(), '0/0')::bigint"

	recvV9         = "pg_last_xlog_receive_location"
	replayV9       = "pg_last_xlog_replay_location"
	locationDiffV9 = "pg_xlog_location_diff"

	recvV10         = "pg_last_wal_receive_lsn"
	replayV10       = "pg_last_wal_replay_lsn"
	locationDiffV10 = "pg_wal_lsn_diff"
)

type ReplicationInfo struct {
	ReceiveLocation uint64
	ReplayLocation  uint64
}

// GetBestTarget
func GetBestTarget(clientset *kubernetes.Clientset, clusterName, namespace string) (*v1.Pod, *v1beta1.Deployment, error) {

	var err error

	//get all the replica deployment pods for this cluster
	var pod v1.Pod
	var deployment v1beta1.Deployment

	//get all the deployments that are replicas for this clustername

	var pods *v1.PodList

	selector := LABEL_PG_CLUSTER + "=" + clusterName + "," + LABEL_PRIMARY + "=false"

	pods, err = kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		return &pod, &deployment, err
	}

	if len(pods.Items) == 0 {
		return &pod, &deployment, errors.New("no replica pods found for cluster " + clusterName)
	}

	for _, p := range pods.Items {
		pod = p
		log.Debug("pod found for replica " + pod.Name)
		if len(pods.Items) == 1 {
			log.Debug("only 1 pod found for failover best match..using it by default")
			return &pod, &deployment, err
		}

		for _, c := range pod.Spec.Containers {
			log.Debug("container " + c.Name + " found in pod")
		}

	}

	return &pod, &deployment, err
}

// GetPod determines the best target to fail to
func GetPod(clientset *kubernetes.Clientset, deploymentName, namespace string) (*v1.Pod, error) {

	var err error

	var pod *v1.Pod
	var pods *v1.PodList

	selector := "replica-name=" + deploymentName
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

func GetRepStatus(restclient *rest.RESTClient, clientset *kubernetes.Clientset, dep *v1beta1.Deployment, namespace string) (uint64, uint64) {
	var receiveLocation, replayLocation uint64

	//get the pods for this deployment
	selector := "primary=false,replica-name=" + dep.Name
	podList, err := kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		log.Error(err.Error())
		return receiveLocation, replayLocation
	}

	if len(podList.Items) != 1 {
		log.Debug("no replicas found for dep " + dep.Name)
		return receiveLocation, replayLocation
	}

	pod := podList.Items[0]

	//get the crd for this dep
	cluster := crv1.Pgcluster{}
	var clusterfound bool
	clusterfound, err = kubeapi.Getpgcluster(restclient, &cluster, dep.ObjectMeta.Labels[LABEL_PG_CLUSTER], namespace)
	if err != nil || !clusterfound {
		log.Error("Getpgcluster error: " + err.Error())
		return receiveLocation, replayLocation
	}

	//get the postgres secret for this dep
	var secretInfo []msgs.ShowUserSecret
	secretInfo, err = GetSecrets(clientset, &cluster, namespace)
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
		log.Error("postgres secret not found for " + dep.Name)
		return receiveLocation, replayLocation
	}

	port := "5432"
	databaseName := "postgres"
	target := getSQLTarget(&pod, pgSecret.Username, pgSecret.Password, port, databaseName)
	var repInfo *ReplicationInfo
	repInfo, err = GetReplicationInfo(target)
	if err != nil {
		log.Error(err)
		return receiveLocation, replayLocation
	}

	receiveLocation = repInfo.ReceiveLocation
	replayLocation = repInfo.ReplayLocation
	return receiveLocation, replayLocation
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
	var replicationInfoQuery string
	var recvLocation uint64
	var replayLocation uint64

	if version < 100000 {
		replicationInfoQuery = fmt.Sprintf(
			replInfoQueryFormat,
			locationDiffV9, recvV9,
			locationDiffV9, replayV9,
		)
	} else {
		replicationInfoQuery = fmt.Sprintf(
			replInfoQueryFormat,
			locationDiffV10, recvV10,
			locationDiffV10, replayV10,
		)
	}

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

	return &ReplicationInfo{recvLocation, replayLocation}, nil
}
