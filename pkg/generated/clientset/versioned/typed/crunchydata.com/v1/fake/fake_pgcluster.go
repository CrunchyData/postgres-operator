/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	crunchydatacomv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePgclusters implements PgclusterInterface
type FakePgclusters struct {
	Fake *FakeCrunchydataV1
	ns   string
}

var pgclustersResource = schema.GroupVersionResource{Group: "crunchydata.com", Version: "v1", Resource: "pgclusters"}

var pgclustersKind = schema.GroupVersionKind{Group: "crunchydata.com", Version: "v1", Kind: "Pgcluster"}

// Get takes name of the pgcluster, and returns the corresponding pgcluster object, and an error if there is any.
func (c *FakePgclusters) Get(name string, options v1.GetOptions) (result *crunchydatacomv1.Pgcluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(pgclustersResource, c.ns, name), &crunchydatacomv1.Pgcluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*crunchydatacomv1.Pgcluster), err
}

// List takes label and field selectors, and returns the list of Pgclusters that match those selectors.
func (c *FakePgclusters) List(opts v1.ListOptions) (result *crunchydatacomv1.PgclusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(pgclustersResource, pgclustersKind, c.ns, opts), &crunchydatacomv1.PgclusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &crunchydatacomv1.PgclusterList{ListMeta: obj.(*crunchydatacomv1.PgclusterList).ListMeta}
	for _, item := range obj.(*crunchydatacomv1.PgclusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested pgclusters.
func (c *FakePgclusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(pgclustersResource, c.ns, opts))

}

// Create takes the representation of a pgcluster and creates it.  Returns the server's representation of the pgcluster, and an error, if there is any.
func (c *FakePgclusters) Create(pgcluster *crunchydatacomv1.Pgcluster) (result *crunchydatacomv1.Pgcluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(pgclustersResource, c.ns, pgcluster), &crunchydatacomv1.Pgcluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*crunchydatacomv1.Pgcluster), err
}

// Update takes the representation of a pgcluster and updates it. Returns the server's representation of the pgcluster, and an error, if there is any.
func (c *FakePgclusters) Update(pgcluster *crunchydatacomv1.Pgcluster) (result *crunchydatacomv1.Pgcluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(pgclustersResource, c.ns, pgcluster), &crunchydatacomv1.Pgcluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*crunchydatacomv1.Pgcluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePgclusters) UpdateStatus(pgcluster *crunchydatacomv1.Pgcluster) (*crunchydatacomv1.Pgcluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(pgclustersResource, "status", c.ns, pgcluster), &crunchydatacomv1.Pgcluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*crunchydatacomv1.Pgcluster), err
}

// Delete takes name of the pgcluster and deletes it. Returns an error if one occurs.
func (c *FakePgclusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(pgclustersResource, c.ns, name), &crunchydatacomv1.Pgcluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePgclusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(pgclustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &crunchydatacomv1.PgclusterList{})
	return err
}

// Patch applies the patch and returns the patched pgcluster.
func (c *FakePgclusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *crunchydatacomv1.Pgcluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pgclustersResource, c.ns, name, pt, data, subresources...), &crunchydatacomv1.Pgcluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*crunchydatacomv1.Pgcluster), err
}
