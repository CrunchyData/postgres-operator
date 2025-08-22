// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// storeDesiredRequest saves the appropriate request value to the PostgresCluster
// status. If the value has grown, create an Event.
func (r *Reconciler) storeDesiredRequest(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	volumeType, host, desiredRequest, desiredRequestBackup string,
) string {
	var current resource.Quantity
	var previous resource.Quantity
	var err error
	log := logging.FromContext(ctx)

	// Parse the desired request from the cluster's status.
	if desiredRequest != "" {
		current, err = resource.ParseQuantity(desiredRequest)
		if err != nil {
			log.Error(err, "Unable to parse "+volumeType+" volume request from status ("+
				desiredRequest+") for "+cluster.Name+"/"+host)
			// If there was an error parsing the value, treat as unset (equivalent to zero).
			desiredRequest = ""
			current, _ = resource.ParseQuantity("")

		}
	}

	// Parse the desired request from the status backup.
	if desiredRequestBackup != "" {
		previous, err = resource.ParseQuantity(desiredRequestBackup)
		if err != nil {
			log.Error(err, "Unable to parse "+volumeType+" volume request from status backup ("+
				desiredRequestBackup+") for "+cluster.Name+"/"+host)
			// If there was an error parsing the value, treat as unset (equivalent to zero).
			desiredRequestBackup = ""
			previous, _ = resource.ParseQuantity("")

		}
	}

	// determine if the appropriate volume limit is set
	limitSet := limitIsSet(cluster, volumeType, host)

	if limitSet && current.Value() > previous.Value() {
		r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "VolumeAutoGrow",
			"%s volume expansion to %v requested for %s/%s.",
			volumeType, current.String(), cluster.Name, host)
	}

	// If the desired size was not observed, update with previously stored value.
	// This can happen in scenarios where the annotation on the Pod is missing
	// such as when the cluster is shutdown or a Pod is in the middle of a restart.
	if desiredRequest == "" {
		desiredRequest = desiredRequestBackup
	}

	return desiredRequest
}

// limitIsSet determines if the limit is set for a given volume type and returns
// a corresponding boolean value
func limitIsSet(cluster *v1beta1.PostgresCluster, volumeType, instanceSetName string) bool {

	var limitSet bool

	switch {

	// Cycle through the instance sets to ensure the correct limit is identified.
	case volumeType == "pgData":
		for _, specInstance := range cluster.Spec.InstanceSets {
			if specInstance.Name == instanceSetName {
				limitSet = !specInstance.DataVolumeClaimSpec.Resources.Limits.Storage().IsZero()
			}
		}

	// VolumeType for the repository host volumes should be in the form 'repoN'
	// where N is 1-4. As above, cycle through any defined repositories and ensure
	// the correct limit is identified.
	case strings.HasPrefix(volumeType, "repo"):
		for _, specRepo := range cluster.Spec.Backups.PGBackRest.Repos {
			if specRepo.Name == volumeType && specRepo.Volume != nil {
				limitSet = !specRepo.Volume.VolumeClaimSpec.Resources.Limits.Storage().IsZero()
			}
		}
	}
	// TODO: Add case for pgWAL

	return limitSet

}

// setVolumeSize compares the potential sizes from the cluster status, volume request
// and volume limit and sets the appropriate current value.
func (r *Reconciler) setVolumeSize(ctx context.Context, cluster *v1beta1.PostgresCluster,
	spec *corev1.PersistentVolumeClaimSpec, volumeType, host string) {

	log := logging.FromContext(ctx)

	// Store the limit for this instance set. This value will not change below.
	volumeLimitFromSpec := spec.Resources.Limits.Storage()

	// This value will capture our desired update.
	volumeRequestSize := spec.Resources.Requests.Storage()

	// A limit of 0 is ignorned, so the volume request is used.
	if volumeLimitFromSpec.IsZero() {
		return
	}

	// If the request value is greater than the set limit, use the limit and issue
	// a warning event.
	if volumeRequestSize.Value() > volumeLimitFromSpec.Value() {
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "VolumeRequestOverLimit",
			"%s volume request (%v) for %s/%s is greater than set limit (%v). Limit value will be used.",
			volumeType, volumeRequestSize, cluster.Name, host, volumeLimitFromSpec)

		spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: *resource.NewQuantity(volumeLimitFromSpec.Value(), resource.BinarySI),
		}
		// Otherwise, if the feature gate is not enabled, do not autogrow.
	} else if feature.Enabled(ctx, feature.AutoGrowVolumes) {

		// determine the appropriate volume request based on what's set in the status
		if dpv, err := getDesiredVolumeSize(
			cluster, volumeType, host, volumeRequestSize,
		); err != nil {
			log.Error(err, "For "+cluster.Name+"/"+host+
				": Unable to parse "+volumeType+" volume request: "+dpv)
		}

		// If the volume request size is greater than or equal to the limit and the
		// limit is not zero, update the request size to the limit value.
		// If the user manually requests a lower limit that is smaller than the current
		// or requested volume size, it will be ignored in favor of the limit value.
		if volumeRequestSize.Value() >= volumeLimitFromSpec.Value() {

			r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "VolumeLimitReached",
				"%s volume(s) for %s/%s are at size limit (%v).", volumeType,
				cluster.Name, host, volumeLimitFromSpec)

			// If the volume size request is greater than the limit, issue an
			// additional event warning.
			if volumeRequestSize.Value() > volumeLimitFromSpec.Value() {
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "DesiredVolumeAboveLimit",
					"The desired size (%v) for the %s/%s %s volume(s) is greater than the size limit (%v).",
					volumeRequestSize, cluster.Name, host, volumeType, volumeLimitFromSpec)
			}

			volumeRequestSize = volumeLimitFromSpec
		}
		spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: *resource.NewQuantity(volumeRequestSize.Value(), resource.BinarySI),
		}
	}
}

// getDesiredVolumeSize compares the volume request size to the suggested autogrow
// size stored in the status and updates the value when the status value is larger.
func getDesiredVolumeSize(cluster *v1beta1.PostgresCluster,
	volumeType, instanceSpecName string,
	volumeRequestSize *resource.Quantity) (string, error) {

	switch {
	case volumeType == "pgData":
		for i := range cluster.Status.InstanceSets {
			if instanceSpecName == cluster.Status.InstanceSets[i].Name {
				for _, dpv := range cluster.Status.InstanceSets[i].DesiredPGDataVolume {
					if dpv != "" {
						desiredRequest, err := resource.ParseQuantity(dpv)
						if err == nil {
							if desiredRequest.Value() > volumeRequestSize.Value() {
								*volumeRequestSize = desiredRequest
							}
						} else {
							return dpv, err
						}
					}
				}
			}
		}

	// VolumeType for the repository host volumes should be in the form 'repoN'
	// where N is 1-4. As above, cycle through any defined repositories and ensure
	// the correct limit is identified.
	case strings.HasPrefix(volumeType, "repo"):
		if cluster.Status.PGBackRest == nil {
			return "", errors.New("PostgresCluster.Status.PGBackRest is nil")
		}
		for i := range cluster.Status.PGBackRest.Repos {
			if volumeType == cluster.Status.PGBackRest.Repos[i].Name {
				dpv := cluster.Status.PGBackRest.Repos[i].DesiredRepoVolume
				if dpv != "" {
					desiredRequest, err := resource.ParseQuantity(dpv)
					if err == nil {
						if desiredRequest.Value() > volumeRequestSize.Value() {
							*volumeRequestSize = desiredRequest
						}
					} else {
						return dpv, err
					}
				}
			}
		}
	}
	// TODO: Add case for pgWAL
	return "", nil
}
