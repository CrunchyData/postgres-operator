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

	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// mount path for SSH configuration
	sshConfigPath = "/sshd"

	// config file for the SSH client
	sshConfig = "config"
	// config file for the SSHD service
	sshdConfig = "sshd_config"

	// private key file name
	privateKey = "id_ecdsa"
	// public key file name
	publicKey = "id_ecdsa.pub"
	// SSH configuration volume
	sshConfigVol = "sshd"

	// suffix used with postgrescluster name for associated configmap.
	// for instance, if the cluster is named 'mycluster', the
	// configmap will be named 'mycluster-ssh-config'
	sshCMNameSuffix = "%s-ssh-config"

	// suffix used with postgrescluster name for associated secret.
	// for instance, if the cluster is named 'mycluster', the
	// secret will be named 'mycluster-ssh'
	sshSecretNameSuffix = "%s-ssh"
)

// sshKey stores byte slices that represent private and public ssh keys
// used to populate the postgrescluster's SSH secret
type sshKey struct {
	Private []byte
	Public  []byte
}

// CreateSSHConfigMapStruct creates a configmap struct with SSHD service and SSH client
// configuration settings in the data field.
func CreateSSHConfigMapStruct(postgresCluster *v1alpha1.PostgresCluster) v1.ConfigMap {

	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(sshCMNameSuffix, postgresCluster.GetName()),
			Namespace: postgresCluster.GetNamespace(),
		},
	}

	// create an empty map for the config data
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

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

// CreateSSHSecretStruct creates the secret containing the new public private key pair to use
// when connecting to and from the pgBackRest repo pod.
func CreateSSHSecretStruct(postgresCluster *v1alpha1.PostgresCluster) (v1.Secret, error) {

	secret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(sshSecretNameSuffix, postgresCluster.GetName()),
			Namespace: postgresCluster.GetNamespace(),
		},
		Type: "Opaque",
	}

	// get the key byte slices
	keys, err := getKeys()
	if err != nil {
		return secret, err
	}

	// create an empty map for the key data
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	// if the public key data map is not ok, populate with the public key
	if _, ok := secret.Data[publicKey]; !ok {
		secret.Data[publicKey] = keys.Public
	}

	// if the private key data map is not ok, populate with the private key
	if _, ok := secret.Data[privateKey]; !ok {
		//secret.Data[privateKey] = keys.Private
		secret.Data[privateKey] = keys.Private
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

	configString := `Port 2022
HostKey /sshd/id_ecdsa
SyslogFacility AUTHPRIV
PermitRootLogin no
StrictModes no
PubkeyAuthentication yes
AuthorizedKeysFile	/sshd/authorized_keys
PasswordAuthentication no
ChallengeResponseAuthentication yes
UsePAM yes
X11Forwarding yes
PidFile /tmp/sshd.pid

# Accept locale-related environment variables
AcceptEnv LANG LC_CTYPE LC_NUMERIC LC_TIME LC_COLLATE LC_MONETARY LC_MESSAGES
AcceptEnv LC_PAPER LC_NAME LC_ADDRESS LC_TELEPHONE LC_MEASUREMENT
AcceptEnv LC_IDENTIFICATION LC_ALL LANGUAGE
AcceptEnv XMODIFIERS

# override default of no subsystems
Subsystem	sftp	/usr/libexec/openssh/sftp-server
`
	return configString
}

// getSSHDConfigString returns a string consisting of the basic required configuration
// for the SSH client
func getSSHConfigString() string {

	configString := `Host *
	StrictHostKeyChecking no
	IdentityFile /sshd/id_ecdsa
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
