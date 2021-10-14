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

package pgadmin

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPod(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	pod := new(corev1.PodSpec)
	pvc := new(corev1.PersistentVolumeClaim)

	call := func() { Pod(cluster, pod, pvc) }

	t.Run("Disabled", func(t *testing.T) {
		before := pod.DeepCopy()
		call()

		// No change when pgAdmin is not requested in the spec.
		assert.DeepEqual(t, before, pod)
	})

	t.Run("Defaults", func(t *testing.T) {
		cluster.Spec.UserInterface = new(v1beta1.UserInterfaceSpec)
		cluster.Spec.UserInterface.PGAdmin = new(v1beta1.PGAdminPodSpec)
		cluster.Default()

		call()

		assert.Assert(t, marshalEquals(pod, strings.Trim(`
containers:
- env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin
  - name: PGADMIN_SETUP_PASSWORD
    value: admin
  livenessProbe:
    initialDelaySeconds: 15
    periodSeconds: 20
    tcpSocket:
      port: 5050
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  readinessProbe:
    initialDelaySeconds: 20
    periodSeconds: 10
    tcpSocket:
      port: 5050
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /tmp
    name: tmp
  - mountPath: /etc/httpd/run
    name: tmp
  - mountPath: /var/log/pgadmin
    name: pgadmin-log
  - mountPath: /var/lib/pgadmin
    name: pgadmin-data
volumes:
- emptyDir:
    medium: Memory
  name: tmp
- emptyDir:
    medium: Memory
  name: pgadmin-log
- name: pgadmin-data
  persistentVolumeClaim:
    claimName: ""
		`, "\t\n")+"\n"))

		// No change when called again.
		before := pod.DeepCopy()
		call()
		assert.DeepEqual(t, before, pod)
	})

	t.Run("Customizations", func(t *testing.T) {
		cluster.Spec.ImagePullPolicy = corev1.PullAlways
		cluster.Spec.UserInterface.PGAdmin.Image = "new-image"
		cluster.Spec.UserInterface.PGAdmin.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		}

		call()

		assert.Assert(t, marshalEquals(pod,
			strings.Trim(`
containers:
- env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin
  - name: PGADMIN_SETUP_PASSWORD
    value: admin
  image: new-image
  imagePullPolicy: Always
  livenessProbe:
    initialDelaySeconds: 15
    periodSeconds: 20
    tcpSocket:
      port: 5050
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  readinessProbe:
    initialDelaySeconds: 20
    periodSeconds: 10
    tcpSocket:
      port: 5050
  resources:
    requests:
      cpu: 100m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /tmp
    name: tmp
  - mountPath: /etc/httpd/run
    name: tmp
  - mountPath: /var/log/pgadmin
    name: pgadmin-log
  - mountPath: /var/lib/pgadmin
    name: pgadmin-data
volumes:
- emptyDir:
    medium: Memory
  name: tmp
- emptyDir:
    medium: Memory
  name: pgadmin-log
- name: pgadmin-data
  persistentVolumeClaim:
    claimName: ""
			`, "\t\n")+"\n"))
	})
}
