package kubeapi

/*
 Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	log "github.com/sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
)

// GetpgclustersBySelector gets a list of pgclusters by selector
func GetpgclustersBySelector(client *rest.RESTClient, clusterList *crv1.PgclusterList, selector, namespace string) error {

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

	log.Debugf("myselector is %s", myselector.String())

	err = client.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Param("labelSelector", myselector.String()).
		Do().
		Into(clusterList)
	if kerrors.IsNotFound(err) {
		log.Debugf("clusters for %s not found", myselector.String())
	}
	if err != nil {
		log.Error("error getting list of clusters " + err.Error())
	}

	return err
}

// Getpgclusters gets a list of pgclusters
func Getpgclusters(client *rest.RESTClient, clusterList *crv1.PgclusterList, namespace string) error {

	err := client.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Do().Into(clusterList)
	if err != nil {
		log.Error("error getting list of clusters " + err.Error())
		return err
	}

	return err
}

// Getpgcluster gets a pgcluster by name
func Getpgcluster(client *rest.RESTClient, cluster *crv1.Pgcluster, name, namespace string) (bool, error) {

	err := client.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(cluster)
	if kerrors.IsNotFound(err) {
		log.Debugf("cluster %s not found", name)
		return false, err
	}
	if err != nil {
		log.Error("error getting cluster " + err.Error())
		return false, err
	}

	return true, err
}

// Deletepgcluster deletes pgcluster by name
func Deletepgcluster(client *rest.RESTClient, name, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().
		Error()
	if err != nil {
		log.Error("error deleting pgcluster " + err.Error())
		return err
	}

	log.Debugf("deleted pgcluster %s", name)
	return err
}

// Createpgcluster creates a pgcluster
func Createpgcluster(client *rest.RESTClient, cluster *crv1.Pgcluster, namespace string) error {

	result := crv1.Pgcluster{}

	err := client.Post().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Body(cluster).
		Do().
		Into(&result)
	if err != nil {
		log.Error("error creating pgcluster " + err.Error())
	}

	return err
}

// Updatepgcluster updates a pgcluster
func Updatepgcluster(client *rest.RESTClient, cluster *crv1.Pgcluster, name, namespace string) error {

	err := client.Put().
		Name(name).
		Namespace(namespace).
		Resource(crv1.PgclusterResourcePlural).
		Body(cluster).
		Do().
		Error()
	if err != nil {
		log.Error("error updating pgcluster " + err.Error())
	}

	log.Debugf("updated pgcluster %s", cluster.Name)
	return err
}
