package diag

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/solo-io/kdiag/pkg/manager"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	redirectExample = `
	Redirect incoming our outgoing connections of pod locally. If redirecting incoming connections,
	and no ports are specified, redirects will be set for ports in "listen" state.

	Examples:

	Redirect outgoing connections to ports 15010 15012 15014 to localhost, from a pod with the label 
	app=productpage in the namespace bookinfo.

	%[1]s redir -l app=productpage -n bookinfo --outgoing 15010 15012 15014

	Redirect all listening ports from an istiod pod to localhost:
	%[1]s redir -l app=istiod -n istio-system
`
)

type portPair struct {
	localPort  uint16
	remotePort uint16
}

// RedirOptions provides information required to update
// the current context on a user's KUBECONFIG
type RedirOptions struct {
	*DiagOptions
	args      []string
	portPairs []portPair

	outgoing bool
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
		Short:        "Redirect incoming or outgoing connections of pod locally",
		Example:      fmt.Sprintf(redirectExample, CommandName()),
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
	cmd.Flags().BoolVar(&o.outgoing, "outgoing", false, "when set, redirects outgoing connections instead of incoming ones")
	return cmd
}

// Complete sets all information required for updating the current context
func (o *RedirOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	for _, portString := range o.args {

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

		o.portPairs = append(o.portPairs, portPair{
			localPort:  uint16(localPort),
			remotePort: uint16(remotePort),
		})

	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *RedirOptions) Validate() error {
	if o.outgoing && len(o.portPairs) == 0 {
		return fmt.Errorf("must specify at least one port pair to redirect")
	}

	return ValidateSinglePodFlags(o.DiagOptions)
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (o *RedirOptions) Run() error {
	mgr := manager.NewEmephemeralContainerManager(o.clientset.CoreV1())

	_, err := mgr.EnsurePodManaged(o.ctx, o.resultingContext.Namespace, o.podName, o.dbgContainerImage, o.targetContainerName, o.pullPolicy)
	if err != nil {
		return fmt.Errorf("failed to ensure pod managed: %v", err)
	}
	ctx := o.ctx
	mgrmgr, err := manager.NewManager(ctx, o.restConfig, o.clientset, o.Out, o.ErrOut, o.podName, o.resultingContext.Namespace, mgr.ContainerName())
	if err != nil {
		return err
	}

	if len(o.portPairs) == 0 {
		ports, err := mgrmgr.GetListeneningPorts(o.ctx)
		if err != nil {
			return err
		}
		o.portPairs = lo.Map(ports, func(port uint16, _ int) portPair {
			return portPair{
				localPort:  port,
				remotePort: port,
			}
		})
	}

	if len(o.portPairs) == 0 {
		return fmt.Errorf("no ports to redirect")
	}

	errGroup, ctx := errgroup.WithContext(o.ctx)

	for _, portPair := range o.portPairs {
		portPair := portPair

		direction := "incoming"
		if o.outgoing {
			direction = "outgoing"
		}

		fmt.Fprintf(o.Out, "redirecting %s traffic from %s:%d to localhost:%d\n", direction, o.podName, portPair.remotePort, portPair.localPort)

		errGroup.Go(func() error {
			if o.outgoing {
				return mgrmgr.RedirectOutgoingTraffic(ctx, portPair.remotePort, portPair.localPort)
			} else {
				return mgrmgr.RedirectIncomingTraffic(ctx, portPair.remotePort, portPair.localPort)
			}
		})
	}
	err = errGroup.Wait()
	if err != nil {
		fmt.Fprintf(o.ErrOut, "failed to redirect traffic: %v\n", err)
		return err
	}
	return nil
}
