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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/crunchydata/postgres-operator/internal/apis/crunchydata.com/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PgtaskLister helps list Pgtasks.
type PgtaskLister interface {
	// List lists all Pgtasks in the indexer.
	List(selector labels.Selector) (ret []*v1.Pgtask, err error)
	// Pgtasks returns an object that can list and get Pgtasks.
	Pgtasks(namespace string) PgtaskNamespaceLister
	PgtaskListerExpansion
}

// pgtaskLister implements the PgtaskLister interface.
type pgtaskLister struct {
	indexer cache.Indexer
}

// NewPgtaskLister returns a new PgtaskLister.
func NewPgtaskLister(indexer cache.Indexer) PgtaskLister {
	return &pgtaskLister{indexer: indexer}
}

// List lists all Pgtasks in the indexer.
func (s *pgtaskLister) List(selector labels.Selector) (ret []*v1.Pgtask, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Pgtask))
	})
	return ret, err
}

// Pgtasks returns an object that can list and get Pgtasks.
func (s *pgtaskLister) Pgtasks(namespace string) PgtaskNamespaceLister {
	return pgtaskNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PgtaskNamespaceLister helps list and get Pgtasks.
type PgtaskNamespaceLister interface {
	// List lists all Pgtasks in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.Pgtask, err error)
	// Get retrieves the Pgtask from the indexer for a given namespace and name.
	Get(name string) (*v1.Pgtask, error)
	PgtaskNamespaceListerExpansion
}

// pgtaskNamespaceLister implements the PgtaskNamespaceLister
// interface.
type pgtaskNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Pgtasks in the indexer for a given namespace.
func (s pgtaskNamespaceLister) List(selector labels.Selector) (ret []*v1.Pgtask, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Pgtask))
	})
	return ret, err
}

// Get retrieves the Pgtask from the indexer for a given namespace and name.
func (s pgtaskNamespaceLister) Get(name string) (*v1.Pgtask, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("pgtask"), name)
	}
	return obj.(*v1.Pgtask), nil
}
