package util

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CloneParameterBackrestPVCSize is the parameter name for the Backrest PVC
	// size parameter
	CloneParameterBackrestPVCSize = "backrestPVCSize"
	// CloneParameterEnableMetrics if set to true, enables metrics collection in
	// a newly created cluster
	CloneParameterEnableMetrics = "enableMetrics"
	// CloneParameterPVCSize is the parameter name for the PVC parameter for
	// primary and replicas
	CloneParameterPVCSize = "pvcSize"
)

// CloneTask allows you to create a Pgtask CRD with the appropriate options
type CloneTask struct {
	BackrestPVCSize       string
	BackrestStorageSource string
	EnableMetrics         bool
	PGOUser               string
	PVCSize               string
	SourceClusterName     string
	TargetClusterName     string
	TaskStepLabel         string
	TaskType              string
	Timestamp             time.Time
	WorkflowID            string
}

// newCloneTask returns a new instance of a Pgtask CRD
func (clone CloneTask) Create() *crv1.Pgtask {
	// get the one-time gneerated task name
	taskName := clone.taskName()

	// sigh...set a "boolean" for enabling metrics
	enableMetrics := "false"
	if clone.EnableMetrics {
		enableMetrics = "true"
	}

	return &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: clone.TargetClusterName,
				config.LABEL_PGOUSER:    clone.PGOUser,
				config.LABEL_PGO_CLONE:  "true",
				clone.TaskStepLabel:     "true",
			},
		},
		Spec: crv1.PgtaskSpec{
			Name:     taskName,
			TaskType: clone.TaskType,
			Parameters: map[string]string{
				CloneParameterBackrestPVCSize: clone.BackrestPVCSize,
				"backrestStorageType":         clone.BackrestStorageSource,
				CloneParameterEnableMetrics:   enableMetrics,
				CloneParameterPVCSize:         clone.PVCSize,
				"sourceClusterName":           clone.SourceClusterName,
				"targetClusterName":           clone.TargetClusterName,
				"taskName":                    taskName,
				"timestamp":                   clone.Timestamp.Format(time.RFC3339),
				crv1.PgtaskWorkflowID:         clone.WorkflowID,
			},
		},
	}
}

// taskName generates the task name, which uses the "TaskType" and
// "TargetClusterName" properties, with a little bit of entropy
func (clone CloneTask) taskName() string {
	// create a task name based on the step we are on in the process, with some
	// entropy
	uid := RandStringBytesRmndr(4)
	return fmt.Sprintf("%s-%s-%s", clone.TaskType, clone.TargetClusterName, uid)
}
