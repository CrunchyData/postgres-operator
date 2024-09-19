// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
  - |-
    monitor() {
    export PGADMIN_SETUP_PASSWORD="$(date +%s | sha256sum | base64 | head -c 32)"
    PGADMIN_DIR=/usr/local/lib/python3.11/site-packages/pgadmin4
    APP_RELEASE=$(cd $PGADMIN_DIR && python3 -c "import config; print(config.APP_RELEASE)")

    echo "Running pgAdmin4 Setup"
    if [ $APP_RELEASE -eq 7 ]; then
        python3 ${PGADMIN_DIR}/setup.py
    else
        python3 ${PGADMIN_DIR}/setup.py setup-db
    fi

    echo "Starting pgAdmin4"
    PGADMIN4_PIDFILE=/tmp/pgadmin4.pid
    if [ $APP_RELEASE -eq 7 ]; then
        pgadmin4 &
    else
        gunicorn -c /etc/pgadmin/gunicorn_config.py --chdir $PGADMIN_DIR pgAdmin4:app &
    fi
    echo $! > $PGADMIN4_PIDFILE

    loadServerCommand() {
        if [ $APP_RELEASE -eq 7 ]; then
            python3 ${PGADMIN_DIR}/setup.py --load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json --user admin@pgadmin.postgres-operator.svc --replace
        else
            python3 ${PGADMIN_DIR}/setup.py load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json --user admin@pgadmin.postgres-operator.svc --replace
        fi
    }
    loadServerCommand

    exec {fd}<> <(:||:)
    while read -r -t 5 -u "${fd}" ||:; do
        if [[ "${cluster_file}" -nt "/proc/self/fd/${fd}" ]] && loadServerCommand
        then
            exec {fd}>&- && exec {fd}<> <(:||:)
            stat --format='Loaded shared servers dated %y' "${cluster_file}"
        fi
        if [[ ! -d /proc/$(cat $PGADMIN4_PIDFILE) ]]
        then
            if [[ $APP_RELEASE -eq 7 ]]; then
                pgadmin4 &
            else
                gunicorn -c /etc/pgadmin/gunicorn_config.py --chdir $PGADMIN_DIR pgAdmin4:app &
            fi
            echo $! > $PGADMIN4_PIDFILE
            echo "Restarting pgAdmin4"
        fi
    done
    }; export cluster_file="$1"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgadmin
  - /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin@pgadmin.postgres-operator.svc
  - name: KRB5_CONFIG
    value: /etc/pgadmin/conf.d/krb5.conf
  - name: KRB5RCACHEDIR
    value: /tmp
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  readinessProbe:
    httpGet:
      path: /login
      port: 5050
      scheme: HTTP
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
    (umask a-w && echo "$2" > /etc/pgadmin/gunicorn_config.py)
  - startup
  - |
    import glob, json, re, os
    DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin-settings.json') as _f:
        _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
    if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password'):
        with open('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password') as _f:
            LDAP_BIND_PASSWORD = _f.read()
    if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/config-database-uri'):
        with open('/etc/pgadmin/conf.d/~postgres-operator/config-database-uri') as _f:
            CONFIG_DATABASE_URI = _f.read()
  - |
    import json, re
    with open('/etc/pgadmin/conf.d/~postgres-operator/gunicorn-config.json') as _f:
        _conf, _data = re.compile(r'[a-z_]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
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
    seccompProfile:
      type: RuntimeDefault
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
        - key: gunicorn-config.json
          path: ~postgres-operator/gunicorn-config.json
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
  - |-
    monitor() {
    export PGADMIN_SETUP_PASSWORD="$(date +%s | sha256sum | base64 | head -c 32)"
    PGADMIN_DIR=/usr/local/lib/python3.11/site-packages/pgadmin4
    APP_RELEASE=$(cd $PGADMIN_DIR && python3 -c "import config; print(config.APP_RELEASE)")

    echo "Running pgAdmin4 Setup"
    if [ $APP_RELEASE -eq 7 ]; then
        python3 ${PGADMIN_DIR}/setup.py
    else
        python3 ${PGADMIN_DIR}/setup.py setup-db
    fi

    echo "Starting pgAdmin4"
    PGADMIN4_PIDFILE=/tmp/pgadmin4.pid
    if [ $APP_RELEASE -eq 7 ]; then
        pgadmin4 &
    else
        gunicorn -c /etc/pgadmin/gunicorn_config.py --chdir $PGADMIN_DIR pgAdmin4:app &
    fi
    echo $! > $PGADMIN4_PIDFILE

    loadServerCommand() {
        if [ $APP_RELEASE -eq 7 ]; then
            python3 ${PGADMIN_DIR}/setup.py --load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json --user admin@pgadmin.postgres-operator.svc --replace
        else
            python3 ${PGADMIN_DIR}/setup.py load-servers /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json --user admin@pgadmin.postgres-operator.svc --replace
        fi
    }
    loadServerCommand

    exec {fd}<> <(:||:)
    while read -r -t 5 -u "${fd}" ||:; do
        if [[ "${cluster_file}" -nt "/proc/self/fd/${fd}" ]] && loadServerCommand
        then
            exec {fd}>&- && exec {fd}<> <(:||:)
            stat --format='Loaded shared servers dated %y' "${cluster_file}"
        fi
        if [[ ! -d /proc/$(cat $PGADMIN4_PIDFILE) ]]
        then
            if [[ $APP_RELEASE -eq 7 ]]; then
                pgadmin4 &
            else
                gunicorn -c /etc/pgadmin/gunicorn_config.py --chdir $PGADMIN_DIR pgAdmin4:app &
            fi
            echo $! > $PGADMIN4_PIDFILE
            echo "Restarting pgAdmin4"
        fi
    done
    }; export cluster_file="$1"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgadmin
  - /etc/pgadmin/conf.d/~postgres-operator/pgadmin-shared-clusters.json
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin@pgadmin.postgres-operator.svc
  - name: KRB5_CONFIG
    value: /etc/pgadmin/conf.d/krb5.conf
  - name: KRB5RCACHEDIR
    value: /tmp
  image: new-image
  imagePullPolicy: Always
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  readinessProbe:
    httpGet:
      path: /login
      port: 5050
      scheme: HTTP
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
    seccompProfile:
      type: RuntimeDefault
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
    (umask a-w && echo "$2" > /etc/pgadmin/gunicorn_config.py)
  - startup
  - |
    import glob, json, re, os
    DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin-settings.json') as _f:
        _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
        if type(_data) is dict:
            globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
    if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password'):
        with open('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password') as _f:
            LDAP_BIND_PASSWORD = _f.read()
    if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/config-database-uri'):
        with open('/etc/pgadmin/conf.d/~postgres-operator/config-database-uri') as _f:
            CONFIG_DATABASE_URI = _f.read()
  - |
    import json, re
    with open('/etc/pgadmin/conf.d/~postgres-operator/gunicorn-config.json') as _f:
        _conf, _data = re.compile(r'[a-z_]+'), json.load(_f)
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
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
        - key: gunicorn-config.json
          path: ~postgres-operator/gunicorn-config.json
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
    - key: gunicorn-config.json
      path: ~postgres-operator/gunicorn-config.json
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
