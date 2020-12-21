package apiserver

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"errors"
	"fmt"
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ErrMessageLimitInvalid indicates that a limit is lower than the request
	ErrMessageLimitInvalid = `limit %q is lower than the request %q`
	// ErrMessagePVCSize provides a standard error message when a PVCSize is not
	// specified to the Kubernetes stnadard
	ErrMessagePVCSize = `could not parse PVC size "%s": %s (hint: try a value like "1Gi")`
	// ErrMessageReplicas provides a standard error message when the count of
	// replicas is incorrect
	ErrMessageReplicas = `must have at least %d replica(s)`
)

var (
	backrestStorageTypes = []string{"local", "s3"}
	// ErrDBContainerNotFound is an error that indicates that a "database" container
	// could not be found in a specific pod
	ErrDBContainerNotFound = errors.New("\"database\" container not found in pod")
	// ErrStandbyNotAllowed contains the error message returned when an API call is not
	// permitted because it involves a cluster that is in standby mode
	ErrStandbyNotAllowed = errors.New("Action not permitted because standby mode is enabled")

	// ErrMethodNotAllowed represents the error that is thrown when a feature is disabled within the
	// current Operator install
	ErrMethodNotAllowed = errors.New("This method has is not allowed in the current PostgreSQL " +
		"Operator installation")
)

func CreateRMDataTask(clusterName, replicaName, taskName string, deleteBackups, deleteData, isReplica, isBackup bool, ns, clusterPGHAScope string) error {
	ctx := context.TODO()
	var err error

	// create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = taskName
	spec.TaskType = crv1.PgtaskDeleteData

	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_DELETE_DATA] = strconv.FormatBool(deleteData)
	spec.Parameters[config.LABEL_DELETE_BACKUPS] = strconv.FormatBool(deleteBackups)
	spec.Parameters[config.LABEL_IS_REPLICA] = strconv.FormatBool(isReplica)
	spec.Parameters[config.LABEL_IS_BACKUP] = strconv.FormatBool(isBackup)
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[config.LABEL_REPLICA_NAME] = replicaName
	spec.Parameters[config.LABEL_PGHA_SCOPE] = clusterPGHAScope

	newInstance := &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_RMDATA] = "true"

	_, err = Clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, newInstance, metav1.CreateOptions{})
	if err != nil {
		log.Error(err)
		return err
	}

	return err
}

func GetBackrestStorageTypes() []string {
	return backrestStorageTypes
}

// IsValidPVC determines if a PVC with the name provided exits
func IsValidPVC(pvcName, ns string) bool {
	ctx := context.TODO()

	pvc, err := Clientset.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return false
	}
	if err != nil {
		log.Error(err)
		return false
	}
	return pvc != nil
}

// ValidateResourceRequestLimit validates that a Kubernetes Requests/Limit pair
// is valid, both by validating the values are valid quantity values, and then
// by checking that the limit >= request. This also needs to check against the
// configured values for a request, which must be provided as a value
func ValidateResourceRequestLimit(request, limit string, defaultQuantity resource.Quantity) error {
	// ensure that the request/limit are valid, as this simplifies the rest of
	// this code. We know that the defaultRequest is already valid at this point,
	// as otherwise the Operator will fail to load
	if err := ValidateQuantity(request); err != nil {
		return err
	}

	if err := ValidateQuantity(limit); err != nil {
		return err
	}

	// parse the quantities so we can compare
	requestQuantity, _ := resource.ParseQuantity(request)
	limitQuantity, _ := resource.ParseQuantity(limit)

	if requestQuantity.IsZero() {
		requestQuantity = defaultQuantity
	}

	// if limit and request are nonzero and the limit is less than the request,
	// error
	if !limitQuantity.IsZero() && !requestQuantity.IsZero() && limitQuantity.Cmp(requestQuantity) == -1 {
		return fmt.Errorf(ErrMessageLimitInvalid, limitQuantity.String(), requestQuantity.String())
	}

	return nil
}

// ValidateQuantity runs the Kubernetes "ParseQuantity" function on a string
// and determine whether or not it is a valid quantity object. Returns an error
// if it is invalid, along with the error message.
//
// If it is empty, it returns no error
//
// See: https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go
func ValidateQuantity(quantity string) error {
	if quantity == "" {
		return nil
	}

	_, err := resource.ParseQuantity(quantity)
	return err
}

// FindStandbyClusters takes a list of pgcluster structs and returns a slice containing the names
// of those clusters that are in standby mode as indicated by whether or not the standby prameter
// in the pgcluster spec is true.
func FindStandbyClusters(clusterList crv1.PgclusterList) (standbyClusters []string) {
	standbyClusters = make([]string, 0)
	for _, cluster := range clusterList.Items {
		if cluster.Spec.Standby {
			standbyClusters = append(standbyClusters, cluster.Name)
		}
	}
	return
}

// PGClusterListHasStandby determines if a PgclusterList has any standby clusters, specifically
// returning "true" if one or more standby clusters exist, along with a slice of strings
// containing the names of the clusters in standby mode
func PGClusterListHasStandby(clusterList crv1.PgclusterList) (bool, []string) {
	standbyClusters := FindStandbyClusters(clusterList)
	return len(FindStandbyClusters(clusterList)) > 0, standbyClusters
}
