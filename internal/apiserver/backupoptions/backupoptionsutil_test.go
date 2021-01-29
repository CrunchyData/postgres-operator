package backupoptions

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

import "testing"

func TestIsValidCompressType(t *testing.T) {
	tests := []struct {
		compressType string
		expected     bool
	}{
		{compressType: "bz2", expected: true},
		{compressType: "gz", expected: true},
		{compressType: "none", expected: true},
		{compressType: "lz4", expected: true},
		{compressType: "zst", expected: false},
		{compressType: "bogus", expected: false},
	}

	for _, test := range tests {
		t.Run(test.compressType, func(t *testing.T) {
			if isValidCompressType(test.compressType) != test.expected {
				t.Fatalf("expected %q to be %t", test.compressType, test.expected)
			}
		})
	}
}
