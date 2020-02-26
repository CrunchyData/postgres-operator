package job

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	benchmarkoperator "github.com/crunchydata/postgres-operator/operator/benchmark"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
)

// rmdataUpdateHandler is responsible for handling updates to benchmark jobs
func (c *Controller) handleBenchmarkUpdate(job *apiv1.Job) error {

	labels := job.GetObjectMeta().GetLabels()

	log.Debugf("jobController onUpdate benchmark job case")
	log.Debugf("got a benchmark job status=%d", job.Status.Succeeded)

	status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"
	if job.Status.Succeeded == 0 {
		status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
	}

	if job.Status.Failed > 0 {
		status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
	}

	err := util.Patch(c.JobClient, patchURL, status, patchResource, job.Name, job.ObjectMeta.Namespace)
	if err != nil {
		log.Error("error in patching pgtask " + labels["workflowName"] + err.Error())
	}

	benchmarkoperator.UpdateWorkflow(c.JobClient, labels["workflowName"], job.ObjectMeta.Namespace, crv1.JobCompletedStatus)

	//publish event benchmark completed
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventBenchmarkCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: job.ObjectMeta.Namespace,
			Username:  job.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventBenchmarkCompleted,
		},
		Clustername: labels[config.LABEL_PG_CLUSTER],
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

	return nil
}
