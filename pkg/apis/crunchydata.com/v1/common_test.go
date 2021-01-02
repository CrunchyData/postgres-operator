package v1

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"reflect"
	"testing"
)

func TestPgStorageSpecGetSupplementalGroups(t *testing.T) {
	{
		groups := PgStorageSpec{}.GetSupplementalGroups()
		if len(groups) != 0 {
			t.Errorf("expected none, got %v", groups)
		}
	}
	{
		groups := PgStorageSpec{SupplementalGroups: "99"}.GetSupplementalGroups()
		if expected := []int64{99}; !reflect.DeepEqual(expected, groups) {
			t.Errorf("expected %v, got %v", expected, groups)
		}
	}
	{
		groups := PgStorageSpec{SupplementalGroups: "7,8,9"}.GetSupplementalGroups()
		if expected := []int64{7, 8, 9}; !reflect.DeepEqual(expected, groups) {
			t.Errorf("expected %v, got %v", expected, groups)
		}
	}
	{
		groups := PgStorageSpec{SupplementalGroups: "  "}.GetSupplementalGroups()
		if len(groups) != 0 {
			t.Errorf("expected none, got %v", groups)
		}
	}
	{
		groups := PgStorageSpec{SupplementalGroups: ", 5 "}.GetSupplementalGroups()
		if expected := []int64{5}; !reflect.DeepEqual(expected, groups) {
			t.Errorf("expected %v, got %v", expected, groups)
		}
	}
}
