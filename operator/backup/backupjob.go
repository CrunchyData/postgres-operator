package backup

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
	log "github.com/Sirupsen/logrus"
	"os"

	v1batch "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ProcessJobs processes a backup job lifecycle
func ProcessJobs(clientset *kubernetes.Clientset, restclient *rest.RESTClient, namespace string) {

	lo := meta_v1.ListOptions{LabelSelector: "pgbackup=true"}
	fw, err := clientset.Batch().Jobs(namespace).Watch(lo)
	if err != nil {
		log.Error("fatal error in ProcessJobs " + err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infof("got a backup job watch event %v\n", event.Type)

		switch event.Type {
		case watch.Added:
			gotjob := event.Object.(*v1batch.Job)
			log.Infof("pgbackup job added=%d\n", gotjob.Status.Succeeded)
		case watch.Deleted:
			gotjob := event.Object.(*v1batch.Job)
			log.Infof("pgbackup job deleted=%d\n", gotjob.Status.Succeeded)
		case watch.Error:
			log.Infof("pgbackup job watch error event")
		case watch.Modified:
			gotjob := event.Object.(*v1batch.Job)
			log.Infof("pgbackup job modified=%d\n", gotjob.Status.Succeeded)
			if gotjob.Status.Succeeded == 1 {
				dbname := gotjob.ObjectMeta.Labels["pg-database"]
				log.Infoln("pgbackup job " + gotjob.Name + " succeeded" + " marking " + dbname + " completed")
			}
		default:
			log.Infoln("backup job unknown watch event %v\n", event.Type)
		}

		return false, nil
	})

	if err4 != nil {
		log.Error("error in ProcessJobs " + err4.Error())
	}

}
