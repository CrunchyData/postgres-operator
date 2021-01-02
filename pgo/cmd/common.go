package cmd

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

// unitType is used to group together the unit types
type unitType int

// values for the headings
const (
	headingCapacity     = "CAPACITY"
	headingCluster      = "CLUSTER"
	headingClusterIP    = "CLUSTER IP"
	headingErrorMessage = "ERROR"
	headingExpires      = "EXPIRES"
	headingExternalIP   = "EXTERNAL IP"
	headingInstance     = "INSTANCE"
	headingPassword     = "PASSWORD"
	headingPercentUsed  = "% USED"
	headingPod          = "POD"
	headingPVC          = "PVC"
	headingService      = "SERVICE"
	headingStatus       = "STATUS"
	headingPVCType      = "TYPE"
	headingUsed         = "USED"
	headingUsername     = "USERNAME"
)

// unitSize recommends the unit we will use to size things
const unitSize = 1024

// the collection of unittypes, from byte to yottabyte
const (
	unitB unitType = iota
	unitKB
	unitMB
	unitGB
	unitTB
	unitPB
	unitEB
	unitZB
	unitYB
)

// unitTypeToString converts the unit types to strings
var unitTypeToString = map[unitType]string{
	unitB:  "B",
	unitKB: "KiB",
	unitMB: "MiB",
	unitGB: "GiB",
	unitTB: "TiB",
	unitPB: "PiB",
	unitEB: "EiB",
	unitZB: "ZiB",
	unitYB: "YiB",
}

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

// getSizeAndUnit determines the best size to return based on the best unit
// where unit is KB, MB, GB, etc...
func getSizeAndUnit(size int64) (float64, unitType) {
	// set the unit
	var unit unitType
	// iterate through each tier, which we will initialize as "bytes"
	normalizedSize := float64(size)

	// We keep dividing by "unitSize" which is 1024. Once it is less than the unit
	// size, that is the normalized unit we will use.
	// The astute observer will note that "du" returns in units of 1024, but we
	// want to attempt to keep things in the 3-digit area
	//
	// of course, eventually this will get too big...so bail after yotta bytes
	for unit = unitB; normalizedSize > unitSize && unit < unitYB; unit++ {
		normalizedSize /= unitSize
	}

	return normalizedSize, unit
}

// getUnitString maps the raw value of the unit to its corresponding
// abbreviation
func getUnitString(unit unitType) string {
	return unitTypeToString[unit]
}

// printJSON renders a JSON response
func printJSON(response interface{}) {
	if content, err := json.MarshalIndent(response, "", "  "); err != nil {
		fmt.Printf(`{"error": "%s"}`, err.Error())
	} else {
		fmt.Println(string(content))
	}
}
