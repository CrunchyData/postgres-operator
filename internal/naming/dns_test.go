/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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
	"context"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestInstancePodDNSNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	instance := &appsv1.StatefulSet{}
	instance.Namespace = "some-place"
	instance.Name = "cluster-name-id"
	instance.Spec.ServiceName = "cluster-pods"

	names := InstancePodDNSNames(ctx, instance)
	assert.Assert(t, len(names) > 0)

	assert.DeepEqual(t, names[1:], []string{
		"cluster-name-id-0.cluster-pods.some-place.svc",
		"cluster-name-id-0.cluster-pods.some-place",
		"cluster-name-id-0.cluster-pods",
	})

	assert.Assert(t, len(names[0]) > len(names[1]), "expected FQDN first, got %q", names[0])
	assert.Assert(t, strings.HasPrefix(names[0], names[1]+"."), "wrong FQDN: %q", names[0])
	assert.Assert(t, strings.HasSuffix(names[0], "."), "expected root, got %q", names[0])
}

func TestServiceDNSNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	service := &corev1.Service{}
	service.Namespace = "baltia"
	service.Name = "the-primary"

	names := ServiceDNSNames(ctx, service)
	assert.Assert(t, len(names) > 0)

	assert.DeepEqual(t, names[1:], []string{
		"the-primary.baltia.svc",
		"the-primary.baltia",
		"the-primary",
	})

	assert.Assert(t, len(names[0]) > len(names[1]), "expected FQDN first, got %q", names[0])
	assert.Assert(t, strings.HasPrefix(names[0], names[1]+"."), "wrong FQDN: %q", names[0])
	assert.Assert(t, strings.HasSuffix(names[0], "."), "expected root, got %q", names[0])
}
