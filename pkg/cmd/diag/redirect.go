package diag

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/solo-io/kdiag/pkg/manager"
	"github.com/spf13/cobra"
)

// RedirOptions provides information required to update
// the current context on a user's KUBECONFIG
type RedirOptions struct {
	*DiagOptions
	args       []string
	localPort  uint16
	remotePort uint16
}

// NewRedirOptions provides an instance of RedirOptions with default values
func NewRedirOptions(diagOptions *DiagOptions) *RedirOptions {
	return &RedirOptions{
		DiagOptions: diagOptions,
	}
}

// NewCmdDiag provides a cobra command wrapping RedirOptions
func NewCmdRedir(diagOptions *DiagOptions) *cobra.Command {
	o := NewRedirOptions(diagOptions)

	cmd := &cobra.Command{
		Use:          "redir podport:localport",
		Short:        "View or set the current Diag",
		Example:      fmt.Sprintf(diagExample, "kubectl"),
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
	return cmd
}

// Complete sets all information required for updating the current context
func (o *RedirOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	if len(o.args) != 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	portString := o.args[0]
	parts := strings.Split(portString, ":")
	var localString, remoteString string
	if len(parts) == 1 {
		localString = parts[0]
		remoteString = parts[0]
	} else if len(parts) == 2 {
		localString = parts[1]
		if localString == "" {
			// support :5000
			localString = "0"
		}
		remoteString = parts[0]
	} else {
		return fmt.Errorf("invalid port format '%s'", portString)
	}
	localPort, err := strconv.ParseUint(localString, 10, 16)
	if err != nil {
		return fmt.Errorf("error parsing local port '%s': %s", localString, err)
	}

	remotePort, err := strconv.ParseUint(remoteString, 10, 16)
	if err != nil {
		return fmt.Errorf("error parsing remote port '%s': %s", remoteString, err)
	}
	if remotePort == 0 {
		return fmt.Errorf("remote port must be > 0")
	}

	o.localPort = uint16(localPort)
	o.remotePort = uint16(remotePort)

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *RedirOptions) Validate() error {
	return ValidateSinglePodFlags(o.DiagOptions)
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (o *RedirOptions) Run() error {

	mgr := manager.NewEmephemeralContainerManager(o.clientset.CoreV1())

	_, err := mgr.EnsurePodManaged(o.ctx, o.resultingContext.Namespace, o.podName, o.dbgContainerImage, "")
	if err != nil {
		return fmt.Errorf("failed to ensure pod managed: %v", err)
	}
	ctx := o.ctx
	mgrmgr, err := manager.NewManager(ctx, o.restConfig, o.clientset, o.Out, o.ErrOut, o.podName, o.resultingContext.Namespace, mgr.ContainerName())
	if err != nil {
		return err
	}

	return mgrmgr.RedirectTraffic(ctx, o.remotePort, o.localPort)
}
