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

package pgadmin

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestConfigMap(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	config := new(corev1.ConfigMap)

	t.Run("Disabled", func(t *testing.T) {
		before := config.DeepCopy()
		assert.NilError(t, ConfigMap(cluster, config))

		// No change when pgAdmin is not requested in the spec.
		assert.DeepEqual(t, before, config)
	})

	t.Run("Defaults", func(t *testing.T) {
		cluster.Spec.UserInterface = new(v1beta1.UserInterfaceSpec)
		cluster.Spec.UserInterface.PGAdmin = new(v1beta1.PGAdminPodSpec)
		cluster.Default()

		assert.NilError(t, ConfigMap(cluster, config))

		assert.Assert(t, cmp.MarshalMatches(config.Data, `
pgadmin-settings.json: |
  {
    "SERVER_MODE": true
  }
		`))
	})

	t.Run("Customizations", func(t *testing.T) {
		cluster.Spec.UserInterface = new(v1beta1.UserInterfaceSpec)
		cluster.Spec.UserInterface.PGAdmin = new(v1beta1.PGAdminPodSpec)
		cluster.Spec.UserInterface.PGAdmin.Config.Settings = map[string]interface{}{
			"some":       "thing",
			"UPPER_CASE": false,
		}
		cluster.Default()

		assert.NilError(t, ConfigMap(cluster, config))

		assert.Assert(t, cmp.MarshalMatches(config.Data, `
pgadmin-settings.json: |
  {
    "SERVER_MODE": true,
    "UPPER_CASE": false,
    "some": "thing"
  }
		`))
	})
}

func TestPod(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	config := new(corev1.ConfigMap)
	pod := new(corev1.PodSpec)
	pvc := new(corev1.PersistentVolumeClaim)

	call := func() { Pod(cluster, config, pod, pvc) }

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

		assert.Assert(t, cmp.MarshalMatches(pod, `
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
  - mountPath: /etc/pgadmin
    name: pgadmin-startup
    readOnly: true
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
  - mountPath: /tmp
    name: tmp
  - mountPath: /etc/httpd/run
    name: tmp
  - mountPath: /var/log/pgadmin
    name: pgadmin-log
  - mountPath: /var/lib/pgadmin
    name: pgadmin-data
initContainers:
- command:
  - bash
  - -ceu
  - --
  - (umask a-w && echo "$1" > /etc/pgadmin/config_system.py)
  - startup
  - |
    import glob, json, re
    DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin.json') as _f:
        _conf, _data = re.compile(r'[A-Z_]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
  name: pgadmin-startup
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin
    name: pgadmin-startup
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
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
- name: pgadmin-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin.json
- emptyDir:
    medium: Memory
    sizeLimit: 32Ki
  name: pgadmin-startup
		`))

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

		assert.Assert(t, cmp.MarshalMatches(pod, `
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
  - mountPath: /etc/pgadmin
    name: pgadmin-startup
    readOnly: true
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
  - mountPath: /tmp
    name: tmp
  - mountPath: /etc/httpd/run
    name: tmp
  - mountPath: /var/log/pgadmin
    name: pgadmin-log
  - mountPath: /var/lib/pgadmin
    name: pgadmin-data
initContainers:
- command:
  - bash
  - -ceu
  - --
  - (umask a-w && echo "$1" > /etc/pgadmin/config_system.py)
  - startup
  - |
    import glob, json, re
    DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin.json') as _f:
        _conf, _data = re.compile(r'[A-Z_]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
  image: new-image
  imagePullPolicy: Always
  name: pgadmin-startup
  resources:
    requests:
      cpu: 100m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin
    name: pgadmin-startup
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
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
- name: pgadmin-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin.json
- emptyDir:
    medium: Memory
    sizeLimit: 32Ki
  name: pgadmin-startup
			`))
	})
}
