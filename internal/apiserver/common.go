package apiserver

/*
Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"strings"

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

// ValidateBackrestStorageTypeForCommand determines if a submitted pgBackRest
// storage value can be used as part of a pgBackRest operation based upon the
// storage types used by the PostgreSQL cluster itself
func ValidateBackrestStorageTypeForCommand(cluster *crv1.Pgcluster, storageTypeStr string) error {
	// first, parse the submitted storage type string to see what we're up against
	storageTypes, err := crv1.ParseBackrestStorageTypes(storageTypeStr)

	// if there is an error parsing the string and it's not due to the string
	// being empty, return the error
	// if it is due to an empty string, then return so that the defaults will be
	// used
	if err != nil {
		if errors.Is(err, crv1.ErrStorageTypesEmpty) {
			return nil
		}
		return err
	}

	// there can only be one storage type used for a command (for now), so ensure
	// this condition is sated
	if len(storageTypes) > 1 {
		return fmt.Errorf("you can only select one storage type")
	}

	// a special case: the list of storage types is empty. if this is not a posix
	// (or local) storage type, then return an error. Otherwise, we can exit here.
	if len(cluster.Spec.BackrestStorageTypes) == 0 {
		if !(storageTypes[0] == crv1.BackrestStorageTypePosix || storageTypes[0] == crv1.BackrestStorageTypeLocal) {
			return fmt.Errorf("%w: choices are: posix", crv1.ErrInvalidStorageType)
		}
		return nil
	}

	// now, see if the select storage type is available in the list of storage
	// types on the cluster
	ok := false
	for _, storageType := range cluster.Spec.BackrestStorageTypes {
		switch storageTypes[0] {
		default:
			ok = ok || (storageType == storageTypes[0])
		case crv1.BackrestStorageTypePosix, crv1.BackrestStorageTypeLocal:
			ok = ok || (storageType == crv1.BackrestStorageTypePosix || storageType == crv1.BackrestStorageTypeLocal)
		}
	}

	if !ok {
		choices := make([]string, len(cluster.Spec.BackrestStorageTypes))
		for i, storageType := range cluster.Spec.BackrestStorageTypes {
			choices[i] = string(storageType)
		}

		return fmt.Errorf("%w: choices are: %s",
			crv1.ErrInvalidStorageType, strings.Join(choices, " "))
	}

	return nil
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
