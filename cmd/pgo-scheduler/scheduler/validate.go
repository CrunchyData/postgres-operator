package scheduler

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"errors"
	"fmt"
	"strings"

	cv3 "github.com/robfig/cron/v3"
)

func validate(s ScheduleTemplate) error {
	if err := ValidateSchedule(s.Schedule); err != nil {
		return err
	}

	if err := ValidateScheduleType(s.Type); err != nil {
		return err
	}

	if err := ValidateBackRestSchedule(s.Type, s.Deployment, s.Label, s.PGBackRest.Type,
		s.PGBackRest.StorageType); err != nil {
		return err
	}

	if err := ValidatePolicySchedule(s.Type, s.Policy.Name, s.Policy.Database); err != nil {
		return err
	}

	return nil
}

// ValidateSchedule validates that the cron syntax is valid
// We use the standard format here...
func ValidateSchedule(schedule string) error {
	parser := cv3.NewParser(cv3.Minute | cv3.Hour | cv3.Dom | cv3.Month | cv3.Dow)

	if _, err := parser.Parse(schedule); err != nil {
		return fmt.Errorf("%s is not a valid schedule: ", schedule)
	}
	return nil
}

func ValidateScheduleType(schedule string) error {
	scheduleTypes := []string{
		"pgbackrest",
		"policy",
	}

	schedule = strings.ToLower(schedule)
	for _, scheduleType := range scheduleTypes {
		if schedule == scheduleType {
			return nil
		}
	}

	return fmt.Errorf("%s is not a valid schedule type", schedule)
}

func ValidateBackRestSchedule(scheduleType, deployment, label, backupType, storageType string) error {
	if scheduleType == "pgbackrest" {
		if deployment == "" && label == "" {
			return errors.New("Deployment or Label required for pgBackRest schedules")
		}

		if backupType == "" {
			return errors.New("Backup Type required for pgBackRest schedules")
		}

		validBackupTypes := []string{"full", "incr", "diff"}

		var valid bool
		for _, bType := range validBackupTypes {
			if backupType == bType {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("pgBackRest Backup Type invalid: %s", backupType)
		}

		validStorageTypes := []string{"local", "s3", "gcs"}
		for _, sType := range validStorageTypes {
			if storageType == sType {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("pgBackRest Backup Type invalid: %s", backupType)
		}
	}
	return nil
}

func ValidatePolicySchedule(scheduleType, policy, database string) error {
	if scheduleType == "policy" {
		if database == "" {
			return errors.New("Database name required for policy schedules")
		}
		if policy == "" {
			return errors.New("Policy name required for policy schedules")
		}
	}
	return nil
}
