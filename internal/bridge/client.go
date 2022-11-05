/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
)

const defaultAPI = "https://api.crunchybridge.com"

var errAuthentication = errors.New("authentication failed")

type Client struct {
	http.Client
	wait.Backoff

	BaseURL url.URL
	Version string
}

// NewClient creates a Client with backoff settings that amount to
// ~10 attempts over ~2 minutes. A default is used when apiURL is not
// an acceptable URL.
func NewClient(apiURL, version string) *Client {
	// Use the default URL when the argument (1) does not parse at all, or
	// (2) has the wrong scheme, or (3) has no hostname.
	base, err := url.Parse(apiURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" {
		base, _ = url.Parse(defaultAPI)
	}

	return &Client{
		Backoff: wait.Backoff{
			Duration: time.Second,
			Factor:   1.6,
			Jitter:   0.2,
			Steps:    10,
			Cap:      time.Minute,
		},
		BaseURL: *base,
		Version: version,
	}
}

// doWithBackoff performs HTTP requests until:
//  1. ctx is cancelled,
//  2. the server returns a status code below 500, "Internal Server Error", or
//  3. the backoff is exhausted.
//
// Be sure to close the [http.Response] Body when the returned error is nil.
// See [http.Client.Do] for more details.
func (c *Client) doWithBackoff(
	ctx context.Context, method, path string, body []byte, headers http.Header,
) (
	*http.Response, error,
) {
	var response *http.Response

	// Prepare a copy of the passed in headers so we can manipulate them.
	if headers = headers.Clone(); headers == nil {
		headers = make(http.Header)
	}

	// Send a value that identifies this PATCH or POST request so it is safe to
	// retry when the server does not respond.
	// - https://docs.crunchybridge.com/api-concepts/idempotency/
	if method == http.MethodPatch || method == http.MethodPost {
		headers.Set("Idempotency-Key", string(uuid.NewUUID()))
	}

	headers.Set("User-Agent", "PGO/"+c.Version)
	url := c.BaseURL.JoinPath(path).String()

	err := wait.ExponentialBackoff(c.Backoff, func() (bool, error) {
		// NOTE: The [net/http] package treats an empty [bytes.Reader] the same as nil.
		request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))

		if err == nil {
			request.Header = headers.Clone()

			//nolint:bodyclose // This response is returned to the caller.
			response, err = c.Client.Do(request)
		}

		// An error indicates there was no response from the server, and the
		// request may not have finished. The "Idempotency-Key" header above
		// makes it safe to retry in this case.
		finished := err == nil

		// When the request finishes with a server error, discard the body and retry.
		// - https://docs.crunchybridge.com/api-concepts/getting-started/#status-codes
		if finished && response.StatusCode >= 500 {
			_ = response.Body.Close()
			finished = false
		}

		// Stop when the context is cancelled.
		return finished, ctx.Err()
	})

	// Discard the response body when there is a timeout from backoff.
	if response != nil && err != nil {
		_ = response.Body.Close()
	}

	// Return the last response, if any.
	// Return the cancellation or timeout from backoff, if any.
	return response, err
}

// doWithRetry performs HTTP requests until:
//  1. ctx is cancelled,
//  2. the server returns a status code below 500, "Internal Server Error",
//     that is not 429, "Too many requests", or
//  3. the backoff is exhausted.
//
// Be sure to close the [http.Response] Body when the returned error is nil.
// See [http.Client.Do] for more details.
func (c *Client) doWithRetry(
	ctx context.Context, method, path string, body []byte, headers http.Header,
) (
	*http.Response, error,
) {
	response, err := c.doWithBackoff(ctx, method, path, body, headers)

	// Retry the request when the server responds with "Too many requests".
	// - https://docs.crunchybridge.com/api-concepts/getting-started/#status-codes
	// - https://docs.crunchybridge.com/api-concepts/getting-started/#rate-limiting
	for err == nil && response.StatusCode == 429 {
		seconds, _ := strconv.Atoi(response.Header.Get("Retry-After"))

		// Only retry when the response indicates how long to wait.
		if seconds <= 0 {
			break
		}

		// Discard the "Too many requests" response body, and retry.
		_ = response.Body.Close()

		// Create a channel that sends after the delay indicated by the API.
		timer := time.NewTimer(time.Duration(seconds) * time.Second)
		defer timer.Stop()

		// Wait for the delay or context cancellation, whichever comes first.
		select {
		case <-timer.C:
			// Try the request again. Check it in the loop condition.
			response, err = c.doWithBackoff(ctx, method, path, body, headers)
			timer.Stop()

		case <-ctx.Done():
			// Exit the loop and return the context cancellation.
			err = ctx.Err()
		}
	}

	return response, err
}

func (c *Client) CreateAuthObject(ctx context.Context, authn AuthObject) (AuthObject, error) {
	var result AuthObject

	response, err := c.doWithRetry(ctx, "POST", "/vendor/operator/auth-objects", nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + authn.Secret},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		// 401, Unauthorized
		case response.StatusCode == 401:
			err = fmt.Errorf("%w: %s", errAuthentication, body)

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

func (c *Client) CreateInstallation(ctx context.Context) (Installation, error) {
	var result Installation

	response, err := c.doWithRetry(ctx, "POST", "/vendor/operator/installations", nil, http.Header{
		"Accept": []string{"application/json"},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}
