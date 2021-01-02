package kubeapi

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

import "k8s.io/apimachinery/pkg/api/errors"

// IsAlreadyExists returns true if the err indicates that a resource already exists.
func IsAlreadyExists(err error) bool { return errors.IsAlreadyExists(err) }

// IsNotFound returns true if err indicates that a resource was not found.
func IsNotFound(err error) bool { return errors.IsNotFound(err) }
