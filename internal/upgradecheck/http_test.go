package upgradecheck

/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

	"github.com/wojas/genericr"
	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"github.com/crunchydata/postgres-operator/internal/logging"
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

type MockCacheClient struct {
	works bool
}

func (cc *MockCacheClient) WaitForCacheSync(ctx context.Context) bool {
	return cc.works
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

		res, header, err := checkForUpgrades(ctx, "4.7.3", backoff,
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

		res, header, err := checkForUpgrades(ctx, "4.7.3", backoff,
			fakeClient, cfg, false)
		// Two failed calls because of env var
		assert.Equal(t, counter, 2)
		assert.Equal(t, res, "")
		assert.Equal(t, err.Error(), `whoops`)
		checkData(t, header)
	})

	t.Run("recovers from panic", func(t *testing.T) {
		var counter int
		// A panicking call
		funcFoo = func() (*http.Response, error) {
			counter++
			panic(fmt.Errorf("oh no!"))
		}

		res, header, err := checkForUpgrades(ctx, "4.7.3", backoff,
			fakeClient, cfg, false)
		// One call because of panic
		assert.Equal(t, counter, 1)
		assert.Equal(t, res, "")
		assert.Equal(t, err.Error(), `oh no!`)
		// no http response returned, so don't perform full check
		assert.Assert(t, header == "")
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

		res, header, err := checkForUpgrades(ctx, "4.7.3", backoff,
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

		res, header, err := checkForUpgrades(ctx, "4.7.3", backoff,
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
	const testUpgradeCheckURL = "http://localhost:8080"

	t.Run("panic from checkForUpgrades doesn't bubble up", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// capture logs
		var calls []string
		ctx = logging.NewContext(ctx, genericr.New(func(input genericr.Entry) {
			calls = append(calls, input.Message)
		}))

		// A panicking call
		funcFoo = func() (*http.Response, error) {
			panic(fmt.Errorf("oh no!"))
		}

		go CheckForUpgradesScheduler(ctx, "4.7.3", testUpgradeCheckURL, fakeClient, cfg, false,
			&MockCacheClient{works: true})
		time.Sleep(1 * time.Second)
		cancel()

		// Sleeping leads to some non-deterministic results, but we expect at least 1 execution
		// plus one log for the failure to apply the configmap
		assert.Assert(t, len(calls) >= 2)
		assert.Equal(t, calls[1], `could not complete upgrade check`)
	})

	t.Run("cache sync fail leads to log and exit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// capture logs
		var calls []string
		ctx = logging.NewContext(ctx, genericr.New(func(input genericr.Entry) {
			calls = append(calls, input.Message)
		}))

		// Set loop time to 1s and sleep for 2s before sending the done signal -- though the cache sync
		// failure will exit the func before the sleep ends
		upgradeCheckPeriod = 1 * time.Second
		go CheckForUpgradesScheduler(ctx, "4.7.3", testUpgradeCheckURL, fakeClient, cfg, false,
			&MockCacheClient{works: false})
		time.Sleep(2 * time.Second)
		cancel()

		assert.Assert(t, len(calls) == 1)
		assert.Equal(t, calls[0], `unable to sync cache for upgrade check`)
	})

	t.Run("successful log each loop, ticker works", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// capture logs
		var calls []string
		ctx = logging.NewContext(ctx, genericr.New(func(input genericr.Entry) {
			calls = append(calls, input.Message)
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
		upgradeCheckPeriod = 1 * time.Second
		go CheckForUpgradesScheduler(ctx, "4.7.3", testUpgradeCheckURL, fakeClient, cfg, false,
			&MockCacheClient{works: true})
		time.Sleep(2 * time.Second)
		cancel()

		// Sleeping leads to some non-deterministic results, but we expect at least 2 executions
		// plus one log for the failure to apply the configmap
		assert.Assert(t, len(calls) >= 4)
		assert.Equal(t, calls[1], `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`)
		assert.Equal(t, calls[3], `{"pgo_versions":[{"tag":"v5.0.4"},{"tag":"v5.0.3"},{"tag":"v5.0.2"},{"tag":"v5.0.1"},{"tag":"v5.0.0"}]}`)
	})
}
