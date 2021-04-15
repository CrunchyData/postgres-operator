package backrest

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
)

func TestGCSRepoTypeCLIOptionExists(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		options := []string{
			`--repo-type=gcs`,
			`--repo-type="gcs"`,
			`--repo-type='gcs'`,
			`--repo-type=gcs --another-option=yes`,
			`--another-option=yes --repo-type=gcs`,
		}

		for _, opts := range options {
			t.Run(opts, func(t *testing.T) {
				if !GCSRepoTypeCLIOptionExists(opts) {
					t.Error("expected valid options to return true")
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		options := []string{
			`--repo-type=s3`,
			`--repo-type=posix`,
			`--repo-typo=gcs`,
		}

		for _, opts := range options {
			t.Run(opts, func(t *testing.T) {
				if GCSRepoTypeCLIOptionExists(opts) {
					t.Error("expected invalid options to return false")
				}
			})
		}
	})
}

func TestS3RepoTypeCLIOptionExists(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		options := []string{
			`--repo-type=s3`,
			`--repo-type="s3"`,
			`--repo-type='s3'`,
			`--repo-type=s3 --another-option=yes`,
			`--another-option=yes --repo-type=s3`,
		}

		for _, opts := range options {
			t.Run(opts, func(t *testing.T) {
				if !S3RepoTypeCLIOptionExists(opts) {
					t.Error("expected valid options to return true")
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		options := []string{
			`--repo-type=gcs`,
			`--repo-type=posix`,
			`--repo-typo=s3`,
		}

		for _, opts := range options {
			t.Run(opts, func(t *testing.T) {
				if S3RepoTypeCLIOptionExists(opts) {
					t.Error("expected invalid options to return false")
				}
			})
		}
	})
}
