package diag

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddSinglePodFlags(cmd *cobra.Command, o *DiagOptions) {
	cmd.PersistentFlags().StringVar(&o.podName, "pod", "", "podname to diagnose")
	cmd.PersistentFlags().StringVarP(&o.labelSelector, "labels", "l", "", "select a pod by label. an arbitrary pod will be selected, with preference to newer pods.")
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

	return nil
}