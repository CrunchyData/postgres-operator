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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// knownHostsKey is the name of the 'known_hosts' file
	knownHostsKey = "ssh_known_hosts"

	// mount path for SSH configuration
	sshConfigPath = "/etc/ssh"

	// config file for the SSH client
	sshConfig = "ssh_config"
	// config file for the SSHD service
	sshdConfig = "sshd_config"

	// private key file name
	privateKey = "id_ecdsa"
	// public key file name
	publicKey = "id_ecdsa.pub"
	// SSH configuration volume
	sshConfigVol = "sshd"
)

// sshKey stores byte slices that represent private and public ssh keys
// used to populate the postgrescluster's SSH secret
type sshKey struct {
	Private []byte
	Public  []byte
}

// CreateSSHConfigMapIntent creates a configmap struct with SSHD service and SSH client
// configuration settings in the data field.
func CreateSSHConfigMapIntent(postgresCluster *v1beta1.PostgresCluster) v1.ConfigMap {

	meta := naming.PGBackRestSSHConfig(postgresCluster)
	meta.Annotations = naming.Merge(
		postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Archive.PGBackRest.Metadata.GetAnnotationsOrNil())
	meta.Labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Archive.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestRepoHostLabels(postgresCluster.GetName()),
	)

	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta,
	}

	// create an empty map for the config data
	initialize.StringMap(&cm.Data)

	// if the SSH config data map is not ok, populate with the configuration string
	if _, ok := cm.Data[sshConfig]; !ok {
		cm.Data[sshConfig] = getSSHConfigString()
	}

	// if the SSHD config data map is not ok, populate with the configuration string
	if _, ok := cm.Data[sshdConfig]; !ok {
		cm.Data[sshdConfig] = getSSHDConfigString()
	}

	return cm
}

// CreateSSHSecretIntent creates the secret containing the new public private key pair to use
// when connecting to and from the pgBackRest repo pod.
func CreateSSHSecretIntent(postgresCluster *v1beta1.PostgresCluster,
	currentSSHSecret *v1.Secret) (v1.Secret, error) {

	meta := naming.PGBackRestSSHSecret(postgresCluster)
	meta.Annotations = naming.Merge(
		postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Archive.PGBackRest.Metadata.GetAnnotationsOrNil())
	meta.Labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Archive.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestRepoHostLabels(postgresCluster.GetName()),
	)

	secret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: meta,
		Type:       "Opaque",
	}

	var privKeyExists, pubKeyExists bool
	if currentSSHSecret != nil {
		_, privKeyExists = currentSSHSecret.Data[privateKey]
		_, pubKeyExists = currentSSHSecret.Data[publicKey]
	}
	var keys sshKey
	var err error
	if pubKeyExists && privKeyExists {
		keys = sshKey{
			Private: currentSSHSecret.Data[privateKey],
			Public:  currentSSHSecret.Data[publicKey],
		}
	} else {
		// get the key byte slices
		keys, err = getKeys()
		if err != nil {
			return secret, err
		}
	}

	// create an empty map for the key data
	initialize.ByteMap(&secret.Data)
	// if the public key data map is not ok, populate with the public key
	if _, ok := secret.Data[publicKey]; !ok {
		secret.Data[publicKey] = keys.Public
	}

	// if the private key data map is not ok, populate with the private key
	if _, ok := secret.Data[privateKey]; !ok {
		secret.Data[privateKey] = keys.Private
	}

	// if the known_hosts is not ok, populate with the knownHosts key
	if _, ok := secret.Data[knownHostsKey]; !ok {
		secret.Data[knownHostsKey] = []byte(fmt.Sprintf(
			"*.%s %s", naming.ClusterPodService(postgresCluster).Name, string(keys.Public)))
	}

	return secret, nil
}

// SSHConfigVolumeAndMount creates a volume and mount configuration from the SSHD configuration configmap
// and secret that will be used by the postgrescluster when connecting to the pgBackRest repo pod
func SSHConfigVolumeAndMount(sshConfigMap *v1.ConfigMap, sshSecret *v1.Secret, pod *v1.PodSpec, containerName string) {
	// Note: the 'container' string will be 'database' for the PostgreSQL database container,
	// otherwise it will be 'backrest'
	var (
		sshConfigVP []v1.VolumeProjection
	)

	volume := v1.Volume{Name: sshConfigVol}
	volume.Projected = &v1.ProjectedVolumeSource{}

	// Add our projections after those specified in the CR. Items later in the
	// list take precedence over earlier items (that is, last write wins).
	// - https://docs.openshift.com/container-platform/latest/nodes/containers/nodes-containers-projected-volumes.html
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	volume.Projected.Sources = append(
		sshConfigVP,
		v1.VolumeProjection{
			ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: sshConfigMap.Name,
				},
				Items: []v1.KeyToPath{{
					Key:  sshConfig,
					Path: "./" + sshConfig,
				}, {
					Key:  sshdConfig,
					Path: "./" + sshdConfig,
				}},
			},
			Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: sshConfigMap.Name,
				},
				Items: []v1.KeyToPath{{
					Key:  privateKey,
					Path: "./" + privateKey,
				}, {
					Key:  publicKey,
					Path: "./" + publicKey,
				}},
			},
		},
	)

	mount := v1.VolumeMount{
		Name:      volume.Name,
		MountPath: sshConfigPath,
		ReadOnly:  true,
	}

	pod.Volumes = mergeVolumes(pod.Volumes, volume)

	container := findOrAppendContainer(&pod.Containers, containerName)

	container.VolumeMounts = mergeVolumeMounts(container.VolumeMounts, mount)
}

// getSSHDConfigString returns a string consisting of the basic required configuration
// for the SSHD service
func getSSHDConfigString() string {

	// please note that the ForceCommand setting ensures nss_wrapper env vars are set when
	// executing commands as required for OpenShift compatibility:
	// https://access.redhat.com/articles/4859371
	configString := `AuthorizedKeysFile /etc/ssh/id_ecdsa.pub
ForceCommand NSS_WRAPPER_SUBDIR=postgres . /opt/crunchy/bin/nss_wrapper_env.sh && $SSH_ORIGINAL_COMMAND
HostKey /etc/ssh/id_ecdsa
PasswordAuthentication no
PermitRootLogin no
PidFile /tmp/sshd.pid
Port 2022
PubkeyAuthentication yes
StrictModes no
`
	return configString
}

// getSSHDConfigString returns a string consisting of the basic required configuration
// for the SSH client
func getSSHConfigString() string {

	configString := `Host *
StrictHostKeyChecking yes
IdentityFile /etc/ssh/id_ecdsa
Port 2022
User postgres
`
	return configString
}

// getKeys returns public/private byte slices of a ECDSA keypair using a P-521 curve
// formatted to be readable by OpenSSH
func getKeys() (sshKey, error) {
	var keys sshKey

	ecdsaPriv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return sshKey{}, err
	}

	pkiPriv := pki.NewPrivateKey(ecdsaPriv)

	keys.Private, err = pkiPriv.MarshalText()
	if err != nil {
		return sshKey{}, err
	}
	keys.Public, err = getECDSAPublicKey(&pkiPriv.PrivateKey.PublicKey)
	if err != nil {
		return sshKey{}, err
	}

	return keys, nil

}

// getECDSAPublicKey returns the ECDSA public key
// serialized for inclusion in an OpenSSH authorized_keys file
func getECDSAPublicKey(key *ecdsa.PublicKey) ([]byte, error) {
	pubKey, err := ssh.NewPublicKey(key)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(pubKey), nil
}
