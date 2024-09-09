// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/initialize"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type TestBridgeClient struct {
	ApiKey          string                                       `json:"apiKey,omitempty"`
	TeamId          string                                       `json:"teamId,omitempty"`
	Clusters        []*bridge.ClusterApiResource                 `json:"clusters,omitempty"`
	ClusterRoles    []*bridge.ClusterRoleApiResource             `json:"clusterRoles,omitempty"`
	ClusterStatuses map[string]*bridge.ClusterStatusApiResource  `json:"clusterStatuses,omitempty"`
	ClusterUpgrades map[string]*bridge.ClusterUpgradeApiResource `json:"clusterUpgrades,omitempty"`
}

func (tbc *TestBridgeClient) ListClusters(ctx context.Context, apiKey, teamId string) ([]*bridge.ClusterApiResource, error) {

	if apiKey == tbc.ApiKey && teamId == tbc.TeamId {
		return tbc.Clusters, nil
	}

	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) UpgradeCluster(ctx context.Context, apiKey, id string, clusterRequestPayload *bridge.PostClustersUpgradeRequestPayload,
) (*bridge.ClusterUpgradeApiResource, error) {
	// look for cluster
	var desiredCluster *bridge.ClusterApiResource
	clusterFound := false
	for _, cluster := range tbc.Clusters {
		if cluster.ID == id {
			desiredCluster = cluster
			clusterFound = true
		}
	}
	if !clusterFound {
		return nil, errors.New("cluster not found")
	}

	// happy path
	if apiKey == tbc.ApiKey {
		result := &bridge.ClusterUpgradeApiResource{
			ClusterID: id,
			Team:      tbc.TeamId,
		}
		if clusterRequestPayload.Plan != desiredCluster.Plan {
			result.Operations = []*v1beta1.UpgradeOperation{
				{
					Flavor:       "maintenance",
					StartingFrom: "",
					State:        "in_progress",
				},
			}
		} else if clusterRequestPayload.PostgresVersion != intstr.FromInt(desiredCluster.MajorVersion) {
			result.Operations = []*v1beta1.UpgradeOperation{
				{
					Flavor:       "major_version_upgrade",
					StartingFrom: "",
					State:        "in_progress",
				},
			}
		} else if clusterRequestPayload.Storage != desiredCluster.Storage {
			result.Operations = []*v1beta1.UpgradeOperation{
				{
					Flavor:       "resize",
					StartingFrom: "",
					State:        "in_progress",
				},
			}
		}
		return result, nil
	}
	// sad path
	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) UpgradeClusterHA(ctx context.Context, apiKey, id, action string,
) (*bridge.ClusterUpgradeApiResource, error) {
	// look for cluster
	var desiredCluster *bridge.ClusterApiResource
	clusterFound := false
	for _, cluster := range tbc.Clusters {
		if cluster.ID == id {
			desiredCluster = cluster
			clusterFound = true
		}
	}
	if !clusterFound {
		return nil, errors.New("cluster not found")
	}

	// happy path
	if apiKey == tbc.ApiKey {
		result := &bridge.ClusterUpgradeApiResource{
			ClusterID: id,
			Team:      tbc.TeamId,
		}
		if action == "enable-ha" && !*desiredCluster.IsHA {
			result.Operations = []*v1beta1.UpgradeOperation{
				{
					Flavor:       "ha_change",
					StartingFrom: "",
					State:        "enabling_ha",
				},
			}
		} else if action == "disable-ha" && *desiredCluster.IsHA {
			result.Operations = []*v1beta1.UpgradeOperation{
				{
					Flavor:       "ha_change",
					StartingFrom: "",
					State:        "disabling_ha",
				},
			}
		} else {
			return nil, errors.New("no change detected")
		}
		return result, nil
	}
	// sad path
	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) UpdateCluster(ctx context.Context, apiKey, id string, clusterRequestPayload *bridge.PatchClustersRequestPayload,
) (*bridge.ClusterApiResource, error) {
	// look for cluster
	var desiredCluster *bridge.ClusterApiResource
	clusterFound := false
	for _, cluster := range tbc.Clusters {
		if cluster.ID == id {
			desiredCluster = cluster
			clusterFound = true
		}
	}
	if !clusterFound {
		return nil, errors.New("cluster not found")
	}

	// happy path
	if apiKey == tbc.ApiKey {
		desiredCluster.ClusterName = clusterRequestPayload.Name
		desiredCluster.IsProtected = clusterRequestPayload.IsProtected
		return desiredCluster, nil
	}
	// sad path
	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) CreateCluster(ctx context.Context, apiKey string,
	clusterRequestPayload *bridge.PostClustersRequestPayload) (*bridge.ClusterApiResource, error) {

	if apiKey == tbc.ApiKey && clusterRequestPayload.Team == tbc.TeamId && clusterRequestPayload.Name != "" &&
		clusterRequestPayload.Plan != "" {
		cluster := &bridge.ClusterApiResource{
			ID:           fmt.Sprint(len(tbc.Clusters)),
			Host:         "example.com",
			IsHA:         initialize.Bool(clusterRequestPayload.IsHA),
			MajorVersion: clusterRequestPayload.PostgresVersion.IntValue(),
			ClusterName:  clusterRequestPayload.Name,
			Plan:         clusterRequestPayload.Plan,
			Provider:     clusterRequestPayload.Provider,
			Region:       clusterRequestPayload.Region,
			Storage:      clusterRequestPayload.Storage,
		}
		tbc.Clusters = append(tbc.Clusters, cluster)

		return cluster, nil
	}

	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) GetCluster(ctx context.Context, apiKey, id string) (*bridge.ClusterApiResource, error) {

	if apiKey == tbc.ApiKey {
		for _, cluster := range tbc.Clusters {
			if cluster.ID == id {
				return cluster, nil
			}
		}
	}

	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) GetClusterStatus(ctx context.Context, apiKey, id string) (*bridge.ClusterStatusApiResource, error) {

	if apiKey == tbc.ApiKey {
		return tbc.ClusterStatuses[id], nil
	}

	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) GetClusterUpgrade(ctx context.Context, apiKey, id string) (*bridge.ClusterUpgradeApiResource, error) {

	if apiKey == tbc.ApiKey {
		return tbc.ClusterUpgrades[id], nil
	}

	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) GetClusterRole(ctx context.Context, apiKey, clusterId, roleName string) (*bridge.ClusterRoleApiResource, error) {

	if apiKey == tbc.ApiKey {
		for _, clusterRole := range tbc.ClusterRoles {
			if clusterRole.ClusterId == clusterId && clusterRole.Name == roleName {
				return clusterRole, nil
			}
		}
	}

	return nil, errors.New("boom")
}

func (tbc *TestBridgeClient) DeleteCluster(ctx context.Context, apiKey, clusterId string) (*bridge.ClusterApiResource, bool, error) {
	alreadyDeleted := true
	var cluster *bridge.ClusterApiResource

	if apiKey == tbc.ApiKey {
		for i := len(tbc.Clusters) - 1; i >= 0; i-- {
			if tbc.Clusters[i].ID == clusterId {
				cluster = tbc.Clusters[i]
				alreadyDeleted = false
				tbc.Clusters = append(tbc.Clusters[:i], tbc.Clusters[i+1:]...)
				return cluster, alreadyDeleted, nil
			}
		}
	} else {
		return nil, alreadyDeleted, errors.New("boom")
	}

	return nil, alreadyDeleted, nil
}
