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

func (h *httpAPI) ShowPgoRole(ctx context.Context, r msgs.ShowPgoRoleRequest) (msgs.ShowPgoRoleResponse, error) {
	u := h.client.URL("pgoroleshow", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.ShowPgoRoleResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowPgoRoleResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowPgoRoleResponse{}, err
	}

	var response msgs.ShowPgoRoleResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) CreatePgoRole(ctx context.Context, r msgs.CreatePgoRoleRequest) (msgs.CreatePgoRoleResponse, error) {
	u := h.client.URL("pgorolecreate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreatePgoRoleResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreatePgoRoleResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreatePgoRoleResponse{}, err
	}

	var response msgs.CreatePgoRoleResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) DeletePgoRole(ctx context.Context, r msgs.DeletePgoRoleRequest) (msgs.DeletePgoRoleResponse, error) {
	u := h.client.URL("pgoroledelete", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.DeletePgoRoleResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeletePgoRoleResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeletePgoRoleResponse{}, err
	}

	var response msgs.DeletePgoRoleResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) UpdatePgoRole(ctx context.Context, r msgs.UpdatePgoRoleRequest) (msgs.UpdatePgoRoleResponse, error) {
	u := h.client.URL("pgoroleupdate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.UpdatePgoRoleResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.UpdatePgoRoleResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.UpdatePgoRoleResponse{}, err
	}

	var response msgs.UpdatePgoRoleResponse
	return response, json.Unmarshal(body, &response)
}
