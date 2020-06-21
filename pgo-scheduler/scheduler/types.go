package scheduler

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	cv2 "github.com/robfig/cron/v3"
)

var kubeClient *kubernetes.Clientset
var restClient *rest.RESTClient

type Scheduler struct {
	entries       map[string]cv2.EntryID
	CronClient    *cv2.Cron
	label         string
	namespace     string
	namespaceList []string
	scheduleTypes []string
}

type ScheduleTemplate struct {
	Version    string    `json:"version"`
	Name       string    `json:"name"`
	Created    time.Time `json:"created"`
	Schedule   string    `json:"schedule"`
	Namespace  string    `json:"namespace"`
	Type       string    `json:"type"`
	Cluster    string    `json:"cluster"`
	PGBackRest `json:"pgbackrest,omitempty"`
	Policy     `json:"policy,omitempty"`
}

type PGBackRest struct {
	Deployment  string `json:"deployment"`
	Label       string `json:"label"`
	Container   string `json:"container"`
	Type        string `json:"type"`
	StorageType string `json:"storageType,omitempty"`
	Options     string `json:"options"`
}

type Policy struct {
	Secret      string `json:"secret"`
	Name        string `json:"name"`
	ImagePrefix string `json:"imagePrefix"`
	ImageTag    string `json:"imageTag"`
	Database    string `json:"database"`
}

type PolicyTemplate struct {
	JobName        string
	ClusterName    string
	PGOImagePrefix string
	PGOImageTag    string
	PGHost         string
	PGPort         string
	PGDatabase     string
	PGUserSecret   string
	PGSQLConfigMap string
}
