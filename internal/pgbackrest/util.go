// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"fmt"
	"hash/fnv"
	"io"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// maxPGBackrestRepos is the maximum number of repositories that can be configured according to the
// multi-repository solution implemented within pgBackRest
const maxPGBackrestRepos = 4

// RepoHostVolumeDefined determines whether not at least one pgBackRest dedicated
// repository host volume has been defined in the PostgresCluster manifest.
func RepoHostVolumeDefined(postgresCluster *v1beta1.PostgresCluster) bool {
	for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
		if repo.Volume != nil {
			return true
		}
	}
	return false
}

// CalculateConfigHashes calculates hashes for any external pgBackRest repository configuration
// present in the PostgresCluster spec (e.g. configuration for Azure, GCR and/or S3 repositories).
// Additionally it returns a hash of the hashes for each external repository.
func CalculateConfigHashes(
	postgresCluster *v1beta1.PostgresCluster) (map[string]string, string, error) {

	hashFunc := func(repoOpts []string) (string, error) {
		return safeHash32(func(w io.Writer) (err error) {
			for _, o := range repoOpts {
				_, err = w.Write([]byte(o))
			}
			return
		})
	}

	var err error
	repoConfigHashes := make(map[string]string)
	for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
		// hashes are only calculated for external repo configs
		if repo.Volume != nil {
			continue
		}

		var hash, name string
		switch {
		case repo.Azure != nil:
			hash, err = hashFunc([]string{repo.Azure.Container})
			name = repo.Name
		case repo.GCS != nil:
			hash, err = hashFunc([]string{repo.GCS.Bucket})
			name = repo.Name
		case repo.S3 != nil:
			hash, err = hashFunc([]string{repo.S3.Bucket, repo.S3.Endpoint, repo.S3.Region})
			name = repo.Name
		default:
			return map[string]string{}, "", errors.New("found unexpected repo type")
		}
		if err != nil {
			return map[string]string{}, "", errors.WithStack(err)
		}
		repoConfigHashes[name] = hash
	}

	configHashes := []string{}
	// ensure we always process in the same order
	for i := 1; i <= maxPGBackrestRepos; i++ {
		configName := fmt.Sprintf("repo%d", i)
		if _, ok := repoConfigHashes[configName]; ok {
			configHashes = append(configHashes, repoConfigHashes[configName])
		}
	}
	configHash, err := hashFunc(configHashes)
	if err != nil {
		return map[string]string{}, "", errors.WithStack(err)
	}

	return repoConfigHashes, configHash, nil
}

// safeHash32 runs content and returns a short alphanumeric string that
// represents everything written to w. The string is unlikely to have bad words
// and is safe to store in the Kubernetes API. This is the same algorithm used
// by ControllerRevision's "controller.kubernetes.io/hash".
func safeHash32(content func(w io.Writer) error) (string, error) {
	hash := fnv.New32()
	if err := content(hash); err != nil {
		return "", err
	}
	return rand.SafeEncodeString(fmt.Sprint(hash.Sum32())), nil
}
