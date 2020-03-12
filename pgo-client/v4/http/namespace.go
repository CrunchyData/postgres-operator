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
	"context"
	"encoding/json"
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
)

func (h *httpAPI) ShowNamespace(ctx context.Context, r msgs.ShowNamespaceRequest) (msgs.ShowNamespaceResponse, error) {
	u := h.client.URL("namespace", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.ShowNamespaceResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowNamespaceResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowNamespaceResponse{}, err
	}

	var response msgs.ShowNamespaceResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) CreateNamespace(ctx context.Context, r msgs.CreateNamespaceRequest) (msgs.CreateNamespaceResponse, error) {
	u := h.client.URL("namespacecreate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.CreateNamespaceResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreateNamespaceResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreateNamespaceResponse{}, err
	}

	var response msgs.CreateNamespaceResponse
	return response, json.Unmarshal(body, &response)
}

func (h httpAPI) DeleteNamespace(ctx context.Context, r msgs.DeleteNamespaceRequest) (msgs.DeleteNamespaceResponse, error) {
	u := h.client.URL("namespacedelete", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.DeleteNamespaceResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeleteNamespaceResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeleteNamespaceResponse{}, err
	}

	var response msgs.DeleteNamespaceResponse
	return response, json.Unmarshal(body, &response)

}
func (h *httpAPI) UpdateNamespace(ctx context.Context, r msgs.UpdateNamespaceRequest) (msgs.UpdateNamespaceResponse, error) {
	u := h.client.URL("namespaceupdate", nil, nil)
	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.UpdateNamespaceResponse{}, err
	}

	resp, body, err := h.client.Do(req)

	if err != nil {
		return msgs.UpdateNamespaceResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.UpdateNamespaceResponse{}, err
	}

	var response msgs.UpdateNamespaceResponse
	return response, json.Unmarshal(body, &response)
}
