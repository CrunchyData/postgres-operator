package upgradecheck

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
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/logging"
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

	// upgradeCheckURL can be set using the CHECK_FOR_UPGRADES_URL env var
	upgradeCheckURL    = "https://operator-maestro.crunchydata.com/pgo-versions"
	upgradeCheckPeriod = 24 * time.Hour
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Creating an interface for cache with WaitForCacheSync to allow easier mocking
type CacheWithWait interface {
	WaitForCacheSync(ctx context.Context) bool
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

func checkForUpgrades(log logr.Logger, versionString string, backoff wait.Backoff,
	crclient crclient.Client, ctx context.Context, cfg *rest.Config,
	isOpenShift bool) (message string, err error) {
	var res *http.Response
	var bodyBytes []byte
	var headerPayloadStruct *clientUpgradeData

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
		// generateHeader always returns some sort of struct, using defaults/nil values
		// in case some of the checks return errors
		headerPayloadStruct = generateHeader(ctx, cfg, crclient,
			log, versionString, isOpenShift)
		req, err = addHeader(req, headerPayloadStruct)
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
func CheckForUpgradesScheduler(channel chan bool,
	versionString, url string, crclient crclient.Client,
	cfg *rest.Config, isOpenShift bool,
	cacheClient CacheWithWait,
) {
	ctx := context.Background()
	log := logging.FromContext(ctx)
	defer func() {
		if err := recover(); err != nil {
			log.V(1).Info("encountered panic in upgrade check",
				"response", err,
			)
		}
	}()

	// set the URL for the check for upgrades endpoint if provided
	if url != "" {
		upgradeCheckURL = url
	}

	// Since we pass the client to this function before we start the manager
	// in cmd/postgres-operator/main.go, we want to make sure cache is synced
	// before using the client.
	// If the cache fails to sync, that probably indicates a more serious problem
	// with the manager starting, so we don't have to worry about restarting or retrying
	// this process -- simply log and return
	if synced := cacheClient.WaitForCacheSync(ctx); !synced {
		log.V(1).Info("unable to sync cache for upgrade check")
		return
	}

	info, err := checkForUpgrades(log, versionString, backoff,
		crclient, ctx, cfg, isOpenShift)
	if err != nil {
		log.V(1).Info("could not complete upgrade check",
			"response", err.Error())
	} else {
		log.V(1).Info(info)
	}

	ticker := time.NewTicker(upgradeCheckPeriod)
	for {
		select {
		case <-ticker.C:
			info, err = checkForUpgrades(log, versionString, backoff,
				crclient, ctx, cfg, isOpenShift)
			if err != nil {
				log.V(1).Info("could not complete scheduled upgrade check",
					"response", err.Error())
			} else {
				log.V(1).Info(info)
			}
		case <-channel:
			ticker.Stop()
			return
		}
	}
}
