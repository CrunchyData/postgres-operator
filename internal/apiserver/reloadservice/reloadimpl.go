package reloadservice

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reload ...
// pgo reload mycluster
// pgo reload all
// pgo reload --selector=name=mycluster
func Reload(request *msgs.ReloadRequest, ns, username string) msgs.ReloadResponse {
	ctx := context.TODO()

	log.Debugf("Reload %v", request)

	var clusterNames []string
	var errorMsgs []string

	resp := msgs.ReloadResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	if request.Selector != "" {
		clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		} else if len(clusterList.Items) == 0 {
			resp.Results = append(resp.Results, "no clusters found with that selector")
			return resp
		}

		for _, cluster := range clusterList.Items {
			clusterNames = append(clusterNames, cluster.Spec.Name)
		}
	} else {
		clusterNames = request.Args
	}

	for _, clusterName := range clusterNames {

		log.Debugf("reload requested for cluster %s", clusterName)

		cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).
			Get(ctx, clusterName, metav1.GetOptions{})
		// maintain same "is not found" error message for backwards compatibility
		if kerrors.IsNotFound(err) {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s was not found, verify cluster name", clusterName))
			continue
		} else if err != nil {
			errorMsgs = append(errorMsgs, err.Error())
			continue
		}

		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s %s", clusterName, msgs.UpgradeError))
			continue
		}

		// now reload the cluster, providing any targets specified
		patroniClient := patroni.NewPatroniClient(apiserver.RESTConfig, apiserver.Clientset,
			cluster.GetName(), ns)
		if err := patroniClient.ReloadCluster(); err != nil {
			errorMsgs = append(errorMsgs, err.Error())
			continue
		}

		resp.Results = append(resp.Results, fmt.Sprintf("reload performed on %s", clusterName))

		if err := publishReloadClusterEvent(cluster.GetName(), ns, username); err != nil {
			log.Error(err.Error())
			errorMsgs = append(errorMsgs, err.Error())
		}
	}

	if len(errorMsgs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = strings.Join(errorMsgs, "\n")
	}

	return resp
}

// publishReloadClusterEvent publishes an event when a cluster is reloaded
func publishReloadClusterEvent(clusterName, username, namespace string) error {

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventReloadClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventReloadCluster,
		},
		Clustername: clusterName,
	}

	if err := events.Publish(f); err != nil {
		return err
	}

	return nil
}
