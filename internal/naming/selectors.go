// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// AsSelector is a wrapper around metav1.LabelSelectorAsSelector() which converts
// the LabelSelector API type into something that implements labels.Selector.
func AsSelector(s metav1.LabelSelector) (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(&s)
}

// AnyCluster selects things for any PostgreSQL cluster.
func AnyCluster() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: LabelCluster, Operator: metav1.LabelSelectorOpExists},
		},
	}
}

// Cluster selects things for cluster.
func Cluster(cluster string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster,
		},
	}
}

// ClusterBackupJobs selects things for all existing backup jobs in cluster.
func ClusterBackupJobs(cluster string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: LabelPGBackRestBackup, Operator: metav1.LabelSelectorOpExists},
		},
	}
}

// ClusterDataForPostgresAndPGBackRest selects things for PostgreSQL data and
// things for pgBackRest data.
func ClusterDataForPostgresAndPGBackRest(cluster string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      LabelData,
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{DataPostgres, DataPGBackRest},
		}},
	}
}

// ClusterInstance selects things for a single instance in a cluster.
func ClusterInstance(cluster, instance string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster:  cluster,
			LabelInstance: instance,
		},
	}
}

// ClusterInstances selects things for PostgreSQL instances in cluster.
func ClusterInstances(cluster string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: LabelInstance, Operator: metav1.LabelSelectorOpExists},
		},
	}
}

// ClusterInstanceSet selects things for set in cluster.
func ClusterInstanceSet(cluster, set string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster:     cluster,
			LabelInstanceSet: set,
		},
	}
}

// ClusterInstanceSets selects things for sets in a cluster.
func ClusterInstanceSets(cluster string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: LabelInstanceSet, Operator: metav1.LabelSelectorOpExists},
		},
	}
}

// ClusterPatronis selects things labeled for Patroni in cluster.
func ClusterPatronis(cluster *v1beta1.PostgresCluster) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster.Name,
			LabelPatroni: PatroniScope(cluster),
		},
	}
}

// ClusterPGBouncerSelector selects things labeled for PGBouncer in cluster.
func ClusterPGBouncerSelector(cluster *v1beta1.PostgresCluster) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster.Name,
			LabelRole:    RolePGBouncer,
		},
	}
}

// ClusterPostgresUsers selects things labeled for PostgreSQL users in cluster.
func ClusterPostgresUsers(cluster string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: cluster,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			// The now-deprecated default PostgreSQL user secret lacks a LabelRole.
			// The existence of a LabelPostgresUser matches it and current secrets.
			{Key: LabelPostgresUser, Operator: metav1.LabelSelectorOpExists},
		},
	}
}

// CrunchyBridgeClusterPostgresRoles selects things labeled for CrunchyBridgeCluster
// PostgreSQL roles in cluster.
func CrunchyBridgeClusterPostgresRoles(clusterName string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCluster: clusterName,
			LabelRole:    RoleCrunchyBridgeClusterPostgresRole,
		},
	}
}
