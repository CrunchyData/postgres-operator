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

func (h *httpAPI) CreatePgBouncer(ctx context.Context, r msgs.CreatePgbouncerRequest) (msgs.CreatePgbouncerResponse, error) {
	u := h.client.URL("pgbouncer", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreatePgbouncerResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreatePgbouncerResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreatePgbouncerResponse{}, err
	}

	var response msgs.CreatePgbouncerResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) DeletePgBouncer(ctx context.Context, r msgs.DeletePgbouncerRequest) (msgs.DeletePgbouncerResponse, error) {
	u := h.client.URL("pgbouncer", nil, nil)

	req, err := h.createRequest(ctx, http.MethodDelete, u.String(), r)
	if err != nil {
		return msgs.DeletePgbouncerResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeletePgbouncerResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeletePgbouncerResponse{}, err
	}

	var response msgs.DeletePgbouncerResponse
	return response, json.Unmarshal(body, &response)
}

// ShowPgBouncer makes an API call to the "show pgbouncer" apiserver endpoint
// and provides the results either using the ShowPgBouncer response format which
// may include an error
func (h *httpAPI) ShowPgBouncer(ctx context.Context, r msgs.ShowPgBouncerRequest) (msgs.ShowPgBouncerResponse, error) {
	u := h.client.URL("pgbouncer/show", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	var response msgs.ShowPgBouncerResponse
	return response, json.Unmarshal(body, &response)
}

// UpdatePgBouncer makes an API call to the "update pgbouncer" apiserver
// endpoint and provides the results
func (h *httpAPI) UpdatePgBouncer(ctx context.Context, r msgs.UpdatePgBouncerRequest) (msgs.UpdatePgBouncerResponse, error) {
	u := h.client.URL("pgbouncer", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPut, u.String(), r)
	if err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	var response msgs.UpdatePgBouncerResponse
	return response, json.Unmarshal(body, &response)
}
