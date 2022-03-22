package diag

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yuval-k/kdiag/pkg/version"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	diagExample = `
	%[1]s diag --namespace=default diag port-redirect mypod 8080:80
`

	errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")
)

// DiagOptions provides information required to update
// the current context on a user's KUBECONFIG
type DiagOptions struct {
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context

	userSpecifiedCluster   string
	userSpecifiedContext   string
	userSpecifiedAuthInfo  string
	userSpecifiedNamespace string

	rawConfig api.Config

	restConfig       *rest.Config
	clientset        *kubernetes.Clientset
	resultingContext *api.Context

	dbgContainerImage string
	podName           string
	labelSelector     string
	genericclioptions.IOStreams
}

// NewDiagOptions provides an instance of DiagOptions with default values
func NewDiagOptions(streams genericclioptions.IOStreams) *DiagOptions {
	return &DiagOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdDiag provides a cobra command wrapping DiagOptions
func NewCmdDiag(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDiagOptions(streams)

	cmd := &cobra.Command{
		Use:          "diag [new-Diag] [flags]",
		Short:        "View or set the current Diag",
		Example:      fmt.Sprintf(diagExample, "kubectl"),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			o.ctx = cmd.Context()
			if err := o.Complete(cmd); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&o.dbgContainerImage, "dbg-image", "r.h.yuval.dev/utils:"+version.Version, "default dbg container image")
	cmd.PersistentFlags().StringVar(&o.podName, "pod", "", "podname to diagnose")
	AddLabelSelectorFlagVar(cmd, &o.labelSelector)
	cmd.PersistentFlags().StringVarP(&o.labelSelector, "labels", "l", "", "select a pod by label. an arbitrary pod will be selected, with preference to ready pods / newer pods.")
	o.configFlags.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(
		NewCmdRedir(o),
		NewCmdShell(o),
	)

	return cmd
}

// Complete sets all information required for updating the current context
func (o *DiagOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	o.userSpecifiedNamespace, err = cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	o.userSpecifiedContext, err = cmd.Flags().GetString("context")
	if err != nil {
		return err
	}

	o.userSpecifiedCluster, err = cmd.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	o.userSpecifiedAuthInfo, err = cmd.Flags().GetString("user")
	if err != nil {
		return err
	}

	var currentContext *api.Context
	var exists bool

	if o.userSpecifiedContext != "" {
		currentContext, exists = o.rawConfig.Contexts[o.userSpecifiedContext]
	} else {
		currentContext, exists = o.rawConfig.Contexts[o.rawConfig.CurrentContext]
	}

	if !exists {
		return fmt.Errorf("context doesn't exist")
	}

	o.resultingContext = currentContext.DeepCopy()
	if o.userSpecifiedNamespace != "" {
		o.resultingContext.Namespace = o.userSpecifiedNamespace
	}
	if o.resultingContext.Namespace == "" {
		o.resultingContext.Namespace = "default"
	}

	o.restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: o.configFlags.ToRawKubeConfigLoader().ConfigAccess().GetDefaultFilename()},
		&clientcmd.ConfigOverrides{
			CurrentContext: o.userSpecifiedContext,
		}).ClientConfig()
	if err != nil {
		return err
	}

	o.clientset, err = kubernetes.NewForConfig(o.restConfig)
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *DiagOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}
	havePodName := len(o.podName) == 0
	haveLabelSelector := len(o.labelSelector) == 0

	if havePodName == haveLabelSelector {
		return fmt.Errorf("one of pod-name,label-selector must be provided, but not both")
	}
	if !havePodName {
		pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{LabelSelector: o.labelSelector})
		if err != nil {
			return err
		}
		o.podName = pl.Items[0].Name
	}

	return nil
}

func AddLabelSelectorFlagVar(cmd *cobra.Command, p *string) {
	cmd.Flags().StringVarP(p, "selector", "l", *p, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.")
}
