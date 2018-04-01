package kubeapi

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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
)

// GetpgingestsBySelector gets a list of pgingests by selector
func GetpgingestsBySelector(client *rest.RESTClient, ingestList *crv1.PgingestList, selector, namespace string) error {

	var err error

	myselector := labels.Everything()

	if selector != "" {
		myselector, err = labels.Parse(selector)
		if err != nil {
			log.Error("could not parse selector value ")
			log.Error(err)
			return err
		}
	}

	log.Debug("myselector is " + myselector.String())

	err = client.Get().
		Resource(crv1.PgingestResourcePlural).
		Namespace(namespace).
		Param("labelSelector", myselector.String()).
		Do().
		Into(ingestList)
	if err != nil {
		log.Error("error getting list of ingests " + err.Error())
	}

	return err
}

// Getpgingests gets a list of pgingests
func Getpgingests(client *rest.RESTClient, ingestList *crv1.PgingestList, namespace string) error {

	err := client.Get().
		Resource(crv1.PgingestResourcePlural).
		Namespace(namespace).
		Do().Into(ingestList)
	if err != nil {
		log.Error("error getting list of ingests " + err.Error())
		return err
	}

	return err
}

// Getpgingest gets a pgingest by name
func Getpgingest(client *rest.RESTClient, ingest *crv1.Pgingest, name, namespace string) (bool, error) {

	err := client.Get().
		Resource(crv1.PgingestResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(ingest)
	if kerrors.IsNotFound(err) {
		log.Debug("ingest " + name + " not found")
		return false, err
	}
	if err != nil {
		log.Error("error getting ingest " + err.Error())
		return false, err
	}

	return true, err
}

// DeleteAllpgingest deletes all pgingests
func DeleteAllpgingest(client *rest.RESTClient, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgingestResourcePlural).
		Namespace(namespace).
		Do().
		Error()
	if err != nil {
		log.Error("error deleting all pgingest " + err.Error())
		return err
	}

	return err
}

// Deletepgingest deletes pgingest by name
func Deletepgingest(client *rest.RESTClient, name, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgingestResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().
		Error()
	if err != nil {
		log.Error("error deleting pgingest " + err.Error())
		return err
	}

	log.Debug("deleted pgingest " + name)
	return err
}

// Createpgingest creates a pgingest
func Createpgingest(client *rest.RESTClient, ingest *crv1.Pgingest, namespace string) error {

	result := crv1.Pgingest{}

	err := client.Post().
		Resource(crv1.PgingestResourcePlural).
		Namespace(namespace).
		Body(ingest).
		Do().
		Into(&result)
	if err != nil {
		log.Error("error creating pgingest " + err.Error())
	}

	return err
}
