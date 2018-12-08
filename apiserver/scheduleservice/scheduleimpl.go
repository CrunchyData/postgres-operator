package scheduleservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type scheduleRequest struct {
	Request  *msgs.CreateScheduleRequest
	Response *msgs.CreateScheduleResponse
}

func getClusterPrimaryPod(cluster string) (string, error) {
	var podName string
	//selector := fmt.Sprintf("%s=true,%s=%s", util.LABEL_PRIMARY, util.LABEL_PG_CLUSTER, cluster)
	selector := fmt.Sprintf("%s=%s,%s=%s", util.LABEL_SERVICE_NAME, cluster, util.LABEL_PG_CLUSTER, cluster)
	log.Debugf("selector in scheduler is %s", selector)
	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return podName, err
	}

	if len(pods.Items) == 0 {
		return podName, errors.New("No primary pods found")
	}

	if len(pods.Items) > 1 {
		return podName, errors.New("More than one primary pod found")
	}

	for _, container := range pods.Items[0].Status.ContainerStatuses {
		if container.Name == "database" {
			if container.Ready {
				podName = pods.Items[0].Name
			} else {
				return podName, errors.New("Database pod is not ready")

			}
		}
	}
	return podName, nil
}

func (s scheduleRequest) createBackRestSchedule(cluster *crv1.Pgcluster) *PgScheduleSpec {
	name := fmt.Sprintf("%s-%s-%s", cluster.Name, s.Request.ScheduleType, s.Request.PGBackRestType)
	schedule := &PgScheduleSpec{
		Name:      name,
		Cluster:   cluster.Name,
		Version:   "v1",
		Created:   time.Now().Format(time.RFC3339),
		Schedule:  s.Request.Schedule,
		Type:      s.Request.ScheduleType,
		Namespace: apiserver.Namespace,
		PGBackRest: PGBackRest{
			Label:     fmt.Sprintf("pg-cluster=%s,service-name=%s", cluster.Name, cluster.Name),
			Container: "database",
			Type:      s.Request.PGBackRestType,
		},
	}
	return schedule
}

func (s scheduleRequest) createBaseBackupSchedule(cluster *crv1.Pgcluster) *PgScheduleSpec {
	name := fmt.Sprintf("backup-%s", cluster.Name)

	_, exists, err := kubeapi.GetPVC(apiserver.Clientset, s.Request.PVCName, apiserver.Namespace)
	if err != nil {
		s.Response.Status.Code = msgs.Error
		s.Response.Status.Msg = err.Error()
		return &PgScheduleSpec{}
	} else if !exists {
		s.Response.Status.Code = msgs.Error
		s.Response.Status.Msg = fmt.Sprintf("PVC does not exist for backup: %s", s.Request.PVCName)
		return &PgScheduleSpec{}
	}

	imageTag := s.Request.CCPImageTag
	if imageTag == "" {
		imageTag = apiserver.Pgo.Cluster.CCPImageTag
	}

	schedule := &PgScheduleSpec{
		Name:      name,
		Cluster:   cluster.Name,
		Version:   "v1",
		Created:   time.Now().Format(time.RFC3339),
		Schedule:  s.Request.Schedule,
		Type:      s.Request.ScheduleType,
		Namespace: apiserver.Namespace,
		PGBaseBackup: PGBaseBackup{
			BackupHost:   cluster.Spec.PrimaryHost,
			BackupPort:   cluster.Spec.Port,
			BackupVolume: s.Request.PVCName,
			ImagePrefix:  "crunchydata",
			ImageTag:     imageTag,
			Secret:       cluster.Spec.PrimarySecretName,
		},
	}
	return schedule
}

//  CreateSchedule
func CreateSchedule(request *msgs.CreateScheduleRequest) msgs.CreateScheduleResponse {
	log.Debugf("Create schedule called clusterName is %s", request.ClusterName)
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
	if sr.Request.ClusterName != "" {
		if sr.Request.Selector != "" {
			sr.Request.Selector += ","
		}
		sr.Request.Selector += fmt.Sprintf("pg-cluster=%s,service-name=%s", sr.Request.ClusterName, sr.Request.ClusterName)
	}

	clusterList := crv1.PgclusterList{}
	err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, sr.Request.Selector, apiserver.Namespace)
	if err != nil {
		sr.Response.Status.Code = msgs.Error
		sr.Response.Status.Msg = fmt.Sprintf("Could not get cluster via selector: %s", err)
		return *sr.Response
	}

	log.Debug("Making schedules")
	var schedules []*PgScheduleSpec
	for _, cluster := range clusterList.Items {
		switch sr.Request.ScheduleType {
		case "pgbackrest":
			schedule := sr.createBackRestSchedule(&cluster)
			schedules = append(schedules, schedule)
		case "pgbasebackup":
			schedule := sr.createBaseBackupSchedule(&cluster)
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
		blob, err := json.Marshal(schedule)
		if err != nil {
			sr.Response.Status.Code = msgs.Error
			sr.Response.Status.Msg = err.Error()
		}

		_, exists := kubeapi.GetConfigMap(apiserver.Clientset, schedule.Name, schedule.Namespace)
		if exists {
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
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   schedule.Name,
				Labels: labels,
			},
			Data: data,
		}

		err = kubeapi.CreateConfigMap(apiserver.Clientset, configmap, schedule.Namespace)
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
func DeleteSchedule(request *msgs.DeleteScheduleRequest) msgs.DeleteScheduleResponse {
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
		sr.Status.Msg = fmt.Sprintf("Cluster name, schedule name or selector must be provided")
		return *sr
	}

	schedules := []string{}
	var err error
	if request.ScheduleName != "" {
		schedules = append(schedules, request.ScheduleName)
	} else {
		schedules, err = getSchedules(request.ClusterName, request.Selector)
		if err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = err.Error()
			return *sr
		}
	}

	log.Debug("Deleting configMaps")
	for _, schedule := range schedules {
		err := kubeapi.DeleteConfigMap(apiserver.Clientset, schedule, apiserver.Namespace)
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
func ShowSchedule(request *msgs.ShowScheduleRequest) msgs.ShowScheduleResponse {
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
		sr.Status.Msg = fmt.Sprintf("Cluster name, schedule name or selector must be provided")
		return *sr
	}

	schedules := []string{}
	var err error
	if request.ScheduleName != "" {
		schedules = append(schedules, request.ScheduleName)
	} else {
		schedules, err = getSchedules(request.ClusterName, request.Selector)
		if err != nil {
			sr.Status.Code = msgs.Error
			sr.Status.Msg = err.Error()
			return *sr
		}
	}

	log.Debug("Parsing configMaps")
	for _, schedule := range schedules {
		cm, exists := kubeapi.GetConfigMap(apiserver.Clientset, schedule, apiserver.Namespace)
		if !exists {
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
		} else if blob.Type == "pgbasebackup" {
			results += fmt.Sprintf("\n\tbackup-host: %s", blob.PGBaseBackup.BackupHost)
			results += fmt.Sprintf("\n\tbackup-volume: %s", blob.PGBaseBackup.BackupVolume)
		}
		sr.Results = append(sr.Results, results)
	}
	return *sr
}

func getSchedules(clusterName, selector string) ([]string, error) {
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
	list, ok := kubeapi.ListConfigMap(apiserver.Clientset, label, apiserver.Namespace)
	if !ok {
		return nil, fmt.Errorf("No schedules found for selector: %s", label)
	}

	for _, cm := range list.Items {
		schedules = append(schedules, cm.Name)
	}

	return schedules, nil
}
