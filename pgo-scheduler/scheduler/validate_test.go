package scheduler

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
		schedule, deployment, label, backupType string
		valid                                   bool
	}{
		{"pgbackrest", "testdeployment", "", "full", true},
		{"pgbackrest", "", "testlabel=label", "diff", true},
		{"pgbasebackup", "", "", "", false},
		{"policy", "", "", "", false},
		{"pgbackrest", "", "", "", false},
		{"pgbackrest", "", "", "full", false},
		{"pgbackrest", "testdeployment", "", "", false},
		{"pgbackrest", "", "testlabel=label", "", false},
		{"pgbackrest", "testdeployment", "", "foobar", false},
		{"pgbackrest", "", "testlabel=label", "foobar", false},
	}

	for i, test := range tests {
		err := ValidateBackRestSchedule(test.schedule, test.deployment, test.label, test.backupType)
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
