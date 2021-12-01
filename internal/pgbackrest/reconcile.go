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
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// AddRepoVolumesToPod adds pgBackRest repository volumes to the provided Pod template spec, while
// also adding associated volume mounts to the containers specified.
func AddRepoVolumesToPod(postgresCluster *v1beta1.PostgresCluster, template *corev1.PodTemplateSpec,
	repoPVCNames map[string]string, containerNames ...string) error {

	for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
		// we only care about repos created using PVCs
		if repo.Volume == nil {
			continue
		}

		var repoVolName string
		if repoPVCNames[repo.Name] != "" {
			// if there is an existing volume for this PVC, use it
			repoVolName = repoPVCNames[repo.Name]
		} else {
			// use the default name to create a new volume
			repoVolName = naming.PGBackRestRepoVolume(postgresCluster,
				repo.Name).Name
		}
		template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
			Name: repo.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: repoVolName},
			},
		})

		for _, name := range containerNames {
			var containerFound bool
			var index int
			for index = range template.Spec.Containers {
				if template.Spec.Containers[index].Name == name {
					containerFound = true
					break
				}
			}
			if !containerFound {
				return errors.Errorf("Unable to find container %q when adding pgBackRest repo volumes",
					name)
			}
			template.Spec.Containers[index].VolumeMounts =
				append(template.Spec.Containers[index].VolumeMounts, corev1.VolumeMount{
					Name:      repo.Name,
					MountPath: "/pgbackrest/" + repo.Name,
				})
		}
	}

	return nil
}

// AddConfigToInstancePod adds and mounts the pgBackRest configuration volumes
// for an instance of cluster to pod. The database container and any pgBackRest
// containers must already be in pod.
func AddConfigToInstancePod(
	cluster *v1beta1.PostgresCluster, pod *corev1.PodSpec,
) {
	configmap := corev1.VolumeProjection{ConfigMap: &corev1.ConfigMapProjection{}}
	configmap.ConfigMap.Name = naming.PGBackRestConfig(cluster).Name
	configmap.ConfigMap.Items = []corev1.KeyToPath{
		{Key: CMInstanceKey, Path: CMInstanceKey},
		{Key: ConfigHashKey, Path: ConfigHashKey},
	}

	// As the cluster transitions from having a repository host to having none,
	// PostgreSQL instances that have not rolled out expect to mount client
	// certificates. Specify those files are optional so the configuration
	// volumes stay valid and Kubernetes propagates their contents to those pods.
	secret := corev1.VolumeProjection{Secret: &corev1.SecretProjection{}}
	secret.Secret.Name = naming.PGBackRestSecret(cluster).Name
	secret.Secret.Optional = initialize.Bool(true)

	if DedicatedRepoHostEnabled(cluster) {
		// TODO(cbandy): add server config file
		secret.Secret.Items = append(secret.Secret.Items, clientCertificates()...)
	}

	// Start with a copy of projections specified in the cluster. Items later in
	// the list take precedence over earlier items (that is, last write wins).
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	sources := append([]corev1.VolumeProjection{},
		cluster.Spec.Backups.PGBackRest.Configuration...)

	if len(secret.Secret.Items) > 0 {
		sources = append(sources, configmap, secret)
	} else {
		sources = append(sources, configmap)
	}

	addConfigVolumeAndMounts(pod, sources)
}

// AddConfigToRepoPod adds and mounts the pgBackRest configuration volume for
// the dedicated repository host of cluster to pod. The pgBackRest containers
// must already be in pod.
func AddConfigToRepoPod(
	cluster *v1beta1.PostgresCluster, pod *corev1.PodSpec,
) {
	configmap := corev1.VolumeProjection{ConfigMap: &corev1.ConfigMapProjection{}}
	configmap.ConfigMap.Name = naming.PGBackRestConfig(cluster).Name
	configmap.ConfigMap.Items = []corev1.KeyToPath{
		{Key: CMRepoKey, Path: CMRepoKey},
		{Key: ConfigHashKey, Path: ConfigHashKey},
	}

	secret := corev1.VolumeProjection{Secret: &corev1.SecretProjection{}}
	secret.Secret.Name = naming.PGBackRestSecret(cluster).Name
	secret.Secret.Items = append(secret.Secret.Items, clientCertificates()...)

	// Start with a copy of projections specified in the cluster. Items later in
	// the list take precedence over earlier items (that is, last write wins).
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	sources := append([]corev1.VolumeProjection{},
		cluster.Spec.Backups.PGBackRest.Configuration...)

	addConfigVolumeAndMounts(pod, append(sources, configmap, secret))
}

// AddConfigToRestorePod adds and mounts the pgBackRest configuration volume to
// read from repositories of repoCluster to pod. The pgBackRest containers must
// already be in pod.
func AddConfigToRestorePod(
	repoCluster *v1beta1.PostgresCluster, pod *corev1.PodSpec,
) {
	configmap := corev1.VolumeProjection{ConfigMap: &corev1.ConfigMapProjection{}}
	configmap.ConfigMap.Name = naming.PGBackRestConfig(repoCluster).Name
	configmap.ConfigMap.Items = []corev1.KeyToPath{
		// TODO(cbandy): This may be the instance configuration of a cluster
		// different from the one we are building/creating. For now the
		// stanza options are "pg1-path", "pg1-port", and "pg1-socket-path"
		// and these are safe enough to use across different clusters running
		// the same PostgreSQL version. When that list grows, consider changing
		// this to use local stanza options and remote repository options.
		{Key: CMInstanceKey, Path: CMInstanceKey},
	}

	secret := corev1.VolumeProjection{Secret: &corev1.SecretProjection{}}
	secret.Secret.Name = naming.PGBackRestSecret(repoCluster).Name

	if DedicatedRepoHostEnabled(repoCluster) {
		secret.Secret.Items = append(secret.Secret.Items, clientCertificates()...)
	}

	// Start with a copy of projections specified in the cluster. Items later in
	// the list take precedence over earlier items (that is, last write wins).
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	sources := append([]corev1.VolumeProjection{},
		repoCluster.Spec.Backups.PGBackRest.Configuration...)

	addConfigVolumeAndMounts(pod, append(sources, configmap, secret))
}

// addConfigVolumeAndMounts adds the config projections to pod as the
// configuration volume. It mounts that volume to the database container and
// all pgBackRest containers in pod.
func addConfigVolumeAndMounts(
	pod *corev1.PodSpec, config []corev1.VolumeProjection,
) {
	configVolumeMount := corev1.VolumeMount{
		Name:      "pgbackrest-config",
		MountPath: configDirectory,
		ReadOnly:  true,
	}

	configVolume := corev1.Volume{
		Name: configVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{Sources: config},
		},
	}

	for i := range pod.Containers {
		container := &pod.Containers[i]

		switch container.Name {
		case
			naming.ContainerDatabase,
			naming.PGBackRestRepoContainerName,
			naming.PGBackRestRestoreContainerName:

			container.VolumeMounts = append(container.VolumeMounts, configVolumeMount)
		}
	}

	pod.Volumes = append(pod.Volumes, configVolume)
}

// AddSSHToPod populates a Pod template Spec with with the container and volumes needed to enable
// SSH within a Pod.  It will also mount the SSH configuration to any additional containers specified.
func AddSSHToPod(postgresCluster *v1beta1.PostgresCluster, template *corev1.PodTemplateSpec,
	enableSSHD bool, resources corev1.ResourceRequirements,
	additionalVolumeMountContainers ...string) error {

	sshConfigs := []corev1.VolumeProjection{}
	// stores all SSH configurations (ConfigMaps & Secrets)
	if postgresCluster.Spec.Backups.PGBackRest.RepoHost == nil ||
		postgresCluster.Spec.Backups.PGBackRest.RepoHost.SSHConfiguration == nil {
		sshConfigs = append(sshConfigs, corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: naming.PGBackRestSSHConfig(postgresCluster).Name,
				},
			},
		})
	} else {
		sshConfigs = append(sshConfigs, corev1.VolumeProjection{
			ConfigMap: postgresCluster.Spec.Backups.PGBackRest.RepoHost.SSHConfiguration,
		})
	}
	if postgresCluster.Spec.Backups.PGBackRest.RepoHost == nil ||
		postgresCluster.Spec.Backups.PGBackRest.RepoHost.SSHSecret == nil {
		sshConfigs = append(sshConfigs, corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: naming.PGBackRestSSHSecret(postgresCluster).Name,
				},
			},
		})
	} else {
		sshConfigs = append(sshConfigs, corev1.VolumeProjection{
			Secret: postgresCluster.Spec.Backups.PGBackRest.RepoHost.SSHSecret,
		})
	}
	template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
		Name: naming.PGBackRestSSHVolume,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources:     sshConfigs,
				DefaultMode: initialize.Int32(0o040),
			},
		},
	})

	sshVolumeMount := corev1.VolumeMount{
		Name:      naming.PGBackRestSSHVolume,
		MountPath: sshConfigPath,
		ReadOnly:  true,
	}

	// Only add the SSHD container if requested.  Sometimes (e.g. when running a restore Job) it is
	// not necessary to run a full SSHD server, but the various SSH configs are still needed.
	if enableSSHD {
		container := corev1.Container{
			Command:         []string{"/usr/sbin/sshd", "-D", "-e"},
			Image:           config.PGBackRestContainerImage(postgresCluster),
			ImagePullPolicy: postgresCluster.Spec.ImagePullPolicy,
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(2022),
					},
				},
			},
			Name:            naming.PGBackRestRepoContainerName,
			VolumeMounts:    []corev1.VolumeMount{sshVolumeMount},
			SecurityContext: initialize.RestrictedSecurityContext(),
			Resources:       resources,
		}

		// Mount PostgreSQL volumes if they are present in the template.
		postgresMounts := map[string]corev1.VolumeMount{
			postgres.DataVolumeMount().Name: postgres.DataVolumeMount(),
			postgres.WALVolumeMount().Name:  postgres.WALVolumeMount(),
		}
		for i := range template.Spec.Volumes {
			if mount, ok := postgresMounts[template.Spec.Volumes[i].Name]; ok {
				container.VolumeMounts = append(container.VolumeMounts, mount)
			}
		}

		template.Spec.Containers = append(template.Spec.Containers, container)
	}

	for _, name := range additionalVolumeMountContainers {
		var containerFound bool
		var index int
		for index = range template.Spec.Containers {
			if template.Spec.Containers[index].Name == name {
				containerFound = true
				break
			}
		}
		if !containerFound {
			return errors.Errorf("Unable to find container %q when adding pgBackRest to Pod",
				name)
		}
		template.Spec.Containers[index].VolumeMounts =
			append(template.Spec.Containers[index].VolumeMounts, sshVolumeMount)
	}

	return nil
}

// addServerContainerAndVolume adds the TLS server container and certificate
// projections to pod. Any PostgreSQL data and WAL volumes in pod are also mounted.
func addServerContainerAndVolume(
	cluster *v1beta1.PostgresCluster, pod *corev1.PodSpec,
	certificates []corev1.VolumeProjection, resources *corev1.ResourceRequirements,
) {
	serverVolumeMount := corev1.VolumeMount{
		Name:      "pgbackrest-server",
		MountPath: serverMountPath,
		ReadOnly:  true,
	}

	serverVolume := corev1.Volume{
		Name: serverVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{Sources: certificates},
		},
	}

	container := corev1.Container{
		Name:            naming.PGBackRestRepoContainerName,
		Command:         []string{"pgbackrest", "server-start"},
		Image:           config.PGBackRestContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		SecurityContext: initialize.RestrictedSecurityContext(),

		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"pgbackrest", "server-ping"},
				},
			},
		},

		VolumeMounts: []corev1.VolumeMount{serverVolumeMount},
	}

	if resources != nil {
		container.Resources = *resources
	}

	// Mount PostgreSQL volumes that are present in pod.
	postgresMounts := map[string]corev1.VolumeMount{
		postgres.DataVolumeMount().Name: postgres.DataVolumeMount(),
		postgres.WALVolumeMount().Name:  postgres.WALVolumeMount(),
	}
	for i := range pod.Volumes {
		if mount, ok := postgresMounts[pod.Volumes[i].Name]; ok {
			container.VolumeMounts = append(container.VolumeMounts, mount)
		}
	}

	pod.Containers = append(pod.Containers, container)
	pod.Volumes = append(pod.Volumes, serverVolume)
}

// AddServerToInstancePod adds the TLS server container and volume to pod for
// an instance of cluster. Any PostgreSQL volumes must already be in pod.
func AddServerToInstancePod(
	cluster *v1beta1.PostgresCluster, pod *corev1.PodSpec,
	instanceCertificateSecretName string,
) {
	certificates := []corev1.VolumeProjection{{
		Secret: &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: instanceCertificateSecretName,
			},
			Items: instanceServerCertificates(),
		},
	}}

	var resources *corev1.ResourceRequirements
	if sidecars := cluster.Spec.Backups.PGBackRest.Sidecars; sidecars != nil && sidecars.PGBackRest != nil {
		resources = sidecars.PGBackRest.Resources
	}

	addServerContainerAndVolume(cluster, pod, certificates, resources)
}

// AddServerToRepoPod adds the TLS server container and volume to pod for
// the dedicated repository host of cluster.
func AddServerToRepoPod(
	cluster *v1beta1.PostgresCluster, pod *corev1.PodSpec,
) {
	certificates := []corev1.VolumeProjection{{
		Secret: &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: naming.PGBackRestSecret(cluster).Name,
			},
			Items: repositoryServerCertificates(),
		},
	}}

	var resources *corev1.ResourceRequirements
	if cluster.Spec.Backups.PGBackRest.RepoHost != nil {
		resources = &cluster.Spec.Backups.PGBackRest.RepoHost.Resources
	}

	addServerContainerAndVolume(cluster, pod, certificates, resources)
}

// InstanceCertificates populates the shared Secret with certificates needed to run pgBackRest.
func InstanceCertificates(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inRoot pki.Certificate,
	inDNS pki.Certificate, inDNSKey pki.PrivateKey,
	outInstanceCertificates *corev1.Secret,
) error {
	var err error

	if DedicatedRepoHostEnabled(inCluster) {
		initialize.ByteMap(&outInstanceCertificates.Data)

		if err == nil {
			outInstanceCertificates.Data[certInstanceSecretKey], err = certFile(inDNS)
		}
		if err == nil {
			outInstanceCertificates.Data[certInstancePrivateKeySecretKey], err = certPrivateKey(inDNSKey)
		}
	}

	return err
}

// ReplicaCreateCommand returns the command that can initialize the PostgreSQL
// data directory on an instance from one of cluster's repositories. It returns
// nil when no repository is available.
func ReplicaCreateCommand(
	cluster *v1beta1.PostgresCluster, instance *v1beta1.PostgresInstanceSetSpec,
) []string {
	command := func(repoName string) []string {
		return []string{
			"pgbackrest", "restore", "--delta",
			"--stanza=" + DefaultStanzaName,
			"--repo=" + strings.TrimPrefix(repoName, "repo"),
			"--link-map=pg_wal=" + postgres.WALDirectory(cluster, instance),
		}
	}

	if cluster.Spec.Standby != nil && cluster.Spec.Standby.Enabled {
		// Patroni initializes standby clusters using the same command it uses
		// for any replica. Assume the repository in the spec has a stanza
		// and can be used to restore. The repository name is validated by the
		// Kubernetes API and begins with "repo".
		//
		// NOTE(cbandy): A standby cluster cannot use "online" stanza-create
		// nor create backups because every instance is always in recovery.
		return command(cluster.Spec.Standby.RepoName)
	}

	if cluster.Status.PGBackRest != nil {
		for _, repo := range cluster.Status.PGBackRest.Repos {
			if repo.ReplicaCreateBackupComplete {
				return command(repo.Name)
			}
		}
	}

	return nil
}

// RepoVolumeMount returns the name and mount path of the pgBackRest repo volume.
func RepoVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "pgbackrest-repo", MountPath: repoMountPath}
}

// Secret populates the pgBackRest Secret.
func Secret(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inRepoHost *appsv1.StatefulSet,
	inRoot *pki.RootCertificateAuthority,
	inSecret *corev1.Secret,
	outSecret *corev1.Secret,
) error {
	var err error

	// Save the CA and generate a TLS client certificate for the entire cluster.
	if inRepoHost != nil {
		initialize.ByteMap(&outSecret.Data)

		// The server verifies its "tls-server-auth" option contains the common
		// name (CN) of the certificate presented by a client. The entire
		// cluster uses a single client certificate so the "tls-server-auth"
		// option can stay the same when PostgreSQL instances and repository
		// hosts are added or removed.
		leaf := pki.NewLeafCertificate("", nil, nil)
		leaf.CommonName = clientCommonName(inCluster)
		leaf.DNSNames = []string{leaf.CommonName}

		if err == nil {
			var parse error
			if data, ok := inSecret.Data[certClientSecretKey]; parse == nil && ok {
				leaf.Certificate, parse = pki.ParseCertificate(data)
			}
			if data, ok := inSecret.Data[certClientPrivateKeySecretKey]; parse == nil && ok {
				leaf.PrivateKey, parse = pki.ParsePrivateKey(data)
			}
			if parse != nil || pki.LeafCertIsBad(ctx, leaf, inRoot, inCluster.Namespace) {
				err = errors.WithStack(leaf.Generate(inRoot))
			}
		}

		if err == nil {
			outSecret.Data[certAuthoritySecretKey], err = certAuthorities(*inRoot.Certificate)
		}
		if err == nil {
			outSecret.Data[certClientPrivateKeySecretKey], err = certPrivateKey(*leaf.PrivateKey)
		}
		if err == nil {
			outSecret.Data[certClientSecretKey], err = certFile(*leaf.Certificate)
		}
	}

	// Generate a TLS server certificate for each repository host.
	if inRepoHost != nil {
		// The client verifies the "pg-host" or "repo-host" option it used is
		// present in the DNS names of the server certificate.
		leaf := pki.NewLeafCertificate("", nil, nil)
		leaf.DNSNames = naming.RepoHostPodDNSNames(ctx, inRepoHost)
		leaf.CommonName = leaf.DNSNames[0] // FQDN

		if err == nil {
			var parse error
			if data, ok := inSecret.Data[certRepoSecretKey]; parse == nil && ok {
				leaf.Certificate, parse = pki.ParseCertificate(data)
			}
			if data, ok := inSecret.Data[certRepoPrivateKeySecretKey]; parse == nil && ok {
				leaf.PrivateKey, parse = pki.ParsePrivateKey(data)
			}
			if parse != nil || pki.LeafCertIsBad(ctx, leaf, inRoot, inCluster.Namespace) {
				err = errors.WithStack(leaf.Generate(inRoot))
			}
		}

		if err == nil {
			outSecret.Data[certRepoPrivateKeySecretKey], err = certPrivateKey(*leaf.PrivateKey)
		}
		if err == nil {
			outSecret.Data[certRepoSecretKey], err = certFile(*leaf.Certificate)
		}
	}

	return err
}
