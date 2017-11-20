package util

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"math/rand"
	"strings"
	"time"
)

const lowercharset = "abcdefghijklmnopqrstuvwxyz"

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// CreateDatabaseSecrets create pgroot, pgprimary, and pguser secrets
func CreateDatabaseSecrets(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {

	//pgroot
	username := "postgres"
	suffix := crv1.RootSecretSuffix

	var secretName string
	var err error

	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.RootPassword, namespace)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	}

	cl.Spec.RootSecretName = secretName
	err = Patch(restclient, "/spec/rootsecretname", secretName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster" + err.Error())
	}

	///primary
	username = "primaryuser"
	suffix = crv1.PrimarySecretSuffix

	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PrimaryPassword, namespace)
	if err != nil {
		log.Error("error creating secret2" + err.Error())
	}

	cl.Spec.PrimarySecretName = secretName
	err = Patch(restclient, "/spec/primarysecretname", secretName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster " + err.Error())
	}

	///pguser
	username = "testuser"
	suffix = crv1.UserSecretSuffix

	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.Password, namespace)
	if err != nil {
		log.Error("error creating secret " + err.Error())
	}

	cl.Spec.UserSecretName = secretName
	err = Patch(restclient, "/spec/usersecretname", secretName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster " + err.Error())
	}

	return err
}

// CreateSecret create the secret, user, and primary secrets
func CreateSecret(clientset *kubernetes.Clientset, db, secretName, username, password, namespace string) error {

	var enUsername = username
	var enPassword = GeneratePassword(10)
	if password != "" {
		log.Debug("using user specified password for secret " + secretName)
		enPassword = password
	}

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels["pg-database"] = db
	secret.Data = make(map[string][]byte)
	secret.Data["username"] = []byte(enUsername)
	secret.Data["password"] = []byte(enPassword)

	_, err := clientset.Core().Secrets(namespace).Create(&secret)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	} else {
		log.Debug("created secret " + secret.Name)
	}

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

// DeleteDatabaseSecrets delete pgroot, pgprimary, and pguser secrets
func DeleteDatabaseSecrets(clientset *kubernetes.Clientset, db, namespace string) {

	options := meta_v1.DeleteOptions{}
	secretName := db + crv1.PrimarySecretSuffix
	err := clientset.Core().Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
	secretName = db + crv1.RootSecretSuffix
	err = clientset.Core().Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
	secretName = db + crv1.UserSecretSuffix
	err = clientset.Core().Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
}

// GetPasswordFromSecret will fetch the password from a user secret
func GetPasswordFromSecret(clientset *kubernetes.Clientset, namespace string, secretName string) (string, error) {

	if clientset == nil {
		log.Errorln("clientset is nil")
	}
	log.Infoln("namespace=" + namespace)
	log.Infoln("secretName=" + secretName)

	options := meta_v1.GetOptions{}
	secret, err := clientset.Core().Secrets(namespace).Get(secretName, options)
	if errors.IsNotFound(err) {
		log.Error("not found error secret " + secretName)
		return "", err
	}

	return string(secret.Data["password"][:]), err

}

// CopySecrets will copy a secret to another secret
func CopySecrets(clientset *kubernetes.Clientset, namespace string, fromCluster, toCluster string) error {

	log.Debug("CopySecrets " + fromCluster + " to " + toCluster)
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + fromCluster}
	secrets, err := clientset.Core().Secrets(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return err
	}

	for _, s := range secrets.Items {
		log.Debug("found secret : " + s.ObjectMeta.Name)
		secret := v1.Secret{}
		secret.Name = strings.Replace(s.ObjectMeta.Name, fromCluster, toCluster, 1)
		secret.ObjectMeta.Labels = make(map[string]string)
		secret.ObjectMeta.Labels["pg-database"] = toCluster
		secret.Data = make(map[string][]byte)
		secret.Data["username"] = s.Data["username"][:]
		secret.Data["password"] = s.Data["password"][:]

		_, err := clientset.Core().Secrets(namespace).Create(&secret)
		if err != nil {
			log.Error("error creating secret" + err.Error())
		} else {
			log.Debug("created secret " + secret.Name)
		}

	}
	return err

}

// CreateUserSecret will create a new secret holding a user credential
func CreateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string) error {

	var err error

	secretName := clustername + "-" + username + "-secret"
	err = CreateSecret(clientset, clustername, secretName, username, password, namespace)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	}

	return err
}

// UpdateUserSecret updates a user secret with a new password
func UpdateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string) error {

	var err error

	secretName := clustername + "-" + username + "-secret"

	//delete current secret
	options := meta_v1.DeleteOptions{}
	err = clientset.Core().Secrets(namespace).Delete(secretName, &options)
	if err == nil {
		log.Debug("deleted secret " + secretName)
		//create secret with updated password
		err = CreateUserSecret(clientset, clustername, username, password, namespace)
		if err != nil {
			log.Error("error creating secret" + err.Error())
		} else {
			log.Debug("created secret " + secretName)
		}
	} else {
		log.Error("error deleting secret" + err.Error())
	}

	return err
}

// DeleteUserSecret will delete a user secret
func DeleteUserSecret(clientset *kubernetes.Clientset, clustername, username, namespace string) error {
	//delete current secret
	secretName := clustername + "-" + username + "-secret"

	options := meta_v1.DeleteOptions{}
	err := clientset.Core().Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Debug("deleted secret " + secretName)
	}
	return err
}
