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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"strconv"
)

// Strategy ....
type Strategy interface {
	Scale(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgreplica, string, string, *crv1.Pgcluster) error
	AddCluster(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, string, string) error
	Failover(*kubernetes.Clientset, *rest.RESTClient, string, *crv1.Pgtask, string, *rest.Config) error
	CreateReplica(string, *kubernetes.Clientset, *crv1.Pgcluster, string, string, string) error
	DeleteCluster(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, string) error

	MinorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
	MajorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
	MajorUpgradeFinalize(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
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
	Name               string
	ClusterName        string
	Port               string
	PgMode             string
	CCPImagePrefix     string
	CCPImageTag        string
	Database           string
	OperatorLabels     string
	DataPathOverride   string
	ArchiveMode        string
	ArchivePVCName     string
	ArchiveTimeout     string
	PVCName            string
	BackupPVCName      string
	BackupPath         string
	RootSecretName     string
	UserSecretName     string
	PrimarySecretName  string
	SecurityContext    string
	ContainerResources string
	NodeSelector       string
	ConfVolume         string
	CollectAddon       string
	//next 2 are for the replica deployment only
	Replicas    string
	PrimaryHost string
}

// ReplicaSuffix ...
const ReplicaSuffix = "-replica"

var strategyMap map[string]Strategy

func init() {
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

	var pvcName string

	_, found, err := kubeapi.GetPVC(clientset, cl.Spec.Name, namespace)
	if found {
		log.Debugf("pvc [%s] already present from previous cluster with this same name, will not recreate\n", cl.Spec.Name)
		pvcName = cl.Spec.Name
	} else {
		pvcName, err = pvc.CreatePVC(clientset, &cl.Spec.PrimaryStorage, cl.Spec.Name, cl.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debug("created primary pvc [" + pvcName + "]")
	}

	if cl.Spec.UserLabels["archive"] == "true" {
		_, err := pvc.CreatePVC(clientset, &cl.Spec.PrimaryStorage, cl.Spec.Name+"-xlog", cl.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}

	log.Debug("creating Pgcluster object strategy is [" + cl.Spec.Strategy + "]")

	var err1, err2, err3 error
	if cl.Spec.SecretFrom != "" {
		_, cl.Spec.RootPassword, err1 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SecretFrom+crv1.RootSecretSuffix)
		_, cl.Spec.PrimaryPassword, err3 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SecretFrom+crv1.PrimarySecretSuffix)
		_, cl.Spec.Password, err2 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SecretFrom+crv1.UserSecretSuffix(cl.Spec.User))
		if err1 != nil || err2 != nil || err3 != nil {
			log.Error("error getting secrets using SecretFrom " + cl.Spec.SecretFrom)
			return
		}
	}

	var userPassword string
	_, _, userPassword, err = util.CreateDatabaseSecrets(clientset, client, cl, namespace)
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

	//add pgpool deployment if requested
	if cl.Spec.UserLabels["crunchy-pgpool"] == "true" {
		//generate a secret for pgpool using the user credential
		secretName := cl.Spec.Name + "-pgpool-secret"
		primaryName := cl.Spec.Name
		replicaName := cl.Spec.Name + "-replica"
		user := "testuser"
		if cl.Spec.User != "" {
			user = cl.Spec.User
		}
		err = CreatePgpoolSecret(clientset, primaryName, replicaName, primaryName, secretName, user, userPassword, namespace)
		if err != nil {
			log.Error(err)
			return
		}
		//create the pgpool deployment using that credential
		AddPgpool(clientset, client, cl, namespace, secretName)
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

	err := kubeapi.Deletepgupgrade(client, cl.Spec.Name, namespace)
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

// ScaleBase ...
func ScaleBase(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace string) {
	var err error

	if replica.Spec.Status == crv1.UpgradeCompletedStatus {
		log.Warn("crv1 pgreplica " + replica.Spec.Name + " is already marked complete, will not recreate")
		return
	}

	//get the pgcluster CRD to base the replica off of
	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(client, &cluster,
		replica.Spec.ClusterName, namespace)
	if err != nil {
		return
	}

	//create the PVC
	pvcName, err := pvc.CreatePVC(clientset, &replica.Spec.ReplicaStorage, replica.Spec.Name, cluster.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	if cluster.Spec.UserLabels["archive"] == "true" {
		_, err := pvc.CreatePVC(clientset, &cluster.Spec.PrimaryStorage, replica.Spec.Name+"-xlog", cluster.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}

	log.Debug("created replica pvc [" + pvcName + "]")

	//update the replica CRD pvcname
	err = util.Patch(client, "/spec/replicastorage/name", pvcName, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	log.Debug("creating Pgreplica object strategy is [" + cluster.Spec.Strategy + "]")

	if cluster.Spec.Strategy == "" {
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cluster.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for replica creation" + cluster.Spec.Strategy)
		return
	}

	//create the replica service if it doesnt exist
	serviceName := replica.Spec.ClusterName + "-replica"
	serviceFields := ServiceTemplateFields{
		Name:        serviceName,
		ClusterName: replica.Spec.ClusterName,
		Port:        cluster.Spec.Port,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	//instantiate the replica
	strategy.Scale(clientset, client, replica, namespace, pvcName, &cluster)

	//update the replica CRD status
	err = util.Patch(client, "/spec/status", crv1.UpgradeCompletedStatus, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}

}
