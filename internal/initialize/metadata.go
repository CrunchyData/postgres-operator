/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package initialize

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Annotations initializes the Annotations of object when they are nil.
func Annotations(object metav1.Object) {
	if object != nil && object.GetAnnotations() == nil {
		object.SetAnnotations(make(map[string]string))
	}
}

// Labels initializes the Labels of object when they are nil.
func Labels(object metav1.Object) {
	if object != nil && object.GetLabels() == nil {
		object.SetLabels(make(map[string]string))
	}
}
