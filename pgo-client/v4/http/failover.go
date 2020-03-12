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

func (h *httpAPI) CreateFailover(ctx context.Context, r msgs.CreateFailoverRequest) (msgs.CreateFailoverResponse, error) {
	u := h.client.URL("failover", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreateFailoverResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreateFailoverResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreateFailoverResponse{}, err
	}

	var response msgs.CreateFailoverResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) QueryFailover(ctx context.Context, r msgs.QueryFailoverRequest) (msgs.QueryFailoverResponse, error) {
	vars := map[string]string{"name": r.ClusterName}
	params := map[string]string{
		"namespace": r.Namespace,
		"version":   r.ClientVersion,
	}

	u := h.client.URL("failover/{name}", vars, params)

	req, err := h.createRequest(ctx, http.MethodGet, u.String(), r)
	if err != nil {
		return msgs.QueryFailoverResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.QueryFailoverResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.QueryFailoverResponse{}, err
	}

	var response msgs.QueryFailoverResponse
	return response, json.Unmarshal(body, &response)
}
