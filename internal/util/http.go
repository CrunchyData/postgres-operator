package util

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

const (
	clientHeader = "X-Client-Upgrade-State"
)

var (
	client HTTPClient

	// With these Backoff settings, wait.ExponentialBackoff will
	// * use one second as the base time;
	// * increase delays between calls by a power of 2 (1, 2, 4, etc.);
	// * and retry four times.
	// Note that there is no indeterminacy here since there is no Jitter set).
	// With these parameters, the calls will occur at 0, 1, 3, and 7 seconds
	// (i.e., at 1, 2, and 4 second delays for the retries).
	backoff = wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   float64(2),
		Steps:    4,
	}

	// TODO(benjaminjb): Get real URL
	upgradeCheckURL    = "http://localhost:8080"
	upgradeCheckPeriod = 24 * time.Hour
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	// Since we create this client once during startup,
	// we want each connection to be fresh, hence the non-default transport
	// with DisableKeepAlives set to true
	// See https://github.com/golang/go/issues/43905 and https://github.com/golang/go/issues/23427
	// for discussion of problems with long-lived connections
	client = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
}

// Extensible struct for client upgrade data
type clientUpgradeData struct {
	Version string `json:"version"`
}

func addHeader(req *http.Request, versionString string) (*http.Request, error) {
	upgradeInfo := &clientUpgradeData{
		Version: versionString,
	}
	marshaled, err := json.Marshal(upgradeInfo)
	if err == nil {
		upgradeInfoString := string(marshaled)
		req.Header.Add(clientHeader, upgradeInfoString)
	}

	return req, err
}

func checkForUpgrades(versionString string, backoff wait.Backoff) (message string, err error) {
	var res *http.Response
	var bodyBytes []byte

	// Guard against panics within the checkForUpgrades function to allow the
	// checkForUpgradesScheduler to reschedule a check
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = fmt.Errorf("%s", panicErr)
		}
	}()

	// Prep request
	req, err := http.NewRequest("GET",
		upgradeCheckURL,
		nil)
	if err == nil {
		req, err = addHeader(req, versionString)
	}

	// wait.ExponentialBackoff will retry the func according to the backoff object until
	// (a) func returns done as true or
	// (b) the backoff settings are exhausted,
	// i.e., the process hits the cap for time or the number of steps
	// The anonymous function here sets certain preexisting variables (res, err)
	// which are then used by the surrounding `checkForUpgrades` function as part of the return
	if err == nil {
		_ = wait.ExponentialBackoff(
			backoff,
			func() (done bool, backoffErr error) {
				// We can't close the body of this response in this block since we use it outside
				// so ignore this linting error
				res, err = client.Do(req) //nolint:bodyclose
				// This is a very basic check, ignoring nuances around
				// certain StatusCodes that should either prevent or impact retries
				if err == nil && res.StatusCode == http.StatusOK {
					return true, nil
				}
				if err == nil {
					err = fmt.Errorf("received StatusCode %d", res.StatusCode)
				}
				// Return false, nil to continue checking
				return false, nil
			})
	}
	// If the final value of err is nil and the final res.StatusCode is OK,
	// we can go on with reading the body of the response
	if err == nil && res.StatusCode == http.StatusOK {
		defer res.Body.Close()
		bodyBytes, err = ioutil.ReadAll(res.Body)
	}

	// TODO: Parse response and log info for user on potential upgrades
	if err == nil {
		return string(bodyBytes), nil
	}

	return "", err
}

// CheckForUpgradesScheduler invokes the check func when the operator starts
// and then on the given period schedule
func CheckForUpgradesScheduler(versionString string, channel chan bool) {
	ctx := context.Background()
	log := logging.FromContext(ctx)
	defer func() {
		if err := recover(); err != nil {
			log.Error(
				fmt.Errorf("%s", err),
				"Error in scheduling upgrade checks",
			)
		}
	}()

	info, err := checkForUpgrades(versionString, backoff)
	if err != nil {
		log.Error(err, err.Error())
	} else {
		log.V(1).Info(info)
	}

	ticker := time.NewTicker(upgradeCheckPeriod)
	for {
		select {
		case <-ticker.C:
			info, err = checkForUpgrades(versionString, backoff)
			if err != nil {
				log.Error(err, err.Error())
			} else {
				log.V(1).Info(info)
			}
		case <-channel:
			ticker.Stop()
			return
		}
	}
}
