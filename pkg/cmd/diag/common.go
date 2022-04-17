package diag

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddSinglePodFlags(cmd *cobra.Command, o *DiagOptions) {
	cmd.PersistentFlags().StringVar(&o.podName, "pod", "", "podname to diagnose")
	cmd.PersistentFlags().StringVarP(&o.targetContainerName, "target", "t", "", "target container to diagnose, defaults to first container in pod")
	cmd.PersistentFlags().StringVarP(&o.labelSelector, "labels", "l", "", "select a pod by label. an arbitrary pod will be selected, with preference to newer pods")
	cmd.PersistentFlags().StringVar(&o.pullPolicyString, "pull-policy", string(corev1.PullIfNotPresent), "image pull policy for the ephemeral container. defaults to IfNotPresent")
}

func ValidateSinglePodFlags(o *DiagOptions) error {
	havePodName := len(o.podName) != 0
	haveLabelSelector := len(o.labelSelector) != 0

	if havePodName == haveLabelSelector {
		return fmt.Errorf("one of pod-name,label-selector must be provided, but not both")
	}
	if !havePodName {
		pl, err := o.clientset.CoreV1().Pods(o.resultingContext.Namespace).List(o.ctx, metav1.ListOptions{LabelSelector: o.labelSelector})
		if err != nil {
			return err
		}
		pods := pl.Items
		if len(pods) == 0 {
			return fmt.Errorf("no pods found")
		}
		sort.Slice(pods, func(i, j int) bool {
			return pods[i].CreationTimestamp.Before(&pods[j].CreationTimestamp)
		})
		o.podName = pods[len(pods)-1].Name
	}

	switch o.pullPolicyString {
	case string(corev1.PullIfNotPresent), string(corev1.PullAlways), string(corev1.PullNever):
		// ok
		o.pullPolicy = corev1.PullPolicy(o.pullPolicyString)
	default:
		return fmt.Errorf("invalid pull-policy: %s", o.pullPolicyString)
	}

	return nil
}

func CommandName() string {
	if strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-") {
		return "kubectl diag"
	}
	return "kdiag"
}
