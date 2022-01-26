package upgradecheck

/*
 Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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
	"io"
	"net/http"
	"time"

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

func checkForUpgrades(ctx context.Context, versionString string, backoff wait.Backoff,
	crclient crclient.Client, cfg *rest.Config,
	isOpenShift bool) (message string, header string, err error) {
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
			versionString, isOpenShift)
		req, err = addHeader(req, headerPayloadStruct)
	}

	// wait.ExponentialBackoff will retry the func according to the backoff object until
	// (a) func returns done as true or
	// (b) the backoff settings are exhausted,
	// i.e., the process hits the cap for time or the number of steps
	// The anonymous function here sets certain preexisting variables (bodyBytes, err, status)
	// which are then used by the surrounding `checkForUpgrades` function as part of the return
	var bodyBytes []byte
	var status int

	if err == nil {
		_ = wait.ExponentialBackoff(
			backoff,
			func() (done bool, backoffErr error) {
				var res *http.Response
				res, err = client.Do(req)

				if err == nil {
					defer res.Body.Close()
					status = res.StatusCode

					// This is a very basic check, ignoring nuances around
					// certain StatusCodes that should either prevent or impact retries
					if status == http.StatusOK {
						bodyBytes, err = io.ReadAll(res.Body)
						return true, nil
					}
				}

				// Return false, nil to continue checking
				return false, nil
			})
	}

	// We received responses, but none of them were 200 OK.
	if err == nil && status != http.StatusOK {
		err = fmt.Errorf("received StatusCode %d", status)
	}

	// TODO: Parse response and log info for user on potential upgrades
	return string(bodyBytes), req.Header.Get(clientHeader), err
}

// CheckForUpgradesScheduler invokes the check func when the operator starts
// and then on the given period schedule. It stops when the context is cancelled.
func CheckForUpgradesScheduler(ctx context.Context,
	versionString, url string, crclient crclient.Client,
	cfg *rest.Config, isOpenShift bool,
	cacheClient CacheWithWait,
) {
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

	info, header, err := checkForUpgrades(ctx, versionString, backoff,
		crclient, cfg, isOpenShift)
	if err != nil {
		log.V(1).Info("could not complete upgrade check",
			"response", err.Error())
	} else {
		log.Info(info, clientHeader, header)
	}

	ticker := time.NewTicker(upgradeCheckPeriod)
	for {
		select {
		case <-ticker.C:
			info, header, err = checkForUpgrades(ctx, versionString, backoff,
				crclient, cfg, isOpenShift)
			if err != nil {
				log.V(1).Info("could not complete scheduled upgrade check",
					"response", err.Error())
			} else {
				log.Info(info, clientHeader, header)
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
