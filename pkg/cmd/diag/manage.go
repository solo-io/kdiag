package diag

import (
	"fmt"

	"github.com/solo-io/kdiag/pkg/manager"
	"github.com/spf13/cobra"
)

var (
	manageExample = `
	Start the ephemeral container to manage a pod.
	commands that need it do it automatically. this is mostly useful for testing / development cycle.
`
)

// ManageOptions provides information required to update
// the current context on a user's KUBECONFIG
type ManageOptions struct {
	*DiagOptions
}

// NewManageOptions provides an instance of ManageOptions with default values
func NewManageOptions(diagOptions *DiagOptions) *ManageOptions {
	return &ManageOptions{
		DiagOptions: diagOptions,
	}
}

// NewCmdDiag provides a cobra command wrapping ManageOptions
func NewCmdManage(diagOptions *DiagOptions) *cobra.Command {
	o := NewManageOptions(diagOptions)

	cmd := &cobra.Command{
		Use: "manage",
		// hide as this command is pretty useless except for debugging
		Hidden:       true,
		Short:        "Start the ephemeral container to manage a pod",
		Example:      fmt.Sprintf(manageExample, "kubectl"),
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
func (o *ManageOptions) Complete(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("no arguments are allowed")
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *ManageOptions) Validate() error {
	return ValidateSinglePodFlags(o.DiagOptions)
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (o *ManageOptions) Run() error {

	// exec!
	mgr := manager.NewEmephemeralContainerManager(o.clientset.CoreV1())

	_, err := mgr.EnsurePodManaged(o.ctx, o.resultingContext.Namespace, o.podName, o.dbgContainerImage, o.targetContainerName, o.pullPolicy)
	if err != nil {
		return fmt.Errorf("failed to ensure pod managed: %v", err)
	}

	fmt.Fprintf(o.Out, "%s container deployed to manage pod %s\n", mgr.ContainerName(), o.podName)
	return nil
}
