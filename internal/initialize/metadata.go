// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
