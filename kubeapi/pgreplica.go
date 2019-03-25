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

// GetpgreplicasBySelector gets a list of pgreplicas by selector
func GetpgreplicasBySelector(client *rest.RESTClient, replicaList *crv1.PgreplicaList, selector, namespace string) error {

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
		Resource(crv1.PgreplicaResourcePlural).
		Namespace(namespace).
		Param("labelSelector", myselector.String()).
		Do().
		Into(replicaList)
	if err != nil {
		log.Error("error getting list of replicas " + err.Error())
	}

	return err
}

// Getpgreplicas gets a list of pgreplicas
func Getpgreplicas(client *rest.RESTClient, replicaList *crv1.PgreplicaList, namespace string) error {

	err := client.Get().
		Resource(crv1.PgreplicaResourcePlural).
		Namespace(namespace).
		Do().Into(replicaList)
	if err != nil {
		log.Error("error getting list of replicas " + err.Error())
		return err
	}

	return err
}

// Getpgreplica gets a pgreplica by name
func Getpgreplica(client *rest.RESTClient, replica *crv1.Pgreplica, name, namespace string) (bool, error) {

	err := client.Get().
		Resource(crv1.PgreplicaResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(replica)
	if kerrors.IsNotFound(err) {
		log.Debugf("replica %s not found", name)
		return false, err
	}
	if err != nil {
		log.Error("error getting replica " + err.Error())
		return false, err
	}

	return true, err
}

// Deletepgreplica deletes pgreplica by name
func Deletepgreplica(client *rest.RESTClient, name, namespace string) error {

	err := client.Delete().
		Resource(crv1.PgreplicaResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().
		Error()
	if err != nil {
		log.Error("error deleting pgreplica " + err.Error())
		return err
	}

	return err
}

// Createpgreplica creates a pgreplica
func Createpgreplica(client *rest.RESTClient, replica *crv1.Pgreplica, namespace string) error {

	result := crv1.Pgreplica{}

	err := client.Post().
		Resource(crv1.PgreplicaResourcePlural).
		Namespace(namespace).
		Body(replica).
		Do().
		Into(&result)
	if err != nil {
		log.Error("error creating pgreplica " + err.Error())
	}

	return err
}

// Updatepgreplica updates a pgreplica
func Updatepgreplica(client *rest.RESTClient, replica *crv1.Pgreplica, name, namespace string) error {

	err := client.Put().
		Name(name).
		Namespace(namespace).
		Resource(crv1.PgreplicaResourcePlural).
		Body(replica).
		Do().
		Error()
	if err != nil {
		log.Error("error updating pgreplica " + err.Error())
	}

	log.Debugf("updated pgreplica %s", replica.Name)
	return err
}
