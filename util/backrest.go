package util

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
	"errors"
	"fmt"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

// ValidateBackrestStorageTypeOnBackupRestore checks to see if the pgbackrest storage type provided
// when performing either pgbackrest backup or restore is valid.  This includes ensuring the value
// provided is a valid storage type (e.g. "s3" and/or "local").  This also includes ensuring the
// storage type specified (e.g. "s3" or "local") is enabled in the current cluster.  And finally,
// validation is ocurring for a restore, the ensure only one storage type is selected.
func ValidateBackrestStorageTypeOnBackupRestore(newBackRestStorageType,
	currentBackRestStorageType string, restore bool) error {

	if newBackRestStorageType != "" && !IsValidBackrestStorageType(newBackRestStorageType) {
		return fmt.Errorf("Invalid value provided for pgBackRest storage type. The following "+
			"values are allowed: %s", "\""+strings.Join(crv1.BackrestStorageTypes, "\", \"")+"\"")
	} else if newBackRestStorageType != "" &&
		strings.Contains(newBackRestStorageType, "s3") &&
		!strings.Contains(currentBackRestStorageType, "s3") {
		return errors.New("Storage type 's3' not allowed. S3 storage is not enabled for " +
			"pgBackRest in thiscluster")
	} else if (newBackRestStorageType == "" ||
		strings.Contains(newBackRestStorageType, "local")) &&
		(currentBackRestStorageType != "" &&
			!strings.Contains(currentBackRestStorageType, "local")) {
		return errors.New("Storage type 'local' not allowed. Local storage is not enabled for " +
			"pgBackRest in this cluster. If this cluster uses S3 storage only, specify 's3' " +
			"for the pgBackRest storage type.")
	}

	// storage type validation that is only applicable for restores
	if restore && newBackRestStorageType != "" &&
		len(strings.Split(newBackRestStorageType, ",")) > 1 {
		return fmt.Errorf("Multiple storage types cannot be selected cannot be select when "+
			"performing a restore. Please select one of the following: %s",
			"\""+strings.Join(crv1.BackrestStorageTypes, "\", \"")+"\"")
	}

	return nil
}

// IsValidBackrestStorageType determines if the storage source string contains valid pgBackRest
// storage type values
func IsValidBackrestStorageType(storageType string) bool {
	isValid := true
	for _, storageType := range strings.Split(storageType, ",") {
		if !IsStringOneOf(storageType, crv1.BackrestStorageTypes...) {
			isValid = false
			break
		}
	}
	return isValid
}
