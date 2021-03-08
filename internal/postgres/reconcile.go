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

package postgres

import (
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

// AddPGDATAVolumeToPod adds pgBackRest repository volumes to the provided Pod template spec, while
// also adding associated volume mounts to the containers specified.
func AddPGDATAVolumeToPod(postgresCluster *v1alpha1.PostgresCluster, template *v1.PodTemplateSpec,
	claimName string, containerNames ...string) error {

	if claimName == "" {
		return errors.WithStack(errors.New("claimName must not be empty"))
	}

	pgdataVolume := v1.Volume{
		Name: naming.PGDATAVolume,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	}
	template.Spec.Volumes = append(template.Spec.Volumes, pgdataVolume)

	for _, name := range containerNames {
		var containerFound bool
		var index int
		for index = range template.Spec.Containers {
			if template.Spec.Containers[index].Name == name {
				containerFound = true
				break
			}
		}
		if !containerFound {
			return fmt.Errorf("Unable to find container %q when adding pgBackRest repo volumes",
				name)
		}
		template.Spec.Containers[index].VolumeMounts =
			append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
				Name:      naming.PGDATAVolume,
				MountPath: naming.PGDATAVMountPath,
			})
	}

	return nil
}
