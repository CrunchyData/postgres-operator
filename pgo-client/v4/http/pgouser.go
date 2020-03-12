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

func (h *httpAPI) ShowPgoUser(ctx context.Context, r msgs.ShowPgoUserRequest) (msgs.ShowPgoUserResponse, error) {
	u := h.client.URL("pgousershow", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.ShowPgoUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowPgoUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowPgoUserResponse{}, err
	}

	var response msgs.ShowPgoUserResponse
	return response, json.Unmarshal(body, &response)

}
func (h *httpAPI) CreatePgoUser(ctx context.Context, r msgs.CreatePgoUserRequest) (msgs.CreatePgoUserResponse, error) {
	u := h.client.URL("pgousercreate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreatePgoUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreatePgoUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreatePgoUserResponse{}, err
	}

	var response msgs.CreatePgoUserResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) DeletePgoUser(ctx context.Context, r msgs.DeletePgoUserRequest) (msgs.DeletePgoUserResponse, error) {
	u := h.client.URL("pgouserdelete", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.DeletePgoUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeletePgoUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeletePgoUserResponse{}, err
	}

	var response msgs.DeletePgoUserResponse
	return response, json.Unmarshal(body, &response)

}

func (h *httpAPI) UpdatePgoUser(ctx context.Context, r msgs.UpdatePgoUserRequest) (msgs.UpdatePgoUserResponse, error) {
	u := h.client.URL("pgouserupdate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.UpdatePgoUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.UpdatePgoUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.UpdatePgoUserResponse{}, err
	}

	var response msgs.UpdatePgoUserResponse
	return response, json.Unmarshal(body, &response)
}
