// +build envtest

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

package postgrescluster

import (
	"context"
	"fmt"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAddPGBackRestToInstancePodSpec(t *testing.T) {

	clusterName := "hippo"
	clusterUID := types.UID("hippouid")
	namespace := "test-add-pgbackrest-to-instance-pod-spec"
	pgBackRestImage := "hippo-image"

	// create a PostgresCluster to test with
	postgresCluster := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       clusterUID,
		},
		Spec: v1alpha1.PostgresClusterSpec{
			Archive: v1alpha1.Archive{
				PGBackRest: v1alpha1.PGBackRestArchive{
					Repos: []v1alpha1.RepoVolume{{
						Name: "repo1",
						VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
							AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					}, {
						Name: "repo2",
						VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
							AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceStorage: resource.MustParse("2Gi"),
								},
							},
						},
					}},
				},
			},
		},
	}

	instance := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hippo-instance-abc",
			Namespace: namespace,
		},
	}

	testCases := []struct {
		repoHost  *v1alpha1.RepoHost
		sshConfig *v1.ConfigMapProjection
		sshSecret *v1.SecretProjection
	}{{
		repoHost: nil,
	}, {
		repoHost: &v1alpha1.RepoHost{
			Image: pgBackRestImage,
		},
	}, {
		repoHost: &v1alpha1.RepoHost{
			Dedicated: &v1alpha1.DedicatedRepo{
				Resources: &v1.ResourceRequirements{},
			},
			Image: pgBackRestImage,
		},
	}, {
		repoHost: nil,
		sshConfig: &v1.ConfigMapProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-config.conf"}},
		sshSecret: &v1.SecretProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-secret.conf"}},
	}, {
		repoHost: &v1alpha1.RepoHost{
			Image: pgBackRestImage,
		},
		sshConfig: &v1.ConfigMapProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-config.conf"}},
		sshSecret: &v1.SecretProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-secret.conf"}},
	}, {
		repoHost: &v1alpha1.RepoHost{
			Dedicated: &v1alpha1.DedicatedRepo{
				Resources: &v1.ResourceRequirements{},
			},
			Image: pgBackRestImage,
		},
		sshConfig: &v1.ConfigMapProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-config.conf"}},
		sshSecret: &v1.SecretProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-secret.conf"}},
	}}

	for _, tc := range testCases {
		repoHost := (tc.repoHost != nil)
		dedicated := (tc.repoHost != nil && tc.repoHost.Dedicated != nil)
		customConfig := (tc.sshConfig != nil)
		customSecret := (tc.sshSecret != nil)
		t.Run(fmt.Sprintf("repoHost:%t, dedicated:%t", repoHost, dedicated), func(t *testing.T) {

			postgresCluster.Spec.Archive.PGBackRest.RepoHost = tc.repoHost
			template := &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: naming.ContainerDatabase}},
				},
			}

			if repoHost {
				if customConfig {
					postgresCluster.Spec.Archive.PGBackRest.RepoHost.SSHConfiguration = tc.sshConfig
				}
				if customSecret {
					postgresCluster.Spec.Archive.PGBackRest.RepoHost.SSHSecret = tc.sshSecret
				}
			}

			err := addPGBackRestToInstancePodSpec(postgresCluster, template, instance)
			assert.NilError(t, err)

			// if there is no dedicated repo host configured, verfiy pgBackRest repos are mounted to the
			// instance Pod
			if !dedicated {
				for _, r := range postgresCluster.Spec.Archive.PGBackRest.Repos {
					var foundRepo bool
					for _, v := range template.Spec.Volumes {
						if r.Name == v.Name {
							foundRepo = true
							break
						}
					}
					assert.Assert(t, foundRepo)

					for _, c := range template.Spec.Containers {
						if c.Name == naming.ContainerDatabase ||
							c.Name == naming.PGBackRestRepoContainerName {
							var foundRepoVolMount bool
							for _, vm := range c.VolumeMounts {
								if vm.Name == r.Name {
									foundRepoVolMount = true
									break
								}
							}
							assert.Assert(t, foundRepoVolMount)
						}
					}
				}
			}

			// if a repo host is configured, then verify SSH is enabled
			if repoHost {

				// verify the ssh volume
				var foundSSHVolume bool
				var sshVolume v1.Volume
				for _, v := range template.Spec.Volumes {
					if v.Name == naming.PGBackRestSSHVolume {
						foundSSHVolume = true
						sshVolume = v
						break
					}
				}
				assert.Assert(t, foundSSHVolume)

				// verify the ssh config and secret
				var foundSSHConfigVolume, foundSSHSecretVolume bool
				defaultConfigName := naming.PGBackRestSSHConfig(postgresCluster).Name
				defaultSecretName := naming.PGBackRestSSHSecret(postgresCluster).Name
				for _, s := range sshVolume.Projected.Sources {
					if s.ConfigMap != nil {
						if (!customConfig && s.ConfigMap.Name == defaultConfigName) ||
							(customConfig && s.ConfigMap.Name == tc.sshConfig.Name) {
							foundSSHConfigVolume = true
						}
					} else if s.Secret != nil {
						if (!customSecret && s.Secret.Name == defaultSecretName) ||
							(customSecret && s.Secret.Name == tc.sshSecret.Name) {
							foundSSHSecretVolume = true
						}
					}
				}
				assert.Assert(t, foundSSHConfigVolume)
				assert.Assert(t, foundSSHSecretVolume)

				// verify that pgbackrest container is present and that the proper SSH volume mount in
				// present in all containers
				var foundSSHContainer bool
				for _, c := range template.Spec.Containers {
					if c.Name == naming.PGBackRestRepoContainerName {
						foundSSHContainer = true
					}
					var foundVolumeMount bool
					for _, vm := range c.VolumeMounts {
						if vm.Name == naming.PGBackRestSSHVolume && vm.MountPath == "/etc/ssh" &&
							vm.ReadOnly == true {
							foundVolumeMount = true
							break
						}
					}
					assert.Assert(t, foundVolumeMount)
				}
				assert.Assert(t, foundSSHContainer)
			}
		})
	}
}

func TestReconcilePGDATAVolume(t *testing.T) {
	ctx := context.Background()

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, _ := setupTestEnv(t, ControllerName)
	r := &Reconciler{
		Client: tClient,
		Tracer: otel.Tracer(ControllerName),
		Owner:  ControllerName,
	}

	t.Cleanup(func() {
		teardownTestEnv(t, tEnv)
	})

	storageClassName := "storage-class1"
	apiGroup := "snapshot.storage.k8s.io"
	testCases := []v1alpha1.PostgresInstanceSetSpec{{
		Name: "instance1",
		VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: new(string),
		},
	}, {
		Name: "instance2",
		VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: resource.MustParse("256Mi"),
				},
			},
			StorageClassName: &storageClassName,
		},
	}, {
		Name: "instance3",
		VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: resource.MustParse("256Mi"),
				},
			},
			StorageClassName: &storageClassName,
			DataSource: &v1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VolumeSnapshot",
				Name:     "pgdata-snap1",
			},
		},
	}}

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })
	namespace := ns.Name

	clusterName := "hippo"
	clusterUID := types.UID("hippouid")
	postgresCluster := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: namespace, UID: clusterUID},
	}

	for _, tc := range testCases {

		t.Run(tc.Name, func(t *testing.T) {

			instance := &appsv1.StatefulSet{ObjectMeta: naming.GenerateInstance(postgresCluster, &tc)}

			err := r.reconcilePGDATAVolume(ctx, postgresCluster, &tc, instance)
			assert.NilError(t, err)

			instancePVC := &v1.PersistentVolumeClaim{}
			err = tClient.Get(ctx, client.ObjectKey{
				Name:      naming.InstancePGDataVolume(instance).Name,
				Namespace: namespace,
			}, instancePVC)
			assert.NilError(t, err)

			assert.DeepEqual(t, tc.VolumeClaimSpec.AccessModes, instancePVC.Spec.AccessModes)
			assert.DeepEqual(t, tc.VolumeClaimSpec.DataSource, instancePVC.Spec.DataSource)
			assert.DeepEqual(t, tc.VolumeClaimSpec.Resources, instancePVC.Spec.Resources)
			assert.DeepEqual(t, tc.VolumeClaimSpec.Selector, instancePVC.Spec.Selector)
			assert.DeepEqual(t, tc.VolumeClaimSpec.StorageClassName, instancePVC.Spec.StorageClassName)
			if tc.VolumeClaimSpec.VolumeMode != nil {
				assert.DeepEqual(t, tc.VolumeClaimSpec.VolumeMode, instancePVC.Spec.VolumeMode)
			}
		})
	}

}
