package util

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// BackrestRepoConfig represents the configuration required to created backrest repo secrets
type BackrestRepoConfig struct {
	// BackrestS3CA is the byte string value of the CA that should be used for the
	// S3 inerfacd pgBackRest repository
	BackrestS3CA        []byte
	BackrestS3Key       string
	BackrestS3KeySecret string
	ClusterName         string
	ClusterNamespace    string
	OperatorNamespace   string
}

// AWSS3Secret is a structured representation for providing  an AWS S3 key and
// key secret
type AWSS3Secret struct {
	AWSS3CA        []byte
	AWSS3Key       string
	AWSS3KeySecret string
}

const (
	// DefaultGeneratedPasswordLength is the length of what a generated password
	// is if it's not set in the pgo.yaml file, and to create some semblance of
	// consistency
	DefaultGeneratedPasswordLength = 24
	// DefaultPasswordValidUntilDays is the number of days until a PostgreSQL user's
	// password expires. If it is not set in the pgo.yaml file, we will use a
	// default of "0" which means that a password will never expire
	DefaultPasswordValidUntilDays = 0
)

// values for the keys used to access the pgBackRest repository Secret
const (
	// three of these are exported, as they are used to help add the information
	// into the templates. Say the last one 10 times fast
	// #nosec: G101
	BackRestRepoSecretKeyAWSS3KeyAWSS3CACert = "aws-s3-ca.crt"
	// #nosec: G101
	BackRestRepoSecretKeyAWSS3KeyAWSS3Key = "aws-s3-key"
	// #nosec: G101
	BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret = "aws-s3-key-secret"
	// the rest are private
	backRestRepoSecretKeyAuthorizedKeys = "authorized_keys"
	backRestRepoSecretKeySSHConfig      = "config"
	// #nosec: G101
	backRestRepoSecretKeySSHDConfig = "sshd_config"
	// #nosec: G101
	backRestRepoSecretKeySSHPrivateKey = "id_ed25519"
	// #nosec: G101
	backRestRepoSecretKeySSHHostPrivateKey = "ssh_host_ed25519_key"
)

const (
	// SQLValidUntilAlways uses a special PostgreSQL value to ensure a password
	// is always valid
	SQLValidUntilAlways = "infinity"
	// SQLValidUntilNever uses a special PostgreSQL value to ensure a password
	// is never valid. This is exportable and used in other places
	SQLValidUntilNever = "-infinity"
	// sqlSetPasswordDefault is the SQL to update the password
	// NOTE: this is safe from SQL injection as we explicitly add the inerpolated
	// string as a MD5 hash or SCRAM verifier. And if you're not doing that,
	// rethink your usage of this
	//
	// The escaping for SQL injections is handled in the SetPostgreSQLPassword
	// function
	// #nosec: G101
	sqlSetPasswordDefault = `ALTER ROLE %s PASSWORD %s;`
)

// ErrMissingConfigAnnotation represents an error thrown when the 'config' annotation is found
// to be missing from the 'config' configMap created to store cluster-wide configuration
var ErrMissingConfigAnnotation error = errors.New("'config' annotation missing from cluster " +
	"configutation")

// CmdStopPostgreSQL is the command used to stop a PostgreSQL instance, which
// uses the "fast" shutdown mode. This needs a data directory appended to it
var cmdStopPostgreSQL = []string{
	"pg_ctl", "stop",
	"-m", "fast", "-D",
}

// CreateBackrestRepoSecrets creates the secrets required to manage the
// pgBackRest repo container
func CreateBackrestRepoSecrets(clientset kubernetes.Interface,
	backrestRepoConfig BackrestRepoConfig) error {
	ctx := context.TODO()

	// first: determine if a Secret already exists. If it does, we are going to
	// work on modifying that Secret.
	secretName := fmt.Sprintf("%s-%s", backrestRepoConfig.ClusterName,
		config.LABEL_BACKREST_REPO_SECRET)
	secret, secretErr := clientset.CoreV1().Secrets(backrestRepoConfig.ClusterNamespace).Get(
		ctx, secretName, metav1.GetOptions{})

	// only return an error if this is a **not** a not found error
	if secretErr != nil && !kerrors.IsNotFound(secretErr) {
		log.Error(secretErr)
		return secretErr
	}

	// determine if we need to create a new secret, i.e. this is a not found error
	newSecret := secretErr != nil
	if newSecret {
		// set up the secret for the cluster that contains the pgBackRest information
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
				Labels: map[string]string{
					config.LABEL_VENDOR:            config.LABEL_CRUNCHY,
					config.LABEL_PG_CLUSTER:        backrestRepoConfig.ClusterName,
					config.LABEL_PGO_BACKREST_REPO: "true",
				},
			},
			Data: map[string][]byte{},
		}
	}

	// next, load the Operator level pgBackRest secret templates, which contain
	// SSHD(...?) and possible S3 credentials
	configs, configErr := clientset.
		CoreV1().Secrets(backrestRepoConfig.OperatorNamespace).
		Get(ctx, config.SecretOperatorBackrestRepoConfig, metav1.GetOptions{})

	if configErr != nil {
		log.Error(configErr)
		return configErr
	}

	// set the SSH/SSHD configuration, if it is not presently set
	for _, key := range []string{backRestRepoSecretKeySSHConfig, backRestRepoSecretKeySSHDConfig} {
		if len(secret.Data[key]) == 0 {
			secret.Data[key] = configs.Data[key]
		}
	}

	// set the SSH keys if any appear to be unset
	if len(secret.Data[backRestRepoSecretKeyAuthorizedKeys]) == 0 ||
		len(secret.Data[backRestRepoSecretKeySSHPrivateKey]) == 0 ||
		len(secret.Data[backRestRepoSecretKeySSHHostPrivateKey]) == 0 {
		// generate the keypair and then assign it to the values in the Secret
		keys, keyErr := NewPrivatePublicKeyPair()

		if keyErr != nil {
			log.Error(keyErr)
			return keyErr
		}

		secret.Data[backRestRepoSecretKeyAuthorizedKeys] = keys.Public
		secret.Data[backRestRepoSecretKeySSHPrivateKey] = keys.Private
		secret.Data[backRestRepoSecretKeySSHHostPrivateKey] = keys.Private
	}

	// Set the S3 credentials
	// If explicit S3 credentials are passed in, use those.
	// If the Secret already has S3 credentials, use those.
	// Otherwise, try to load in the default credentials from the Operator Secret.
	if len(backrestRepoConfig.BackrestS3CA) != 0 {
		secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3CACert] = backrestRepoConfig.BackrestS3CA
	}

	if len(secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3CACert]) == 0 &&
		len(configs.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3CACert]) != 0 {
		secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3CACert] = configs.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3CACert]
	}

	if backrestRepoConfig.BackrestS3Key != "" {
		secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3Key] = []byte(backrestRepoConfig.BackrestS3Key)
	}

	if len(secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3Key]) == 0 &&
		len(configs.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3Key]) != 0 {
		secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3Key] = configs.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3Key]
	}

	if backrestRepoConfig.BackrestS3KeySecret != "" {
		secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret] = []byte(backrestRepoConfig.BackrestS3KeySecret)
	}

	if len(secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret]) == 0 &&
		len(configs.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret]) != 0 {
		secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret] = configs.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret]
	}

	// time to create or update the secret!
	if newSecret {
		_, err := clientset.CoreV1().Secrets(backrestRepoConfig.ClusterNamespace).Create(
			ctx, secret, metav1.CreateOptions{})
		return err
	}

	_, err := clientset.CoreV1().Secrets(backrestRepoConfig.ClusterNamespace).Update(
		ctx, secret, metav1.UpdateOptions{})

	return err
}

// IsAutofailEnabled - returns true if autofail label is set to true, false if not.
func IsAutofailEnabled(cluster *crv1.Pgcluster) bool {
	labels := cluster.ObjectMeta.Labels
	failLabel := labels[config.LABEL_AUTOFAIL]

	log.Debugf("IsAutoFailEnabled: %s", failLabel)

	return failLabel == "true"
}

// GeneratedPasswordValidUntilDays returns the value for the number of days that
// a password is valid for, which is used as part of PostgreSQL's VALID UNTIL
// directive on a user.  It first determines if the user provided this value via
// a configuration file, and if not and/or the value is invalid, uses the
// default value
func GeneratedPasswordValidUntilDays(configuredValidUntilDays string) int {
	// set the generated password length for random password generation
	// note that "configuredPasswordLength" may be an empty string, and as such
	// the below line could fail. That's ok though! as we have a default set up
	validUntilDays, err := strconv.Atoi(configuredValidUntilDays)
	// if there is an error...set it to a default
	if err != nil {
		validUntilDays = DefaultPasswordValidUntilDays
	}

	return validUntilDays
}

// GetPrimaryPod gets the Pod of the primary PostgreSQL instance. If somehow
// the query gets multiple pods, then the first one in the list is returned
func GetPrimaryPod(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (*v1.Pod, error) {
	ctx := context.TODO()

	// set up the selector for the primary pod
	selector := fmt.Sprintf("%s=%s,%s=%s",
		config.LABEL_PG_CLUSTER, cluster.Spec.Name, config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)
	namespace := cluster.Spec.Namespace

	// query the pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	// if there is an error, log it and abort
	if err != nil {
		return nil, err
	}

	// if no pods are retirn, then also raise an error
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("primary pod not found for selector %q", selector)
	}

	// Grab the first pod from the list as this is presumably the primary pod
	pod := pods.Items[0]
	return &pod, nil
}

// GetS3CredsFromBackrestRepoSecret retrieves the AWS S3 credentials, i.e. the key and key
// secret, from a specific cluster's backrest repo secret
func GetS3CredsFromBackrestRepoSecret(clientset kubernetes.Interface, namespace, clusterName string) (AWSS3Secret, error) {
	ctx := context.TODO()
	secretName := fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET)
	s3Secret := AWSS3Secret{}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return s3Secret, err
	}

	// get the S3 secret credentials out of the secret, and return
	s3Secret.AWSS3CA = secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3CACert]
	s3Secret.AWSS3Key = string(secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3Key])
	s3Secret.AWSS3KeySecret = string(secret.Data[BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret])

	return s3Secret, nil
}

// SetPostgreSQLPassword updates the password for a PostgreSQL role in the
// PostgreSQL cluster by executing into the primary Pod and changing it
//
// Note: it is recommended to pre-hash the password (e.g. md5, SCRAM) so that
// way the plaintext password is not logged anywhere. This also avoids potential
// SQL injections
func SetPostgreSQLPassword(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port, username, password, sqlCustom string) error {
	log.Debugf("set PostgreSQL password for user [%s]", username)

	// if custom SQL is not set, use the default SQL
	sqlRaw := sqlCustom

	if sqlRaw == "" {
		sqlRaw = sqlSetPasswordDefault
	}

	// This is safe from SQL injection as we are using constants and a well defined
	// string...well, as long as the function caller does this
	sql := strings.NewReader(fmt.Sprintf(sqlRaw,
		SQLQuoteIdentifier(username), SQLQuoteLiteral(password)))
	cmd := []string{"psql", "-p", port}

	// exec into the pod to run the query
	_, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, sql)

	// if there is an error executing the command, or output in stderr,
	// log the error message and return
	if err != nil {
		log.Error(err)
		return err
	} else if stderr != "" {
		log.Error(stderr)
		return fmt.Errorf(stderr)
	}

	return nil
}

// StopPostgreSQLInstance issues a "fast" shutdown command to the PostgreSQL
// instance. This will immediately terminate any connections and safely shut
// down PostgreSQL so it does not have to start up in crash recovery mode
func StopPostgreSQLInstance(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, instanceName string) error {
	log.Debugf("shutting down PostgreSQL on pod [%s]", pod.Name)

	// append the data directory, which is the name of the instance
	cmd := cmdStopPostgreSQL
	dataDirectory := fmt.Sprintf("%s/%s", config.VOLUME_POSTGRESQL_DATA_MOUNT_PATH, instanceName)
	cmd = append(cmd, dataDirectory)

	// exec into the pod to execute the stop command
	_, stderr, _ := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, nil)

	// if there is error output, assume this is an error and return
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	return nil
}
