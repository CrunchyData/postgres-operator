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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// createRequest is a helper function to create an http request.
func (h *httpAPI) createRequest(ctx context.Context, method, url string, msg interface{}) (*http.Request, error) {

	// First let's marshal the message to json, so that it can be included as
	// part of the request body.
	body, err := json.Marshal(msg)

	if err != nil {
		return nil, err
	}

	// Create the request with the provide context.
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// statusCheck ...
func statusCheck(resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed: %d", resp.StatusCode)
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	return nil
}
