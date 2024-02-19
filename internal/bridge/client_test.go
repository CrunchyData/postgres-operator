/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	gocmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
)

// TestClientBackoff logs the backoff timing chosen by [NewClient] for use
// with `go test -v`.
func TestClientBackoff(t *testing.T) {
	client := NewClient("", "")
	var total time.Duration

	for i := 1; i <= 50 && client.Backoff.Steps > 0; i++ {
		step := client.Backoff.Step()
		total += step

		t.Logf("%02d:%20v%20v", i, step, total)
	}
}

func TestClientURL(t *testing.T) {
	assert.Equal(t, defaultAPI, NewClient("", "").BaseURL.String(),
		"expected the API constant to parse correctly")

	assert.Equal(t, defaultAPI, NewClient("/path", "").BaseURL.String())
	assert.Equal(t, defaultAPI, NewClient("http://:9999", "").BaseURL.String())
	assert.Equal(t, defaultAPI, NewClient("postgres://localhost", "").BaseURL.String())
	assert.Equal(t, defaultAPI, NewClient("postgres://localhost:5432", "").BaseURL.String())

	assert.Equal(t,
		"http://localhost:12345", NewClient("http://localhost:12345", "").BaseURL.String())
}

func TestClientDoWithBackoff(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		var bodies []string
		var requests []http.Request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(body))
			requests = append(requests, *r)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`some-response`))
		}))
		t.Cleanup(server.Close)

		// Client with one attempt, i.e. no backoff.
		client := NewClient(server.URL, "xyz")
		client.Backoff.Steps = 1
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx := context.Background()
		params := url.Values{}
		params.Add("foo", "bar")
		response, err := client.doWithBackoff(ctx,
			"ANY", "/some/path", params, []byte(`the-body`),
			http.Header{"Some": []string{"header"}})

		assert.NilError(t, err)
		assert.Assert(t, response != nil)
		t.Cleanup(func() { _ = response.Body.Close() })

		// Arguments became Request fields, including the client version.
		assert.Equal(t, len(requests), 1)
		assert.Equal(t, bodies[0], "the-body")
		assert.Equal(t, requests[0].Method, "ANY")
		assert.Equal(t, requests[0].URL.String(), "/some/path?foo=bar")
		assert.DeepEqual(t, requests[0].Header.Values("Some"), []string{"header"})
		assert.DeepEqual(t, requests[0].Header.Values("User-Agent"), []string{"PGO/xyz"})

		body, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(body), "some-response")
	})

	t.Run("Idempotency", func(t *testing.T) {
		var bodies []string
		var requests []http.Request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(body))
			requests = append(requests, *r)

			switch len(requests) {
			case 1, 2:
				w.WriteHeader(http.StatusBadGateway)
			default:
				w.WriteHeader(http.StatusNotAcceptable)
			}
		}))
		t.Cleanup(server.Close)

		// Client with brief backoff.
		client := NewClient(server.URL, "")
		client.Backoff.Duration = time.Millisecond
		client.Backoff.Steps = 5
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx := context.Background()
		response, err := client.doWithBackoff(ctx,
			"POST", "/anything", nil, []byte(`any-body`),
			http.Header{"Any": []string{"thing"}})

		assert.NilError(t, err)
		assert.Assert(t, response != nil)
		assert.NilError(t, response.Body.Close())

		assert.Equal(t, len(requests), 3, "expected multiple requests")

		// Headers include an Idempotency-Key.
		assert.Equal(t, bodies[0], "any-body")
		assert.Equal(t, requests[0].Header.Get("Any"), "thing")
		assert.Assert(t, requests[0].Header.Get("Idempotency-Key") != "")

		// Requests are identical, including the Idempotency-Key.
		assert.Equal(t, bodies[0], bodies[1])
		assert.DeepEqual(t, requests[0], requests[1],
			gocmpopts.IgnoreFields(http.Request{}, "Body"),
			gocmpopts.IgnoreUnexported(http.Request{}))

		assert.Equal(t, bodies[1], bodies[2])
		assert.DeepEqual(t, requests[1], requests[2],
			gocmpopts.IgnoreFields(http.Request{}, "Body"),
			gocmpopts.IgnoreUnexported(http.Request{}))

		// Another, identical request gets a new Idempotency-Key.
		response, err = client.doWithBackoff(ctx,
			"POST", "/anything", nil, []byte(`any-body`),
			http.Header{"Any": []string{"thing"}})

		assert.NilError(t, err)
		assert.Assert(t, response != nil)
		assert.NilError(t, response.Body.Close())

		prior := requests[0].Header.Get("Idempotency-Key")
		assert.Assert(t, len(requests) > 3)
		assert.Assert(t, requests[3].Header.Get("Idempotency-Key") != "")
		assert.Assert(t, requests[3].Header.Get("Idempotency-Key") != prior,
			"expected a new idempotency key")
	})

	t.Run("Backoff", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		// Client with brief backoff.
		client := NewClient(server.URL, "")
		client.Backoff.Duration = time.Millisecond
		client.Backoff.Steps = 5
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx := context.Background()
		_, err := client.doWithBackoff(ctx, "POST", "/any", nil, nil, nil) //nolint:bodyclose
		assert.ErrorContains(t, err, "timed out waiting")
		assert.Assert(t, requests > 0, "expected multiple requests")
	})

	t.Run("Cancellation", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)

		// Client with lots of brief backoff.
		client := NewClient(server.URL, "")
		client.Backoff.Duration = time.Millisecond
		client.Backoff.Steps = 100
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		t.Cleanup(cancel)

		_, err := client.doWithBackoff(ctx, "POST", "/any", nil, nil, nil) //nolint:bodyclose
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Assert(t, requests > 0, "expected multiple requests")
	})
}

func TestClientDoWithRetry(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		var bodies []string
		var requests []http.Request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(body))
			requests = append(requests, *r)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`some-response`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "xyz")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx := context.Background()
		params := url.Values{}
		params.Add("foo", "bar")
		response, err := client.doWithRetry(ctx,
			"ANY", "/some/path", params, []byte(`the-body`),
			http.Header{"Some": []string{"header"}})

		assert.NilError(t, err)
		assert.Assert(t, response != nil)
		t.Cleanup(func() { _ = response.Body.Close() })

		// Arguments became Request fields, including the client version.
		assert.Equal(t, len(requests), 1)
		assert.Equal(t, bodies[0], "the-body")
		assert.Equal(t, requests[0].Method, "ANY")
		assert.Equal(t, requests[0].URL.String(), "/some/path?foo=bar")
		assert.DeepEqual(t, requests[0].Header.Values("Some"), []string{"header"})
		assert.DeepEqual(t, requests[0].Header.Values("User-Agent"), []string{"PGO/xyz"})

		body, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(body), "some-response")
	})

	t.Run("Throttling", func(t *testing.T) {
		var bodies []string
		var requests []http.Request
		var times []time.Time
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(body))
			requests = append(requests, *r)
			times = append(times, time.Now())

			switch len(requests) {
			case 1:
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
			default:
				w.WriteHeader(http.StatusOK)
			}
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx := context.Background()
		response, err := client.doWithRetry(ctx,
			"POST", "/anything", nil, []byte(`any-body`),
			http.Header{"Any": []string{"thing"}})

		assert.NilError(t, err)
		assert.Assert(t, response != nil)
		assert.NilError(t, response.Body.Close())

		assert.Equal(t, len(requests), 2, "expected multiple requests")

		// Headers include an Idempotency-Key.
		assert.Equal(t, bodies[0], "any-body")
		assert.Equal(t, requests[0].Header.Get("Any"), "thing")
		assert.Assert(t, requests[0].Header.Get("Idempotency-Key") != "")

		// Requests are identical, except for the Idempotency-Key.
		assert.Equal(t, bodies[0], bodies[1])
		assert.DeepEqual(t, requests[0], requests[1],
			gocmpopts.IgnoreFields(http.Request{}, "Body"),
			gocmpopts.IgnoreUnexported(http.Request{}),
			gocmp.FilterPath(
				func(p gocmp.Path) bool { return p.String() == "Header" },
				gocmpopts.IgnoreMapEntries(
					func(k string, v []string) bool { return k == "Idempotency-Key" },
				),
			),
		)

		prior := requests[0].Header.Get("Idempotency-Key")
		assert.Assert(t, requests[1].Header.Get("Idempotency-Key") != "")
		assert.Assert(t, requests[1].Header.Get("Idempotency-Key") != prior,
			"expected a new idempotency key")

		// Requests are delayed according the server's response.
		// TODO: Mock the clock for faster tests.
		assert.Assert(t, times[0].Add(time.Second).Before(times[1]),
			"expected the second request over 1sec after the first")
	})

	t.Run("Cancellation", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		t.Cleanup(cancel)

		start := time.Now()
		_, err := client.doWithRetry(ctx, "POST", "/any", nil, nil, nil) //nolint:bodyclose
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Assert(t, time.Since(start) < time.Second)
		assert.Equal(t, requests, 1, "expected one request")
	})

	t.Run("UnexpectedResponse", func(t *testing.T) {
		for _, tt := range []struct {
			Name   string
			Send   func(http.ResponseWriter)
			Expect func(testing.TB, http.Response)
		}{
			{
				Name: "NoHeader",
				Send: func(w http.ResponseWriter) {
					w.WriteHeader(http.StatusTooManyRequests)
				},
				Expect: func(t testing.TB, r http.Response) {
					t.Helper()
					assert.Equal(t, r.StatusCode, http.StatusTooManyRequests)
				},
			},
			{
				Name: "ZeroHeader",
				Send: func(w http.ResponseWriter) {
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(http.StatusTooManyRequests)
				},
				Expect: func(t testing.TB, r http.Response) {
					t.Helper()
					assert.Equal(t, r.Header.Get("Retry-After"), "0")
					assert.Equal(t, r.StatusCode, http.StatusTooManyRequests)
				},
			},
			{
				Name: "NegativeHeader",
				Send: func(w http.ResponseWriter) {
					w.Header().Set("Retry-After", "-10")
					w.WriteHeader(http.StatusTooManyRequests)
				},
				Expect: func(t testing.TB, r http.Response) {
					t.Helper()
					assert.Equal(t, r.Header.Get("Retry-After"), "-10")
					assert.Equal(t, r.StatusCode, http.StatusTooManyRequests)
				},
			},
			{
				Name: "TextHeader",
				Send: func(w http.ResponseWriter) {
					w.Header().Set("Retry-After", "bogus")
					w.WriteHeader(http.StatusTooManyRequests)
				},
				Expect: func(t testing.TB, r http.Response) {
					t.Helper()
					assert.Equal(t, r.Header.Get("Retry-After"), "bogus")
					assert.Equal(t, r.StatusCode, http.StatusTooManyRequests)
				},
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					tt.Send(w)
				}))
				t.Cleanup(server.Close)

				client := NewClient(server.URL, "")
				assert.Equal(t, client.BaseURL.String(), server.URL)

				ctx := context.Background()
				response, err := client.doWithRetry(ctx, "POST", "/any", nil, nil, nil)
				assert.NilError(t, err)
				assert.Assert(t, response != nil)
				t.Cleanup(func() { _ = response.Body.Close() })

				tt.Expect(t, *response)

				assert.Equal(t, requests, 1, "expected no retries")
			})
		}
	})
}

func TestClientCreateAuthObject(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		var requests []http.Request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, len(body), 0)
			requests = append(requests, *r)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		ctx := context.Background()
		_, _ = client.CreateAuthObject(ctx, AuthObject{Secret: "sesame"})

		assert.Equal(t, len(requests), 1)
		assert.Equal(t, requests[0].Header.Get("Authorization"), "Bearer sesame")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`some info`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateAuthObject(context.Background(), AuthObject{})
		assert.ErrorContains(t, err, "authentication")
		assert.ErrorContains(t, err, "some info")
		assert.ErrorIs(t, err, errAuthentication)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`some message`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateAuthObject(context.Background(), AuthObject{})
		assert.ErrorContains(t, err, "404 Not Found")
		assert.ErrorContains(t, err, "some message")
	})

	t.Run("NoResponseBody", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateAuthObject(context.Background(), AuthObject{})
		assert.ErrorContains(t, err, "unexpected end")
		assert.ErrorContains(t, err, "JSON")
	})

	t.Run("ResponseNotJSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`asdf`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateAuthObject(context.Background(), AuthObject{})
		assert.ErrorContains(t, err, "invalid")
		assert.ErrorContains(t, err, "asdf")
	})
}

func TestClientCreateInstallation(t *testing.T) {
	t.Run("ErrorResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`any content, any format`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateInstallation(context.Background())
		assert.ErrorContains(t, err, "404 Not Found")
		assert.ErrorContains(t, err, "any content, any format")
	})

	t.Run("NoResponseBody", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateInstallation(context.Background())
		assert.ErrorContains(t, err, "unexpected end")
		assert.ErrorContains(t, err, "JSON")
	})

	t.Run("ResponseNotJSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`asdf`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err := client.CreateInstallation(context.Background())
		assert.ErrorContains(t, err, "invalid")
		assert.ErrorContains(t, err, "asdf")
	})
}
