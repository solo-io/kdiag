package manager

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"time"

	pb "github.com/yuval-k/kdiag/pkg/api/kdiag"
	frwrd "github.com/yuval-k/kdiag/pkg/portforward"
	"github.com/yuval-k/kdiag/pkg/srv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type ProcessInfo struct {
	Pid  int
	Name string
}

type Manager interface {
	GetProcesses() ([]ProcessInfo, error)
	Run(cmd *exec.Cmd) error
	StartInteractiveShell() error
	RedirectTraffic(ctx context.Context, podPort, localPort uint16) error
}
type manager struct {
	RESTConfig   *rest.Config
	clientset    *kubernetes.Clientset
	Out          io.Writer
	ErrOut       io.Writer
	podname      string
	podnamespace string
	container    string
	port         int

	fw     *frwrd.PortForward
	conn   *grpc.ClientConn
	client pb.ManagerClient
}

func NewManager(
	ctx context.Context,
	RESTConfig *rest.Config,
	clientset *kubernetes.Clientset,
	Out io.Writer,
	ErrOut io.Writer, podname, podnamespace, container string) (Manager, error) {
	mgr := &manager{
		RESTConfig:   RESTConfig,
		clientset:    clientset,
		Out:          Out,
		ErrOut:       ErrOut,
		podname:      podname,
		podnamespace: podnamespace,
		container:    container,
	}

	getPortCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	port, err := getPortFromLogs(getPortCtx, clientset.CoreV1().Pods(podnamespace), podname, container)
	cancel()
	if err != nil {
		return nil, err
	}
	err = mgr.connect(ctx, uint16(port))
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (m *manager) connect(ctx context.Context, port uint16) error {
	fw, err := m.newPortForward(ctx, port)
	if err != nil {
		return fmt.Errorf("failed to create port forward to port %d: %w", port, err)
	}
	m.fw = fw
	localPort, err := fw.LocalPort()
	if err != nil {
		return fmt.Errorf("failed to get local port: %w", err)
	}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", localPort), opts...)
	if err != nil {
		return fmt.Errorf("fail to dial: %w", err)
	}
	m.conn = conn
	m.client = pb.NewManagerClient(conn)
	return nil
}

func (m *manager) Close() error {
	if m.conn != nil {
		m.conn.Close()
	}
	// TODO: cleanup the port forward
	//	if m.fw != nil {
	//		m.fw.Close()
	//	}
	return nil
}

func (m *manager) GetProcesses() ([]ProcessInfo, error) {
	panic("not implemented") // TODO: Implement
}

func (m *manager) Run(cmd *exec.Cmd) error {
	panic("not implemented") // TODO: Implement
}

func (m *manager) StartInteractiveShell() error {
	panic("not implemented") // TODO: Implement
}

func (m *manager) RedirectTraffic(ctx context.Context, podPort, localPort uint16) error {
	// connect to the manager in the pod,
	// start a stream and wait for remote connections
	// proxy remote connections to local port

	return srv.Redirect(ctx, m.client, podPort, localPort, m.newPortForward)
}

func (m *manager) newPortForward(ctx context.Context, port uint16) (*frwrd.PortForward, error) {

	fw := &frwrd.PortForward{
		Ctx:          ctx,
		Port:         int(port),
		PodName:      m.podname,
		PodNamespace: m.podnamespace,
		RESTConfig:   m.RESTConfig,
		RESTClient:   m.clientset.RESTClient(),
		Out:          m.Out,
		ErrOut:       m.ErrOut,
	}

	err := fw.ForwardPorts()
	if err != nil {
		return nil, fmt.Errorf("failed to forward port: %w", err)
	}

	select {
	case <-time.After(time.Second * 10):
		return nil, fmt.Errorf("timeout waiting for port forward to start")
	case <-fw.ReadyChannel:
		return fw, nil
	}
}

func getPortFromLogs(ctx context.Context, podclient typedcorev1.PodInterface, podName, container string) (int, error) {

	// now, connect to the manager in the pod:
	currOpts := &corev1.PodLogOptions{
		Container: container,
		Follow:    false,
	}
	readCloser, err := podclient.GetLogs(podName, currOpts).Stream(ctx)
	if err != nil {
		return 0, err
	}
	defer readCloser.Close()
	r := bufio.NewReader(readCloser)
	for {
		bytes, err := r.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return 0, fmt.Errorf("failed to read logs: %w", err)
			}
			return 0, fmt.Errorf("no port found in log")
		}
		submatch := portRegexp.FindSubmatch(bytes)
		if submatch != nil {
			port, err := strconv.Atoi(string(submatch[1]))
			if err != nil {
				return 0, fmt.Errorf("failed to parse port: %w", err)
			}
			return port, nil
		}
	}
}
