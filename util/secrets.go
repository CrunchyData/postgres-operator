package util

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
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
func CreateDatabaseSecrets(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) (string, string, string, error) {

	//pgroot
	pgUser := "postgres"
	suffix := crv1.RootSecretSuffix

	var secretName string
	var err error

	secretName = cl.Spec.Name + suffix
	pgPassword := GeneratePassword(10)
	if cl.Spec.RootPassword != "" {
		log.Debug("using user specified password for secret " + secretName)
		pgPassword = cl.Spec.RootPassword
	}

	err = CreateSecret(clientset, cl.Spec.Name, secretName, pgUser, pgPassword, namespace)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	}

	cl.Spec.RootSecretName = secretName
	err = Patch(restclient, "/spec/rootsecretname", secretName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster" + err.Error())
	}

	///primary
	primaryUser := "primaryuser"
	suffix = crv1.PrimarySecretSuffix

	secretName = cl.Spec.Name + suffix
	primaryPassword := GeneratePassword(10)
	if cl.Spec.PrimaryPassword != "" {
		log.Debug("using user specified password for secret " + secretName)
		primaryPassword = cl.Spec.PrimaryPassword
	}

	err = CreateSecret(clientset, cl.Spec.Name, secretName, primaryUser, primaryPassword, namespace)
	if err != nil {
		log.Error("error creating secret2" + err.Error())
	}

	cl.Spec.PrimarySecretName = secretName
	err = Patch(restclient, "/spec/primarysecretname", secretName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster " + err.Error())
	}

	///pguser
	username := "testuser"
	if cl.Spec.User != "" {
		username = cl.Spec.User
	}
	suffix = crv1.UserSecretSuffix

	secretName = cl.Spec.Name + suffix
	if cl.Spec.UserSecretName != "" {
		secretName = cl.Spec.UserSecretName
		log.Debugf("using user specified user secret name: %v", secretName)
	}

	userPassword := GeneratePassword(10)
	if cl.Spec.Password != "" {
		log.Debugf("using user specified password for secret %v", secretName)
		userPassword = cl.Spec.Password
	}

	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, userPassword, namespace)
	if err != nil {
		log.WithError(err).Errorf("error creating secret %v", secretName)
	}

	if secretName != cl.Spec.UserSecretName {
		cl.Spec.UserSecretName = secretName
		err = Patch(restclient, "/spec/usersecretname", secretName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
		if err != nil {
			log.WithError(err).Errorf("error patching cluster %s", cl.Spec.Name)
		}
	}

	return pgPassword, primaryPassword, userPassword, err
}

// CreateSecret create the secret, user, and primary secrets
func CreateSecret(clientset *kubernetes.Clientset, db, secretName, username, password, namespace string) error {

	var enUsername = username

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels["pg-database"] = db
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

// DeleteDatabaseSecrets delete secrets that match pg-database=somecluster
func DeleteDatabaseSecrets(clientset *kubernetes.Clientset, db, namespace string) {
	//get all that match pg-database=db
	selector := "pg-database=" + db
	secrets, err := kubeapi.GetSecrets(clientset, selector, namespace)
	if err != nil {
		return
	}

	for _, s := range secrets.Items {
		kubeapi.DeleteSecret(clientset, s.ObjectMeta.Name, namespace)
	}
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

	log.Debug("CopySecrets " + fromCluster + " to " + toCluster)
	selector := "pg-database=" + fromCluster

	secrets, err := kubeapi.GetSecrets(clientset, selector, namespace)
	if err != nil {
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

		kubeapi.CreateSecret(clientset, &secret, namespace)

	}
	return err

}

// CreateUserSecret will create a new secret holding a user credential
func CreateUserSecret(clientset *kubernetes.Clientset, clustername, username, password, namespace string) error {

	var err error

	secretName := clustername + "-" + username + "-secret"
	var enPassword = GeneratePassword(10)
	if password != "" {
		log.Debug("using user specified password for secret " + secretName)
		enPassword = password
	}
	err = CreateSecret(clientset, clustername, secretName, username, enPassword, namespace)
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
	err = kubeapi.DeleteSecret(clientset, secretName, namespace)
	if err == nil {
		//create secret with updated password
		err = CreateUserSecret(clientset, clustername, username, password, namespace)
		if err != nil {
			log.Error("UpdateUserSecret error creating secret" + err.Error())
		} else {
			log.Debug("created secret " + secretName)
		}
	}

	return err
}

// DeleteUserSecret will delete a user secret
func DeleteUserSecret(clientset *kubernetes.Clientset, clustername, username, namespace string) error {
	//delete current secret
	secretName := clustername + "-" + username + "-secret"

	err := kubeapi.DeleteSecret(clientset, secretName, namespace)
	return err
}
