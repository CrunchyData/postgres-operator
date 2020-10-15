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
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// UserSecretFormat follows the pattern of how the user information is stored,
// which is "<clusteRName>-<userName>-secret"
const UserSecretFormat = "%s-%s" + crv1.UserSecretSuffix

// The following constants are used as a part of password generation. For more
// information on these selections, please consulting the ASCII man page
// (`man ascii`)
const (
	// passwordCharLower is the lowest ASCII character to use for generating a
	// password, which is 40
	passwordCharLower = 40
	// passwordCharUpper is the highest ASCII character to use for generating a
	// password, which is 126
	passwordCharUpper = 126
)

// passwordCharSelector is a "big int" that we need to select the random ASCII
// character for the password. Since the random integer generator looks for
// values from [0,X), we need to force this to be [40,126]
var passwordCharSelector = big.NewInt(passwordCharUpper - passwordCharLower)

// CreateSecret create the secret, user, and primary secrets
func CreateSecret(clientset kubernetes.Interface, db, secretName, username, password, namespace string) error {

	var enUsername = username

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels["pg-cluster"] = db
	secret.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	secret.Data = make(map[string][]byte)
	secret.Data["username"] = []byte(enUsername)
	secret.Data["password"] = []byte(password)

	_, err := clientset.CoreV1().Secrets(namespace).Create(&secret)

	return err

}

// GeneratePassword generates a password of a given length out of the acceptable
// ASCII characters suitable for a password
func GeneratePassword(length int) (string, error) {
	password := make([]byte, length)

	for i := 0; i < length; i++ {
		char, err := rand.Int(rand.Reader, passwordCharSelector)

		// if there is an error generating the random integer, return
		if err != nil {
			return "", err
		}

		password[i] = byte(passwordCharLower + char.Int64())
	}

	return string(password), nil
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
func GetPasswordFromSecret(clientset kubernetes.Interface, namespace, secretName string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(secret.Data["password"][:]), nil
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

// CreateUserSecret will create a new secret holding a user credential
func CreateUserSecret(clientset kubernetes.Interface, clustername, username, password, namespace string) error {
	secretName := fmt.Sprintf(UserSecretFormat, clustername, username)

	if err := CreateSecret(clientset, clustername, secretName, username, password, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// UpdateUserSecret updates a user secret with a new password. It follows the
// following method:
//
// 1. If the Secret exists, it updates the value of the Secret
// 2. If the Secret does not exist, it creates the secret
func UpdateUserSecret(clientset kubernetes.Interface, clustername, username, password, namespace string) error {
	secretName := fmt.Sprintf(UserSecretFormat, clustername, username)

	// see if the secret already exists
	secret, err := clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})

	// if this returns an error and it's not the "not found" error, return
	// However, if it is the "not found" error, treat this as creating the user
	// secret
	if err != nil {
		if !kubeapi.IsNotFound(err) {
			return err
		}

		return CreateUserSecret(clientset, clustername, username, password, namespace)
	}

	// update the value of "password"
	secret.Data["password"] = []byte(password)

	_, err = clientset.CoreV1().Secrets(secret.Namespace).Update(secret)
	return err
}
