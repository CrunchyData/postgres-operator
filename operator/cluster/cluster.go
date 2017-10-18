// Package cluster holds the cluster TPR logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/rand"
	"strconv"
	"time"
)

// Strategy ....
type Strategy interface {
	AddCluster(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, string, string) error
	CreateReplica(string, *kubernetes.Clientset, *crv1.Pgcluster, string, string, string, bool) error
	DeleteCluster(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, string) error

	MinorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
	MajorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
	MajorUpgradeFinalize(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
	PrepareClone(*kubernetes.Clientset, *rest.RESTClient, string, *crv1.Pgcluster, string) error
	UpdatePolicyLabels(*kubernetes.Clientset, string, string, map[string]string) error
}

// ServiceTemplateFields ...
type ServiceTemplateFields struct {
	Name        string
	ClusterName string
	Port        string
}

// DeploymentTemplateFields ...
type DeploymentTemplateFields struct {
	Name              string
	ClusterName       string
	Port              string
	CCPImageTag       string
	Database          string
	OperatorLabels    string
	DataPathOverride  string
	PVCName           string
	BackupPVCName     string
	BackupPath        string
	RootSecretName    string
	UserSecretName    string
	PrimarySecretName string
	SecurityContext   string
	NodeSelector      string
	//next 2 are for the replica deployment only
	Replicas    string
	PrimaryHost string
}

// ReplicaSuffix ...
const ReplicaSuffix = "-replica"

var strategyMap map[string]Strategy

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
	strategyMap = make(map[string]Strategy)
	strategyMap["1"] = Strategy1{}
}

// AddClusterBase ...
func AddClusterBase(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) {
	var err error

	if cl.Spec.Status == crv1.UpgradeCompletedStatus {
		log.Warn("crv1 pgcluster " + cl.Spec.ClusterName + " is already marked complete, will not recreate")
		return
	}

	pvcName, err := pvc.CreatePVC(clientset, cl.Spec.Name, &cl.Spec.PrimaryStorage, namespace)
	log.Debug("created primary pvc [" + pvcName + "]")

	log.Debug("creating Pgcluster object strategy is [" + cl.Spec.Strategy + "]")

	var err1, err2, err3 error
	if cl.Spec.SecretFrom != "" {
		cl.Spec.RootPassword, err1 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SecretFrom+crv1.RootSecretSuffix)
		cl.Spec.Password, err2 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SecretFrom+crv1.UserSecretSuffix)
		cl.Spec.PrimaryPassword, err3 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SecretFrom+crv1.PrimarySecretSuffix)
		if err1 != nil || err2 != nil || err3 != nil {
			log.Error("error getting secrets using SecretFrom " + cl.Spec.SecretFrom)
			return
		}
	}

	err = util.CreateDatabaseSecrets(clientset, client, cl, namespace)
	if err != nil {
		log.Error("error in create secrets " + err.Error())
		return
	}

	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
		return
	}

	//replaced with ccpimagetag instead of pg version
	//setFullVersion(client, cl, namespace)

	strategy.AddCluster(clientset, client, cl, namespace, pvcName)

	err = util.Patch(client, "/spec/status", crv1.UpgradeCompletedStatus, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}
	err = util.Patch(client, "/spec/PrimaryStorage/name", pvcName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

}

// DeleteClusterBase ...
func DeleteClusterBase(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) {

	log.Debug("deleteCluster called with strategy " + cl.Spec.Strategy)

	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
		return
	}

	util.DeleteDatabaseSecrets(clientset, cl.Spec.Name, namespace)

	strategy.DeleteCluster(clientset, client, cl, namespace)

	err := client.Delete().
		Resource(crv1.PgupgradeResourcePlural).
		Namespace(namespace).
		Name(cl.Spec.Name).
		Do().
		Error()
	if err == nil {
		log.Info("deleted pgupgrade " + cl.Spec.Name)
	} else if kerrors.IsNotFound(err) {
		log.Info("will not delete pgupgrade, not found for " + cl.Spec.Name)
	} else {
		log.Error("error deleting pgupgrade " + cl.Spec.Name + err.Error())
	}

}

// AddUpgradeBase ...
func AddUpgradeBase(clientset *kubernetes.Clientset, client *rest.RESTClient, upgrade *crv1.Pgupgrade, namespace string, cl *crv1.Pgcluster) error {
	var err error

	//get the strategy to use
	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default cluster strategy")
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster upgrade" + cl.Spec.Strategy)
		return err
	}

	//invoke the strategy
	if upgrade.Spec.UpgradeType == "minor" {
		err = strategy.MinorUpgrade(clientset, client, cl, upgrade, namespace)
		if err == nil {
			err = util.Patch(client, "/spec/upgradestatus", crv1.UpgradeCompletedStatus, crv1.PgupgradeResourcePlural, upgrade.Spec.Name, namespace)
		}
	} else if upgrade.Spec.UpgradeType == "major" {
		err = strategy.MajorUpgrade(clientset, client, cl, upgrade, namespace)
	} else {
		log.Error("invalid UPGRADE_TYPE requested for cluster upgrade" + upgrade.Spec.UpgradeType)
		return err
	}
	if err == nil {
		log.Info("updating the pg version after cluster upgrade")
		fullVersion := upgrade.Spec.CCPImageTag
		err = util.Patch(client, "/spec/ccpimagetag", fullVersion, crv1.PgclusterResourcePlural, upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error(err.Error())
		}
	}

	return err

}

// ScaleCluster ...
func ScaleCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, oldcluster *crv1.Pgcluster, namespace string) {

	//log.Debug("updateCluster on pgcluster called..something changed")

	if oldcluster.Spec.Replicas != cl.Spec.Replicas {
		log.Debug("detected change to Replicas for " + cl.Spec.Name + " from " + oldcluster.Spec.Replicas + " to " + cl.Spec.Replicas)
		oldCount, err := strconv.Atoi(oldcluster.Spec.Replicas)
		if err != nil {
			log.Error(err)
			return
		}
		newCount, err := strconv.Atoi(cl.Spec.Replicas)
		if err != nil {
			log.Error(err)
			return
		}
		if oldCount > newCount {
			log.Error("scale down is not implemented yet")
			return
		}
		newReps := newCount - oldCount
		if newReps > 0 {
			serviceName := cl.Spec.Name + "-replica"
			ScaleReplicasBase(serviceName, clientset, cl, newReps, namespace)
		} else {
			log.Error("scale to the same number does nothing")
		}
	}

}

// ScaleReplicasBase ...
func ScaleReplicasBase(serviceName string, clientset *kubernetes.Clientset, cl *crv1.Pgcluster, newReplicas int, namespace string) {

	//create the service if it doesn't exist
	serviceFields := ServiceTemplateFields{
		Name:        serviceName,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
	}

	err := CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	//get the strategy to use
	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default cluster strategy")
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster upgrade" + cl.Spec.Strategy)
		return
	}

	log.Debug("scale up called ")

	for i := 0; i < newReplicas; i++ {
		//generate a unique name suffix
		uniqueName := RandStringBytesRmndr(4)
		depName := cl.Spec.Name + "-replica-" + uniqueName

		//create a PVC
		pvcName, err := pvc.CreatePVC(clientset, depName, &cl.Spec.ReplicaStorage, namespace)
		if err != nil {
			log.Error(err)
			return
		}
		strategy.CreateReplica(serviceName, clientset, cl, depName, pvcName, namespace, false)
	}
}

// RandStringBytesRmndr ...
func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
