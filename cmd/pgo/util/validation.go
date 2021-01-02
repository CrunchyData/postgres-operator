package util

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
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
)

var validResourceName = regexp.MustCompile(`^[a-z0-9.\-]+$`).MatchString

// validates whether a string meets requirements for a valid resource name for kubernetes.
// It can consist of lowercase alphanumeric characters, '-' and '.', per
//
//      https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
//
func IsValidForResourceName(target string) bool {
	log.Debugf("IsValidForResourceName: %s", target)

	return validResourceName(target)
}

// ValidateQuantity runs the Kubernetes "ParseQuantity" function on a string
// and determine whether or not it is a valid quantity object. Returns an error
// if it is invalid, along with the error message pertaining to the specific
// flag.
//
// Does nothing if no value is passed in
//
// See: https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go
func ValidateQuantity(quantity, flag string) error {
	if quantity != "" {
		if _, err := resource.ParseQuantity(quantity); err != nil {
			return fmt.Errorf("Error: \"%s\" - %w", flag, err)
		}
	}

	return nil
}
