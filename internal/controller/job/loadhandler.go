package job

import (
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
)

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

// handleLoadUpdate is responsible for handling updates to load jobs
func (c *Controller) handleLoadUpdate(job *apiv1.Job) error {
	log.Debugf("jobController onUpdate load job case")
	log.Debugf("got a load job status=%d", job.Status.Succeeded)

	if isJobSuccessful(job) {
		log.Debugf("load job succeeded=%d", job.Status.Succeeded)
	}
	return nil
}
