// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"errors"
	"io"
	"testing"

	"gotest.tools/v3/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestSafeHash32(t *testing.T) {
	expected := errors.New("whomp")

	_, err := safeHash32(func(io.Writer) error { return expected })
	assert.Equal(t, err, expected)

	stuff, err := safeHash32(func(w io.Writer) error {
		_, _ = w.Write([]byte(`some stuff`))
		return nil
	})
	assert.NilError(t, err)
	assert.Equal(t, stuff, "574b4c7d87", "expected alphanumeric")

	same, err := safeHash32(func(w io.Writer) error {
		_, _ = w.Write([]byte(`some stuff`))
		return nil
	})
	assert.NilError(t, err)
	assert.Equal(t, same, stuff, "expected deterministic hash")
}

func TestAddDevSHM(t *testing.T) {

	testCases := []struct {
		tcName      string
		podTemplate *corev1.PodTemplateSpec
		expected    bool
	}{{
		tcName: "database and pgbackrest containers",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "database"}, {Name: "pgbackrest"}, {Name: "dontmodify"},
			}}},
		expected: true,
	}, {
		tcName: "database container only",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "database"}, {Name: "dontmodify"}}}},
		expected: true,
	}, {
		tcName: "pgbackest container only",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "dontmodify"}, {Name: "pgbackrest"}}}},
	}, {
		tcName: "other containers",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "dontmodify1"}, {Name: "dontmodify2"}}}},
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			template := tc.podTemplate

			addDevSHM(template)

			found := false

			// check there is an empty dir mounted under the dshm volume
			for _, v := range template.Spec.Volumes {
				if v.Name == "dshm" && v.EmptyDir != nil && v.EmptyDir.Medium == corev1.StorageMediumMemory {
					found = true
					break
				}
			}
			assert.Assert(t, found)

			// check that the database container contains a mount to the shared volume
			// directory
			found = false

		loop:
			for _, c := range template.Spec.Containers {
				if c.Name == naming.ContainerDatabase {
					for _, vm := range c.VolumeMounts {
						if vm.Name == "dshm" && vm.MountPath == "/dev/shm" {
							found = true
							break loop
						}
					}
				}
			}

			assert.Equal(t, tc.expected, found)
		})
	}
}

func TestAddNSSWrapper(t *testing.T) {

	image := "test-image"
	imagePullPolicy := corev1.PullAlways

	expectedResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("200m"),
		}}

	expectedEnv := []corev1.EnvVar{
		{Name: "LD_PRELOAD", Value: "/usr/lib64/libnss_wrapper.so"},
		{Name: "NSS_WRAPPER_PASSWD", Value: "/tmp/nss_wrapper/postgres/passwd"},
		{Name: "NSS_WRAPPER_GROUP", Value: "/tmp/nss_wrapper/postgres/group"},
	}

	expectedPGAdminEnv := []corev1.EnvVar{
		{Name: "LD_PRELOAD", Value: "/usr/lib64/libnss_wrapper.so"},
		{Name: "NSS_WRAPPER_PASSWD", Value: "/tmp/nss_wrapper/pgadmin/passwd"},
		{Name: "NSS_WRAPPER_GROUP", Value: "/tmp/nss_wrapper/pgadmin/group"},
	}

	testCases := []struct {
		tcName                        string
		podTemplate                   *corev1.PodTemplateSpec
		pgadmin                       bool
		resourceProvider              string
		expectedUpdatedContainerCount int
	}{{
		tcName: "database container with pgbackrest sidecar",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: naming.ContainerDatabase, Resources: expectedResources},
				{Name: naming.PGBackRestRepoContainerName, Resources: expectedResources},
				{Name: "dontmodify"},
			}}},
		expectedUpdatedContainerCount: 2,
	}, {
		tcName: "database container only",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: naming.ContainerDatabase, Resources: expectedResources},
				{Name: "dontmodify"}}}},
		expectedUpdatedContainerCount: 1,
	}, {
		tcName: "pgbackest container only",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: naming.PGBackRestRepoContainerName, Resources: expectedResources},
				{Name: "dontmodify"},
			}}},
		expectedUpdatedContainerCount: 1,
	}, {
		tcName: "pgadmin container only",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "dontmodify"}, {Name: "pgadmin"}}}},
		pgadmin:                       true,
		expectedUpdatedContainerCount: 1,
	}, {
		tcName: "restore container only",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: naming.PGBackRestRestoreContainerName, Resources: expectedResources},
				{Name: "dontmodify"},
			}}},
		expectedUpdatedContainerCount: 1,
	}, {
		tcName: "custom database container resources",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "database",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("200m"),
						}}}}}},
		resourceProvider:              "database",
		expectedUpdatedContainerCount: 1,
	}, {
		tcName: "custom pgbackrest container resources",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "pgbackrest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("300m"),
						}}}}}},
		resourceProvider:              "pgbackrest",
		expectedUpdatedContainerCount: 1,
	}, {
		tcName: "custom pgadmin container resources",
		podTemplate: &corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "pgadmin",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("400m"),
						}}}}}},
		pgadmin:                       true,
		resourceProvider:              "pgadmin",
		expectedUpdatedContainerCount: 1,
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			template := tc.podTemplate
			beforeAddNSS := template.DeepCopy().Spec.Containers

			addNSSWrapper(image, imagePullPolicy, template)

			t.Run("container-updated", func(t *testing.T) {
				// Each container that requires the nss_wrapper envs should be updated
				var actualUpdatedContainerCount int
				for i, c := range template.Spec.Containers {
					switch c.Name {
					case naming.ContainerDatabase, naming.PGBackRestRepoContainerName, naming.PGBackRestRestoreContainerName:
						assert.DeepEqual(t, expectedEnv, c.Env)
						actualUpdatedContainerCount++
					case "pgadmin":
						assert.DeepEqual(t, expectedPGAdminEnv, c.Env)
						actualUpdatedContainerCount++
					default:
						assert.DeepEqual(t, beforeAddNSS[i], c)
					}
				}
				// verify database and/or pgbackrest containers updated
				assert.Equal(t, actualUpdatedContainerCount,
					tc.expectedUpdatedContainerCount)
			})

			t.Run("init-container-added", func(t *testing.T) {
				var foundInitContainer bool
				// verify init container command, image & name
				for _, ic := range template.Spec.InitContainers {
					if ic.Name == naming.ContainerNSSWrapperInit {
						if tc.pgadmin {
							assert.Equal(t, pgAdminNSSWrapperPrefix+nssWrapperScript, ic.Command[2]) // ignore "bash -c"
						} else {
							assert.Equal(t, postgresNSSWrapperPrefix+nssWrapperScript, ic.Command[2]) // ignore "bash -c"
						}
						assert.Assert(t, ic.Image == image)
						assert.Assert(t, ic.ImagePullPolicy == imagePullPolicy)
						assert.Assert(t, !cmp.DeepEqual(ic.SecurityContext,
							&corev1.SecurityContext{})().Success())

						if tc.resourceProvider != "" {
							for _, c := range template.Spec.Containers {
								if c.Name == tc.resourceProvider {
									assert.DeepEqual(t, ic.Resources.Requests,
										c.Resources.Requests)
								}
							}
						}
						foundInitContainer = true
						break
					}
				}
				// verify init container is present
				assert.Assert(t, foundInitContainer)
			})
		})
	}
}

func TestJobCompleted(t *testing.T) {

	testCases := []struct {
		job              *batchv1.Job
		expectSuccessful bool
		testDesc         string
	}{{
		job: &batchv1.Job{
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
				}},
			},
		},
		expectSuccessful: true,
		testDesc:         "condition present and true",
	}, {
		job: &batchv1.Job{
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionFalse,
				}},
			},
		},
		expectSuccessful: false,
		testDesc:         "condition present but false",
	}, {
		job: &batchv1.Job{
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
		expectSuccessful: false,
		testDesc:         "condition present but unknown",
	}, {
		job:              &batchv1.Job{},
		expectSuccessful: false,
		testDesc:         "empty conditions",
	}}

	for _, tc := range testCases {
		t.Run(tc.testDesc, func(t *testing.T) {
			// first ensure jobCompleted gives the expected result
			isCompleted := jobCompleted(tc.job)
			assert.Assert(t, isCompleted == tc.expectSuccessful)
		})
	}
}

func TestJobFailed(t *testing.T) {

	testCases := []struct {
		job          *batchv1.Job
		expectFailed bool
		testDesc     string
	}{{
		job: &batchv1.Job{
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobFailed,
					Status: corev1.ConditionTrue,
				}},
			},
		},
		expectFailed: true,
		testDesc:     "condition present and true",
	}, {
		job: &batchv1.Job{
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobFailed,
					Status: corev1.ConditionFalse,
				}},
			},
		},
		expectFailed: false,
		testDesc:     "condition present but false",
	}, {
		job: &batchv1.Job{
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobFailed,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
		expectFailed: false,
		testDesc:     "condition present but unknown",
	}, {
		job:          &batchv1.Job{},
		expectFailed: false,
		testDesc:     "empty conditions",
	}}

	for _, tc := range testCases {
		t.Run(tc.testDesc, func(t *testing.T) {
			// first ensure jobCompleted gives the expected result
			isCompleted := jobFailed(tc.job)
			assert.Assert(t, isCompleted == tc.expectFailed)
		})
	}
}

func TestAddAdditionalVolumesToSpecifiedContainers(t *testing.T) {

	podTemplate := &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "startup"},
				{Name: "config"},
			},
			Containers: []corev1.Container{
				{Name: "database"},
				{Name: "other"},
			}}}

	testCases := []struct {
		tcName                 string
		additionalVolumes      []v1beta1.AdditionalVolume
		expectedContainers     string
		expectedInitContainers string
		expectedVolumes        string
		expectedMissing        []string
	}{{
		tcName: "all",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			ClaimName: "required",
			Name:      "required",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: other
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: config
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{},
	}, {
		tcName: "multiple additional volumes",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			ClaimName: "required",
			Name:      "required",
		}, {
			ClaimName: "also",
			Name:      "other",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other
- name: other
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other
- name: config
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required
- name: volumes-other
  persistentVolumeClaim:
    claimName: also`,
		expectedMissing: []string{},
	}, {
		tcName: "none",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{},
			ClaimName:  "required",
			Name:       "required",
		}},
		expectedContainers: `- name: database
  resources: {}
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{},
	}, {
		tcName: "multiple additional volumes",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			ClaimName: "required",
			Name:      "required",
		}, {
			ClaimName: "also",
			Name:      "other",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other
- name: other
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other
- name: config
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required
- name: volumes-other
  persistentVolumeClaim:
    claimName: also`,
		expectedMissing: []string{},
	}, {
		tcName: "database and startup containers only",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{"database", "startup"},
			ClaimName:  "required",
			Name:       "required",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{},
	}, {
		tcName: "container is missing",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{"database", "startup", "missing", "container"},
			ClaimName:  "required",
			Name:       "required",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{"missing", "container"},
	}, {
		tcName: "readonly",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{"database"},
			ClaimName:  "required",
			Name:       "required",
			ReadOnly:   true,
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
    readOnly: true
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required
    readOnly: true`,
		expectedMissing: []string{},
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			copyPodTemplate := podTemplate.DeepCopy()

			missingContainers := AddAdditionalVolumesToSpecifiedContainers(
				copyPodTemplate,
				tc.additionalVolumes,
			)

			assert.Assert(t, cmp.MarshalMatches(
				copyPodTemplate.Spec.Containers,
				tc.expectedContainers))
			assert.Assert(t, cmp.MarshalMatches(
				copyPodTemplate.Spec.InitContainers,
				tc.expectedInitContainers))
			assert.Assert(t, cmp.MarshalMatches(
				copyPodTemplate.Spec.Volumes,
				tc.expectedVolumes))
			if len(tc.expectedMissing) == 0 {
				assert.Assert(t, cmp.DeepEqual(
					missingContainers,
					tc.expectedMissing))
			} else {
				for _, mc := range tc.expectedMissing {
					assert.Assert(t, cmp.Contains(
						missingContainers,
						mc))
				}
			}
		})
	}
}
