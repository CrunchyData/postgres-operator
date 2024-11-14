// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	GR  = schema.GroupResource
	GV  = schema.GroupVersion
	GVK = schema.GroupVersionKind
	GVR = schema.GroupVersionResource
)

// These functions call the [runtime.DefaultUnstructuredConverter] with some additional type safety.
// An [unstructured.Unstructured] should always be paired with a [client.Object], and
// an [unstructured.UnstructuredList] should always be paired with a [client.ObjectList].

// FromUnstructuredList returns a copy of list by marshaling through JSON.
func FromUnstructuredList[
	// *T implements [client.ObjectList]
	T any, PT interface {
		client.ObjectList
		*T
	},
](list *unstructured.UnstructuredList) (*T, error) {
	result := new(T)
	return result, runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(list.UnstructuredContent(), result)
}

// FromUnstructuredObject returns a copy of object by marshaling through JSON.
func FromUnstructuredObject[
	// *T implements [client.Object]
	T any, PT interface {
		client.Object
		*T
	},
](object *unstructured.Unstructured) (*T, error) {
	result := new(T)
	return result, runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(object.UnstructuredContent(), result)
}

// ToUnstructuredList returns a copy of list by marshaling through JSON.
func ToUnstructuredList(list client.ObjectList) (*unstructured.UnstructuredList, error) {
	content, err := runtime.
		DefaultUnstructuredConverter.
		ToUnstructured(list)

	result := new(unstructured.UnstructuredList)
	result.SetUnstructuredContent(content)
	return result, err
}

// ToUnstructuredObject returns a copy of object by marshaling through JSON.
func ToUnstructuredObject(object client.Object) (*unstructured.Unstructured, error) {
	content, err := runtime.
		DefaultUnstructuredConverter.
		ToUnstructured(object)

	result := new(unstructured.Unstructured)
	result.SetUnstructuredContent(content)
	return result, err
}
