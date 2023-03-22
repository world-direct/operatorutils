package podexec

import (
	"io"

	"github.com/gprossliner/xhdl"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type podExecutor struct {
	restConfig *rest.Config
	pod        *corev1.Pod
	container  string
	command    []string
}

type PodExecutor interface {
	Execute(ctx xhdl.Context, stdout, stderr io.Writer)
}

func New(restConfig *rest.Config, pod *corev1.Pod, container string, command []string) PodExecutor {
	// https://discuss.kubernetes.io/t/go-client-exec-ing-a-shel-command-in-pod/5354/4
	return &podExecutor{restConfig, pod, container, command}
}

func (e *podExecutor) Execute(ctx xhdl.Context, stdout, stderr io.Writer) {

	cs, err := kubernetes.NewForConfig(e.restConfig)
	ctx.Throw(err)

	request := cs.CoreV1().RESTClient().
		Post().
		Namespace(e.pod.Namespace).
		Resource("pods").
		Name(e.pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   e.command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
			Container: e.container,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(e.restConfig, "POST", request.URL())
	ctx.Throw(err)

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
	})

	ctx.Throw(err)
}
