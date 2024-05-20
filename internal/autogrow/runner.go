// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package autogrow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// There are 6 df output columns for df --human-readable: "Filesystem Size Used Avail Use% Mounted on".
// 4 of those columns are relevant.
const (
	dfUsePercentIdx int = 4
	dfSizeIdx       int = 1
)

type exec func(
	namespace, pod, container string,
	stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

type Runner struct {
	client       client.Client
	clientConfig *rest.Config
	clusters     map[string][]string
	log          logr.Logger
	podExec      func(
		namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error
	refresh time.Duration
	stale   []string
}

// Runner implements [Autogrow] and [manager.Runnable].
var (
	_ Autogrow         = (*Runner)(nil)
	_ manager.Runnable = (*Runner)(nil)
)

func NewRunner(clientConfig *rest.Config, log logr.Logger) (*Runner, error) {
	runner := &Runner{
		refresh:      5 * time.Second,
		clientConfig: clientConfig,
		log:          log,
	}

	return runner, nil
}

func (r *Runner) Start(ctx context.Context) error {
	if r.podExec == nil {
		var err error
		r.podExec, err = runtime.NewPodExecutor(r.clientConfig)
		if err != nil {
			return err
		}
	}
	var ticks <-chan time.Time

	ticker := time.NewTicker(r.refresh)
	defer ticker.Stop()
	ticks = ticker.C

	for {
		select {
		case <-ticks:
			r.checkVolumes()
		}
	}
}

func (r *Runner) WatchCluster(clusterNamespace, clusterName string, client client.Client) {
	if r.clusters == nil {
		r.clusters = map[string][]string{}
	}
	if r.client == nil {
		r.client = client
	}
	key := clusterNamespace + "-" + clusterName
	if _, ok := r.clusters[key]; !ok {
		r.clusters[key] = []string{clusterNamespace, clusterName}
	}
}

func (r *Runner) checkVolumes() {
	// If the Runner isn't configured, do nothing.
	if len(r.clusters) == 0 || r.client == nil {
		return
	}

	// Remove stale clusters from queue.
	for _, cluster := range r.stale {
		delete(r.clusters, cluster)
	}
	r.stale = []string{}

	clusters := r.clusters
	var keys []string
	for k := range clusters {
		keys = append(keys, k)
	}
	var wg sync.WaitGroup
	sliceLength := len(keys)
	wg.Add(sliceLength)
	for i := 0; i < sliceLength; i++ {
		go func(i int) {
			defer wg.Done()
			primary, usage, size, err := r.checkVolume(keys, i)
			if err != nil {
				r.log.Error(err, "Unable to check primary pod pgdata volume")
				return
			}
			key := keys[i]
			clusterData := r.clusters[key]
			if len(clusterData) != 2 {
				r.log.Error(errors.New("Cluster not found"), "%s is not watched by the autogrow runner", key)
				return
			}
			clusterNamespace := clusterData[0]
			clusterName := clusterData[1]

			if r.exceedsUsageLimit(usage) {
				r.annotatePGPrimaryPod(primary, clusterNamespace, clusterName, r.client, size)
			}
			return
		}(i)
	}
}

func (r *Runner) getPGPrimaryPod(podList corev1.PodList) corev1.Pod {
	var primary v1.Pod
	for _, pod := range podList.Items {
		if pod.Labels[naming.LabelRole] == naming.RolePatroniLeader {
			primary = pod
		}
	}
	return primary
}

// checkVolume will exec into the primary and run df, returning Use% and Size
func (r *Runner) checkVolume(keys []string, i int) (corev1.Pod, string, string, error) {
	clusters := r.clusters
	k := keys[i]
	cluster := clusters[k]
	clusterNamespace := cluster[0]
	clusterName := cluster[1]
	pods := &corev1.PodList{}
	selector, _ := naming.AsSelector(naming.ClusterInstances(clusterName))
	ctx := context.Background()
	err := errors.WithStack(
		r.client.List(ctx, pods,
			client.InNamespace(clusterNamespace),
			client.MatchingLabelsSelector{Selector: selector},
		))
	var primary v1.Pod
	if len(pods.Items) == 0 {
		// If no pods return, it may indicate that the cluster has been deleted.
		// Queue the key for removal up the stack.
		r.stale = append(r.stale, k)
		return primary, "", "", nil
	}
	// TODO: Refactor to check for pod readiness before trying to exec.
	primary = r.getPGPrimaryPod(*pods)
	podName := fmt.Sprintf(primary.ObjectMeta.Name)
	var stdin, stdout, stderr bytes.Buffer

	dfString := []string{"df", "--human-readable", "/pgdata"}
	r.podExec("postgres-operator", podName, "database", &stdout, &stdin, &stderr, dfString...)

	if stdin.String() != "" && strings.Contains(stdin.String(), "%") {
		dfValues := strings.Split(stdin.String(), "\n")[1]
		percentUse := strings.Fields(dfValues)[dfUsePercentIdx]
		size := strings.Fields(dfValues)[dfSizeIdx]
		return primary, percentUse, size, nil
	}
	return primary, "", "", err
}

func (r *Runner) annotatePGPrimaryPod(primary corev1.Pod, clusterNamespace, clusterName string, cli client.Client, size string) error {
	reg, err := regexp.Compile(`[aA-zZ]*$`)
	unitStr := reg.FindString(size)
	num := strings.Split(size, unitStr)[0]
	sizeFloat, err := strconv.ParseFloat(num, 64)

	newSizeFloat := sizeFloat * 1.5
	newSizeInt := int(newSizeFloat)
	newSizeStr := fmt.Sprintf("%d%s", newSizeInt, unitStr)
	anno := []byte(`{"metadata":{"annotations":{"disk-starvation": "%s"}}}`)
	sizeAnno := fmt.Sprintf(string(anno), newSizeStr)
	err = r.client.Patch(context.TODO(), &primary, client.RawPatch(
		client.Merge.Type(), []byte(sizeAnno)))
	return err
}

// NeedLeaderElection returns true so that r runs only on the single
// [manager.Manager] that is elected leader in the Kubernetes namespace.
func (r *Runner) NeedLeaderElection() bool { return true }

func Enabled(cluster v1beta1.PostgresCluster) bool {
	// TODO: Make this real.
	return true
}

func (r *Runner) exceedsUsageLimit(diskUse string) bool {
	percentString := strings.Split(diskUse, "%")[0]
	percentInt, err := strconv.Atoi(percentString)
	if err != nil {
		r.log.Error(errors.New("df Parse Error"), "Failed to parse df Use%")
		return false
	}
	return percentInt > 75
}
