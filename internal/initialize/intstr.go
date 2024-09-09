// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
