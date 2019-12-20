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

	"gopkg.in/yaml.v2"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/sshutil"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// BackrestRepoConfig represents the configuration required to created backrest repo secrets
type BackrestRepoConfig struct {
	BackrestS3Key       string
	BackrestS3KeySecret string
	ClusterName         string
	ClusterNamespace    string
	OperatorNamespace   string
}

// AWSS3Secret is a representation of the yaml structure found in aws-s3-credentials.yaml
// for providing an AWS S3 key and key secret
type AWSS3Secret struct {
	AWSS3Key       string `yaml:"aws-s3-key"`
	AWSS3KeySecret string `yaml:"aws-s3-key-secret"`
}

// CreateBackrestRepoSecrets creates the secrets required to manage the
// pgBackRest repo container
func CreateBackrestRepoSecrets(clientset *kubernetes.Clientset,
	backrestRepoConfig BackrestRepoConfig) error {

	keys, err := sshutil.NewPrivatePublicKeyPair(config.DEFAULT_BACKREST_SSH_KEY_BITS)
	if err != nil {
		return err
	}

	// Retrieve the S3/SSHD configuration files from secret
	configs, _, err := kubeapi.GetSecret(clientset, "pgo-backrest-repo-config",
		backrestRepoConfig.OperatorNamespace)
	if kerrors.IsNotFound(err) || err != nil {
		return err
	}

	// if an S3 key has been provided via the request, then use key and key secret inlcuded
	// in the request instead of the default credentials provided by 'aws-s3-credentials.yaml'.
	// If an S3 key doesn't exist in the request, the use `aws-s3-credentials.yaml`
	var s3KeySecretData []byte
	if backrestRepoConfig.BackrestS3Key != "" && backrestRepoConfig.BackrestS3KeySecret != "" {
		s3Secret := AWSS3Secret{
			backrestRepoConfig.BackrestS3Key,
			backrestRepoConfig.BackrestS3KeySecret,
		}
		s3KeySecretData, err = yaml.Marshal(s3Secret)
		if err != nil {
			return err
		}
	} else {
		s3KeySecretData = configs.Data["aws-s3-credentials.yaml"]
	}

	secret := v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", backrestRepoConfig.ClusterName,
				config.LABEL_BACKREST_REPO_SECRET),
			Labels: map[string]string{
				config.LABEL_VENDOR:            config.LABEL_CRUNCHY,
				config.LABEL_PG_CLUSTER:        backrestRepoConfig.ClusterName,
				config.LABEL_PGO_BACKREST_REPO: "true",
			},
		},
		Data: map[string][]byte{
			"authorized_keys":         keys.Public,
			"aws-s3-ca.crt":           configs.Data["aws-s3-ca.crt"],
			"aws-s3-credentials.yaml": s3KeySecretData,
			"config":                  configs.Data["config"],
			"id_rsa":                  keys.Private,
			"sshd_config":             configs.Data["sshd_config"],
			"ssh_host_rsa_key":        keys.Private,
		},
	}
	return kubeapi.CreateSecret(clientset, &secret, backrestRepoConfig.ClusterNamespace)
}

// IsAutofailEnabled - returns true if autofail label is set to true, false if not.
func IsAutofailEnabled(cluster *crv1.Pgcluster) bool {

	labels := cluster.ObjectMeta.Labels
	failLabel := labels[config.LABEL_AUTOFAIL]

	log.Debugf("IsAutoFailEnabled: %s", failLabel)

	return failLabel == "true"
}

// GetS3CredsFromBackrestRepoSecret retrieves the AWS S3 credentials, i.e. the key and key
// secret, from a specific cluster's backrest repo secret
func GetS3CredsFromBackrestRepoSecret(clientset *kubernetes.Clientset, clusterName,
	namespace string) (s3Secret AWSS3Secret, err error) {

	currBackrestSecret, _, err := kubeapi.GetSecret(clientset,
		clusterName+"-backrest-repo-config", namespace)
	if err != nil {
		log.Error(err)
		return
	}
	err = yaml.Unmarshal(currBackrestSecret.Data["aws-s3-credentials.yaml"], &s3Secret)
	if err != nil {
		log.Error(err)
		return
	}

	return
}
