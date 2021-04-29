package scheduler

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type PolicyJob struct {
	ccpImageTag    string
	ccpImagePrefix string
	cluster        string
	namespace      string
	secret         string
	policy         string
	database       string
}

func (s *ScheduleTemplate) NewPolicySchedule() PolicyJob {
	return PolicyJob{
		namespace:      s.Namespace,
		cluster:        s.Cluster,
		ccpImageTag:    s.Policy.ImageTag,
		ccpImagePrefix: s.Policy.ImagePrefix,
		secret:         s.Policy.Secret,
		policy:         s.Policy.Name,
		database:       s.Policy.Database,
	}
}

func (p PolicyJob) Run() {
	ctx := context.TODO()
	contextLogger := log.WithFields(log.Fields{
		"namespace": p.namespace,
		"policy":    p.policy,
		"cluster":   p.cluster,
	})

	contextLogger.Info("Running Policy schedule")

	cluster, err := clientset.CrunchydataV1().Pgclusters(p.namespace).Get(ctx, p.cluster, metav1.GetOptions{})
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("error retrieving pgCluster")
		return
	}

	policy, err := clientset.CrunchydataV1().Pgpolicies(p.namespace).Get(ctx, p.policy, metav1.GetOptions{})
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("error retrieving pgPolicy")
		return
	}

	name := fmt.Sprintf("policy-%s-%s-schedule", p.cluster, p.policy)

	// if the cluster is found, check for a annotation indicating it has not been upgraded
	// if the annotation does not exist, then it is a new cluster and proceed as usual
	// if the annotation is set to "true", the cluster has already been upgraded and can proceed but
	// if the annotation is set to "false", this cluster will need to be upgraded before proceeding
	// log the issue, then return
	if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
		contextLogger.WithFields(log.Fields{
			"task": name,
		}).Debug("pgcluster requires an upgrade before scheduled policy task can run")
		return
	}

	filename := fmt.Sprintf("%s.sql", p.policy)
	data := make(map[string]string)
	data[filename] = string(policy.Spec.SQL)

	labels := map[string]string{
		"pg-cluster": p.cluster,
	}
	labels["pg-cluster"] = p.cluster
	labels["pg-policy"] = p.policy
	labels["pg-schedule"] = "true"

	configmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Data: data,
	}

	err = clientset.CoreV1().ConfigMaps(p.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		contextLogger.WithFields(log.Fields{
			"error":     err,
			"configMap": name,
		}).Error("could not delete policy configmap")
		return
	}

	log.Debug("Creating configmap..")
	_, err = clientset.CoreV1().ConfigMaps(p.namespace).Create(ctx, configmap, metav1.CreateOptions{})
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("could not create policy configmap")
		return
	}

	policyJob := PolicyTemplate{
		JobName:        name,
		ClusterName:    p.cluster,
		CCPImagePrefix: util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, p.ccpImagePrefix),
		CCPImageTag:    p.ccpImageTag,
		CustomLabels:   operator.GetLabelsFromMap(util.GetCustomLabels(cluster), false),
		PGHost:         p.cluster,
		PGPort:         cluster.Spec.Port,
		PGDatabase:     p.database,
		PGSQLConfigMap: name,
		PGUserSecret:   p.secret,
		Tolerations:    util.GetTolerations(cluster.Spec.Tolerations),
	}

	var doc bytes.Buffer
	if err := config.PolicyJobTemplate.Execute(&doc, policyJob); err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to render job template")
		return
	}

	deletePropagation := metav1.DeletePropagationForeground
	err = clientset.
		BatchV1().Jobs(p.namespace).
		Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
	if err == nil {
		err = wait.Poll(time.Second/2, time.Minute, func() (bool, error) {
			_, err := clientset.BatchV1().Jobs(p.namespace).Get(ctx, name, metav1.GetOptions{})
			return false, err
		})
	}
	if !kerrors.IsNotFound(err) {
		contextLogger.WithFields(log.Fields{
			"job":   name,
			"error": err,
		}).Error("error deleting policy job")
		return
	}

	newJob := &v1batch.Job{}
	if err := json.Unmarshal(doc.Bytes(), newJob); err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed unmarshaling job template")
		return
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA,
		&newJob.Spec.Template.Spec.Containers[0])

	_, err = clientset.BatchV1().Jobs(p.namespace).Create(ctx, newJob, metav1.CreateOptions{})
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed creating policy job")
		return
	}
}
