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

func (h *httpAPI) ShowUser(ctx context.Context, r msgs.ShowUserRequest) (msgs.ShowUserResponse, error) {
	u := h.client.URL("usershow", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.ShowUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowUserResponse{}, err
	}

	var response msgs.ShowUserResponse
	return response, json.Unmarshal(body, &response)
}
func (h *httpAPI) CreateUser(ctx context.Context, r msgs.CreateUserRequest) (msgs.CreateUserResponse, error) {
	u := h.client.URL("usercreate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreateUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreateUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreateUserResponse{}, err
	}

	var response msgs.CreateUserResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) DeleteUser(ctx context.Context, r msgs.DeleteUserRequest) (msgs.DeleteUserResponse, error) {
	u := h.client.URL("userdelete", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.DeleteUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeleteUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeleteUserResponse{}, err
	}

	var response msgs.DeleteUserResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) UpdateUser(ctx context.Context, r msgs.UpdateUserRequest) (msgs.UpdateUserResponse, error) {
	u := h.client.URL("userupdate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.UpdateUserResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.UpdateUserResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.UpdateUserResponse{}, err
	}

	var response msgs.UpdateUserResponse
	return response, json.Unmarshal(body, &response)
}
