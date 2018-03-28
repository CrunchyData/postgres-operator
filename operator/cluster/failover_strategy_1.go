// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	//crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	//"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	//"k8s.io/api/extensions/v1beta1"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	//"k8s.io/apimachinery/pkg/api/meta"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"text/template"
)

// AddCluster ...
func (r Strategy1) Failover(clientset *kubernetes.Clientset, client *rest.RESTClient, clusterName, target, namespace string) error {

	var podName string
	var err error

	log.Info("strategy 1 Failover called on " + clusterName + " target is " + target)

	if target == "" {
		log.Debug("failover target not set, will use best estimate")
		podName, err = util.GetBestTarget(clientset, clusterName, namespace)
	} else {
		podName, err = util.GetPodName(clientset, target, namespace)
	}
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("best pod to failover to is " + podName)

	//delete the primary deployment
	//trigger the failover on the replica

	return err

}
