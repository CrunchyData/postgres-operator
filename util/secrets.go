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
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"math/rand"
	"strings"
	"time"
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

// GenerateRandString generate a rand lowercase string of a given length
func GenerateRandString(length int) string {
	return stringWithCharset(length, lowercharset)
}

// GetPasswordFromSecret will fetch the username, password from a user secret
func GetPasswordFromSecret(clientset *kubernetes.Clientset, namespace string, secretName string) (string, string, error) {

	if clientset == nil {
		log.Errorln("clientset is nil")
	}
	log.Infoln("namespace=" + namespace)
	log.Infoln("secretName=" + secretName)

	secret, found, err := kubeapi.GetSecret(clientset, secretName, namespace)
	if !found || err != nil {
		return "", "", err
	}

	return string(secret.Data["username"][:]), string(secret.Data["password"][:]), err

}

// CopySecrets will copy a secret to another secret
func CopySecrets(clientset *kubernetes.Clientset, namespace string, fromCluster, toCluster string) error {

	log.Debugf("CopySecrets %s to %s", fromCluster, toCluster)
	selector := "pg-cluster=" + fromCluster

	secrets, err := kubeapi.GetSecrets(clientset, selector, namespace)
	if err != nil {
		return err
	}

	for _, s := range secrets.Items {
		log.Debugf("found secret : %s", s.ObjectMeta.Name)
		secret := v1.Secret{}
		secret.Name = strings.Replace(s.ObjectMeta.Name, fromCluster, toCluster, 1)
		secret.ObjectMeta.Labels = make(map[string]string)
		secret.ObjectMeta.Labels["pg-cluster"] = toCluster
		secret.Data = make(map[string][]byte)
		secret.Data["username"] = s.Data["username"][:]
		secret.Data["password"] = s.Data["password"][:]

		kubeapi.CreateSecret(clientset, &secret, namespace)

	}
	return err

}

// CreateUserSecret will create a new secret holding a user credential
func CreateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string, passwordLength int) error {

	var err error

	secretName := clustername + "-" + username + "-secret"
	var enPassword = GeneratePassword(passwordLength)
	if password != "" {
		log.Debugf("using user specified password for secret %s", secretName)
		enPassword = password
	}
	err = CreateSecret(clientset, clustername, secretName, username, enPassword, namespace)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	}

	return err
}

// UpdateUserSecret updates a user secret with a new password
func UpdateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string, passwordLength int) error {

	var err error

	secretName := clustername + "-" + username + "-secret"

	//delete current secret
	err = kubeapi.DeleteSecret(clientset, secretName, namespace)
	if err == nil {
		//create secret with updated password
		err = CreateUserSecret(clientset, clustername, username, password, namespace, passwordLength)
		if err != nil {
			log.Error("UpdateUserSecret error creating secret" + err.Error())
		} else {
			log.Debugf("created secret %s", secretName)
		}
	}

	return err
}
