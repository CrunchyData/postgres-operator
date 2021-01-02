package restartservice

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
	"context"
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Restart restarts either all PostgreSQL databases within a PostgreSQL cluster (i.e. the primary
// and all replicas) or if targets are specified, just those targets.
// pgo restart mycluster
// pgo restart mycluster --target=mycluster-abcd
func Restart(request *msgs.RestartRequest, pgouser string) msgs.RestartResponse {
	ctx := context.TODO()

	log.Debugf("restart called for %s", request.ClusterName)

	resp := msgs.RestartResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	clusterName := request.ClusterName
	namespace := request.Namespace

	cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).
		Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// check if the current cluster is not upgraded to the deployed
	// Operator version. If not, do not allow the command to complete
	if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("%s %s", cluster.Name, msgs.UpgradeError)
		return resp
	}

	// if a rolling update is requested, this takes a detour to create a pgtask
	// to accomplish this
	if request.RollingUpdate {
		// since a rolling update takes time, this needs to be performed as a
		// separate task
		// Create a pgtask
		task := &crv1.Pgtask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", cluster.Name, config.LABEL_RESTART),
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					config.LABEL_PG_CLUSTER: cluster.Name,
					config.LABEL_PGOUSER:    pgouser,
				},
			},
			Spec: crv1.PgtaskSpec{
				TaskType: crv1.PgtaskRollingUpdate,
				Parameters: map[string]string{
					config.LABEL_PG_CLUSTER: cluster.Name,
				},
			},
		}

		// remove any previous rolling restart, then add a new one
		if err := apiserver.Clientset.CrunchydataV1().Pgtasks(task.Namespace).Delete(ctx, task.Name,
			metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if _, err := apiserver.Clientset.CrunchydataV1().Pgtasks(cluster.Namespace).Create(ctx, task,
			metav1.CreateOptions{}); err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
		}

		return resp
	}

	var restartResults []patroni.RestartResult
	// restart either the whole cluster, or just any targets specified
	patroniClient := patroni.NewPatroniClient(apiserver.RESTConfig, apiserver.Clientset,
		cluster.GetName(), namespace)
	if len(request.Targets) > 0 {
		restartResults, err = patroniClient.RestartInstances(request.Targets...)
	} else {
		restartResults, err = patroniClient.RestartCluster()
	}
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	restartDetails := msgs.RestartDetail{ClusterName: clusterName}
	for _, restartResult := range restartResults {

		instanceDetail := msgs.InstanceDetail{InstanceName: restartResult.Instance}
		if restartResult.Error != nil {
			instanceDetail.Error = true
			instanceDetail.ErrorMessage = restartResult.Error.Error()
		}

		restartDetails.Instances = append(restartDetails.Instances, instanceDetail)
	}

	resp.Result = restartDetails

	return resp
}

// QueryRestart queries a cluster for instances available to use as as targets for a PostgreSQL restart.
// pgo restart mycluster --query
func QueryRestart(clusterName, namespace string) msgs.QueryRestartResponse {
	ctx := context.TODO()

	log.Debugf("query restart called for %s", clusterName)

	resp := msgs.QueryRestartResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).
		Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// Get information about the current status of all of all cluster instances. This is
	// handled by a helper function, that will return the information in a struct with the
	// key elements to help the user understand the current state of the instances in a cluster
	replicationStatusRequest := util.ReplicationStatusRequest{
		RESTConfig:  apiserver.RESTConfig,
		Clientset:   apiserver.Clientset,
		Namespace:   namespace,
		ClusterName: clusterName,
	}

	// get a list of all the Pods...note that we can included "busted" pods as
	// by including the primary, we're getting all of the database pods anyway.
	replicationStatusResponse, err := util.ReplicationStatus(replicationStatusRequest, true, true)
	if err != nil {
		log.Error(err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// if there are no results, return the response as is
	if len(replicationStatusResponse.Instances) == 0 {
		return resp
	}

	// iterate through response results to create the API response
	for _, instance := range replicationStatusResponse.Instances {
		// create an result for the response
		resp.Results = append(resp.Results, msgs.RestartTargetSpec{
			Name:           instance.Name,
			Node:           instance.Node,
			Status:         instance.Status,
			ReplicationLag: instance.ReplicationLag,
			Timeline:       instance.Timeline,
			PendingRestart: instance.PendingRestart,
			Role:           instance.Role,
		})
	}

	resp.Standby = cluster.Spec.Standby

	return resp
}
