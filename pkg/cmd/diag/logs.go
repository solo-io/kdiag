package diag

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/solo-io/kdiag/pkg/logs"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	logExample = `
	The use case for this command is when you want to see the impact of an action over the container logs.
	As such, this command tails and follows the logs while a command is executed.
	for example, get all the logs from the "istio-proxy" container in the bookinfo namespace:
	while executing a curl command:

	%[1]s logs -n bookinfo --all -c istio-proxy -- curl http://foo.bar.com

	You can also use the following syntax to get the logs from a specific container.
	
	This examples gets the logs from the "istio-proxy" container from all the pods with the app=productpage label

	%[1]s logs -n bookinfo -l app=productpage:istio-proxy -- curl http://foo.bar.com
`
)

// LogsOptions provides information required to update
// the current context on a user's KUBECONFIG

type LogsOptions struct {
	*DiagOptions

	podNames       []string
	labelSelectors []string
	all            bool
	containerName  string
	args           []string

	podAndContainerNames []logs.PodAndContainerName
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
		Short:        "View logs from multiple containers",
		Example:      fmt.Sprintf(logExample, CommandName()),
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
	cmd.Flags().StringArrayVar(&o.podNames, "pod", nil, "podname to view logs of. you can use podname:containername to specify container name")
	cmd.Flags().StringArrayVarP(&o.labelSelectors, "labels", "l", nil, "select a pods to watch logs by label. you can use k=v:containername to specify container name")
	cmd.Flags().BoolVarP(&o.all, "all", "a", false, "select all pods in the namespace")
	cmd.Flags().StringVarP(&o.containerName, "container", "c", "", "default container name to use for logs. defaults to first container in the pod")
	return cmd
}

// Complete sets all information required for updating the current context
func (o *LogsOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	return nil
}

func (o *LogsOptions) getContainerName(ls string) (string, string) {
	if index := strings.LastIndexByte(ls, ':'); index > 0 {
		return ls[:index], ls[index+1:]
	}
	return ls, o.containerName
}

// Validate ensures that all required arguments and flag values are provided
func (o *LogsOptions) Validate() error {
	// alias here so less to type
	type podCntnrName = logs.PodAndContainerName
	if o.all {
		pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		o.podAndContainerNames = lo.Map(pl.Items, func(p corev1.Pod, _ int) podCntnrName {
			return podCntnrName{PodName: p.Name, ContainerName: o.containerName}
		})
	} else {
		for _, ls := range o.labelSelectors {
			ls, c := o.getContainerName(ls)

			pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{LabelSelector: ls})
			if err != nil {
				return err
			}
			o.podAndContainerNames = append(o.podAndContainerNames, lo.Map(pl.Items, func(p corev1.Pod, _ int) podCntnrName {
				return podCntnrName{PodName: p.Name, ContainerName: c}
			})...)
		}
		for _, podName := range o.podNames {
			n, c := o.getContainerName(podName)
			o.podAndContainerNames = append(o.podAndContainerNames, podCntnrName{PodName: n, ContainerName: c})
		}
	}

	if len(o.podAndContainerNames) == 0 {
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
	return printer.PrintLogs(o.ctx, o.clientset.CoreV1().Pods(o.resultingContext.Namespace), o.podAndContainerNames)
}
