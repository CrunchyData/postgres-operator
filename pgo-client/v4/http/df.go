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

func (h *httpAPI) Df(ctx context.Context, r msgs.DfRequest) (msgs.DfResponse, error) {
	u := h.client.URL("df", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.DfResponse{}, err
	}

	resp, body, err := h.client.Do(req)

	if err != nil {
		return msgs.DfResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DfResponse{}, err
	}

	var response msgs.DfResponse
	return response, json.Unmarshal(body, &response)
}
