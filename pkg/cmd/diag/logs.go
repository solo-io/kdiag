package diag

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/solo-io/kdiag/pkg/logs"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	logExample = `
	%[1]s --namespace=default logs --pod mypod
`
)

// LogsOptions provides information required to update
// the current context on a user's KUBECONFIG
type LogsOptions struct {
	*DiagOptions

	podNames       []string
	labelSelectors []string
	all            bool
	args           []string
}

// NewLogsOptions provides an instance of LogsOptions with default values
func NewLogsOptions(diagOptions *DiagOptions) *LogsOptions {
	return &LogsOptions{
		DiagOptions: diagOptions,
	}
}

// NewCmdDiag provides a cobra command wrapping LogsOptions
func NewCmdLogs(diagOptions *DiagOptions) *cobra.Command {
	o := NewLogsOptions(diagOptions)

	cmd := &cobra.Command{
		Use:          "logs",
		Short:        "View or set the current Diag",
		Example:      fmt.Sprintf(logExample, "kubectl diag"),
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
	cmd.PersistentFlags().StringArrayVar(&o.podNames, "pod", nil, "podname to diagnose")
	cmd.PersistentFlags().StringArrayVarP(&o.labelSelectors, "labels", "l", nil, "select a pods by label.")
	cmd.PersistentFlags().BoolVarP(&o.all, "all", "a", false, "select all pods in the namespace.")
	return cmd
}

// Complete sets all information required for updating the current context
func (o *LogsOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *LogsOptions) Validate() error {
	if o.all {
		pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		o.podNames = lo.Map(pl.Items, func(p corev1.Pod, _ int) string { return p.Name })
	} else {
		for _, ls := range o.labelSelectors {
			pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{LabelSelector: ls})
			if err != nil {
				return err
			}
			o.podNames = append(o.podNames, lo.Map(pl.Items, func(p corev1.Pod, _ int) string { return p.Name })...)
		}
	}

	if len(o.podNames) == 0 {
		return fmt.Errorf("no pods found")
	}

	return nil
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (o *LogsOptions) Run() error {
	printer := logs.MultiLogPrinter{
		Out:    o.Out,
		ErrOut: o.ErrOut,
		In:     o.In,
		Args:   o.args,
	}
	return printer.PrintLogs(o.ctx, o.clientset.CoreV1().Pods(o.resultingContext.Namespace), o.podNames)
}
