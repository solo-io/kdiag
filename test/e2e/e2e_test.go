package e2e_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

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

		ctx     context.Context
		devNull *os.File
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

		devNull, err = os.Open(os.DevNull)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		if devNull != nil {
			devNull.Close()
		}
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
			pod := pl.Items[0]
			if !pod.DeletionTimestamp.IsZero() {
				return errors.New("pod being deleted")
			}
			if pod.Status.Phase == corev1.PodRunning {
				return nil
			}
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					return nil
				}
			}
			return fmt.Errorf("no pod is running and ready")

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

		// run redir command
		root := diag.NewCmdDiag(genericclioptions.IOStreams{In: devNull, Out: GinkgoWriter, ErrOut: GinkgoWriter})
		root.SetArgs([]string{
			"-l", labelSelector, "redir", "80:8989",
		})

		ctx, cancel := context.WithCancel(context.Background())
		received1 := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			err := root.ExecuteContext(ctx)
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

	It("should redirect outgoing traffic to us", func() {
		received := make(chan struct{})
		var once sync.Once
		go http.ListenAndServe("localhost:8990", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			once.Do(func() {
				close(received)
			})
		}))

		// run redir command
		root := diag.NewCmdDiag(genericclioptions.IOStreams{In: devNull, Out: GinkgoWriter, ErrOut: GinkgoWriter})
		root.SetArgs([]string{
			"-l", "app=curl", "redir", "--outgoing", "80:8990",
		})

		ctx, cancel := context.WithCancel(context.Background())
		received1 := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			err := root.ExecuteContext(ctx)
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

	It("should have a top in the shell even though its not in the image", func() {
		out := &bytes.Buffer{}
		root := diag.NewCmdDiag(genericclioptions.IOStreams{In: devNull, Out: out, ErrOut: GinkgoWriter})
		root.SetArgs([]string{"shell", "-l", "app=curl", "--", "-c", "top -n 1"})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := root.ExecuteContext(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("do curl"))
	})

	It("should show logs from both apps a top in the shell even though its not in the image", func() {
		out := &bytes.Buffer{}
		root := diag.NewCmdDiag(genericclioptions.IOStreams{In: devNull, Out: out, ErrOut: out})
		root.SetArgs([]string{"logs", "--all"})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		root.ExecuteContext(ctx)

		Eventually(out.String(), "10s").Should(ContainSubstring("HTTP/1.1 200 OK"))             // curl
		Eventually(out.String(), "10s").Should(ContainSubstring(`HEAD / HTTP/1.1" 200 0 "-" `)) //nginx
		Expect(ctx.Err()).To(Equal(context.DeadlineExceeded))
	})

	It("should show logs using a command", func() {
		// use safe writer here as both command output and log output are written here,
		// and can happen concurrently.
		out := &SafeWriter{}

		root := diag.NewCmdDiag(genericclioptions.IOStreams{In: devNull, Out: out, ErrOut: out})

		// get the node port and node ip:
		nodes, err := clientset.CoreV1().Nodes().List(ctx, v1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(nodes.Items).ToNot(BeEmpty())

		svcs, err := clientset.CoreV1().Services(ns).List(ctx, v1.ListOptions{LabelSelector: labelSelector})
		Expect(err).NotTo(HaveOccurred())
		Expect(svcs.Items).ToNot(BeEmpty())

		ip := nodes.Items[0].Status.Addresses[0].Address
		port := svcs.Items[0].Spec.Ports[0].NodePort
		args := []string{"logs", "-l", labelSelector, "--drain-duration", "2s", "--", "curl", fmt.Sprintf("http://%s:%d/test", ip, port), "--retry", "3", "--max-time", "5"}
		root.SetArgs(args)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fmt.Fprintln(GinkgoWriter, "logs command:", args, "time", time.Now().Format(time.RFC3339))
		err = root.ExecuteContext(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(ctx.Err()).NotTo(HaveOccurred())
		Expect(out.Buff.String()).To(ContainSubstring(`GET /test HTTP/1.1" 404`)) //nginx
	})
})

type SafeWriter struct {
	Buff bytes.Buffer
	m    sync.Mutex
}

func (w *SafeWriter) Write(p []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.Buff.Write(p)
}
