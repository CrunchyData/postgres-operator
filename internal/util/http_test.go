package util

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

func TestAddHeader(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		req := &http.Request{
			Header: http.Header{},
		}
		versionString := "1.2.3"

		result, _ := addHeader(req, versionString)
		assert.DeepEqual(t,
			result.Header[clientHeader],
			[]string{`{"version":"1.2.3"}`},
		)
	})
}

type MockClient struct {
	Timeout time.Duration
}

var funcFoo func() (*http.Response, error)

// Do is the mock request that will return a mock success
// TODO(benjaminjb): return to this when Operator Version API is complete
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return funcFoo()
}

func TestCheckForUpgrades(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// A successful call
		funcFoo = func() (*http.Response, error) {
			json := `{"display_name":"PGO 5.0.3","major_version":5}`
			r := io.NopCloser(strings.NewReader(json))
			return &http.Response{
				Body:       r,
				StatusCode: http.StatusOK,
			}, nil
		}

		res, err := checkForUpgrades("4.7.3", backoff)
		assert.NilError(t, err)
		assert.Equal(t, res, `{"display_name":"PGO 5.0.3","major_version":5}`)
	})

	t.Run("total failure, err sending", func(t *testing.T) {
		var counter int
		// A call returning errors
		funcFoo = func() (*http.Response, error) {
			counter++
			return &http.Response{}, errors.New("whoops")
		}

		res, err := checkForUpgrades("4.7.3", backoff)
		// Two failed calls because of env var
		assert.Equal(t, counter, 2)
		assert.Equal(t, res, "")
		assert.Equal(t, err.Error(), `whoops`)
	})

	t.Run("recovers from panic", func(t *testing.T) {
		var counter int
		// A panicking call
		funcFoo = func() (*http.Response, error) {
			counter++
			panic(fmt.Errorf("oh no!"))
		}

		res, err := checkForUpgrades("4.7.3", backoff)
		// One call because of panic
		assert.Equal(t, counter, 1)
		assert.Equal(t, res, "")
		assert.Equal(t, err.Error(), `oh no!`)
	})

	t.Run("total failure, bad StatusCode", func(t *testing.T) {
		var counter int
		// A call returning bad StatusCode
		funcFoo = func() (*http.Response, error) {
			counter++
			return &http.Response{
				StatusCode: http.StatusBadRequest,
			}, nil
		}

		res, err := checkForUpgrades("4.7.3", backoff)
		assert.Equal(t, res, "")
		// Two failed calls because of env var
		assert.Equal(t, counter, 2)
		assert.Equal(t, err.Error(), `received StatusCode 400`)
	})

	t.Run("one failure, then success", func(t *testing.T) {
		var counter int
		// A call returning bad StatusCode the first time
		// and a successful response the second time
		funcFoo = func() (*http.Response, error) {
			if counter < 1 {
				counter++
				return &http.Response{
					StatusCode: http.StatusBadRequest,
				}, nil
			}
			counter++
			json := `{"display_name":"PGO 5.0.3","major_version":5}`
			r := io.NopCloser(strings.NewReader(json))
			return &http.Response{
				Body:       r,
				StatusCode: http.StatusOK,
			}, nil
		}

		res, err := checkForUpgrades("4.7.3", backoff)
		assert.Equal(t, counter, 2)
		assert.NilError(t, err)
		assert.Equal(t, res, `{"display_name":"PGO 5.0.3","major_version":5}`)
	})
}

func TestCheckForUpgradesScheduler(t *testing.T) {
	t.Run("panic from checkForUpgrades doesn't bubble up", func(t *testing.T) {
		done := make(chan bool, 1)
		// capture logs
		var calls []string
		logging.SetLogFunc(1, func(input genericr.Entry) {
			calls = append(calls, input.Message)
		})

		// A panicking call
		funcFoo = func() (*http.Response, error) {
			panic(fmt.Errorf("oh no!"))
		}

		go CheckForUpgradesScheduler("4.7.3", done)
		time.Sleep(1 * time.Second)
		done <- true

		// Sleeping leads to some non-deterministic results, but we expect at least 1 execution
		assert.Assert(t, len(calls) >= 1)
		assert.Equal(t, calls[0], `oh no!`)
	})

	t.Run("successful log each loop, ticker works", func(t *testing.T) {
		done := make(chan bool, 1)
		// capture logs
		var calls []string
		logging.SetLogFunc(1, func(input genericr.Entry) {
			calls = append(calls, input.Message)
		})

		// A successful call
		funcFoo = func() (*http.Response, error) {
			json := `{"display_name":"PGO 5.0.3","major_version":5}`
			r := io.NopCloser(strings.NewReader(json))
			return &http.Response{
				Body:       r,
				StatusCode: http.StatusOK,
			}, nil
		}

		// Set loop time to 1s and sleep for 2s before sending the done signal
		upgradeCheckPeriod = 1 * time.Second
		go CheckForUpgradesScheduler("4.7.3", done)
		time.Sleep(2 * time.Second)
		done <- true

		// Sleeping leads to some non-deterministic results, but we expect at least 2 executions
		assert.Assert(t, len(calls) >= 2)
		assert.Equal(t, calls[0], `{"display_name":"PGO 5.0.3","major_version":5}`)
		assert.Equal(t, calls[1], `{"display_name":"PGO 5.0.3","major_version":5}`)
	})
}
