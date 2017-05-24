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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/pkg/api/v1"
	"math/rand"
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
	suffix := "-pgroot-secret"
	secretName := cl.Spec.Name + suffix
	err := CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PG_ROOT_PASSWORD, namespace)
	if err != nil {
		log.Error(err.Error())
	}
	cl.Spec.PGROOT_SECRET_NAME = secretName
	err = Patch(tprclient, "/spec/pgrootsecretname", secretName, "pgclusters", cl.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

	///pgmaster
	username = "master"
	suffix = "-pgmaster-secret"
	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PG_MASTER_PASSWORD, namespace)
	if err != nil {
		log.Error(err.Error())
	}
	cl.Spec.PGMASTER_SECRET_NAME = secretName
	err = Patch(tprclient, "/spec/pgmastersecretname", secretName, "pgclusters", cl.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

	///pguser
	username = "testuser"
	suffix = "-pguser-secret"
	secretName = cl.Spec.Name + suffix
	err = CreateSecret(clientset, cl.Spec.Name, secretName, username, cl.Spec.PG_PASSWORD, namespace)
	if err != nil {
		log.Error(err.Error())
	}
	cl.Spec.PGUSER_SECRET_NAME = secretName
	err = Patch(tprclient, "/spec/pgusersecretname", secretName, "pgclusters", cl.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
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
		log.Error(err.Error())
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
	secretName := db + "-pgmaster-secret"
	err := clientset.Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
	secretName = db + "-pgroot-secret"
	err = clientset.Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
	secretName = db + "-pguser-secret"
	err = clientset.Secrets(namespace).Delete(secretName, &options)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("deleted secret " + secretName)
	}
}
