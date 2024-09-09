// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package upgradecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr/funcr"
	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func init() {
	client = &MockClient{Timeout: 1}
	// set backoff to two steps, 1 second apart for testing
	backoff = wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   float64(1),
		Steps:    2,
	}
}

type MockClient struct {
	Timeout time.Duration
}

var funcFoo func() (*http.Response, error)

// Do is the mock request that will return a mock success
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return funcFoo()
}

func TestCheckForUpgrades(t *testing.T) {
	fakeClient := setupFakeClientWithPGOScheme(t, false)
	ctx := logging.NewContext(context.Background(), logging.Discard())
	cfg := &rest.Config{}

	// Pass *testing.T to allows the correct messages from the assert package
	// in the event of certain failures.
	checkData := func(t *testing.T, header string) {
		data := clientUpgradeData{}
		err := json.Unmarshal([]byte(header), &data)
		assert.NilError(t, err)
		assert.Assert(t, data.DeploymentID != "")
		assert.Equal(t, data.PGOVersion, "4.7.3")
	}

	t.Run("success", func(t *testing.T) {
		// A successful call
		funcFoo = func() (*http.Response, error) {
			json := `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(json)),
				StatusCode: http.StatusOK,
			}, nil
		}

		res, header, err := checkForUpgrades(ctx, "", "4.7.3", backoff,
			fakeClient, cfg, false)
		assert.NilError(t, err)
		assert.Equal(t, res, `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`)
		checkData(t, header)
	})

	t.Run("total failure, err sending", func(t *testing.T) {
		var counter int
		// A call returning errors
		funcFoo = func() (*http.Response, error) {
			counter++
			return &http.Response{}, errors.New("whoops")
		}

		res, header, err := checkForUpgrades(ctx, "", "4.7.3", backoff,
			fakeClient, cfg, false)
		// Two failed calls because of env var
		assert.Equal(t, counter, 2)
		assert.Equal(t, res, "")
		assert.Equal(t, err.Error(), `whoops`)
		checkData(t, header)
	})

	t.Run("total failure, bad StatusCode", func(t *testing.T) {
		var counter int
		// A call returning bad StatusCode
		funcFoo = func() (*http.Response, error) {
			counter++
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader("")),
				StatusCode: http.StatusBadRequest,
			}, nil
		}

		res, header, err := checkForUpgrades(ctx, "", "4.7.3", backoff,
			fakeClient, cfg, false)
		assert.Equal(t, res, "")
		// Two failed calls because of env var
		assert.Equal(t, counter, 2)
		assert.Equal(t, err.Error(), `received StatusCode 400`)
		checkData(t, header)
	})

	t.Run("one failure, then success", func(t *testing.T) {
		var counter int
		// A call returning bad StatusCode the first time
		// and a successful response the second time
		funcFoo = func() (*http.Response, error) {
			if counter < 1 {
				counter++
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("")),
					StatusCode: http.StatusBadRequest,
				}, nil
			}
			counter++
			json := `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(json)),
				StatusCode: http.StatusOK,
			}, nil
		}

		res, header, err := checkForUpgrades(ctx, "", "4.7.3", backoff,
			fakeClient, cfg, false)
		assert.Equal(t, counter, 2)
		assert.NilError(t, err)
		assert.Equal(t, res, `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`)
		checkData(t, header)
	})
}

// TODO(benjaminjb): Replace `fake` with envtest
func TestCheckForUpgradesScheduler(t *testing.T) {
	fakeClient := setupFakeClientWithPGOScheme(t, false)
	_, server := setupVersionServer(t, true)
	defer server.Close()
	cfg := &rest.Config{Host: server.URL}

	t.Run("panic from checkForUpgrades doesn't bubble up", func(t *testing.T) {
		ctx := context.Background()

		// capture logs
		var calls []string
		ctx = logging.NewContext(ctx, funcr.NewJSON(func(object string) {
			calls = append(calls, object)
		}, funcr.Options{
			Verbosity: 1,
		}))

		// A panicking call
		funcFoo = func() (*http.Response, error) {
			panic(fmt.Errorf("oh no!"))
		}

		s := CheckForUpgradesScheduler{
			Client: fakeClient,
			Config: cfg,
		}
		s.check(ctx)

		assert.Equal(t, len(calls), 2)
		assert.Assert(t, cmp.Contains(calls[1], `encountered panic in upgrade check`))
	})

	t.Run("successful log each loop, ticker works", func(t *testing.T) {
		ctx := context.Background()

		// capture logs
		var calls []string
		ctx = logging.NewContext(ctx, funcr.NewJSON(func(object string) {
			calls = append(calls, object)
		}, funcr.Options{
			Verbosity: 1,
		}))

		// A successful call
		funcFoo = func() (*http.Response, error) {
			json := `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(json)),
				StatusCode: http.StatusOK,
			}, nil
		}

		// Set loop time to 1s and sleep for 2s before sending the done signal
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		s := CheckForUpgradesScheduler{
			Client:  fakeClient,
			Config:  cfg,
			Refresh: 1 * time.Second,
		}
		assert.ErrorIs(t, context.DeadlineExceeded, s.Start(ctx))

		// Sleeping leads to some non-deterministic results, but we expect at least 2 executions
		// plus one log for the failure to apply the configmap
		assert.Assert(t, len(calls) >= 4)

		assert.Assert(t, cmp.Contains(calls[1], `{\"pgo_versions\":[{\"tag\":\"v5.0.4\"},{\"tag\":\"v5.0.3\"},{\"tag\":\"v5.0.2\"},{\"tag\":\"v5.0.1\"},{\"tag\":\"v5.0.0\"}]}`))
		assert.Assert(t, cmp.Contains(calls[3], `{\"pgo_versions\":[{\"tag\":\"v5.0.4\"},{\"tag\":\"v5.0.3\"},{\"tag\":\"v5.0.2\"},{\"tag\":\"v5.0.1\"},{\"tag\":\"v5.0.0\"}]}`))
	})
}

func TestCheckForUpgradesSchedulerLeaderOnly(t *testing.T) {
	// CheckForUpgradesScheduler should implement this interface.
	var s manager.LeaderElectionRunnable = new(CheckForUpgradesScheduler)

	assert.Assert(t, s.NeedLeaderElection(),
		"expected to only run on the leader")
}
