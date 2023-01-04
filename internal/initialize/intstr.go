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
	"k8s.io/apimachinery/pkg/util/intstr"
)

// IntOrStringInt32 returns an *intstr.IntOrString containing i.
func IntOrStringInt32(i int32) *intstr.IntOrString {
	return IntOrString(intstr.FromInt(int(i)))
}

// IntOrStringString returns an *intstr.IntOrString containing s.
func IntOrStringString(s string) *intstr.IntOrString {
	return IntOrString(intstr.FromString(s))
}

// IntOrString returns a pointer to the provided IntOrString
func IntOrString(ios intstr.IntOrString) *intstr.IntOrString {
	return &ios
}
