/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

// ClusterPrimary selects things for the Primary PostgreSQL instance.
func ClusterPrimary(cluster string) metav1.LabelSelector {
	s := ClusterInstances(cluster)
	s.MatchLabels[LabelRole] = RolePatroniLeader
	return s
}
