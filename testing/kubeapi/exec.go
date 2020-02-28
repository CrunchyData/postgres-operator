package kubeapi

import (
	"bytes"
	"io"

	core_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// Exec returns the stdout and stderr from running a command inside an existing
// container.
func (k *KubeAPI) Exec(namespace, pod, container string, stdin io.Reader, command ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	request := k.Client.CoreV1().RESTClient().Post().
		Resource("pods").SubResource("exec").
		Namespace(namespace).Name(pod).
		VersionedParams(&core_v1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.Config, "POST", request.URL())

	if err == nil {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: &stdout,
			Stderr: &stderr,
		})
	}

	return stdout.String(), stderr.String(), err
}

// PodExec returns the stdout and stderr from running a command inside the first
// container of an existing pod.
func (k *KubeAPI) PodExec(namespace, name string, stdin io.Reader, command ...string) (string, string, error) {
	pod, err := k.GetPod(namespace, name)

	if err != nil {
		return "", "", err
	}

	return k.Exec(pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, stdin, command...)
}
