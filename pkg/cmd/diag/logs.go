package diag

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogsOptions provides information required to update
// the current context on a user's KUBECONFIG
type LogsOptions struct {
	*DiagOptions

	podNames       []string
	labelSelectors []string
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
	cmd.PersistentFlags().StringArrayVar(&o.podNames, "pod", nil, "podname to diagnose")
	cmd.PersistentFlags().StringArrayVarP(&o.labelSelectors, "labels", "l", nil, "select a pod by label. an arbitrary pod will be selected, with preference to newer pods.")
	return cmd
}

// Complete sets all information required for updating the current context
func (o *LogsOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *LogsOptions) Validate() error {
	for _, ls := range o.labelSelectors {
		pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{LabelSelector: ls})
		if err != nil {
			return err
		}
		o.podNames = append(o.podNames, lo.Map(pl.Items, func(p corev1.Pod, _ int) string { return p.Name })...)
	}

	if len(o.podNames) == 0 {
		return fmt.Errorf("no pods found")
	}

	return nil
}

type logEntry struct {
	podName string
	err     error
	log     string
	done    bool
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (o *LogsOptions) Run() error {

	// exec!
	zero := int64(0)
	ctx := o.ctx
	logEntries := make(chan logEntry)

	var wg sync.WaitGroup
	for _, podName := range o.podNames {
		// get the logs from the pod
		currOpts := &corev1.PodLogOptions{
			Container: "",
			Follow:    true,
			TailLines: &zero,
		}
		readCloser, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).GetLogs(podName, currOpts).Stream(ctx)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()
			defer readCloser.Close()
			r := bufio.NewReader(readCloser)
			for {
				bytes, err := r.ReadBytes('\n')
				if err != nil {
					if err != io.EOF {
						err := fmt.Errorf("failed to read logs: %w", err)
						logEntries <- logEntry{podName: podName, err: err}
					}
					logEntries <- logEntry{podName: podName, done: true}
				}

				logline := strings.TrimSuffix(string(bytes), "\n")
				logEntries <- logEntry{podName: podName, log: logline}
			}
		}(podName)
	}
	go func() {
		wg.Wait()
		close(logEntries)
	}()

	go func() {
		for entry := range logEntries {
			if entry.err != nil {
				fmt.Fprintf(o.ErrOut, "error reading logs for %s: %v\n", entry.podName, entry.err)
				continue
			}
			if entry.done {
				fmt.Fprintf(o.Out, "pod %s is done\n", entry.podName)
			}
			fmt.Fprintf(o.Out, "%s: %s\n", entry.podName, entry.log)
		}
	}()

	if len(o.args) > 0 {
		cmd := exec.CommandContext(ctx, o.args[0], o.args[1:]...)
		cmd.Stderr = o.ErrOut
		cmd.Stdout = o.Out
		cmd.Stdin = o.In
		err := cmd.Start()
		if err != nil {
			return err
		}
		cmd.Wait()
	} else {
		<-ctx.Done()
	}

	// print the logs
	// run command

	return nil
}
