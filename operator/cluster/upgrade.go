package cluster

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	v1batch "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

// AddUpgrade creates a pgupgrade job
func AddUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgupgrade, namespace string) {
	cl := crv1.Pgcluster{}

	//not a db so get the pgcluster CRD
	_, err := kubeapi.Getpgcluster(restclient, &cl,
		upgrade.Spec.Name, namespace)
	if err != nil {
		return
	}

	err = AddUpgradeBase(clientset, restclient, upgrade, namespace, &cl)
	if err != nil {
		log.Error("error adding upgrade" + err.Error())
	} else {
		//update the upgrade CRD status to submitted
		err = util.Patch(restclient, "/spec/upgradestatus", crv1.UpgradeSubmittedStatus, "pgupgrades", upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error("error patching upgrade" + err.Error())
		}
	}

}

// DeleteUpgrade deletes a pgupgrade job
func DeleteUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, upgrade *crv1.Pgupgrade, namespace string) {
	var jobName = "upgrade-" + upgrade.Spec.Name
	log.Debugf("deleting Job with Name= %s in namespace %s", jobName, namespace)

	//delete the job
	kubeapi.DeleteJob(clientset, jobName, namespace)
}

// MajorUpgradeProcess process major upgrade completions
// this watcher will look for completed upgrade jobs
// and when this occurs, will update the upgrade CRD status to
// completed and spin up the database or cluster using the newly
// upgraded data files
func MajorUpgradeProcess(clientset *kubernetes.Clientset, restclient *rest.RESTClient, namespace string) {

	log.Info("MajorUpgradeProcess watch starting...")

	lo := meta_v1.ListOptions{LabelSelector: "pgupgrade=true"}
	fw, err := clientset.Batch().Jobs(namespace).Watch(lo)
	if err != nil {
		log.Error("error watching upgrade job" + err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infoln("got a pgupgrade job watch event")

		switch event.Type {
		case watch.Added:
			gotjob := event.Object.(*v1batch.Job)
			log.Infof("pgupgrade job added=%d\n", gotjob.Status.Succeeded)
		case watch.Deleted:
			gotjob := event.Object.(*v1batch.Job)
			log.Infof("pgupgrade job deleted=%d\n", gotjob.Status.Succeeded)
		case watch.Error:
			log.Infof("pgupgrade job watch error event")
		case watch.Modified:
			gotjob := event.Object.(*v1batch.Job)
			log.Infof("pgupgrade job modified=%d\n", gotjob.Status.Succeeded)
			if gotjob.Status.Succeeded == 1 {
				log.Infoln("pgupgrade job " + gotjob.Name + " succeeded")
				finishUpgrade(clientset, restclient, gotjob, namespace)

			}
		default:
			log.Infoln("unknown watch event %v\n", event.Type)
		}

		return false, nil
	})

	if err4 != nil {
		log.Error("error in major upgrade " + err4.Error())
	}

}

// finishUpgrade performs the final part of a major upgrade
func finishUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, job *v1batch.Job, namespace string) {

	var cl crv1.Pgcluster
	var upgrade crv1.Pgupgrade

	//from the job get the db and upgrade CRDs
	//pgdatabase name is from the pg-database label value in the job
	// it represents the cluster name or the database name
	//pgupgrade name is from the pg-database label value in the job
	name := job.ObjectMeta.Labels["pg-database"]
	if name == "" {
		log.Error("name was empty in the pg-database label for the upgrade job")
		return
	}

	_, err := kubeapi.Getpgupgrade(restclient, &upgrade, name, namespace)
	if err != nil {
		return
	}
	log.Debugf("%s pgupgrade crv1 is found", name)

	_, err = kubeapi.Getpgcluster(restclient, &cl, name, namespace)
	if err != nil {
		return
	}
	log.Debugf("pgcluster crv1 is found", name)

	var clusterStrategy Strategy

	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default strategy")
	}

	clusterStrategy, ok := strategyMap[cl.Spec.Strategy]

	if ok {
		log.Info("strategy found")

	} else {
		log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
		return
	}

	err = clusterStrategy.MajorUpgradeFinalize(clientset, restclient, &cl, &upgrade, namespace)
	if err != nil {
		log.Error("error in major upgrade finalize" + err.Error())
	}

	if err == nil {
		//update the upgrade CRD status to completed
		err = util.Patch(restclient, "/spec/upgradestatus", crv1.UpgradeCompletedStatus, "pgupgrades", upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error("error in patch upgrade " + err.Error())
		}

	}

}
