package scheduleservice

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"fmt"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/apiserver/backupoptions"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type scheduleRequest struct {
	Request  *msgs.CreateScheduleRequest
	Response *msgs.CreateScheduleResponse
}

func (s scheduleRequest) createBackRestSchedule(cluster *crv1.Pgcluster, ns string) *PgScheduleSpec {
	name := fmt.Sprintf("%s-%s-%s", cluster.Name, s.Request.ScheduleType, s.Request.PGBackRestType)

	if err := apiserver.ValidateBackrestStorageTypeForCommand(cluster, s.Request.BackrestStorageType); err != nil {
		s.Response.Status.Code = msgs.Error
		s.Response.Status.Msg = err.Error()
		return &PgScheduleSpec{}
	}

	schedule := &PgScheduleSpec{
		Name:      name,
		Cluster:   cluster.Name,
		Version:   "v1",
		Created:   time.Now().Format(time.RFC3339),
		Schedule:  s.Request.Schedule,
		Type:      s.Request.ScheduleType,
		Namespace: ns,
		PGBackRest: PGBackRest{
			Label:       fmt.Sprintf("pg-cluster=%s,name=%s,deployment-name=%s", cluster.Name, cluster.Name, cluster.Name),
			Container:   "database",
			Type:        s.Request.PGBackRestType,
			StorageType: s.Request.BackrestStorageType,
			Options:     s.Request.ScheduleOptions,
		},
	}
	return schedule
}

func (s scheduleRequest) createPolicySchedule(cluster *crv1.Pgcluster, ns string) *PgScheduleSpec {
	name := fmt.Sprintf("%s-%s-%s", cluster.Name, s.Request.ScheduleType, s.Request.PolicyName)

	err := util.ValidatePolicy(apiserver.Clientset, ns, s.Request.PolicyName)
	if err != nil {
		s.Response.Status.Code = msgs.Error
		s.Response.Status.Msg = fmt.Sprintf("policy %s not found", s.Request.PolicyName)
		return &PgScheduleSpec{}
	}

	if s.Request.Secret == "" {
		s.Request.Secret = crv1.UserSecretName(cluster, crv1.PGUserSuperuser)
	}

	schedule := &PgScheduleSpec{
		Name:      name,
		Cluster:   cluster.Name,
		Version:   "v1",
		Created:   time.Now().Format(time.RFC3339),
		Schedule:  s.Request.Schedule,
		Type:      s.Request.ScheduleType,
		Namespace: ns,
		Policy: Policy{
			Name:        s.Request.PolicyName,
			Database:    s.Request.Database,
			Secret:      s.Request.Secret,
			ImagePrefix: util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, apiserver.Pgo.Cluster.CCPImagePrefix),
			ImageTag:    apiserver.Pgo.Cluster.CCPImageTag,
		},
	}
	return schedule
}

//  CreateSchedule
func CreateSchedule(request *msgs.CreateScheduleRequest, ns string) msgs.CreateScheduleResponse {
	ctx := context.TODO()

	log.Debugf("Create schedule called: %s", request.ClusterName)
	sr := &scheduleRequest{
		Request: request,
		Response: &msgs.CreateScheduleResponse{
			Status: msgs.Status{
				Code: msgs.Ok,
				Msg:  "",
			},
			Results: make([]string, 0),
		},
	}

	log.Debug("Getting cluster")
	var selector string
	if sr.Request.ClusterName != "" {
		selector = fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, sr.Request.ClusterName)
	} else if sr.Request.Selector != "" {
		selector = sr.Request.Selector
	}

	clusterList, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		sr.Response.Status.Code = msgs.Error
		sr.Response.Status.Msg = fmt.Sprintf("Could not get cluster via selector: %s", err)
		return *sr.Response
	}

	// validate schedule options
	if sr.Request.ScheduleOptions != "" {
		err := backupoptions.ValidateBackupOpts(sr.Request.ScheduleOptions, request)
		if err != nil {
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = err.Error()
			return *sr.Response
		}
	}

	log.Debug("Making schedules")
	var schedules []*PgScheduleSpec
	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]
		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = cluster.Name + msgs.UpgradeError
			return *sr.Response
		}
		switch sr.Request.ScheduleType {
		case "pgbackrest":
			schedule := sr.createBackRestSchedule(cluster, ns)
			schedules = append(schedules, schedule)
		case "policy":
			schedule := sr.createPolicySchedule(cluster, ns)
			schedules = append(schedules, schedule)
		default:
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = fmt.Sprintf("Schedule type unknown: %s", sr.Request.ScheduleType)
			return *sr.Response
		}

		if sr.Response.Status.Code == msgs.Error {
			return *sr.Response
		}
	}

	log.Debug("Marshalling schedules")
	for _, schedule := range schedules {
		log.Debug(schedule.Name, schedule.Cluster)
		blob, err := json.Marshal(schedule)
		if err != nil {
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = err.Error()
		}

		log.Debug("Getting configmap..")
		_, err = apiserver.Clientset.CoreV1().ConfigMaps(schedule.Namespace).Get(ctx, schedule.Name, metav1.GetOptions{})
		if err == nil {
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = fmt.Sprintf("Schedule %s already exists", schedule.Name)
			return *sr.Response
		}

		labels := make(map[string]string)
		labels["pg-cluster"] = schedule.Cluster
		labels["crunchy-scheduler"] = "true"

		data := make(map[string]string)
		data[schedule.Name] = string(blob)

		configmap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:   schedule.Name,
				Labels: labels,
			},
			Data: data,
		}

		log.Debug("Creating configmap..")
		_, err = apiserver.Clientset.CoreV1().ConfigMaps(schedule.Namespace).Create(ctx, configmap, metav1.CreateOptions{})
		if err != nil {
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = err.Error()
			return *sr.Response
		}

		msg := fmt.Sprintf("created schedule %s for cluster %s", configmap.ObjectMeta.Name, schedule.Cluster)
		sr.Response.Results = append(sr.Response.Results, msg)
	}
	return *sr.Response
}

//  DeleteSchedule ...
func DeleteSchedule(request *msgs.DeleteScheduleRequest, ns string) msgs.DeleteScheduleResponse {
	ctx := context.TODO()

	log.Debug("Deleted schedule called")

	sr := &msgs.DeleteScheduleResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
		Results: make([]string, 0),
	}

	if request.ScheduleName == "" && request.ClusterName == "" && request.Selector == "" {
		sr.Status.Code = msgs.Error
		sr.Status.Msg = "Cluster name, schedule name or selector must be provided"
		return *sr
	}

	schedules := []string{}
	var err error
	if request.ScheduleName != "" {
		schedules = append(schedules, request.ScheduleName)
	} else {
		schedules, err = getSchedules(request.ClusterName, request.Selector, ns)
		if err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = err.Error()
			return *sr
		}
	}

	log.Debug("Deleting configMaps")
	for _, schedule := range schedules {
		err := apiserver.Clientset.CoreV1().ConfigMaps(ns).Delete(ctx, schedule, metav1.DeleteOptions{})
		if err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = fmt.Sprintf("Could not delete ConfigMap %s: %s", schedule, err)
			return *sr
		}
		msg := fmt.Sprintf("deleted schedule %s", schedule)
		sr.Results = append(sr.Results, msg)
	}

	return *sr
}

//  ShowSchedule ...
func ShowSchedule(request *msgs.ShowScheduleRequest, ns string) msgs.ShowScheduleResponse {
	ctx := context.TODO()

	log.Debug("Show schedule called")

	sr := &msgs.ShowScheduleResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
		Results: make([]string, 0),
	}

	if request.ScheduleName == "" && request.ClusterName == "" && request.Selector == "" {
		sr.Status.Code = msgs.Error
		sr.Status.Msg = "Cluster name, schedule name or selector must be provided"
		return *sr
	}

	schedules := []string{}
	var err error
	if request.ScheduleName != "" {
		schedules = append(schedules, request.ScheduleName)
	} else {
		schedules, err = getSchedules(request.ClusterName, request.Selector, ns)
		if err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = err.Error()
			return *sr
		}
	}

	log.Debug("Parsing configMaps")
	for _, schedule := range schedules {
		cm, err := apiserver.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, schedule, metav1.GetOptions{})
		if err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = fmt.Sprintf("Could not delete ConfigMap %s: %s", schedule, err)
			return *sr
		}

		var blob PgScheduleSpec
		log.Debug(cm.Data[schedule])
		if err := json.Unmarshal([]byte(cm.Data[schedule]), &blob); err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = fmt.Sprintf("Could not parse schedule json %s: %s", schedule, err)
			return *sr
		}

		results := fmt.Sprintf("%s:\n\tschedule: %s\n\tschedule-type: %s", blob.Name, blob.Schedule, blob.Type)
		if blob.Type == "pgbackrest" {
			results += fmt.Sprintf("\n\tbackup-type: %s", blob.PGBackRest.Type)
		}
		sr.Results = append(sr.Results, results)
	}
	return *sr
}

func getSchedules(clusterName, selector, ns string) ([]string, error) {
	ctx := context.TODO()
	schedules := []string{}
	label := "crunchy-scheduler=true"
	if clusterName == "all" {
	} else if clusterName != "" {
		label += fmt.Sprintf(",pg-cluster=%s", clusterName)
	}

	if selector != "" {
		label += fmt.Sprintf(",%s", selector)
	}

	log.Debugf("Finding configMaps with selector: %s", label)
	list, err := apiserver.Clientset.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, fmt.Errorf("No schedules found for selector: %s", label)
	}

	for _, cm := range list.Items {
		schedules = append(schedules, cm.Name)
	}

	return schedules, nil
}
