package job

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

import (
	apiv1 "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// isBackoffLimitExceeded returns true if the jobs backoff limit has been exceeded
func isBackoffLimitExceeded(job *apiv1.Job) bool {
	if job.Spec.BackoffLimit != nil {
		return job.Status.Failed >= *job.Spec.BackoffLimit
	}
	return false
}

// isJobSuccessful returns true if the job provided completed successfully.  Otherwise
// it returns false.  Per the Kubernetes documentation, "the completion time is only set
// when the job finishes successfully".  Therefore, the presence of a completion time can
// be utilized to determine whether or not the job was successful.
func isJobSuccessful(job *apiv1.Job) bool {
	return job.Status.CompletionTime != nil
}

// isJobInForegroundDeletion determines if a job is currently being deleted using foreground
// cascading deletion, as indicated by the presence of value “foregroundDeletion” in the jobs
// metadata.finalizers.
func isJobInForegroundDeletion(job *apiv1.Job) bool {
	for _, finalizer := range job.Finalizers {
		if finalizer == meta_v1.FinalizerDeleteDependents {
			return true
		}
	}
	return false
}
