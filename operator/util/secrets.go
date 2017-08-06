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

package util

import (
	//"encoding/base64"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/tpr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/pkg/api/v1"
	"math/rand"
	"strings"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

//create pgroot, pgmaster, and pguser secrets
func CreateDatabaseSecrets(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, cl *tpr.PgCluster, namespace string) error {

	//pgroot
	username := "postgres"
	suffix := tpr.PGROOT_SECRET_SUFFIX

	var secretName string
	var err error

	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PG_ROOT_PASSWORD, namespace)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	}

	cl.Spec.PGROOT_SECRET_NAME = secretName
	err = Patch(tprclient, "/spec/pgrootsecretname", secretName, tpr.CLUSTER_RESOURCE, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster" + err.Error())
	}

	///pgmaster
	username = "master"
	suffix = tpr.PGMASTER_SECRET_SUFFIX

	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PG_MASTER_PASSWORD, namespace)
	if err != nil {
		log.Error("error creating secret2" + err.Error())
	}

	cl.Spec.PGMASTER_SECRET_NAME = secretName
	err = Patch(tprclient, "/spec/pgmastersecretname", secretName, tpr.CLUSTER_RESOURCE, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster " + err.Error())
	}

	///pguser
	username = "testuser"
	suffix = tpr.PGUSER_SECRET_SUFFIX

	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PG_PASSWORD, namespace)
	if err != nil {
		log.Error("error creating secret " + err.Error())
	}

	cl.Spec.PGUSER_SECRET_NAME = secretName
	err = Patch(tprclient, "/spec/pgusersecretname", secretName, tpr.CLUSTER_RESOURCE, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error patching cluster " + err.Error())
	}

	return err
}

//create the secret, user, and master secrets
func CreateSecret(clientset *kubernetes.Clientset, db, secretName, username, password, namespace string) error {

	//var enUsername = base64.StdEncoding.EncodeToString([]byte(username))
	var enUsername = username
	//var enPassword = base64.StdEncoding.EncodeToString([]byte(generatePassword(10)))
	var enPassword = generatePassword(10)
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

	_, err := clientset.Secrets(namespace).Create(&secret)
	if err != nil {
		log.Error("error creating secret" + err.Error())
	} else {
		log.Info("created secret secret")
	}

	return err

}

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

//generate a password of a given length
func generatePassword(length int) string {
	return stringWithCharset(length, charset)
}

//delete pgroot, pgmaster, and pguser secrets
func DeleteDatabaseSecrets(clientset *kubernetes.Clientset, db, namespace string) {

	options := v1.DeleteOptions{}
	secretName := db + tpr.PGMASTER_SECRET_SUFFIX
	err := clientset.Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
	secretName = db + tpr.PGROOT_SECRET_SUFFIX
	err = clientset.Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
	secretName = db + tpr.PGUSER_SECRET_SUFFIX
	err = clientset.Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error("error deleting secret" + err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
}

func GetPasswordFromSecret(clientset *kubernetes.Clientset, namespace string, secretName string) (string, error) {
	secret, err := clientset.Secrets(namespace).Get(secretName)
	if errors.IsNotFound(err) {
		log.Error("not found error secret " + secretName)
		return "", err
	}

	return string(secret.Data["password"][:]), err

}

func CopySecrets(clientset *kubernetes.Clientset, namespace string, fromCluster, toCluster string) error {

	log.Debug("CopySecrets " + fromCluster + " to " + toCluster)
	lo := v1.ListOptions{LabelSelector: "pg-database=" + fromCluster}
	secrets, err := clientset.Secrets(namespace).List(lo)
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

		_, err := clientset.Secrets(namespace).Create(&secret)
		if err != nil {
			log.Error("error creating secret" + err.Error())
		} else {
			log.Info("created secret " + secret.Name)
		}

	}
	return err

}
