package job

/*
Copyright 2019 - 2022 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
)

func publishBackupComplete(clusterName, clusterIdentifier, username, backuptype, namespace, path string) {
	topics := make([]string, 2)
	topics[0] = events.EventTopicCluster
	topics[1] = events.EventTopicBackup

	f := events.EventCreateBackupCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreateBackupCompleted,
		},
		Clustername: clusterName,
		BackupType:  backuptype,
		Path:        path,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}

func publishRestoreComplete(clusterName, identifier, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventRestoreClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventRestoreClusterCompleted,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}

func publishDeleteClusterComplete(clusterName, identifier, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventDeleteClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeleteClusterCompleted,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}
