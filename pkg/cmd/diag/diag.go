package diag

import (
	"context"
	"fmt"
	"os"

	"github.com/solo-io/kdiag/pkg/version"
	"github.com/spf13/cobra"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
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

	dbgContainerImage   string
	podName             string
	targetContainerName string
	labelSelector       string
	pullPolicyString    string
	pullPolicy          corev1.PullPolicy
	genericclioptions.IOStreams
}

// NewDiagOptions provides an instance of DiagOptions with default values
func NewDiagOptions(streams genericclioptions.IOStreams) *DiagOptions {
	return &DiagOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

func versionStr() string {

	if len(version.VersionPrerelease) > 0 {
		return fmt.Sprintf("%s-%s %s", version.Version, version.VersionPrerelease, version.Commit)
	}
	return fmt.Sprintf("%s %s", version.Version, version.Commit)
}

// NewCmdDiag provides a cobra command wrapping DiagOptions
func NewCmdDiag(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDiagOptions(streams)

	cmd := &cobra.Command{
		Use:               "diag [new-Diag] [flags]",
		Short:             "View or set the current Diag",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
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
		Version: versionStr(),
	}

	defaultImage := "ghcr.io/solo-io/kdiag:" + version.Version
	if envImg := os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_DBG_IMAGE"); len(envImg) != 0 {
		defaultImage = envImg
	}
	cmd.PersistentFlags().StringVar(&o.dbgContainerImage, "dbg-image", defaultImage, "default dbg container image")

	o.configFlags.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(
		NewCmdRedir(o),
		NewCmdShell(o),
		NewCmdManage(o),
		NewCmdLogs(o),
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

	return nil
}

func AddLabelSelectorFlagVar(cmd *cobra.Command, p *string) {
	cmd.Flags().StringVarP(p, "selector", "l", *p, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.")
}
