/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"context"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func (r *Reconciler) reconcilePGTune(ctx context.Context, cluster *v1beta1.PostgresCluster) error {
	if !cluster.Spec.AutoPGTune {
		return nil
	}

	if cluster.Spec.InstanceSets == nil ||
		cluster.Spec.InstanceSets[0].Resources.Requests == nil {
		return nil
	}

	initPatroniForPGTune(cluster)
	//err := errors.WithStack(r.apply(ctx, cluster))

	return nil
}

/* initPatroniForPGTune initializes empty Spec.Patroni, for cases when AutoPGTune is
enabled but Patroni is not.
AutoPGTune property can be added under Patroni to make this function unneccessary,
however it is less intuitive. */
func initPatroniForPGTune(cluster *v1beta1.PostgresCluster) {
	if cluster.Spec.Patroni == nil {
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{}
	}
}
