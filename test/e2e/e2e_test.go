package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/solo-io/kdiag/pkg/cmd/diag"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = Describe("E2e", func() {
	var (
		ns        string
		clientset *kubernetes.Clientset

		ctx context.Context
	)
	const (
		labelSelector = "app=nginx"
	)
	// we assume we have a cluster setup, with nginx pod and curl that curls it
	// every second. i.e. we assume that make create-test-env was run

	BeforeEach(func() {
		// get our client:

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
		restConfig, err := clientConfig.ClientConfig()
		Expect(err).NotTo(HaveOccurred())

		clientset, err = kubernetes.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())

		ns, _, err = clientConfig.Namespace()
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
	})

	BeforeEach(func() {
		// delete existing pods so we have a clean slate
		clientset.CoreV1().Pods(ns).DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: labelSelector})

		Eventually(func() error {

			pl, err := clientset.CoreV1().Pods(ns).List(ctx, v1.ListOptions{LabelSelector: labelSelector})
			if err != nil {
				return err
			}
			if len(pl.Items) != 1 {
				return errors.New("other pod still exists")
			}
			for _, pod := range pl.Items {
				if pod.DeletionTimestamp != nil {
					continue
				}
				if pod.Status.Phase == corev1.PodRunning {
					return nil
				}
			}
			return fmt.Errorf("no pod is running")

		}, "20s", "1s").Should(Succeed())

	})

	It("should redirect traffic to us", func() {
		received := make(chan struct{})
		var once sync.Once
		go http.ListenAndServe("localhost:8989", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			once.Do(func() {
				close(received)
			})
		}))
		f, err := os.Open(os.DevNull)
		Expect(err).NotTo(HaveOccurred())
		defer f.Close()

		// run redir command
		root := diag.NewCmdDiag(genericclioptions.IOStreams{In: f, Out: GinkgoWriter, ErrOut: GinkgoWriter})
		root.SetArgs([]string{
			"-l", labelSelector, "redir", "80:8989",
		})

		ctx, cancel := context.WithCancel(context.Background())
		received1 := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			err = root.ExecuteContext(ctx)
			if errors.Is(err, context.Canceled) {
				Expect(err).NotTo(HaveOccurred())
			}
			close(received1)
		}()
		// the curl pod should hit the nginx pod every second
		Eventually(received, "20s").Should(BeClosed())
		cancel()
		Eventually(received1, "10s").Should(BeClosed())
	})

})
