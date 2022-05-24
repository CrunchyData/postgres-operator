package upgradeservice

/*
Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"fmt"
	"testing"
)

func TestUpgradeTagValid(t *testing.T) {
	tests := []struct {
		fromTag  string
		toTag    string
		expected bool
	}{
		// Bad tags
		// Too short
		{fromTag: "ubi8-12-4.7.4", toTag: "ubi8-12.10-4.7.5", expected: false},
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12-4.7.5", expected: false},
		// Too long
		{fromTag: "ubi8-12.9.10.3-4.7.4", toTag: "ubi8-12.10-4.7.5", expected: false},
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12.10.3.4-4.7.5", expected: false},
		// Not digits
		{fromTag: "ubi8-12.9.hello-4.7.4", toTag: "ubi8-12.10-4.7.5", expected: false},
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12.10.hello-4.7.5", expected: false},
		{fromTag: "ubi8-12.hello-4.7.4", toTag: "ubi8-12-4.7.5", expected: false},
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12.hello-4.7.5", expected: false},
		// Mismatched major version
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-13.10-4.7.5", expected: false},
		{fromTag: "ubi8-14.9-4.7.4", toTag: "ubi8-13.10-4.7.5", expected: false},
		// Patch should be absent if comparing minor values
		{fromTag: "ubi8-12.9.3-4.7.4", toTag: "ubi8-12.10-4.7.5", expected: false},
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12.10.3-4.7.5", expected: false},
		// From values higher than to values for minor or patch values
		// Note values chosen here are 9=>10 to test that int conversion occurs
		{fromTag: "ubi8-12.10-4.7.4", toTag: "ubi8-12.9-4.7.5", expected: false},
		{fromTag: "ubi8-12.7.10-4.7.4", toTag: "ubi8-12.7.9-4.7.5", expected: false},
		// Patch value partially absent
		{fromTag: "ubi8-12.9.3-4.7.4", toTag: "ubi8-12.9-4.7.5", expected: false},
		{fromTag: "ubi8-12.10-4.7.4", toTag: "ubi8-12.10.3-4.7.5", expected: false},
		// Valid
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12.10-4.7.5", expected: true},
		{fromTag: "ubi8-12.9-4.7.4", toTag: "ubi8-12.9-4.7.5", expected: true},
		{fromTag: "ubi8-12.9.9-4.7.4", toTag: "ubi8-12.9.10-4.7.5", expected: true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s=>%s", test.fromTag, test.toTag), func(t *testing.T) {
			if upgradeTagValid(test.fromTag, test.toTag) != test.expected {
				t.Fatalf("expected %t", test.expected)
			}
		})
	}
}
