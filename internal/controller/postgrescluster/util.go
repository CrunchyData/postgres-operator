package postgrescluster

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

var tmpDirSizeLimit = resource.MustParse("16Mi")

// addTMPEmptyDir adds a "tmp" EmptyDir volume to the provided Pod template, while then also adding a
// volume mount at /tmp for all containers defined within the Pod template
// The '/tmp' directory is currently utilized for the following:
//  * A temporary location for instance PGDATA volumes until real volumes are implemented
//  * The location of the SSHD pid file
//  * As the pgBackRest lock directory (this is the default lock location for pgBackRest)
func addTMPEmptyDir(template *v1.PodTemplateSpec) {

	template.Spec.Volumes = append(template.Spec.Volumes, v1.Volume{
		Name: "tmp",
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{
				SizeLimit: &tmpDirSizeLimit,
			},
		},
	})

	for i := range template.Spec.Containers {
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts,
			v1.VolumeMount{
				Name:      "tmp",
				MountPath: "/tmp",
			})
	}
}

// setStatusConditions updates the provided PostgresCluster status with the provided
// conditions. This includes populating the PostgresCluster status with any conditions that
// do not yet exist, or updating them as needed if they already exist.
func setStatusConditions(status *v1alpha1.PostgresClusterStatus,
	conditions ...metav1.Condition) {
	for i := range conditions {
		// please note the observedGeneration will not be properly set via SetStatusCondition()
		// until apimachinery is updated v0.20
		meta.SetStatusCondition(&status.Conditions, conditions[i])
	}
}

// updateReconcileResult creates a new Result based on the new and existing results provided to it.
// This includes setting "Requeue" to true in the Result if set to true in the new Result but not
// in the existing Result, while also updating RequeueAfter if the RequeueAfter value for the new
// result is less the the RequeueAfter value for the existing Result.
func updateReconcileResult(currResult, newResult reconcile.Result) reconcile.Result {

	if newResult.Requeue {
		currResult.Requeue = true
	}
	if newResult.RequeueAfter < currResult.RequeueAfter {
		currResult.RequeueAfter = newResult.RequeueAfter
	}

	return currResult
}
