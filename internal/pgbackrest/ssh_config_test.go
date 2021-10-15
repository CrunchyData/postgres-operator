//go:build envtest
// +build envtest

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

package pgbackrest

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// TestKeys validates public/private byte slices returned by
// getKeys() are of the expected type and use the expected curve
func TestKeys(t *testing.T) {

	testKeys, err := getKeys()
	assert.NilError(t, err)

	t.Run("test private key", func(t *testing.T) {
		block, _ := pem.Decode(testKeys.Private)

		if assert.Check(t, block != nil) {
			private, err := x509.ParseECPrivateKey(block.Bytes)

			assert.NilError(t, err)
			assert.Equal(t, fmt.Sprintf("%T", private), "*ecdsa.PrivateKey")
			assert.Equal(t, private.Params().BitSize, 521)
		}
	})

	t.Run("test public key", func(t *testing.T) {
		pub, _, _, _, err := ssh.ParseAuthorizedKey(testKeys.Public)

		assert.NilError(t, err)
		assert.Equal(t, pub.Type(), "ecdsa-sha2-nistp521")
		assert.Equal(t, fmt.Sprintf("%T", pub), "*ssh.ecdsaPublicKey")
	})

}

// TestSSHDConfiguration verifies the default SSH/SSHD configurations
// are created. These include the secret containing the public and private
// keys, the configmap containing the SSH client config file and SSHD
// sshd_config file, their respective contents, the project volume and
// the volume mount
func TestSSHDConfiguration(t *testing.T) {

	// set cluster name and namespace values in postgrescluster spec
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testclustername,
			Namespace: "postgres-operator-test-" + rand.String(6),
		},
	}

	// the initially created configmap
	var sshCMInitial corev1.ConfigMap
	// the returned configmap
	var sshCMReturned corev1.ConfigMap
	// pod spec for testing projected volumes and volume mounts
	pod := &corev1.PodSpec{}
	// initially created secret
	var secretInitial corev1.Secret
	// returned secret
	var secretReturned corev1.Secret

	t.Run("ssh configmap and secret checks", func(t *testing.T) {

		// setup the test environment and ensure a clean teardown
		testEnv, testClient := setupTestEnv(t)

		// define the cleanup steps to run once the tests complete
		t.Cleanup(func() {
			teardownTestEnv(t, testEnv)
		})

		ns := &corev1.Namespace{}
		ns.Name = naming.PGBackRestConfig(postgresCluster).Namespace
		ns.Labels = labels.Set{"postgres-operator-test": ""}
		assert.NilError(t, testClient.Create(context.Background(), ns))
		t.Cleanup(func() { assert.Check(t, testClient.Delete(context.Background(), ns)) })

		t.Run("create ssh configmap struct", func(t *testing.T) {
			sshCMInitial = CreateSSHConfigMapIntent(postgresCluster)

			// check that there is configmap data
			assert.Assert(t, sshCMInitial.Data != nil)
		})

		t.Run("create ssh secret struct", func(t *testing.T) {

			// declare this locally so ':=' operation will not result in a
			// locally scoped 'secretInitial' variable
			var err error

			secretInitial, err = CreateSSHSecretIntent(postgresCluster, nil,
				naming.ClusterPodService(postgresCluster).Name, ns.GetName())

			assert.NilError(t, err)

			// check that there is configmap data
			assert.Assert(t, secretInitial.Data != nil)
		})

		t.Run("create ssh configmap", func(t *testing.T) {

			// create the configmap
			err := testClient.Patch(context.Background(), &sshCMInitial, client.Apply, client.ForceOwnership, client.FieldOwner(testFieldOwner))

			assert.NilError(t, err)
		})

		t.Run("create ssh secret", func(t *testing.T) {

			// create the secret
			err := testClient.Patch(context.Background(), &secretInitial, client.Apply, client.ForceOwnership, client.FieldOwner(testFieldOwner))

			assert.NilError(t, err)
		})

		t.Run("get ssh configmap", func(t *testing.T) {

			objectKey := client.ObjectKey{
				Namespace: naming.PGBackRestSSHConfig(postgresCluster).Namespace,
				Name:      naming.PGBackRestSSHConfig(postgresCluster).Name,
			}

			err := testClient.Get(context.Background(), objectKey, &sshCMReturned)

			assert.NilError(t, err)
		})

		t.Run("get ssh secret", func(t *testing.T) {

			objectKey := client.ObjectKey{
				Namespace: naming.PGBackRestSSHSecret(postgresCluster).Namespace,
				Name:      naming.PGBackRestSSHSecret(postgresCluster).Name,
			}

			err := testClient.Get(context.Background(), objectKey, &secretReturned)

			assert.NilError(t, err)
		})

		// finally, verify initial and returned match
		assert.Assert(t, reflect.DeepEqual(sshCMInitial.Data, sshCMReturned.Data))
		assert.Assert(t, reflect.DeepEqual(secretInitial.Data, secretReturned.Data))

	})

	t.Run("check ssh config", func(t *testing.T) {

		assert.Equal(t, getCMData(sshCMReturned, sshConfig),
			`Host *
StrictHostKeyChecking yes
IdentityFile /etc/ssh/id_ecdsa
Port 2022
User postgres
`)
	})

	t.Run("check sshd config", func(t *testing.T) {

		assert.Equal(t, getCMData(sshCMReturned, sshdConfig),
			`AuthorizedKeysFile /etc/ssh/id_ecdsa.pub
ForceCommand NSS_WRAPPER_SUBDIR=postgres . /opt/crunchy/bin/nss_wrapper_env.sh && $SSH_ORIGINAL_COMMAND
HostKey /etc/ssh/id_ecdsa
PasswordAuthentication no
PermitRootLogin no
PidFile /tmp/sshd.pid
Port 2022
PubkeyAuthentication yes
StrictModes no
`)
	})

	t.Run("check sshd volume", func(t *testing.T) {

		SSHConfigVolumeAndMount(&sshCMReturned, &secretReturned, pod, "database")

		assert.Assert(t, simpleMarshalContains(&pod.Volumes, strings.TrimSpace(`
		- name: sshd
  projected:
    sources:
    - configMap:
        items:
        - key: ssh_config
          path: ./ssh_config
        - key: sshd_config
          path: ./sshd_config
        name: `+postgresCluster.GetName()+`-ssh-config
      secret:
        items:
        - key: id_ecdsa
          path: ./id_ecdsa
        - key: id_ecdsa.pub
          path: ./id_ecdsa.pub
        name: `+postgresCluster.GetName()+`-ssh-config
`)+"\n"))
	})

	t.Run("check sshd volume mount", func(t *testing.T) {

		SSHConfigVolumeAndMount(&sshCMReturned, &secretReturned, pod, "database")

		container := findOrAppendContainer(&pod.Containers, "database")

		assert.Assert(t, simpleMarshalContains(container.VolumeMounts, strings.TrimSpace(`
		- mountPath: /etc/ssh
  name: sshd
  readOnly: true
		`)+"\n"))
	})
}
