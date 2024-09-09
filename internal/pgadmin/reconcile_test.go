// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
- command:
  - bash
  - -c
  - |-
    CRUNCHY_DIR=${CRUNCHY_DIR:-'/opt/crunchy'}
    PGADMIN_DIR=/usr/lib/python3.6/site-packages/pgadmin4-web
    APACHE_PIDFILE='/tmp/httpd.pid'
    export PATH=$PATH:/usr/pgsql-*/bin

    RED="\033[0;31m"
    GREEN="\033[0;32m"
    RESET="\033[0m"

    function enable_debugging() {
        if [[ ${CRUNCHY_DEBUG:-false} == "true" ]]
        then
            echo_info "Turning debugging on.."
            export PS4='+(${BASH_SOURCE}:${LINENO})> ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
            set -x
        fi
    }

    function env_check_err() {
        if [[ -z ${!1} ]]
        then
            echo_err "$1 environment variable is not set, aborting."
            exit 1
        fi
    }

    function echo_info() {
        echo -e "${GREEN?}$(date) INFO: ${1?}${RESET?}"
    }

    function echo_err() {
        echo -e "${RED?}$(date) ERROR: ${1?}${RESET?}"
    }

    function err_check {
        RC=${1?}
        CONTEXT=${2?}
        ERROR=${3?}

        if [[ ${RC?} != 0 ]]
        then
            echo_err "${CONTEXT?}: ${ERROR?}"
            exit ${RC?}
        fi
    }

    function trap_sigterm() {
        echo_info "Doing trap logic.."
        echo_warn "Clean shutdown of Apache.."
        /usr/sbin/httpd -k stop
        kill -SIGINT $(head -1 $APACHE_PIDFILE)
    }

    enable_debugging
    trap 'trap_sigterm' SIGINT SIGTERM

    env_check_err "PGADMIN_SETUP_EMAIL"
    env_check_err "PGADMIN_SETUP_PASSWORD"

    if [[ ${ENABLE_TLS:-false} == 'true' ]]
    then
        echo_info "TLS enabled. Applying https configuration.."
        if [[ ( ! -f /certs/server.key ) || ( ! -f /certs/server.crt ) ]]
        then
            echo_err "ENABLE_TLS true but /certs/server.key or /certs/server.crt not found, aborting"
            exit 1
        fi
        cp "${CRUNCHY_DIR}/conf/pgadmin-https.conf" /var/lib/pgadmin/pgadmin.conf
    else
        echo_info "TLS disabled. Applying http configuration.."
        cp "${CRUNCHY_DIR}/conf/pgadmin-http.conf" /var/lib/pgadmin/pgadmin.conf
    fi

    cp "${CRUNCHY_DIR}/conf/config_local.py" /var/lib/pgadmin/config_local.py

    if [[ -z "${SERVER_PATH}" ]]
    then
        sed -i "/RedirectMatch/d" /var/lib/pgadmin/pgadmin.conf
    fi

    sed -i "s|SERVER_PATH|${SERVER_PATH:-/}|g" /var/lib/pgadmin/pgadmin.conf
    sed -i "s|SERVER_PORT|${SERVER_PORT:-5050}|g" /var/lib/pgadmin/pgadmin.conf
    sed -i "s/^DEFAULT_SERVER_PORT.*/DEFAULT_SERVER_PORT = ${SERVER_PORT:-5050}/" /var/lib/pgadmin/config_local.py
    sed -i "s|\"pg\":.*|\"pg\": \"/usr/pgsql-${PGVERSION?}/bin\",|g" /var/lib/pgadmin/config_local.py

    cd ${PGADMIN_DIR?}

    if [[ ! -f /var/lib/pgadmin/pgadmin4.db ]]
    then
        echo_info "Setting up pgAdmin4 database.."
        python3 setup.py > /tmp/pgadmin4.stdout 2> /tmp/pgadmin4.stderr
        err_check "$?" "pgAdmin4 Database Setup" "Could not create pgAdmin4 database: \n$(cat /tmp/pgadmin4.stderr)"
    fi

    echo_info "Starting Apache web server.."
    /usr/sbin/httpd -D FOREGROUND &
    echo $! > $APACHE_PIDFILE

    wait
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin
  - name: PGADMIN_SETUP_PASSWORD
    value: admin
  - name: KRB5_CONFIG
    value: /etc/pgadmin/conf.d/krb5.conf
  - name: KRB5RCACHEDIR
    value: /tmp
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
    name: pgadmin-startup
    readOnly: true
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
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
    import glob, json, re, os
    DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin.json') as _f:
        _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
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
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgadmin
    name: pgadmin-startup
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
volumes:
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
		cluster.Spec.UserInterface.PGAdmin.Config.Files = []corev1.VolumeProjection{{
			Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{
				Name: "test",
			}},
		}}
		cluster.Spec.UserInterface.PGAdmin.Config.LDAPBindPassword = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "podtest",
			},
			Key: "podtestpw",
		}

		call()

		assert.Assert(t, cmp.MarshalMatches(pod, `
containers:
- command:
  - bash
  - -c
  - |-
    CRUNCHY_DIR=${CRUNCHY_DIR:-'/opt/crunchy'}
    PGADMIN_DIR=/usr/lib/python3.6/site-packages/pgadmin4-web
    APACHE_PIDFILE='/tmp/httpd.pid'
    export PATH=$PATH:/usr/pgsql-*/bin

    RED="\033[0;31m"
    GREEN="\033[0;32m"
    RESET="\033[0m"

    function enable_debugging() {
        if [[ ${CRUNCHY_DEBUG:-false} == "true" ]]
        then
            echo_info "Turning debugging on.."
            export PS4='+(${BASH_SOURCE}:${LINENO})> ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
            set -x
        fi
    }

    function env_check_err() {
        if [[ -z ${!1} ]]
        then
            echo_err "$1 environment variable is not set, aborting."
            exit 1
        fi
    }

    function echo_info() {
        echo -e "${GREEN?}$(date) INFO: ${1?}${RESET?}"
    }

    function echo_err() {
        echo -e "${RED?}$(date) ERROR: ${1?}${RESET?}"
    }

    function err_check {
        RC=${1?}
        CONTEXT=${2?}
        ERROR=${3?}

        if [[ ${RC?} != 0 ]]
        then
            echo_err "${CONTEXT?}: ${ERROR?}"
            exit ${RC?}
        fi
    }

    function trap_sigterm() {
        echo_info "Doing trap logic.."
        echo_warn "Clean shutdown of Apache.."
        /usr/sbin/httpd -k stop
        kill -SIGINT $(head -1 $APACHE_PIDFILE)
    }

    enable_debugging
    trap 'trap_sigterm' SIGINT SIGTERM

    env_check_err "PGADMIN_SETUP_EMAIL"
    env_check_err "PGADMIN_SETUP_PASSWORD"

    if [[ ${ENABLE_TLS:-false} == 'true' ]]
    then
        echo_info "TLS enabled. Applying https configuration.."
        if [[ ( ! -f /certs/server.key ) || ( ! -f /certs/server.crt ) ]]
        then
            echo_err "ENABLE_TLS true but /certs/server.key or /certs/server.crt not found, aborting"
            exit 1
        fi
        cp "${CRUNCHY_DIR}/conf/pgadmin-https.conf" /var/lib/pgadmin/pgadmin.conf
    else
        echo_info "TLS disabled. Applying http configuration.."
        cp "${CRUNCHY_DIR}/conf/pgadmin-http.conf" /var/lib/pgadmin/pgadmin.conf
    fi

    cp "${CRUNCHY_DIR}/conf/config_local.py" /var/lib/pgadmin/config_local.py

    if [[ -z "${SERVER_PATH}" ]]
    then
        sed -i "/RedirectMatch/d" /var/lib/pgadmin/pgadmin.conf
    fi

    sed -i "s|SERVER_PATH|${SERVER_PATH:-/}|g" /var/lib/pgadmin/pgadmin.conf
    sed -i "s|SERVER_PORT|${SERVER_PORT:-5050}|g" /var/lib/pgadmin/pgadmin.conf
    sed -i "s/^DEFAULT_SERVER_PORT.*/DEFAULT_SERVER_PORT = ${SERVER_PORT:-5050}/" /var/lib/pgadmin/config_local.py
    sed -i "s|\"pg\":.*|\"pg\": \"/usr/pgsql-${PGVERSION?}/bin\",|g" /var/lib/pgadmin/config_local.py

    cd ${PGADMIN_DIR?}

    if [[ ! -f /var/lib/pgadmin/pgadmin4.db ]]
    then
        echo_info "Setting up pgAdmin4 database.."
        python3 setup.py > /tmp/pgadmin4.stdout 2> /tmp/pgadmin4.stderr
        err_check "$?" "pgAdmin4 Database Setup" "Could not create pgAdmin4 database: \n$(cat /tmp/pgadmin4.stderr)"
    fi

    echo_info "Starting Apache web server.."
    /usr/sbin/httpd -D FOREGROUND &
    echo $! > $APACHE_PIDFILE

    wait
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin
  - name: PGADMIN_SETUP_PASSWORD
    value: admin
  - name: KRB5_CONFIG
    value: /etc/pgadmin/conf.d/krb5.conf
  - name: KRB5RCACHEDIR
    value: /tmp
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
    name: pgadmin-startup
    readOnly: true
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
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
    import glob, json, re, os
    DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
    with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin.json') as _f:
        _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
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
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgadmin
    name: pgadmin-startup
  - mountPath: /etc/pgadmin/conf.d
    name: pgadmin-config
    readOnly: true
volumes:
- emptyDir:
    medium: Memory
  name: pgadmin-log
- name: pgadmin-data
  persistentVolumeClaim:
    claimName: ""
- name: pgadmin-config
  projected:
    sources:
    - secret:
        name: test
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin.json
    - secret:
        items:
        - key: podtestpw
          path: ~postgres-operator/ldap-bind-password
        name: podtest
- emptyDir:
    medium: Memory
    sizeLimit: 32Ki
  name: pgadmin-startup
			`))
	})
}
