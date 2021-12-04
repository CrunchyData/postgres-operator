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

package pgbackrest

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestClientCommonName(t *testing.T) {
	t.Parallel()

	cluster := &metav1.ObjectMeta{UID: uuid.NewUUID()}
	cn := clientCommonName(cluster)

	assert.Assert(t, cmp.Regexp("^[-[:xdigit:]]{36}$", string(cluster.UID)),
		"expected Kubernetes UID to be a UUID string")

	assert.Assert(t, cmp.Regexp("^[[:print:]]{1,64}$", cn),
		"expected printable ASCII within 64 characters for %q", cluster)

	assert.Assert(t, strings.HasPrefix(cn, "pgbackrest@"),
		`expected %q to begin with "pgbackrest@" for %q`, cn, cluster)
}
