// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"io"
	"math/rand"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestCalculateConfigHashes(t *testing.T) {

	hashFunc := func(opts []string) (string, error) {
		return safeHash32(func(w io.Writer) (err error) {
			for _, o := range opts {
				_, err = w.Write([]byte(o))
			}
			return
		})
	}

	azureOpts, gcsOpts := []string{"container"}, []string{"container"}
	s3Opts := []string{"bucket", "endpoint", "region"}

	preCalculatedRepo1AzureHash, err := hashFunc(azureOpts)
	assert.NilError(t, err)
	preCalculatedRepo2GCSHash, err := hashFunc(gcsOpts)
	assert.NilError(t, err)
	preCalculatedRepo3S3Hash, err := hashFunc(s3Opts)
	assert.NilError(t, err)
	preCalculatedConfigHash, err := hashFunc([]string{preCalculatedRepo1AzureHash,
		preCalculatedRepo2GCSHash, preCalculatedRepo3S3Hash})
	assert.NilError(t, err)

	// create a PostgresCluster to test with
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-hashes",
			Namespace: "calculate-config-hashes",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Azure: &v1beta1.RepoAzure{
							Container: azureOpts[0],
						},
					}, {
						Name: "repo2",
						GCS: &v1beta1.RepoGCS{
							Bucket: gcsOpts[0],
						},
					}, {
						Name: "repo3",
						S3: &v1beta1.RepoS3{
							Bucket:   s3Opts[0],
							Endpoint: s3Opts[1],
							Region:   s3Opts[2],
						},
					}},
				},
			},
		},
	}

	configHashMap, configHash, err := CalculateConfigHashes(postgresCluster)
	assert.NilError(t, err)
	assert.Equal(t, preCalculatedConfigHash, configHash)
	assert.Equal(t, preCalculatedRepo1AzureHash, configHashMap["repo1"])
	assert.Equal(t, preCalculatedRepo2GCSHash, configHashMap["repo2"])
	assert.Equal(t, preCalculatedRepo3S3Hash, configHashMap["repo3"])

	// call CalculateConfigHashes multiple times to ensure consistent results
	for i := 0; i < 10; i++ {
		hashMap, hash, err := CalculateConfigHashes(postgresCluster)
		assert.NilError(t, err)
		assert.Equal(t, configHash, hash)
		assert.Equal(t, configHashMap["repo1"], hashMap["repo1"])
		assert.Equal(t, configHashMap["repo2"], hashMap["repo2"])
		assert.Equal(t, configHashMap["repo3"], hashMap["repo3"])
	}

	// shuffle the repo slice in order to ensure the same result is returned regardless of the
	// order of the repos slice
	shuffleCluster := postgresCluster.DeepCopy()
	for i := 0; i < 10; i++ {
		repos := shuffleCluster.Spec.Backups.PGBackRest.Repos
		rand.Shuffle(len(repos), func(i, j int) {
			repos[i], repos[j] = repos[j], repos[i]
		})
		_, hash, err := CalculateConfigHashes(shuffleCluster)
		assert.NilError(t, err)
		assert.Equal(t, configHash, hash)
	}

	// now modify some values in each repo and confirm we see a different result
	for i := 0; i < 3; i++ {
		modCluster := postgresCluster.DeepCopy()
		switch i {
		case 0:
			modCluster.Spec.Backups.PGBackRest.Repos[i].Azure.Container = "modified-container"
		case 1:
			modCluster.Spec.Backups.PGBackRest.Repos[i].GCS.Bucket = "modified-bucket"
		case 2:
			modCluster.Spec.Backups.PGBackRest.Repos[i].S3.Bucket = "modified-bucket"
		}
		hashMap, hash, err := CalculateConfigHashes(modCluster)
		assert.NilError(t, err)
		assert.Assert(t, configHash != hash)
		assert.NilError(t, err)
		repo := "repo" + strconv.Itoa(i+1)
		assert.Assert(t, hashMap[repo] != configHashMap[repo])
	}
}
