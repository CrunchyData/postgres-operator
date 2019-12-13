package util

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/sshutil"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateBackrestRepoSecrets creates the secrets required to manage the
// pgBackRest repo container
func CreateBackrestRepoSecrets(clientset *kubernetes.Clientset, operatorNamespace, clusterNamespace, clusterName string) error {
	keys, err := sshutil.NewPrivatePublicKeyPair(config.DEFAULT_BACKREST_SSH_KEY_BITS)
	if err != nil {
		return err
	}

	// Retrieve the S3/SSHD configuration files from secret
	configs, _, err := kubeapi.GetSecret(clientset, "pgo-backrest-repo-config", operatorNamespace)
	if kerrors.IsNotFound(err) || err != nil {
		return err
	}

	secret := v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET),
			Labels: map[string]string{
				config.LABEL_VENDOR:            config.LABEL_CRUNCHY,
				config.LABEL_PG_CLUSTER:        clusterName,
				config.LABEL_PGO_BACKREST_REPO: "true",
			},
		},
		Data: map[string][]byte{
			"authorized_keys":         keys.Public,
			"aws-s3-ca.crt":           configs.Data["aws-s3-ca.crt"],
			"aws-s3-credentials.yaml": configs.Data["aws-s3-credentials.yaml"],
			"config":                  configs.Data["config"],
			"id_rsa":                  keys.Private,
			"sshd_config":             configs.Data["sshd_config"],
			"ssh_host_rsa_key":        keys.Private,
		},
	}
	return kubeapi.CreateSecret(clientset, &secret, clusterNamespace)
}

// IsAutofailEnabled - returns true if autofail label is set to true, false if not.
func IsAutofailEnabled(cluster *crv1.Pgcluster) bool {

	labels := cluster.ObjectMeta.Labels
	failLabel := labels[config.LABEL_AUTOFAIL]

	log.Debugf("IsAutoFailEnabled: %s", failLabel)

	return failLabel == "true"
}
