// Copyright 2018 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type KubeAPI struct {
	Client *kubernetes.Clientset
	Config *rest.Config
}

const backrestCommand = "pgbackrest"

const (
	backrestBackupCommand       = "backup"
	backrestInfoCommand         = "info"
	backrestStanzaCreateCommand = "stanza-create"
)

const (
	repoTypeFlagGCS   = "--repo1-type=gcs"
	repoTypeFlagS3    = "--repo1-type=s3"
	noRepoS3VerifyTLS = "--no-repo1-s3-verify-tls"
)

const containerNameDefault = "database"

const (
	pgtaskBackrestStanzaCreate = "stanza-create"
	pgtaskBackrestInfo         = "info"
	pgtaskBackrestBackup       = "backup"
)

type config struct {
	command, commandOpts, container, namespace, podName, repoType, selector string
	compareHash, localGCSStorage, localS3Storage, s3VerifyTLS               bool
}

func main() {
	ctx := context.Background()
	log.Info("crunchy-pgbackrest starts")

	debugFlag, _ := strconv.ParseBool(os.Getenv("CRUNCHY_DEBUG"))
	if debugFlag {
		log.SetLevel(log.DebugLevel)
	}
	log.Infof("debug flag set to %t", debugFlag)

	config, err := NewConfig()
	if err != nil {
		panic(err)
	}

	k, err := NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// first load any configuration provided (e.g. via environment variables)
	cfg := loadConfiguration(ctx, k)

	// create the proper pgBackRest command
	cmd := createPGBackRestCommand(cfg)
	log.Infof("command to execute is [%s]", strings.Join(cmd, " "))

	var output, stderr string
	// now run the proper exec command depending on whether or not the config hashes should first
	// be compared prior to executing the PGBackRest command
	if !cfg.compareHash {
		output, stderr, err = runCommand(ctx, k, cfg, cmd)
	} else {
		output, stderr, err = compareHashAndRunCommand(ctx, k, cfg, cmd)
	}

	// log any output and check for errors
	log.Info("output=[" + output + "]")
	log.Info("stderr=[" + stderr + "]")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("crunchy-pgbackrest ends")
}

// Exec returns the stdout and stderr from running a command inside an existing
// container.
func (k *KubeAPI) Exec(ctx context.Context, namespace, pod, container string, stdin io.Reader, command []string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	var Scheme = runtime.NewScheme()
	if err := corev1.AddToScheme(Scheme); err != nil {
		log.Error(err)
		return "", "", err
	}
	var ParameterCodec = runtime.NewParameterCodec(Scheme)

	request := k.Client.CoreV1().RESTClient().Post().
		Resource("pods").SubResource("exec").
		Namespace(namespace).Name(pod).
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
		}, ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.Config, "POST", request.URL())

	if err == nil {
		err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: &stdout,
			Stderr: &stderr,
		})
	}

	return stdout.String(), stderr.String(), err
}

func NewConfig() (*rest.Config, error) {
	// The default loading rules try to read from the files specified in the
	// environment or from the home directory.
	loader := clientcmd.NewDefaultClientConfigLoadingRules()

	// The deferred loader tries an in-cluster config if the default loading
	// rules produce no results.
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &clientcmd.ConfigOverrides{}).ClientConfig()
}

func NewForConfig(config *rest.Config) (*KubeAPI, error) {
	var api KubeAPI
	var err error

	api.Config = config
	api.Client, err = kubernetes.NewForConfig(api.Config)

	return &api, err
}

// getEnvRequired attempts to get an environmental variable that is required
// by this program. If this cannot happen, we fatally exit
func getEnvRequired(envVar string) string {
	val := strings.TrimSpace(os.Getenv(envVar))

	if val == "" {
		log.Fatalf("required environmental variable %q not set, exiting.", envVar)
	}

	log.Debugf("%s set to: %s", envVar, val)

	return val
}

// loadConfiguration loads configuration from the environment as needed to run a pgBackRest
// command
func loadConfiguration(ctx context.Context, kubeapi *KubeAPI) config {
	cfg := config{}

	cfg.namespace = getEnvRequired("NAMESPACE")
	cfg.command = getEnvRequired("COMMAND")

	cfg.container = os.Getenv("CONTAINER")
	if cfg.container == "" {
		cfg.container = containerNameDefault
	}
	log.Debugf("CONTAINER set to: %s", cfg.container)

	cfg.commandOpts = os.Getenv("COMMAND_OPTS")
	log.Debugf("COMMAND_OPTS set to: %s", cfg.commandOpts)

	cfg.selector = os.Getenv("SELECTOR")
	log.Debugf("SELECTOR set to: %s", cfg.selector)

	compareHashEnv := os.Getenv("COMPARE_HASH")
	log.Debugf("COMPARE_HASH set to: %s", compareHashEnv)
	if compareHashEnv != "" {
		// default to false if an error parsing
		cfg.compareHash, _ = strconv.ParseBool(compareHashEnv)
	}

	if cfg.selector == "" {
		// if no selector then a Pod name must be provided
		cfg.podName = getEnvRequired("PODNAME")
	} else {
		// if a selector is provided, then lookup the Pod via the provided selector to get the
		// Pod name
		pods, err := kubeapi.Client.CoreV1().Pods(cfg.namespace).List(ctx,
			metav1.ListOptions{LabelSelector: cfg.selector})
		if err != nil {
			log.Fatal(err)
		}
		if len(pods.Items) != 1 {
			log.Fatalf("found %d Pods using selector, but only expected one", len(pods.Items))
		}
		cfg.podName = pods.Items[0].GetName()
	}
	log.Debugf("PODNAME set to: %s", cfg.podName)

	cfg.repoType = os.Getenv("PGBACKREST_REPO1_TYPE")
	log.Debugf("PGBACKREST_REPO1_TYPE set to: %s", cfg.repoType)

	// determine the setting of PGHA_PGBACKREST_LOCAL_S3_STORAGE
	// we will discard the error and treat the value as "false" if it is not
	// explicitly set
	cfg.localS3Storage, _ = strconv.ParseBool(os.Getenv("PGHA_PGBACKREST_LOCAL_S3_STORAGE"))
	log.Debugf("PGHA_PGBACKREST_LOCAL_S3_STORAGE set to: %t", cfg.localS3Storage)

	// determine the setting of PGHA_PGBACKREST_LOCAL_GCS_STORAGE
	// we will discard the error and treat the value as "false" if it is not
	// explicitly set
	cfg.localGCSStorage, _ = strconv.ParseBool(os.Getenv("PGHA_PGBACKREST_LOCAL_GCS_STORAGE"))
	log.Debugf("PGHA_PGBACKREST_LOCAL_GCS_STORAGE set to: %t", cfg.localGCSStorage)

	// parse the environment variable and store the appropriate boolean value
	// we will discard the error and treat the value as "false" if it is not
	// explicitly set
	cfg.s3VerifyTLS, _ = strconv.ParseBool(os.Getenv("PGHA_PGBACKREST_S3_VERIFY_TLS"))
	log.Debugf("PGHA_PGBACKREST_S3_VERIFY_TLS set to: %t", cfg.s3VerifyTLS)

	return cfg
}

// createPGBackRestCommand form the proper pgBackRest command based on the configuration provided
func createPGBackRestCommand(cfg config) []string {
	cmd := []string{backrestCommand}

	switch cfg.command {
	default:
		log.Fatalf("unsupported backup command specified: %s", cfg.command)
	case pgtaskBackrestStanzaCreate:
		log.Info("backrest stanza-create command requested")
		cmd = append(cmd, backrestStanzaCreateCommand, cfg.commandOpts)
	case pgtaskBackrestInfo:
		log.Info("backrest info command requested")
		cmd = append(cmd, backrestInfoCommand, cfg.commandOpts)
	case pgtaskBackrestBackup:
		log.Info("backrest backup command requested")
		cmd = append(cmd, backrestBackupCommand, cfg.commandOpts)
	}

	if cfg.localS3Storage {
		// if the first backup fails, still attempt the 2nd one
		cmd = append(cmd, ";")
		cmd = append(cmd, cmd...)
		cmd[len(cmd)-1] = repoTypeFlagS3 // a trick to overwite the second ";"
		// pass in the flag to disable TLS verification, if set
		// otherwise, maintain default behavior and verify TLS
		if !cfg.s3VerifyTLS {
			cmd = append(cmd, noRepoS3VerifyTLS)
		}
		log.Info("backrest command will be executed for both local and s3 storage")
	} else if cfg.repoType == "s3" {
		cmd = append(cmd, repoTypeFlagS3)
		// pass in the flag to disable TLS verification, if set
		// otherwise, maintain default behavior and verify TLS
		if !cfg.s3VerifyTLS {
			cmd = append(cmd, noRepoS3VerifyTLS)
		}
		log.Info("s3 flag enabled for backrest command")
	}

	if cfg.localGCSStorage {
		// if the first backup fails, still attempt the 2nd one
		cmd = append(cmd, ";")
		cmd = append(cmd, cmd...)
		cmd[len(cmd)-1] = repoTypeFlagGCS // a trick to overwite the second ";"
		log.Info("backrest command will be executed for both local and gcs storage")
	} else if cfg.repoType == "gcs" {
		cmd = append(cmd, repoTypeFlagGCS)
		log.Info("gcs flag enabled for backrest command")
	}

	return cmd
}

// compareHashAndRunCommand calculates the hash of the pgBackRest configuration locally against
// a hash of the pgBackRest configuration for the container being exec'd into to run a pgBackRest
// command.  Only if the hashes match will the pgBackRest command be run, otherwise and error will
// be written and exit code 1 will be returned.  This is done to ensure a pgBackRest command is only
// run when it can be verified that the exepected configuration is present.
func compareHashAndRunCommand(ctx context.Context, kubeapi *KubeAPI, cfg config, cmd []string) (string, string, error) {

	// the base script used in both the local and exec commands created below
	baseScript := `
export LC_ALL=C
shopt -s globstar
files=(/etc/pgbackrest/conf.d/**)
for i in "${!files[@]}"; do
	[[ -f "${files[$i]}" ]] || unset -v "files[$i]"
done`

	// the script run locally to get the local hash
	localScript := baseScript + `
sha1sum "${files[@]}" | sha1sum
`

	// the script to run remotely via exec
	execScript := baseScript + `
declare -r hash="$1"
local_hash="$(sha1sum "${files[@]}" | sha1sum)" 
if [[ "${local_hash}" != "${hash}" ]]; then
	printf >&2 "hash %s does not match local hash %s" "${hash}" "${local_hash}"; exit 1;
else
	` + strings.Join(cmd, " ") + `
fi
`

	localHashCmd := exec.CommandContext(ctx, "bash", "-ceu", "--", localScript)
	hashOutput, err := localHashCmd.Output()
	if err != nil {
		log.Fatalf("unable to calculate hash for pgBackRest config: %v", err)
	}
	configHash := strings.TrimSuffix(string(hashOutput), "\n")
	log.Debugf("calculated config hash %s", configHash)

	execCmd := []string{"bash", "-ceu", "--", execScript, "-", configHash}
	return kubeapi.Exec(ctx, cfg.namespace, cfg.podName, cfg.container, nil, execCmd)
}

// runCommand runs the provided pgBackRest command according to the configuration
// provided
func runCommand(ctx context.Context, kubeapi *KubeAPI, cfg config, cmd []string) (string, string, error) {
	bashCmd := []string{"bash"}
	reader := strings.NewReader(strings.Join(cmd, " "))
	return kubeapi.Exec(ctx, cfg.namespace, cfg.podName, cfg.container, reader, bashCmd)
}
