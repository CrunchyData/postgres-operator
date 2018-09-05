package kubeapi

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
	v1batch "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetJobs gets a list of jobs using a label selector
func GetJobs(clientset *kubernetes.Clientset, selector, namespace string) (*v1batch.JobList, error) {
	lo := meta_v1.ListOptions{LabelSelector: selector}

	jobs, err := clientset.Batch().Jobs(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting Job list selector[" + selector + "]")
	}
	return jobs, err

}

// GetJob gets a Job by name
func GetJob(clientset *kubernetes.Clientset, name, namespace string) (*v1batch.Job, bool) {
	job, err := clientset.Batch().Jobs(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		log.Error(err)
		return job, false
	}
	if err != nil {
		log.Error(err)
		return job, false
	}

	return job, true
}

// DeleteJob deletes a job
func DeleteJob(clientset *kubernetes.Clientset, jobName, namespace string) error {
	log.Debug("deleting Job with Name=" + jobName + " in namespace " + namespace)
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	//delete the job
	err := clientset.Batch().Jobs(namespace).Delete(jobName,
		&delOptions)
	if err != nil {
		log.Error("error deleting Job " + jobName + err.Error())
		return err
	}

	log.Info("deleted Job " + jobName)
	return err
}

// CreateJob deletes a backup job
func CreateJob(clientset *kubernetes.Clientset, job *v1batch.Job, namespace string) error {
	result, err := clientset.Batch().Jobs(namespace).Create(job)
	if err != nil {
		log.Error("error creating Job " + job.Name + err.Error())
		return err
	}

	log.Info("created Job " + result.Name)
	return err
}

// DeleteJobs deletes all jobs that match a selector
func DeleteJobs(clientset *kubernetes.Clientset, selector, namespace string) error {
	log.Debug("deleting Jobs with selector=" + selector + " in namespace " + namespace)

	//delete the job
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	lo := meta_v1.ListOptions{LabelSelector: selector}

	err := clientset.Batch().Jobs(namespace).DeleteCollection(&delOptions, lo)
	if err != nil {
		log.Error("error deleting Jobs " + selector + err.Error())
		return err
	}

	return err
}
