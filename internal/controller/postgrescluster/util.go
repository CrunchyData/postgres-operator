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
	"fmt"
	"hash/fnv"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var tmpDirSizeLimit = resource.MustParse("16Mi")

const (
	// nssWrapperDir is the directory in a container for the nss_wrapper passwd and group files
	nssWrapperDir = "/tmp/nss_wrapper/%s/%s"
	// uidCommand is the command for setting up nss_wrapper in the container
	nssWrapperCmd = `NSS_WRAPPER_SUBDIR=postgres CRUNCHY_NSS_USERNAME=postgres ` +
		`CRUNCHY_NSS_USER_DESC="postgres" /opt/crunchy/bin/nss_wrapper.sh`
)

// addTMPEmptyDir adds a "tmp" EmptyDir volume to the provided Pod template, while then also adding a
// volume mount at /tmp for all containers defined within the Pod template
// The '/tmp' directory is currently utilized for the following:
//  * A temporary location for instance PGDATA volumes until real volumes are implemented
//  * The location of the SSHD pid file
//  * As the pgBackRest lock directory (this is the default lock location for pgBackRest)
//  * The location where the replication client certificates can be loaded with the proper
//    permissions set
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

	for i := range template.Spec.InitContainers {
		template.Spec.InitContainers[i].VolumeMounts = append(template.Spec.InitContainers[i].VolumeMounts,
			v1.VolumeMount{
				Name:      "tmp",
				MountPath: "/tmp",
			})
	}
}

// addNSSWrapper adds nss_wrapper environment variables to the database and pgBackRest
// containers in the Pod template.  Additionally, an init container is added to the Pod template
// as needed to setup the nss_wrapper. Please note that the nss_wrapper is required for
// compatibility with OpenShift: https://access.redhat.com/articles/4859371.
func addNSSWrapper(postgresCluster *v1beta1.PostgresCluster, template *v1.PodTemplateSpec) {

	for i, c := range template.Spec.Containers {
		switch c.Name {
		case naming.ContainerDatabase, naming.PGBackRestRepoContainerName:
			passwd := fmt.Sprintf(nssWrapperDir, "postgres", "passwd")
			group := fmt.Sprintf(nssWrapperDir, "postgres", "group")
			template.Spec.Containers[i].Env = append(template.Spec.Containers[i].Env, []v1.EnvVar{
				{Name: "LD_PRELOAD", Value: "/usr/lib64/libnss_wrapper.so"},
				{Name: "NSS_WRAPPER_PASSWD", Value: passwd},
				{Name: "NSS_WRAPPER_GROUP", Value: group},
			}...)
		}
	}

	template.Spec.InitContainers = append(template.Spec.InitContainers,
		v1.Container{
			Command: []string{"bash", "-c", nssWrapperCmd},
			Image:   postgresCluster.Spec.Image,
			Name:    naming.ContainerNSSWrapperInit,
		})
}

// safeHash32 runs content and returns a short alphanumeric string that
// represents everything written to w. The string is unlikely to have bad words
// and is safe to store in the Kubernetes API. This is the same algorithm used
// by ControllerRevision's "controller.kubernetes.io/hash".
func safeHash32(content func(w io.Writer) error) (string, error) {
	hash := fnv.New32()
	if err := content(hash); err != nil {
		return "", err
	}
	return rand.SafeEncodeString(fmt.Sprint(hash.Sum32())), nil
}

// setStatusConditions updates the provided PostgresCluster status with the provided
// conditions. This includes populating the PostgresCluster status with any conditions that
// do not yet exist, or updating them as needed if they already exist.
func setStatusConditions(status *v1beta1.PostgresClusterStatus,
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

	if newResult.RequeueAfter != 0 {
		if currResult.RequeueAfter == 0 || newResult.RequeueAfter < currResult.RequeueAfter {
			currResult.RequeueAfter = newResult.RequeueAfter
		}
	}

	return currResult
}
