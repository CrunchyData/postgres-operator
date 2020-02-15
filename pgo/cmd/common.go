package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"
	"reflect"
)

// values for the headings
const (
	headingCapacity     = "CAPACITY"
	headingCluster      = "CLUSTER"
	headingClusterIP    = "CLUSTER IP"
	headingErrorMessage = "ERROR"
	headingExternalIP   = "EXTERNAL IP"
	headingInstance     = "INSTANCE"
	headingPassword     = "PASWORD"
	headingPercentUsed  = "% USED"
	headingPod          = "POD"
	headingPVC          = "PVC"
	headingService      = "SERVICE"
	headingStatus       = "STATUS"
	headingPVCType      = "TYPE"
	headingUsed         = "USED"
	headingUsername     = "USERNAME"
)

// getHeaderLength returns the length of any value in a list, so that
// the maximum length of the header can be determined
func getHeaderLength(value interface{}, fieldName string) int {
	// get the field from the reflection
	r := reflect.ValueOf(value)
	field := reflect.Indirect(r).FieldByName(fieldName)
	return len(field.String())
}

// getMaxLength returns the maxLength of the strings of a particular value in
// the struct. Increases the max length by 1 to include a buffer
func getMaxLength(results []interface{}, title, fieldName string) int {
	maxLength := len(title)

	for _, result := range results {
		length := getHeaderLength(result, fieldName)

		if length > maxLength {
			maxLength = length
		}
	}

	return maxLength + 1
}

// printJSON renders a JSON response
func printJSON(response interface{}) {
	if content, err := json.MarshalIndent(response, "", "  "); err != nil {
		fmt.Printf(`{"error": "%s"}`, err.Error())
	} else {
		fmt.Println(string(content))
	}
}
