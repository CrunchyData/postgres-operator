package pgo_cli_test

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crunchydata/postgres-operator/testing/kubeapi"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/storage/names"
)

type Pool struct {
	*kubeapi.Proxy
	*pgxpool.Pool
}

func (p *Pool) Close() error { p.Pool.Close(); return p.Proxy.Close() }

// clusterConnection opens a PostgreSQL connection to a database pod. Any error
// will cause t to FailNow.
func clusterConnection(t testing.TB, namespace, cluster, dsn string) *Pool {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(namespace, map[string]string{
		"pg-cluster":      cluster,
		"pgo-pg-database": "true",
	})
	require.NoError(t, err)
	require.NotEmpty(t, pods)

	proxy, err := TestContext.Kubernetes.PodPortForward(pods[0].Namespace, pods[0].Name, "5432")
	require.NoError(t, err)

	host, port, err := net.SplitHostPort(proxy.LocalAddr())
	if err != nil {
		proxy.Close()
		require.NoError(t, err)
	}

	pool, err := pgxpool.Connect(context.Background(), dsn+" host="+host+" port="+port)
	if err != nil {
		proxy.Close()
		require.NoError(t, err)
	}

	return &Pool{proxy, pool}
}

// clusterPSQL executes psql commands and/or SQL on a database pod. Any error
// will cause t to FailNow.
func clusterPSQL(t testing.TB, namespace, cluster, psql string) (string, string) {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(namespace, map[string]string{
		"pg-cluster":      cluster,
		"pgo-pg-database": "true",
	})
	require.NoError(t, err)
	require.NotEmpty(t, pods)

	stdout, stderr, err := TestContext.Kubernetes.PodExec(
		pods[0].Namespace, pods[0].Name,
		strings.NewReader(psql), "psql", "-U", "postgres", "-f-")
	require.NoError(t, err)

	return stdout, stderr
}

// clusterPVCs returns a list of persistent volume claims for the cluster. Any
// error will cause t to FailNow.
func clusterPVCs(t testing.TB, namespace, cluster string) []core_v1.PersistentVolumeClaim {
	t.Helper()

	pvcs, err := TestContext.Kubernetes.ListPVCs(namespace, map[string]string{
		"pg-cluster": cluster,
	})
	require.NoError(t, err)

	return pvcs
}

// primaryPods returns a list of PostgreSQL primary pods for the cluster. Any
// error will cause t to FailNow.
func primaryPods(t testing.TB, namespace, cluster string) []core_v1.Pod {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(namespace, map[string]string{
		"pg-cluster": cluster,
		"role":       "master",
	})
	require.NoError(t, err)

	return pods
}

// replicaPods returns a list of PostgreSQL replica pods for the cluster. Any
// error will cause t to FailNow.
func replicaPods(t testing.TB, namespace, cluster string) []core_v1.Pod {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(namespace, map[string]string{
		"pg-cluster": cluster,
		"role":       "replica",
	})
	require.NoError(t, err)

	return pods
}

// requireClusterReady waits until all deployments of cluster are ready. If
// timeout elapses or any error occurs, t will FailNow.
func requireClusterReady(t testing.TB, namespace, cluster string, timeout time.Duration) {
	t.Helper()

	// Give up now if some part of setting up the cluster failed.
	if t.Failed() || cluster == "" {
		t.FailNow()
	}

	ready := func() bool {
		deployments, err := TestContext.Kubernetes.ListDeployments(namespace, map[string]string{
			"pg-cluster": cluster,
		})
		require.NoError(t, err)

		if len(deployments) == 0 {
			return false
		}

		var database bool
		for _, deployment := range deployments {
			if deployment.Status.Replicas != deployment.Status.ReadyReplicas {
				return false
			}
			if deployment.Labels["pgo-pg-database"] == "true" {
				database = true
			}
		}
		return database
	}

	if !ready() {
		requireWaitFor(t, ready, timeout, time.Second,
			"timeout waiting for %q in %q", cluster, namespace)
	}
}

// requirePgBouncerReady waits until all PgBouncer deployments for cluster are
// ready. If timeout elapses or any error occurs, t will FailNow.
func requirePgBouncerReady(t testing.TB, namespace, cluster string, timeout time.Duration) {
	t.Helper()

	ready := func() bool {
		deployments, err := TestContext.Kubernetes.ListDeployments(namespace, map[string]string{
			"pg-cluster":        cluster,
			"crunchy-pgbouncer": "true",
		})
		require.NoError(t, err)

		if len(deployments) == 0 {
			return false
		}
		for _, deployment := range deployments {
			if deployment.Status.Replicas != deployment.Status.ReadyReplicas {
				return false
			}
		}
		return true
	}

	if !ready() {
		requireWaitFor(t, ready, timeout, time.Second,
			"timeout waiting for PgBouncer of %q in %q", cluster, namespace)
	}
}

// requireReplicasReady waits until all replicas of cluster are ready. If
// timeout elapses or any error occurs, t will FailNow.
func requireReplicasReady(t testing.TB, namespace, cluster string, timeout time.Duration) {
	t.Helper()

	ready := func() bool {
		pods := replicaPods(t, namespace, cluster)

		if len(pods) == 0 {
			return false
		}
		for _, pod := range pods {
			if !kubeapi.IsPodReady(pod) {
				return false
			}
		}
		return true
	}

	if !ready() {
		requireWaitFor(t, ready, timeout, time.Second,
			"timeout waiting for replicas of %q in %q", cluster, namespace)
	}
}

// requireWaitFor calls condition every tick until it returns true. If timeout
// elapses, t will Logf message and args then FailNow. Condition runs in the
// current goroutine so that it may also call t.FailNow.
func requireWaitFor(t testing.TB,
	condition func() bool, timeout, tick time.Duration,
	message string, args ...interface{},
) {
	t.Helper()

	if !waitFor(t, condition, timeout, tick) {
		t.Fatalf(message, args...)
	}
}

// teardownCluster deletes a cluster. It waits sufficiently long after created
// for the delete to go well.
func teardownCluster(t testing.TB, namespace, cluster string, created time.Time) {
	minimum := TestContext.Scale(10 * time.Second)

	if elapsed := time.Since(created); elapsed < minimum {
		time.Sleep(minimum - elapsed)
	}

	_, err := pgo("delete", "cluster", cluster, "-n", namespace, "--no-prompt").Exec(t)
	assert.NoError(t, err, "unable to tear down cluster %q in %q", cluster, namespace)
}

// waitFor calls condition once every tick until it returns true. If timeout
// elapses or t Failed, waitFor returns false. Condition runs in the current
// goroutine so that it may also call t.FailNow.
func waitFor(t testing.TB, condition func() bool, timeout, tick time.Duration) bool {
	t.Helper()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	timer := time.NewTimer(TestContext.Scale(timeout))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
			if t.Failed() {
				return false
			}
		}
	}
}

// withCluster calls during with a function that returns the name of an existing
// cluster. The cluster may not exist until that function is called. When during
// returns, the cluster might be destroyed.
func withCluster(t testing.TB, namespace func() string, during func(func() string)) {
	t.Helper()

	var created time.Time
	var name string
	var once sync.Once

	defer func() {
		if name != "" {
			teardownCluster(t, namespace(), name, created)
		}
	}()

	during(func() string {
		once.Do(func() {
			generated := names.SimpleNameGenerator.GenerateName("pgo-test-")
			_, err := pgo("create", "cluster", generated, "-n", namespace()).Exec(t)

			if assert.NoError(t, err) {
				t.Logf("created cluster %q in %q", generated, namespace())
				created = time.Now()
				name = generated
			}
		})
		return name
	})
}

// withNamespace calls during with a function that returns the name of an
// existing namespace. The namespace may not exist until that function is
// called. When during returns, the namespace and all its contents are destroyed.
func withNamespace(t testing.TB, during func(func() string)) {
	t.Helper()

	// Use the namespace specified in the environment.
	if name := os.Getenv("PGO_NAMESPACE"); name != "" {
		during(func() string { return name })
		return
	}

	// Prepare to cleanup a namespace that might be created.
	var namespace *core_v1.Namespace
	var once sync.Once

	defer func() {
		if namespace != nil {
			err := TestContext.Kubernetes.DeleteNamespace(namespace.Name)
			assert.NoErrorf(t, err, "unable to tear down namespace %q", namespace.Name)
		}
	}()

	during(func() string {
		once.Do(func() {
			ns, err := TestContext.Kubernetes.GenerateNamespace(
				"pgo-test-", map[string]string{"pgo-test": kubeapi.SanitizeLabelValue(t.Name())})

			if assert.NoError(t, err) {
				namespace = ns
				_, err = pgo("update", "namespace", namespace.Name).Exec(t)
				assert.NoErrorf(t, err, "unable to take ownership of namespace %q", namespace.Name)
			}
		})

		return namespace.Name
	})
}
