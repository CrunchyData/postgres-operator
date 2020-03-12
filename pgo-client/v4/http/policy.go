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

func (h *httpAPI) ShowPolicy(ctx context.Context, r msgs.ShowPolicyRequest) (msgs.ShowPolicyResponse, error) {
	u := h.client.URL("showpolicies", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.ShowPolicyResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowPolicyResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowPolicyResponse{}, err
	}

	var response msgs.ShowPolicyResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) CreatePolicy(ctx context.Context, r msgs.CreatePolicyRequest) (msgs.CreatePolicyResponse, error) {
	u := h.client.URL("policies", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreatePolicyResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreatePolicyResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreatePolicyResponse{}, err
	}

	var response msgs.CreatePolicyResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) DeletePolicy(ctx context.Context, r msgs.DeletePolicyRequest) (msgs.DeletePolicyResponse, error) {
	u := h.client.URL("policiesdelete", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.DeletePolicyResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeletePolicyResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeletePolicyResponse{}, err
	}

	var response msgs.DeletePolicyResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) ApplyPolicy(ctx context.Context, r msgs.ApplyPolicyRequest) (msgs.ApplyPolicyResponse, error) {
	u := h.client.URL("policies/apply", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.ApplyPolicyResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ApplyPolicyResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ApplyPolicyResponse{}, err
	}

	var response msgs.ApplyPolicyResponse
	return response, json.Unmarshal(body, &response)
}
