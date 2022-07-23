package diag

import (
	"fmt"

	"github.com/solo-io/kdiag/pkg/manager"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/util/term"
)

var (
	shellExample = `
	Start a shell to a pod. Unlike regular "kubectl exec" this command works even in distroless and
	scratch containers. It does so by using standalone busybox ash binary as the shell. This means
	that the shell is more limited, but works on any container as long as "/proc" is mounted.
	Note that this command needs linux kernel >= 5.3 to work. though this requirement may be relaxed
	in the future if needed.

	For example:

	%[1]s -l app=productpage -n bookinfo -t istio-proxy shell

	Example with arguments to ash:
	
	%[1]s -l app=productpage -n bookinfo -t istio-proxy shell -- -c "top -n 1"

	Start a shell targeting the istio-proxy container in the productpage pod.

	Note: a container is only created once, and may have been created from the previous commands. so specifying
	a different target the second time will have no effect.
`
)

// ShellOptions provides information required to update
// the current context on a user's KUBECONFIG
type ShellOptions struct {
	*DiagOptions
	debugShell bool
	args       []string
}

// NewShellOptions provides an instance of ShellOptions with default values
func NewShellOptions(diagOptions *DiagOptions) *ShellOptions {
	return &ShellOptions{
		DiagOptions: diagOptions,
	}
}

// NewCmdDiag provides a cobra command wrapping ShellOptions
func NewCmdShell(diagOptions *DiagOptions) *cobra.Command {
	o := NewShellOptions(diagOptions)

	cmd := &cobra.Command{
		Use:          "shell",
		Short:        "start a debug shell to the pod with an ephemeral container",
		Example:      fmt.Sprintf(shellExample, CommandName()),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}
	AddSinglePodFlags(cmd, o.DiagOptions)
	cmd.Flags().BoolVar(&o.debugShell, "debug-shell", false, "start a debug shell in the ephemeral container instead of the pod's container")
	// hidden as it used for dev purposes.
	cmd.Flags().MarkHidden("debug-shell")

	return cmd
}

// Complete sets all information required for updating the current context
func (o *ShellOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *ShellOptions) Validate() error {
	return ValidateSinglePodFlags(o.DiagOptions)
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (o *ShellOptions) Run() error {

	// exec!
	mgr := manager.NewEmephemeralContainerManager(o.clientset.CoreV1())

	_, err := mgr.EnsurePodManaged(o.ctx, o.resultingContext.Namespace, o.podName, o.dbgContainerImage, o.targetContainerName, o.pullPolicy)
	if err != nil {
		return fmt.Errorf("failed to ensure pod managed: %v", err)
	}

	execRequest := o.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(o.podName).
		Namespace(o.resultingContext.Namespace).
		SubResource("exec")

	tty := isTty(o.IOStreams.Out)
	var sizeQueue remotecommand.TerminalSizeQueue

	safe := func(fn term.SafeFunc) error {
		return fn()
	}

	if tty {
		t := o.SetupTTY()
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = t.MonitorSize(t.GetSize())
		safe = t.Safe
	}

	// unset p.Err if it was previously set because both stdout and stderr go over p.Out when tty is
	// true
	o.ErrOut = nil

	// run ASH in the namespaces of pid 1, as pid 1 belongs to the target container.
	cmd := []string{"/usr/local/bin/enter", "1", "/usr/local/bin/ash"}
	if o.debugShell {
		cmd = []string{"/bin/bash"}
	}
	cmd = append(cmd, o.args...)

	execRequest.VersionedParams(&corev1.PodExecOptions{
		Container: mgr.ContainerName(),
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    o.ErrOut != nil,
		TTY:       tty,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(o.restConfig, "POST", execRequest.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %v", err)
	}
	fmt.Fprintln(o.Out, "Connecting to pod...")

	fn := func() error {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:             o.In,
			Stdout:            o.Out,
			Stderr:            o.ErrOut,
			Tty:               tty,
			TerminalSizeQueue: sizeQueue,
		})

		var exitCode = 0
		if err != nil {
			if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
				exitCode = exitErr.ExitStatus()
				err = nil
			}
		}
		// TODO:
		_ = exitCode
		return err
	}

	if err := safe(fn); err != nil {
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}

	return nil
}

func (o *ShellOptions) SetupTTY() term.TTY {
	t := term.TTY{
		Parent: nil,
		Out:    o.Out,
		In:     o.In,
		Raw:    true,
	}

	t.In = o.In

	if !t.IsTerminalIn() {
		fmt.Fprintln(o.ErrOut, "Unable to use a TTY - input is not a terminal or the right kind of file")
		return t
	}

	return t
}
