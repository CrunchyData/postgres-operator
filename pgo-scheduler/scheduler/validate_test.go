package scheduler

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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

func TestValidSchedule(t *testing.T) {
	tests := []struct {
		schedule string
		valid    bool
	}{
		{"* * * * *", true},
		{"1 1 1 1 1", true},
		{"1-59/2 * * * *", true},
		{"*/2 * * * *", true},
		{"* * * * * * *", false},
		{"60 * * * *", false},
		{"* 24 * * *", false},
		{"* * 32 * *", false},
	}

	for i, test := range tests {
		err := ValidateSchedule(test.schedule)
		if test.valid && err != nil {
			t.Fatalf("tests[%d] - invalid schedule. expected valid, got invalid: %s",
				i, err)
		} else if !test.valid && err == nil {
			t.Fatalf("tests[%d] - valid schedule. expected invalid, got valid: %s",
				i, err)
		}
	}
}

func TestValidScheduleType(t *testing.T) {
	tests := []struct {
		schedule string
		valid    bool
	}{
		{"pgbackrest", true},
		{"pgbasebackup", true},
		{"policy", true},
		{"PGBACKREST", true},
		{"PGBASEBACKUP", true},
		{"POLICY", true},
		{"pgBackRest", true},
		{"pgBaseBackup", true},
		{"PoLiCY", true},
		{"FOO", false},
		{"BAR", false},
		{"foo", false},
		{"bar", false},
		{"", false},
	}

	for i, test := range tests {
		err := ValidateScheduleType(test.schedule)
		if test.valid && err != nil {
			t.Fatalf("tests[%d] - invalid schedule type. expected valid, got invalid: %s",
				i, err)
		} else if !test.valid && err == nil {
			t.Fatalf("tests[%d] - valid schedule. expected invalid, got valid: %s",
				i, err)
		}
	}
}

func TestValidBackRestSchedule(t *testing.T) {
	tests := []struct {
		schedule, deployment, label, backupType, storageType string
		valid                                                bool
	}{
		{"pgbackrest", "testdeployment", "", "full", "local", true},
		{"pgbackrest", "", "testlabel=label", "diff", "local", true},
		{"pgbackrest", "testdeployment", "", "full", "s3", true},
		{"pgbackrest", "", "testlabel=label", "diff", "s3", true},
		{"pgbasebackup", "", "", "", "local", false},
		{"policy", "", "", "", "local", false},
		{"pgbackrest", "", "", "", "local", false},
		{"pgbackrest", "", "", "full", "local", false},
		{"pgbackrest", "testdeployment", "", "", "local", false},
		{"pgbackrest", "", "testlabel=label", "", "local", false},
		{"pgbackrest", "testdeployment", "", "foobar", "local", false},
		{"pgbackrest", "", "testlabel=label", "foobar", "local", false},
		{"pgbackrest", "", "testlabel=label", "foobar", "", false},
	}

	for i, test := range tests {
		err := ValidateBackRestSchedule(test.schedule, test.deployment, test.label, test.backupType, test.storageType)
		if test.valid && err != nil {
			t.Fatalf("tests[%d] - invalid schedule type. expected valid, got invalid: %s",
				i, err)
		} else if !test.valid && err == nil {
			t.Fatalf("tests[%d] - valid schedule. expected invalid, got valid: %s",
				i, err)
		}
	}
}

func TestValidBaseBackupSchedule(t *testing.T) {
	tests := []struct {
		schedule, pvcName string
		valid             bool
	}{
		{"pgbasebackup", "mypvc", true},
		{"pgbasebackup", "", false},
	}

	for i, test := range tests {
		err := ValidateBaseBackupSchedule(test.schedule, test.pvcName)
		if test.valid && err != nil {
			t.Fatalf("tests[%d] - invalid schedule type. expected valid, got invalid: %s",
				i, err)
		} else if !test.valid && err == nil {
			t.Fatalf("tests[%d] - valid schedule. expected invalid, got valid: %s",
				i, err)
		}
	}
}

func TestValidSQLSchedule(t *testing.T) {
	tests := []struct {
		schedule, policy, database string
		valid                      bool
	}{
		{"policy", "mypolicy", "mydatabase", true},
		{"policy", "", "mydatabase", false},
		{"policy", "mypolicy", "", false},
	}

	for i, test := range tests {
		err := ValidatePolicySchedule(test.schedule, test.policy, test.database)
		if test.valid && err != nil {
			t.Fatalf("tests[%d] - invalid schedule type. expected valid, got invalid: %s",
				i, err)
		} else if !test.valid && err == nil {
			t.Fatalf("tests[%d] - valid schedule. expected invalid, got valid: %s",
				i, err)
		}
	}
}
