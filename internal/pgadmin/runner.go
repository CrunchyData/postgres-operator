package pgadmin

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
	"fmt"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultPath = "/var/lib/pgadmin/pgadmin4.db"
	maxRetries  = 10
)

// queryRunner provides a helper for performing queries against the pgadmin
// sqlite database via Kubernetes Exec functionality
type queryRunner struct {
	BackoffPolicy Backoff
	Namespace     string
	Path          string
	Pod           v1.Pod

	clientset kubernetes.Interface
	apicfg    *rest.Config
	secSalt   string // Cached value of the database-specific security salt
	separator string // Field separator for multi-field queries
	ready     bool   // Flagged true once db has been set up
}

// NewQueryRunner creates a query runner instance with the configuration
// necessary to exec into the named pod in the provided namespace
func NewQueryRunner(clientset kubernetes.Interface, apic *rest.Config, pod v1.Pod) *queryRunner {
	qr := &queryRunner{
		Namespace: pod.ObjectMeta.Namespace,
		Path:      defaultPath,
		Pod:       pod,
		apicfg:    apic,
		clientset: clientset,
		separator: ",",
	}

	// Set up a default policy as an 'intelligent default', creators can
	// override, naturally - default will hit max at n == 10
	qr.BackoffPolicy = ExponentialBackoffPolicy{
		Base:       35 * time.Millisecond,
		JitterMode: JitterSmall,
		Maximum:    2 * time.Second,
		Ratio:      1.5,
	}

	return qr
}

// EnsureReady waits until the database both exists and has content -
// determined by entires in the user table or the timeout has occurred
func (qr *queryRunner) EnsureReady() error {
	// Use cached status to avoid repeated querying
	if qr.ready {
		return nil
	}

	cmd := []string{
		"sqlite3",
		qr.Path,
		"SELECT email FROM user WHERE id=1",
	}

	// short-fuse test, otherwise minimum wait is tick time
	stdout, _, err := kubeapi.ExecToPodThroughAPI(qr.apicfg, qr.clientset,
		cmd, qr.Pod.Spec.Containers[0].Name, qr.Pod.Name, qr.Namespace, nil)
	if len(strings.TrimSpace(stdout)) > 0 && err == nil {
		return nil
	}

	var output string
	var lastError error
	// Extended retries compared to "normal" queries
	for i := 0; i < maxRetries; i++ {
		// exec into the pod to run the query
		stdout, stderr, err := kubeapi.ExecToPodThroughAPI(qr.apicfg, qr.clientset,
			cmd, qr.Pod.Spec.Containers[0].Name, qr.Pod.Name, qr.Namespace, nil)

		if err != nil && !strings.Contains(stderr, "no such table") {
			lastError = fmt.Errorf("%w - %v", err, stderr)
			nextRoundIn := qr.BackoffPolicy.Duration(i)
			log.Debugf("[InitWait attempt %02d]: %v - retry in %v", i, err, nextRoundIn)
			time.Sleep(nextRoundIn)
		} else {
			// trim any space that may be there for an accurate read
			output = strings.TrimSpace(stdout)
			if output == "" || len(strings.TrimSpace(stderr)) > 0 {
				log.Debugf("InitWait stderr: %s", stderr)
				nextRoundIn := qr.BackoffPolicy.Duration(i)
				time.Sleep(nextRoundIn)
			} else {
				qr.ready = true
				lastError = nil
				break
			}
		}
	}
	if lastError != nil && output == "" {
		return fmt.Errorf("error executing query: %w", lastError)
	}

	return nil
}

// Exec performs a query on the database but expects no results
func (qr *queryRunner) Exec(query string) error {
	if err := qr.EnsureReady(); err != nil {
		return err
	}

	cmd := []string{"sqlite3", qr.Path, query}

	var lastError error
	for i := 0; i < maxRetries; i++ {
		// exec into the pod to run the query
		_, stderr, err := kubeapi.ExecToPodThroughAPI(qr.apicfg, qr.clientset,
			cmd, qr.Pod.Spec.Containers[0].Name, qr.Pod.Name, qr.Namespace, nil)
		if err != nil {
			lastError = fmt.Errorf("%w - %v", err, stderr)
			nextRoundIn := qr.BackoffPolicy.Duration(i)
			log.Debugf("[Exec attempt %02d]: %v - retry in %v", i, err, nextRoundIn)
			time.Sleep(nextRoundIn)
		} else {
			lastError = nil
			break
		}
	}
	if lastError != nil {
		return fmt.Errorf("error executing query: %w", lastError)
	}

	return nil
}

// Query performs a query on the database expecting results
func (qr *queryRunner) Query(query string) (string, error) {
	if err := qr.EnsureReady(); err != nil {
		return "", err
	}

	cmd := []string{
		"sqlite3",
		"-separator",
		qr.separator,
		qr.Path,
		query,
	}

	var output string
	var lastError error
	for i := 0; i < maxRetries; i++ {
		// exec into the pod to run the query
		stdout, stderr, err := kubeapi.ExecToPodThroughAPI(qr.apicfg, qr.clientset,
			cmd, qr.Pod.Spec.Containers[0].Name, qr.Pod.Name, qr.Namespace, nil)
		if err != nil {
			lastError = fmt.Errorf("%w - %v", err, stderr)
			nextRoundIn := qr.BackoffPolicy.Duration(i)
			log.Debugf("[Query attempt %02d]: %v - retry in %v", i, err, nextRoundIn)
			time.Sleep(nextRoundIn)
		} else {
			output = strings.TrimSpace(stdout)
			lastError = nil
			break
		}
	}
	if lastError != nil && output == "" {
		return "", fmt.Errorf("error executing query: %w", lastError)
	}

	return output, nil
}

// Separator gets the configured field separator
func (qr *queryRunner) Separator() string {
	return qr.separator
}
