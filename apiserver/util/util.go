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
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSecretPassword ...
func GetSecretPassword(db, suffix, Namespace string) (string, error) {

	var err error

	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + db}
	secrets, err := apiserver.Clientset.Core().Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return "", err
	}

	log.Debug("secrets for " + db)
	secretName := db + suffix
	for _, s := range secrets.Items {
		log.Debug("secret : " + s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretName {
			log.Debug("pgprimary password found")
			return string(s.Data["password"][:]), err
		}
	}

	log.Error("primary secret not found for " + db)
	return "", errors.New("primary secret not found for " + db)

}
