package scheduler

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	cv2 "gopkg.in/robfig/cron.v2"
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
	Version      string    `json:"version"`
	Name         string    `json:"name"`
	Created      time.Time `json:"created"`
	Schedule     string    `json:"schedule"`
	Namespace    string    `json:"namespace"`
	Type         string    `json:"type"`
	Cluster      string    `json:"cluster"`
	PGBackRest   `json:"pgbackrest,omitempty"`
	PGBaseBackup `json:"pgbasebackup,omitempty"`
	Policy       `json:"policy,omitempty"`
}

type PGBaseBackup struct {
	Port         string `json:"backupPort"`
	Secret       string `json:"secret"`
	BackupVolume string `json:"backupVolume"`
	ImagePrefix  string `json:"imagePrefix"`
	ImageTag     string `json:"imageTag"`
}

type PGBackRest struct {
	Deployment string    `json:"deployment"`
	Label      string    `json:"label"`
	Container  string    `json:"container"`
	Type       string    `json:"type"`
	Options    []Options `json:"options"`
}

type Policy struct {
	Secret      string `json:"secret"`
	Name        string `json:"name"`
	ImagePrefix string `json:"imagePrefix"`
	ImageTag    string `json:"imageTag"`
	Database    string `json:"database"`
}

type Options struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type SecurityContext struct {
	FSGroup            int   `json:"fsGroup,omitempty"`
	SupplementalGroups []int `json:"supplementalGroups,omitempty"`
}

type PolicyTemplate struct {
	JobName        string
	ClusterName    string
	COImagePrefix  string
	COImageTag     string
	PGHost         string
	PGPort         string
	PGDatabase     string
	PGUserSecret   string
	PGSQLConfigMap string
}
