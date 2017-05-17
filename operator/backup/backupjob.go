/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package backup

import (
	log "github.com/Sirupsen/logrus"
	"os"

	//"github.com/crunchydata/postgres-operator/operator/database"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	//"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	//"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/cache"
)

func ProcessJobs(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, stopchan chan struct{}, namespace string) {

	lo := v1.ListOptions{LabelSelector: "pgbackup=true"}
	fw, err := clientset.Batch().Jobs(namespace).Watch(lo)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infoln("got a pgupgrade job watch event")

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
				//update the backup TPR status to completed
				err = util.Patch(tprclient, "/spec/backupstatus", tpr.UPGRADE_COMPLETED_STATUS, "pgbackups", dbname, namespace)
				if err != nil {
					log.Error(err.Error())
				}

			}
		default:
			log.Infoln("unknown watch event %v\n", event.Type)
		}

		return false, nil
	})

	if err4 != nil {
		log.Error(err4.Error())
	}

}
