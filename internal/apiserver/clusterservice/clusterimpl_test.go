package clusterservice

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

import (
	"testing"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
)

func TestIsMissingGCSConfig(t *testing.T) {
	setup := func(test func(t *testing.T)) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			apiserver.Pgo.Cluster.BackrestGCSBucket = "abc"
			apiserver.Pgo.Cluster.BackrestGCSEndpoint = "nyc.example.com"

			test(t)
		}
	}

	t.Run("valid", func(t *testing.T) {
		t.Run("defaults present", setup(func(t *testing.T) {
			request := &msgs.CreateClusterRequest{}
			if isMissingGCSConfig(request) {
				t.Errorf("expected no missing configuration, had missing configuration")
			}
		}))

		t.Run("bucket default missing, bucket request present", setup(func(t *testing.T) {
			apiserver.Pgo.Cluster.BackrestGCSBucket = ""
			request := &msgs.CreateClusterRequest{
				BackrestGCSBucket: "def",
			}
			if isMissingGCSConfig(request) {
				t.Errorf("expected no missing configuration, had missing configuration")
			}
		}))

		t.Run("bucket default present, bucket request missing", setup(func(t *testing.T) {
			request := &msgs.CreateClusterRequest{
				BackrestGCSEndpoint: "west.example.com",
			}
			if isMissingGCSConfig(request) {
				t.Errorf("expected no missing configuration, had missing configuration")
			}
		}))

		t.Run("endpoint default missing, endpoint request present", setup(func(t *testing.T) {
			apiserver.Pgo.Cluster.BackrestGCSEndpoint = ""
			request := &msgs.CreateClusterRequest{
				BackrestGCSEndpoint: "west.example.com",
			}
			if isMissingGCSConfig(request) {
				t.Errorf("expected no missing configuration, had missing configuration")
			}
		}))

		t.Run("endpoint default present, endpoint request missing", setup(func(t *testing.T) {
			request := &msgs.CreateClusterRequest{
				BackrestGCSBucket: "def",
			}
			if isMissingGCSConfig(request) {
				t.Errorf("expected no missing configuration, had missing configuration")
			}
		}))
	})

	t.Run("invalid", func(t *testing.T) {
		t.Run("bucket default and bucket request missing", setup(func(t *testing.T) {
			apiserver.Pgo.Cluster.BackrestGCSBucket = ""
			request := &msgs.CreateClusterRequest{
				BackrestGCSEndpoint: "west.example.com",
			}
			if !isMissingGCSConfig(request) {
				t.Errorf("expected missing configuration, instead returned false")
			}
		}))
	})
}

func TestValidateBackrestStorageTypeOnCreate(t *testing.T) {
	setup := func(test func(t *testing.T)) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			apiserver.Pgo.Cluster.BackrestGCSBucket = "abc"
			apiserver.Pgo.Cluster.BackrestGCSEndpoint = "nyc.example.com"
			apiserver.Pgo.Cluster.BackrestS3Bucket = "abc"
			apiserver.Pgo.Cluster.BackrestS3Endpoint = "nyc.example.com"
			apiserver.Pgo.Cluster.BackrestS3Region = "us-east-0"
			test(t)
		}
	}

	t.Run("valid", setup(func(t *testing.T) {
		requests := []struct {
			request  *msgs.CreateClusterRequest
			expected []crv1.BackrestStorageType
		}{
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: ""},
				expected: []crv1.BackrestStorageType{},
			},
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: "posix"},
				expected: []crv1.BackrestStorageType{"posix"},
			},
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: "local"},
				expected: []crv1.BackrestStorageType{"posix"},
			},
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: "gcs"},
				expected: []crv1.BackrestStorageType{"gcs"},
			},
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: "s3"},
				expected: []crv1.BackrestStorageType{"s3"},
			},
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: "posix,s3"},
				expected: []crv1.BackrestStorageType{"posix", "s3"},
			},
			{
				request:  &msgs.CreateClusterRequest{BackrestStorageType: "posix,gcs"},
				expected: []crv1.BackrestStorageType{"posix", "gcs"},
			},
		}

		for _, request := range requests {
			t.Run(request.request.BackrestStorageType, func(t *testing.T) {
				storageTypes, err := validateBackrestStorageTypeOnCreate(request.request)

				if err != nil {
					t.Errorf("expected no error, got: %s", err.Error())
				}

				if len(storageTypes) != len(request.expected) {
					t.Errorf("mismatching lengths in storage types. expected: %d, actual %d",
						len(request.expected), len(storageTypes))
				}

				// check equivalency
				c := 0
				for _, i := range request.expected {
					for _, j := range storageTypes {
						if i == j {
							c += 1
						}
					}
				}

				if c != len(request.expected) {
					t.Errorf("did not match expected storage types. expected: %v, actual: %v",
						request.expected, storageTypes)
				}
			})
		}
	}))

	t.Run("invalid", setup(func(t *testing.T) {
		requests := []struct {
			request *msgs.CreateClusterRequest
			name    string
		}{
			{
				request: &msgs.CreateClusterRequest{BackrestStorageType: "grumpy-cat"},
				name:    "bogus storage type",
			},
			{
				request: &msgs.CreateClusterRequest{BackrestStorageType: "s3,gcs"},
				name:    "gcs and s3",
			},
			{
				request: &msgs.CreateClusterRequest{BackrestStorageType: "s3"},
				name:    "incomplete s3",
			},
			{
				request: &msgs.CreateClusterRequest{BackrestStorageType: "gcs"},
				name:    "incomplete gcs",
			},
			{
				request: &msgs.CreateClusterRequest{BackrestStorageType: "gcs", BackrestGCSKeyType: "not-a-type"},
				name:    "bad gcs key type",
			},
		}

		for _, request := range requests {
			t.Run(request.name, func(t *testing.T) {
				// some additional setup
				switch request.name {
				case "incomplete s3":
					apiserver.Pgo.Cluster.BackrestS3Bucket = ""
				case "incomplete gcs":
					apiserver.Pgo.Cluster.BackrestGCSBucket = ""
				}

				if _, err := validateBackrestStorageTypeOnCreate(request.request); err == nil {
					t.Errorf("error was expected")
				}
			})
		}
	}))
}

func TestValidateStandbyCluster(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		request := &msgs.CreateClusterRequest{
			BackrestRepoPath: "/some/place/nice",
		}

		storageTypes := []string{
			"s3",
			"gcs",
			"posix,s3",
			"posix,gcs",
		}

		for _, storageType := range storageTypes {
			t.Run(storageType, func(t *testing.T) {
				request.BackrestStorageType = storageType
				if err := validateStandbyCluster(request); err != nil {
					t.Errorf("expected no error, got: %s", err.Error())
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Run("no repo path", func(t *testing.T) {
			request := &msgs.CreateClusterRequest{
				BackrestRepoPath:    "",
				BackrestStorageType: "s3",
			}
			if err := validateStandbyCluster(request); err == nil {
				t.Error("expected error")
			}
		})

		request := &msgs.CreateClusterRequest{
			BackrestRepoPath: "/some/place/nice",
		}

		storageTypes := []string{
			"posix",
			"local",
		}

		for _, storageType := range storageTypes {
			t.Run(storageType, func(t *testing.T) {
				request.BackrestStorageType = storageType
				if err := validateStandbyCluster(request); err == nil {
					t.Errorf("expected no error, got: %s", err.Error())
				}
			})
		}
	})
}
