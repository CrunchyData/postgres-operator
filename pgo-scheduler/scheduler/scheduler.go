package scheduler

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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/crunchydata/postgres-operator/apiserver"
	log "github.com/sirupsen/logrus"

	cv2 "gopkg.in/robfig/cron.v2"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func New(label, namespace string, nsList []string, client *kubernetes.Clientset) *Scheduler {
	apiserver.ConnectToKube()
	restClient = apiserver.RESTClient
	kubeClient = client
	cronClient := cv2.New()
	var p phony
	cronClient.AddJob("* * * * *", p)

	return &Scheduler{
		namespace:     namespace,
		label:         label,
		CronClient:    cronClient,
		entries:       make(map[string]cv2.EntryID),
		namespaceList: nsList,
	}
}

func (s *Scheduler) AddSchedule(config *v1.ConfigMap) error {
	name := config.Name + config.Namespace
	if _, ok := s.entries[name]; ok {
		return nil
	}

	if len(config.Data) != 1 {
		return errors.New("Schedule configmaps should contain only one schedule")
	}

	var schedule ScheduleTemplate
	for _, data := range config.Data {
		if err := json.Unmarshal([]byte(data), &schedule); err != nil {
			return fmt.Errorf("Failed unmarhsaling configMap: %s", err)
		}
	}

	if err := validate(schedule); err != nil {
		return fmt.Errorf("Failed to validate schedule: %s", err)
	}

	id, err := s.schedule(schedule)
	if err != nil {
		return fmt.Errorf("Failed to schedule configmap: %s", err)
	}

	log.WithFields(log.Fields{
		"configMap":  string(config.Name),
		"type":       schedule.Type,
		"schedule":   schedule.Schedule,
		"namespace":  schedule.Namespace,
		"deployment": schedule.Deployment,
		"label":      schedule.Label,
		"container":  schedule.Container,
	}).Info("Added new schedule")

	s.entries[name] = id
	return nil
}

func (s *Scheduler) DeleteSchedule(config *v1.ConfigMap) {
	log.WithFields(log.Fields{
		"scheduleName": config.Name,
	}).Info("Removed schedule")

	name := config.Name + config.Namespace
	s.CronClient.Remove(s.entries[name])
	delete(s.entries, name)
}

func (s *Scheduler) schedule(st ScheduleTemplate) (cv2.EntryID, error) {
	var job cv2.Job

	switch st.Type {
	case "pgbackrest":
		job = st.NewBackRestSchedule()
	case "pgbasebackup":
		job = st.NewBaseBackupSchedule()
	case "policy":
		job = st.NewPolicySchedule()
	default:
		var id cv2.EntryID
		return id, fmt.Errorf("schedule type not implemented yet")
	}
	return s.CronClient.AddJob(st.Schedule, job)
}

type phony string

func (p phony) Run() {
	// This is a phony job that register with the cron service
	// that does nothing to prevent a bug that runs newly scheduled
	// jobs multiple times.
	_ = time.Now()
}
