package util

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"crypto/rand"
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
	// passwordCharExclude is a map of characters that we choose to exclude from
	// the password to simplify usage in the shell. There is still enough entropy
	// that exclusion of these characters is OK.
	passwordCharExclude = "`\\"
)

// passwordCharSelector is a "big int" that we need to select the random ASCII
// character for the password. Since the random integer generator looks for
// values from [0,X), we need to force this to be [40,126]
var passwordCharSelector = big.NewInt(passwordCharUpper - passwordCharLower)

// CreateSecret create the secret, user, and primary secrets
func CreateSecret(clientset kubernetes.Interface, db, secretName, username, password, namespace string, labels map[string]string) error {
	ctx := context.TODO()
	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = labels
	secret.ObjectMeta.Labels["pg-cluster"] = db
	secret.ObjectMeta.Labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	secret.Data = make(map[string][]byte)
	secret.Data["username"] = []byte(username)
	secret.Data["password"] = []byte(password)

	_, err := clientset.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{})

	return err
}

// GeneratePassword generates a password of a given length out of the acceptable
// ASCII characters suitable for a password
func GeneratePassword(length int) (string, error) {
	password := make([]byte, length)
	i := 0

	for i < length {
		val, err := rand.Int(rand.Reader, passwordCharSelector)
		// if there is an error generating the random integer, return
		if err != nil {
			return "", err
		}

		char := byte(passwordCharLower + val.Int64())

		// if the character is in the exclusion list, continue
		if idx := strings.IndexAny(string(char), passwordCharExclude); idx > -1 {
			continue
		}

		password[i] = char
		i++
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
	ctx := context.TODO()
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
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
func CreateUserSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster, username, password string) error {
	secretName := crv1.UserSecretName(cluster, username)

	if err := CreateSecret(clientset, cluster.Name, secretName, username, password,
		cluster.Namespace, GetCustomLabels(cluster)); err != nil {
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
func UpdateUserSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster, username, password string) error {
	ctx := context.TODO()
	secretName := crv1.UserSecretName(cluster, username)

	// see if the secret already exists
	secret, err := clientset.CoreV1().Secrets(cluster.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	// if this returns an error and it's not the "not found" error, return
	// However, if it is the "not found" error, treat this as creating the user
	// secret
	if err != nil {
		if !kubeapi.IsNotFound(err) {
			return err
		}

		return CreateUserSecret(clientset, cluster, username, password)
	}

	// update the value of "password"
	secret.Data["password"] = []byte(password)

	_, err = clientset.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}
