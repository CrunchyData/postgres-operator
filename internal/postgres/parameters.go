/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgres

import (
	"strings"
)

// NewParameters returns ParameterSets required by this package.
func NewParameters() Parameters {
	parameters := Parameters{
		Mandatory: NewParameterSet(),
		Default:   NewParameterSet(),
	}

	// Use UNIX domain sockets for local connections.
	// PostgreSQL must be restarted when changing this value.
	parameters.Mandatory.Add("unix_socket_directories", SocketDirectory)

	return parameters
}

// Parameters is a pairing of ParameterSets.
type Parameters struct{ Mandatory, Default *ParameterSet }

// ParameterSet is a collection of PostgreSQL parameters.
// - https://www.postgresql.org/docs/current/config-setting.html
type ParameterSet struct {
	values map[string]string
}

// NewParameterSet returns an empty ParameterSet.
func NewParameterSet() *ParameterSet {
	return &ParameterSet{
		values: make(map[string]string),
	}
}

// AsMap returns a copy of ps as a map.
func (ps ParameterSet) AsMap() map[string]string {
	out := make(map[string]string, len(ps.values))
	for name, value := range ps.values {
		out[name] = value
	}
	return out
}

// DeepCopy returns a copy of ps.
func (ps *ParameterSet) DeepCopy() (out *ParameterSet) {
	return &ParameterSet{
		values: ps.AsMap(),
	}
}

// Add sets parameter name to value.
func (ps *ParameterSet) Add(name, value string) {
	ps.values[ps.normalize(name)] = value
}

// Get returns the value of parameter name and whether or not it was present in ps.
func (ps ParameterSet) Get(name string) (string, bool) {
	value, ok := ps.values[ps.normalize(name)]
	return value, ok
}

// Has returns whether or not parameter name is present in ps.
func (ps ParameterSet) Has(name string) bool {
	_, ok := ps.Get(name)
	return ok
}

func (ParameterSet) normalize(name string) string {
	// All parameter names are case-insensitive.
	// -- https://www.postgresql.org/docs/current/config-setting.html
	return strings.ToLower(name)
}

// Value returns empty string or the value of parameter name if it is present in ps.
func (ps ParameterSet) Value(name string) string {
	value, _ := ps.Get(name)
	return value
}
