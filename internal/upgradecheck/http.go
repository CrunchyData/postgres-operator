// Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package upgradecheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

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
)

const (
	// upgradeCheckURL can be set using the CHECK_FOR_UPGRADES_URL env var
	upgradeCheckURL = "https://operator-maestro.crunchydata.com/pgo-versions"
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

func checkForUpgrades(ctx context.Context, url, versionString string, backoff wait.Backoff,
	crclient crclient.Client, cfg *rest.Config,
	isOpenShift bool) (message string, header string, err error) {
	var headerPayloadStruct *clientUpgradeData

	// Prep request
	req, err := http.NewRequest("GET", url, nil)
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

type CheckForUpgradesScheduler struct {
	Client crclient.Client
	Config *rest.Config

	OpenShift    bool
	Refresh      time.Duration
	URL, Version string
}

// ManagedScheduler creates a [CheckForUpgradesScheduler] and adds it to m.
func ManagedScheduler(m manager.Manager, openshift bool, url, version string) error {
	if url == "" {
		url = upgradeCheckURL
	}

	return m.Add(&CheckForUpgradesScheduler{
		Client:    m.GetClient(),
		Config:    m.GetConfig(),
		OpenShift: openshift,
		Refresh:   24 * time.Hour,
		URL:       url,
		Version:   version,
	})
}

// NeedLeaderElection returns true so that s runs only on the single
// [manager.Manager] that is elected leader in the Kubernetes cluster.
func (s *CheckForUpgradesScheduler) NeedLeaderElection() bool { return true }

// Start checks for upgrades periodically. It blocks until ctx is cancelled.
func (s *CheckForUpgradesScheduler) Start(ctx context.Context) error {
	s.check(ctx)

	ticker := time.NewTicker(s.Refresh)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.check(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *CheckForUpgradesScheduler) check(ctx context.Context) {
	log := logging.FromContext(ctx)

	defer func() {
		if v := recover(); v != nil {
			log.V(1).Info("encountered panic in upgrade check", "response", v)
		}
	}()

	info, header, err := checkForUpgrades(ctx,
		s.URL, s.Version, backoff, s.Client, s.Config, s.OpenShift)

	if err != nil {
		log.V(1).Info("could not complete upgrade check", "response", err.Error())
	} else {
		log.Info(info, clientHeader, header)
	}
}
