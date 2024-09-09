// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

func FromCPU(n int64) *resource.Quantity {
	// Assume the Bridge API returns numbers that can be parsed by the
	// [resource] package.
	if q, err := resource.ParseQuantity(fmt.Sprint(n)); err == nil {
		return &q
	}

	return resource.NewQuantity(0, resource.DecimalSI)
}

// FromGibibytes returns n gibibytes as a [resource.Quantity].
func FromGibibytes(n int64) *resource.Quantity {
	// Assume the Bridge API returns numbers that can be parsed by the
	// [resource] package.
	if q, err := resource.ParseQuantity(fmt.Sprint(n) + "Gi"); err == nil {
		return &q
	}

	return resource.NewQuantity(0, resource.BinarySI)
}

// ToGibibytes returns q rounded up to a non-negative gibibyte.
func ToGibibytes(q resource.Quantity) int64 {
	v := q.Value()

	if v <= 0 {
		return 0
	}

	// https://stackoverflow.com/a/2745086
	return 1 + ((v - 1) >> 30)
}
