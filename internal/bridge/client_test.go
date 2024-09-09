// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	gocmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

var testApiKey = "9012"
var testTeamId = "5678"

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

func TestListClusters(t *testing.T) {
	responsePayload := &ClusterList{
		Clusters: []*ClusterApiResource{},
	}
	firstClusterApiResource := &ClusterApiResource{
		ID: "1234",
	}
	secondClusterApiResource := &ClusterApiResource{
		ID: "2345",
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "GET", "Expected GET method")
			assert.Equal(t, r.URL.Path, "/clusters", "Expected path to be '/clusters'")
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")
			assert.Equal(t, r.URL.Query()["team_id"][0], testTeamId, "Expected query params to contain team id.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.ListClusters(context.Background(), testApiKey, testTeamId)
		assert.NilError(t, err)
	})

	t.Run("OkResponseNoClusters", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusters, err := client.ListClusters(context.Background(), testApiKey, testTeamId)
		assert.NilError(t, err)
		assert.Equal(t, len(clusters), 0)
	})

	t.Run("OkResponseOneCluster", func(t *testing.T) {
		responsePayload.Clusters = append(responsePayload.Clusters, firstClusterApiResource)
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusters, err := client.ListClusters(context.Background(), testApiKey, testTeamId)
		assert.NilError(t, err)
		assert.Equal(t, len(clusters), 1)
		assert.Equal(t, clusters[0].ID, responsePayload.Clusters[0].ID)
	})

	t.Run("OkResponseTwoClusters", func(t *testing.T) {
		responsePayload.Clusters = append(responsePayload.Clusters, secondClusterApiResource)
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusters, err := client.ListClusters(context.Background(), testApiKey, testTeamId)
		assert.NilError(t, err)
		assert.Equal(t, len(clusters), 2)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.ListClusters(context.Background(), testApiKey, testTeamId)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestCreateCluster(t *testing.T) {
	clusterApiResource := &ClusterApiResource{
		ClusterName: "test-cluster1",
	}
	clusterRequestPayload := &PostClustersRequestPayload{
		Name: "test-cluster1",
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var receivedPayload PostClustersRequestPayload
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&receivedPayload)
			assert.NilError(t, err)
			assert.Equal(t, r.Method, "POST", "Expected POST method")
			assert.Equal(t, r.URL.Path, "/clusters", "Expected path to be '/clusters'")
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")
			assert.Equal(t, receivedPayload, *clusterRequestPayload)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.CreateCluster(context.Background(), testApiKey, clusterRequestPayload)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		newCluster, err := client.CreateCluster(context.Background(), testApiKey, clusterRequestPayload)
		assert.NilError(t, err)
		assert.Equal(t, newCluster.ClusterName, clusterApiResource.ClusterName)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.CreateCluster(context.Background(), testApiKey, clusterRequestPayload)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestDeleteCluster(t *testing.T) {
	clusterId := "1234"
	clusterApiResource := &ClusterApiResource{
		ClusterName: "test-cluster1",
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "DELETE", "Expected DELETE method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId, "Expected path to be /clusters/"+clusterId)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, _, err = client.DeleteCluster(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		deletedCluster, deletedAlready, err := client.DeleteCluster(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, deletedCluster.ClusterName, clusterApiResource.ClusterName)
		assert.Equal(t, deletedAlready, false)
	})

	t.Run("GoneResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGone)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, deletedAlready, err := client.DeleteCluster(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, deletedAlready, true)
	})

	t.Run("NotFoundResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, deletedAlready, err := client.DeleteCluster(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, deletedAlready, true)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, _, err = client.DeleteCluster(context.Background(), testApiKey, clusterId)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestGetCluster(t *testing.T) {
	clusterId := "1234"
	clusterApiResource := &ClusterApiResource{
		ClusterName: "test-cluster1",
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "GET", "Expected GET method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId, "Expected path to be /clusters/"+clusterId)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetCluster(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		cluster, err := client.GetCluster(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, cluster.ClusterName, clusterApiResource.ClusterName)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetCluster(context.Background(), testApiKey, clusterId)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestGetClusterStatus(t *testing.T) {
	clusterId := "1234"
	state := "Ready"

	clusterStatusApiResource := &ClusterStatusApiResource{
		State: state,
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterStatusApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "GET", "Expected GET method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId+"/status", "Expected path to be /clusters/"+clusterId+"/status")
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetClusterStatus(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterStatusApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterStatus, err := client.GetClusterStatus(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, clusterStatus.State, state)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterStatusApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetClusterStatus(context.Background(), testApiKey, clusterId)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestGetClusterUpgrade(t *testing.T) {
	clusterId := "1234"
	clusterUpgradeApiResource := &ClusterUpgradeApiResource{
		ClusterID: clusterId,
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "GET", "Expected GET method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId+"/upgrade", "Expected path to be /clusters/"+clusterId+"/upgrade")
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetClusterUpgrade(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterUpgrade, err := client.GetClusterUpgrade(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, clusterUpgrade.ClusterID, clusterId)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetClusterUpgrade(context.Background(), testApiKey, clusterId)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestUpgradeCluster(t *testing.T) {
	clusterId := "1234"
	clusterUpgradeApiResource := &ClusterUpgradeApiResource{
		ClusterID: clusterId,
	}
	clusterUpgradeRequestPayload := &PostClustersUpgradeRequestPayload{
		Plan:             "standard-8",
		PostgresVersion:  intstr.FromInt(15),
		UpgradeStartTime: "start-time",
		Storage:          10,
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var receivedPayload PostClustersUpgradeRequestPayload
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&receivedPayload)
			assert.NilError(t, err)
			assert.Equal(t, r.Method, "POST", "Expected POST method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId+"/upgrade", "Expected path to be /clusters/"+clusterId+"/upgrade")
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")
			assert.Equal(t, receivedPayload, *clusterUpgradeRequestPayload)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.UpgradeCluster(context.Background(), testApiKey, clusterId, clusterUpgradeRequestPayload)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterUpgrade, err := client.UpgradeCluster(context.Background(), testApiKey, clusterId, clusterUpgradeRequestPayload)
		assert.NilError(t, err)
		assert.Equal(t, clusterUpgrade.ClusterID, clusterId)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.UpgradeCluster(context.Background(), testApiKey, clusterId, clusterUpgradeRequestPayload)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestUpgradeClusterHA(t *testing.T) {
	clusterId := "1234"
	action := "enable-ha"
	clusterUpgradeApiResource := &ClusterUpgradeApiResource{
		ClusterID: clusterId,
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "PUT", "Expected PUT method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId+"/actions/"+action,
				"Expected path to be /clusters/"+clusterId+"/actions/"+action)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.UpgradeClusterHA(context.Background(), testApiKey, clusterId, action)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterUpgrade, err := client.UpgradeClusterHA(context.Background(), testApiKey, clusterId, action)
		assert.NilError(t, err)
		assert.Equal(t, clusterUpgrade.ClusterID, clusterId)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterUpgradeApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.UpgradeClusterHA(context.Background(), testApiKey, clusterId, action)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestUpdateCluster(t *testing.T) {
	clusterId := "1234"
	clusterApiResource := &ClusterApiResource{
		ClusterName: "new-cluster-name",
	}
	clusterUpdateRequestPayload := &PatchClustersRequestPayload{
		IsProtected: initialize.Bool(true),
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var receivedPayload PatchClustersRequestPayload
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&receivedPayload)
			assert.NilError(t, err)
			assert.Equal(t, r.Method, "PATCH", "Expected PATCH method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId, "Expected path to be /clusters/"+clusterId)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")
			assert.Equal(t, *receivedPayload.IsProtected, *clusterUpdateRequestPayload.IsProtected)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.UpdateCluster(context.Background(), testApiKey, clusterId, clusterUpdateRequestPayload)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterUpdate, err := client.UpdateCluster(context.Background(), testApiKey, clusterId, clusterUpdateRequestPayload)
		assert.NilError(t, err)
		assert.Equal(t, clusterUpdate.ClusterName, clusterApiResource.ClusterName)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.UpdateCluster(context.Background(), testApiKey, clusterId, clusterUpdateRequestPayload)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestGetClusterRole(t *testing.T) {
	clusterId := "1234"
	roleName := "application"
	clusterRoleApiResource := &ClusterRoleApiResource{
		Name: roleName,
	}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterRoleApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "GET", "Expected GET method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId+"/roles/"+roleName,
				"Expected path to be /clusters/"+clusterId+"/roles/"+roleName)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetClusterRole(context.Background(), testApiKey, clusterId, roleName)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterRoleApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterRole, err := client.GetClusterRole(context.Background(), testApiKey, clusterId, roleName)
		assert.NilError(t, err)
		assert.Equal(t, clusterRole.Name, roleName)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(clusterRoleApiResource)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.GetClusterRole(context.Background(), testApiKey, clusterId, roleName)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestListClusterRoles(t *testing.T) {
	clusterId := "1234"
	responsePayload := &ClusterRoleList{
		Roles: []*ClusterRoleApiResource{},
	}
	applicationClusterRoleApiResource := &ClusterRoleApiResource{}
	postgresClusterRoleApiResource := &ClusterRoleApiResource{}

	t.Run("WeSendCorrectData", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, "GET", "Expected GET method")
			assert.Equal(t, r.URL.Path, "/clusters/"+clusterId+"/roles", "Expected path to be '/clusters/%s/roles'")
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+testApiKey, "Expected Authorization header to contain api key.")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.ListClusterRoles(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
	})

	t.Run("OkResponse", func(t *testing.T) {
		responsePayload.Roles = append(responsePayload.Roles, applicationClusterRoleApiResource, postgresClusterRoleApiResource)
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		clusterRoles, err := client.ListClusterRoles(context.Background(), testApiKey, clusterId)
		assert.NilError(t, err)
		assert.Equal(t, len(clusterRoles), 2)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		responsePayloadJson, err := json.Marshal(responsePayload)
		assert.NilError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(responsePayloadJson)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "")
		assert.Equal(t, client.BaseURL.String(), server.URL)

		_, err = client.ListClusterRoles(context.Background(), testApiKey, clusterId)
		assert.Check(t, err != nil)
		assert.ErrorContains(t, err, "400 Bad Request")
	})
}
