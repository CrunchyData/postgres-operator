package util

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const lowercharset = "abcdefghijklmnopqrstuvwxyz"

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const charsetNumbers = "0123456789"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// CreateSecret create the secret, user, and primary secrets
func CreateSecret(clientset *kubernetes.Clientset, db, secretName, username, password, namespace string) error {

	var enUsername = username

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels["pg-cluster"] = db
	secret.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	secret.Data = make(map[string][]byte)
	secret.Data["username"] = []byte(enUsername)
	secret.Data["password"] = []byte(password)

	err := kubeapi.CreateSecret(clientset, &secret, namespace)

	return err

}

// stringWithCharset returns a generated string value
func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GeneratePassword generate a password of a given length
func GeneratePassword(length int) string {
	return stringWithCharset(length, charset)
}

// GeneratePostgreSQLMD5Password takes a username and a plaintext password and
// returns the PostgreSQL formatted MD5 password, which is:
// "md5" + md5(password+username)
func GeneratePostgreSQLMD5Password(username, password string) string {
	// create the plaintext password/salt that PostgreSQL expects as a byte string
	plaintext := []byte(fmt.Sprintf("%s%s", password, username))
	// set up the password hasher
	hasher := md5.New()
	// add the above plaintext to the hash
	hasher.Write(plaintext)
	// finish the transformation by getting the string value of the MD5 hash and
	// encoding it in hexadecimal for PostgreSQL, appending "md5" to the front
	return fmt.Sprintf("md5%s", hex.EncodeToString(hasher.Sum(nil)))
}

// GenerateRandString generate a rand lowercase string of a given length
func GenerateRandString(length int) string {
	return stringWithCharset(length, lowercharset)
}

// GeneratedPasswordLength returns the value for what the length of a
// randomly generated password should be. It first determines if the user
// provided this value via a configuration file, and if not and/or the value is
// invalid, uses the default value
func GeneratedPasswordLength(configuredPasswordLength string) int {
	// set the generated password length for random password generation
	// note that "configuredPasswordLength" may be an empty string, and as such
	// the below line could fail. That's ok though! as we have a default set up
	generatedPasswordLength, err := strconv.Atoi(configuredPasswordLength)

	// if there is an error...set it to a default
	if err != nil {
		generatedPasswordLength = DefaultGeneratedPasswordLength
	}

	return generatedPasswordLength
}

// GetPasswordFromSecret will fetch the password from a user secret
func GetPasswordFromSecret(clientset *kubernetes.Clientset, namespace, secretName string) (string, error) {
	secret, found, err := kubeapi.GetSecret(clientset, secretName, namespace)

	if !found || err != nil {
		return "", err
	}

	return string(secret.Data["password"][:]), err
}

// IsPostgreSQLUserSystemAccount determines whether or not this is a system
// PostgreSQL user account, as if this returns true, one likely may not want to
// allow a user to directly access the account
// Normalizes the lookup by downcasing it
func IsPostgreSQLUserSystemAccount(username string) bool {
	// go look up and see if the username is in the map
	_, found := crv1.PGUserSystemAccounts[strings.ToLower(username)]
	return found
}

// CloneClusterSecrets will copy the secrets from a cluster into the secrets of
// another cluster
type CloneClusterSecrets struct {
	// any additional selectors that can be added to the query that is made
	AdditionalSelectors []string
	// The Kubernetes Clientset used to make API calls to Kubernetes`
	ClientSet *kubernetes.Clientset
	// The Namespace that the clusters are in
	Namespace string
	// The name of the PostgreSQL cluster that the secrets are originating from
	SourceClusterName string
	// The name of the PostgreSQL cluster that we are copying the secrets to
	TargetClusterName string
}

// Clone performs the actual clone of the secrets between PostgreSQL clusters
func (cs CloneClusterSecrets) Clone() error {
	log.Debugf("clone secrets [%s] to [%s]", cs.SourceClusterName, cs.TargetClusterName)

	// initialize the selector, and add any additional options to it
	selector := fmt.Sprintf("pg-cluster=%s", cs.SourceClusterName)

	for _, additionalSelector := range cs.AdditionalSelectors {
		selector += fmt.Sprintf(",%s", additionalSelector)
	}

	// get all the secrets that exist in the source PostgreSQL cluster
	secrets, err := kubeapi.GetSecrets(cs.ClientSet, selector, cs.Namespace)

	// if this fails, log and return the error
	if err != nil {
		log.Error(err)
		return err
	}

	// iterate through the existing secrets in the cluster, and copy them over
	for _, s := range secrets.Items {
		log.Debugf("found secret : %s", s.ObjectMeta.Name)

		secret := v1.Secret{}

		// create the secret name
		secret.Name = strings.Replace(s.ObjectMeta.Name, cs.SourceClusterName, cs.TargetClusterName, 1)

		// assign the labels
		secret.ObjectMeta.Labels = map[string]string{
			"pg-cluster": cs.TargetClusterName,
		}
		// secret.ObjectMeta.Labels["pg-cluster"] = toCluster

		// copy over the secret
		// secret.Data = make(map[string][]byte)
		secret.Data = map[string][]byte{
			"username": s.Data["username"][:],
			"password": s.Data["password"][:],
		}

		// create the secret
		kubeapi.CreateSecret(cs.ClientSet, &secret, cs.Namespace)
	}

	return nil
}

// CreateUserSecret will create a new secret holding a user credential
func CreateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string) error {
	secretName := clustername + "-" + username + "-secret"

	if err := CreateSecret(clientset, clustername, secretName, username, password, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// UpdateUserSecret updates a user secret with a new password
func UpdateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string) error {

	var err error

	secretName := clustername + "-" + username + "-secret"

	//delete current secret
	err = kubeapi.DeleteSecret(clientset, secretName, namespace)
	if err == nil {
		//create secret with updated password
		err = CreateUserSecret(clientset, clustername, username, password, namespace)
		if err != nil {
			log.Error("UpdateUserSecret error creating secret" + err.Error())
		} else {
			log.Debugf("created secret %s", secretName)
		}
	}

	return err
}
