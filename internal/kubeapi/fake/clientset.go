package fake

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

import (
	fakekubernetes "k8s.io/client-go/kubernetes/fake"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	fakecrunchydata "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned/fake"
	crunchydatav1 "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned/typed/crunchydata.com/v1"
)

type Clientset struct {
	*fakekubernetes.Clientset
	PGOClientset *fakecrunchydata.Clientset
}

var _ kubeapi.Interface = &Clientset{}

// CrunchydataV1 retrieves the CrunchydataV1Client
func (c *Clientset) CrunchydataV1() crunchydatav1.CrunchydataV1Interface {
	return c.PGOClientset.CrunchydataV1()
}
