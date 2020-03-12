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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	api "github.com/crunchydata/postgres-operator/pgo-client/v4"
)

// Client is the interface implemented by clients that are intended to interact
// with a v4 postgres-operator API server.
type Client interface {
	URL(string, map[string]string, map[string]string) *url.URL
	Do(*http.Request) (*http.Response, []byte, error)
}

// Credentials is used to provide the user and password values for basic
// authentication purposes.
type Credentials struct {
	Username string
	Password string
}

// Config defines the properties of the client that can be configured.
type Config struct {
	// Address is the base URL for the postgres-operator API server.
	Address *url.URL

	// Credentials is the basic auth credentials required to authenticate with
	// the postgres-operator API server.
	Credentials Credentials

	// Transport specifies the mechanism by which individual HTTP requests are
	// made.
	Transport http.RoundTripper
}

// NewClient returns a configured HTTP client for use with a v4
// postgres-operator API implementation
func NewClient(cfg Config) (Client, error) {
	client := &http.Client{}

	if cfg.Transport != nil {
		client.Transport = cfg.Transport
	}

	cfg.Address.Path = strings.TrimRight(cfg.Address.Path, "/")

	return &httpClient{
		base:        cfg.Address,
		credentials: cfg.Credentials,
		client:      client,
	}, nil
}

type httpClient struct {
	base        *url.URL
	credentials Credentials
	client      *http.Client
}

// URL returns an endpoint URL using the clients configured base address for the
// postgres-operator API server. This function will construct an appropriate URL
// based on the route path, route variables and query parameters provided.
//
// Route variable substition is supported by this function using simple
// substitution. To utilize route variable substituion simply wrap the variable
// name with a `{}` pair.  Example, `/path/{name}` where `name:foo` would result
// in `/path/foo`. The mapping of the variable name to value should be provided
// in the `vars` map.
//
// Query Parameters are also supported. Example: `/path` where `params` defines
// a query parameter of `foo:bar`, would result in `/path?foo=bar`.
func (c *httpClient) URL(route string, vars map[string]string, params map[string]string) *url.URL {
	u := &url.URL{}
	*u = *c.base

	// Join the route to the base URL path.
	u.Path = path.Join(c.base.Path, route)

	// If route vars were provided then perform their substitution.
	if vars != nil {
		replacements := make([]string, 0, 2*len(vars))
		for name, value := range vars {
			replacements = append(replacements, fmt.Sprintf("{%s}", name), value)
		}
		u.Path = strings.NewReplacer(replacements...).Replace(u.Path)
	}

	// If params were provided, them add them to the URL Query.
	if params != nil {
		query := u.Query()
		for name, value := range params {
			query.Add(name, value)
		}
		u.RawQuery = query.Encode()
	}

	return u
}

// Do performs the request. Upon successful execution of the request, this
// function will also attempt to extract the body of the response.
func (c *httpClient) Do(r *http.Request) (*http.Response, []byte, error) {

	// Basic auth credentials were configured with the client, then ensure that
	// they are included with the outgoing request.
	if c.credentials != (Credentials{}) {
		r.SetBasicAuth(c.credentials.Username, c.credentials.Password)
	}

	// Perform the request.
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// Read the body from the response.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, body, nil
}

type httpAPI struct {
	client Client
}

// NewAPI returns a new API handle configured to use the provided client.
func NewAPI(cfg Config) (api.API, error) {
	client, err := NewClient(cfg)

	if err != nil {
		return nil, err
	}

	return &httpAPI{
		client: client,
	}, nil
}
