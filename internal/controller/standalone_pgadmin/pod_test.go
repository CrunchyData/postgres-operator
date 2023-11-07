// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package standalone_pgadmin

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPod(t *testing.T) {
	t.Parallel()

	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Name = "pgadmin"
	pgadmin.Namespace = "postgres-operator"
	config := new(corev1.ConfigMap)
	testpod := new(corev1.PodSpec)
	pvc := new(corev1.PersistentVolumeClaim)

	call := func() { pod(pgadmin, config, testpod, pvc) }

	t.Run("Defaults", func(t *testing.T) {

		call()

		assert.Assert(t, cmp.MarshalMatches(testpod, `
containers:
- command:
  - bash
  - -ceu
  - --
  - "monitor() {\nPGADMIN_DIR=/usr/local/lib/python3.11/site-packages/pgadmin4\n\necho
    \"Running pgAdmin4 Setup\"\npython3 ${PGADMIN_DIR}/setup.py\n\necho \"Starting
    pgAdmin4\"\nPGADMIN4_PIDFILE=/tmp/pgadmin4.pid\npgadmin4 &\necho $! > $PGADMIN4_PIDFILE\n\npython3
    ${PGADMIN_DIR}/setup.py --load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
    --user admin@pgadmin.postgres-operator.svc --replace\n\nexec {fd}<> <(:)\nwhile
    read -r -t 5 -u \"${fd}\" || true; do\n\tif [ \"${cluster_file}\" -nt \"/proc/self/fd/${fd}\"
    ] && python3 ${PGADMIN_DIR}/setup.py --load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
    --user admin@pgadmin.postgres-operator.svc --replace\n\tthen\n\t\texec {fd}>&-
    && exec {fd}<> <(:)\n\t\tstat --format='Loaded shared servers dated %y' \"${cluster_file}\"\n\tfi\n\tif
    [ ! -d /proc/$(cat $PGADMIN4_PIDFILE) ]\n\tthen\n\t\tpgadmin4 &\n\t\techo $! >
    $PGADMIN4_PIDFILE\n\t\techo \"Restarting pgAdmin4\"\n\tfi\ndone\n}; export cluster_file=\"$1\";
    export -f monitor; exec -a \"$0\" bash -ceu monitor"
  - pgadmin
  - /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin@pgadmin.postgres-operator.svc
  - name: PGADMIN_SETUP_PASSWORD
    valueFrom:
      secretKeyRef:
        key: password
        name: pgadmin-
  - name: PGADMIN_LISTEN_PORT
    value: "5050"
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
  - mountPath: /var/lib/pgadmin
    name: pgadmin-data
  - mountPath: /var/log/pgadmin
    name: pgadmin-log
  - mountPath: /etc/pgadmin
    name: pgadmin-config-system
    readOnly: true
  - mountPath: /tmp
    name: tmp
initContainers:
- command:
  - bash
  - -ceu
  - --
  - |-
    mkdir -p /etc/pgadmin/conf.d
    (umask a-w && echo "$1" > /etc/pgadmin/config_system.py)
  - startup
  - |
    import json, re, os
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin-settings.json') as _f:
        _conf, _data = re.compile(r'[A-Z_]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
    if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password'):
        with open('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password') as _f:
            LDAP_BIND_PASSWORD = _f.read()
  name: pgadmin-startup
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin
    name: pgadmin-config-system
volumes:
- name: pgadmin-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin-settings.json
        - key: pgadmin-shared-clusters.json
          path: ~postgres-operator/pgadmin-shared-clusters.json
- name: pgadmin-data
  persistentVolumeClaim:
    claimName: ""
- emptyDir:
    medium: Memory
  name: pgadmin-log
- emptyDir:
    medium: Memory
    sizeLimit: 32Ki
  name: pgadmin-config-system
- emptyDir:
    medium: Memory
  name: tmp
`))

		// No change when called again.
		before := testpod.DeepCopy()
		call()
		assert.DeepEqual(t, before, testpod)
	})

	t.Run("Customizations", func(t *testing.T) {
		pgadmin.Spec.ImagePullPolicy = corev1.PullAlways
		pgadmin.Spec.Image = initialize.String("new-image")
		pgadmin.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		}

		call()

		assert.Assert(t, cmp.MarshalMatches(testpod, `
containers:
- command:
  - bash
  - -ceu
  - --
  - "monitor() {\nPGADMIN_DIR=/usr/local/lib/python3.11/site-packages/pgadmin4\n\necho
    \"Running pgAdmin4 Setup\"\npython3 ${PGADMIN_DIR}/setup.py\n\necho \"Starting
    pgAdmin4\"\nPGADMIN4_PIDFILE=/tmp/pgadmin4.pid\npgadmin4 &\necho $! > $PGADMIN4_PIDFILE\n\npython3
    ${PGADMIN_DIR}/setup.py --load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
    --user admin@pgadmin.postgres-operator.svc --replace\n\nexec {fd}<> <(:)\nwhile
    read -r -t 5 -u \"${fd}\" || true; do\n\tif [ \"${cluster_file}\" -nt \"/proc/self/fd/${fd}\"
    ] && python3 ${PGADMIN_DIR}/setup.py --load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
    --user admin@pgadmin.postgres-operator.svc --replace\n\tthen\n\t\texec {fd}>&-
    && exec {fd}<> <(:)\n\t\tstat --format='Loaded shared servers dated %y' \"${cluster_file}\"\n\tfi\n\tif
    [ ! -d /proc/$(cat $PGADMIN4_PIDFILE) ]\n\tthen\n\t\tpgadmin4 &\n\t\techo $! >
    $PGADMIN4_PIDFILE\n\t\techo \"Restarting pgAdmin4\"\n\tfi\ndone\n}; export cluster_file=\"$1\";
    export -f monitor; exec -a \"$0\" bash -ceu monitor"
  - pgadmin
  - /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin@pgadmin.postgres-operator.svc
  - name: PGADMIN_SETUP_PASSWORD
    valueFrom:
      secretKeyRef:
        key: password
        name: pgadmin-
  - name: PGADMIN_LISTEN_PORT
    value: "5050"
  image: new-image
  imagePullPolicy: Always
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  resources:
    requests:
      cpu: 100m
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
  - mountPath: /var/lib/pgadmin
    name: pgadmin-data
  - mountPath: /var/log/pgadmin
    name: pgadmin-log
  - mountPath: /etc/pgadmin
    name: pgadmin-config-system
    readOnly: true
  - mountPath: /tmp
    name: tmp
initContainers:
- command:
  - bash
  - -ceu
  - --
  - |-
    mkdir -p /etc/pgadmin/conf.d
    (umask a-w && echo "$1" > /etc/pgadmin/config_system.py)
  - startup
  - |
    import json, re, os
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin-settings.json') as _f:
        _conf, _data = re.compile(r'[A-Z_]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
    if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password'):
        with open('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password') as _f:
            LDAP_BIND_PASSWORD = _f.read()
  image: new-image
  imagePullPolicy: Always
  name: pgadmin-startup
  resources:
    requests:
      cpu: 100m
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin
    name: pgadmin-config-system
volumes:
- name: pgadmin-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin-settings.json
        - key: pgadmin-shared-clusters.json
          path: ~postgres-operator/pgadmin-shared-clusters.json
- name: pgadmin-data
  persistentVolumeClaim:
    claimName: ""
- emptyDir:
    medium: Memory
  name: pgadmin-log
- emptyDir:
    medium: Memory
    sizeLimit: 32Ki
  name: pgadmin-config-system
- emptyDir:
    medium: Memory
  name: tmp
`))
	})
}

func TestPodConfigFiles(t *testing.T) {
	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "some-cm"}}

	pgadmin := v1beta1.PGAdmin{
		Spec: v1beta1.PGAdminSpec{
			Config: v1beta1.StandalonePGAdminConfiguration{Files: []corev1.VolumeProjection{{
				Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{
					Name: "test-secret",
				}},
			}, {
				ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{
					Name: "test-cm",
				}},
			}}},
		},
	}

	projections := podConfigFiles(configmap, pgadmin)
	assert.Assert(t, cmp.MarshalMatches(projections, `
- secret:
    name: test-secret
- configMap:
    name: test-cm
- configMap:
    items:
    - key: pgadmin-settings.json
      path: ~postgres-operator/pgadmin-settings.json
    - key: pgadmin-shared-clusters.json
      path: ~postgres-operator/pgadmin-shared-clusters.json
    name: some-cm
	`))
}

func TestPodSecurityContext(t *testing.T) {
	pgAdminReconciler := &PGAdminReconciler{}

	assert.Assert(t, cmp.MarshalMatches(podSecurityContext(pgAdminReconciler), `
fsGroup: 2
fsGroupChangePolicy: OnRootMismatch
	`))

	pgAdminReconciler.IsOpenShift = true
	assert.Assert(t, cmp.MarshalMatches(podSecurityContext(pgAdminReconciler),
		`fsGroupChangePolicy: OnRootMismatch`))
}
