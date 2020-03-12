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

package http

import (
	"net/url"
	"testing"
)

func TestClientURL(t *testing.T) {
	cases := map[string]struct {
		address  string
		route    string
		expected string
		vars     map[string]string
		params   map[string]string
	}{
		"simple": {
			address:  "http://example.com",
			route:    "endpoint",
			expected: "http://example.com/endpoint",
		},
		"simple with separator": {
			address:  "http://example.com",
			route:    "/endpoint",
			expected: "http://example.com/endpoint",
		},
		"simple with route var": {
			address:  "http://example.com",
			route:    "endpoint/{var}",
			vars:     map[string]string{"var": "value"},
			expected: "http://example.com/endpoint/value",
		},
		"simple with multiple route vars": {
			address:  "http://example.com",
			route:    "endpoint/{var1}/{var2}",
			vars:     map[string]string{"var1": "value1", "var2": "value2"},
			expected: "http://example.com/endpoint/value1/value2",
		},
		"simple with multiple identical route vars": {
			address:  "http://example.com",
			route:    "endpoint/{var}/{var}",
			vars:     map[string]string{"var": "value"},
			expected: "http://example.com/endpoint/value/value",
		},
		"simple with route var and query param": {
			address:  "http://example.com",
			route:    "endpoint/{var}",
			vars:     map[string]string{"var": "value"},
			params:   map[string]string{"foo": "bar"},
			expected: "http://example.com/endpoint/value?foo=bar",
		},
		"simple with port": {
			address:  "http://example.com:8080",
			route:    "endpoint",
			expected: "http://example.com:8080/endpoint",
		},
		"simple with port and route var": {
			address:  "http://example.com:8080",
			route:    "endpoint/{var}",
			vars:     map[string]string{"var": "value"},
			expected: "http://example.com:8080/endpoint/value",
		},
		"simple with port and multiple route vars": {
			address:  "http://example.com:8080",
			route:    "endpoint/{var1}/{var2}",
			vars:     map[string]string{"var1": "value1", "var2": "value2"},
			expected: "http://example.com:8080/endpoint/value1/value2",
		},
		"simple with port and multiple identical route vars": {
			address:  "http://example.com:8080",
			route:    "endpoint/{var}/{var}",
			vars:     map[string]string{"var": "value"},
			expected: "http://example.com:8080/endpoint/value/value",
		},
		"simple with port and separator": {
			address:  "http://example.com:8080",
			route:    "/endpoint",
			expected: "http://example.com:8080/endpoint",
		},
		"prefix": {
			address:  "http://example.com/prefix",
			route:    "endpoint",
			expected: "http://example.com/prefix/endpoint",
		},
		"prefix with separator": {
			address:  "http://example.com/prefix",
			route:    "/endpoint",
			expected: "http://example.com/prefix/endpoint",
		},
		"prefix with port": {
			address:  "http://example.com:8080/prefix",
			route:    "endpoint",
			expected: "http://example.com:8080/prefix/endpoint",
		},
		"prefix with port and separator": {
			address:  "http://example.com:8080/prefix",
			route:    "/endpoint",
			expected: "http://example.com:8080/prefix/endpoint",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			base, err := url.Parse(tc.address)
			if err != nil {
				t.Fatal(err)
			}

			c := httpClient{base: base}
			ep := c.URL(tc.route, tc.vars, tc.params)

			if ep.String() != tc.expected {
				t.Fatalf("expected: '%s', actual: '%s'", tc.expected, ep.String())
			}
		})
	}
}
