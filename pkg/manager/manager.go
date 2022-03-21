package manager

import (
	"os/exec"

	corev1 "k8s.io/api/core/v1"
)

type manager struct {
	podObj    *corev1.Pod
	container string
	port      int
}

func NewManager(podObj *corev1.Pod, container string, port int) (Manager, error) {
	return &manager{
		podObj:    podObj,
		container: container,
		port:      port,
	}, nil
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

func (m *manager) RedirectTraffic(podPort int, localPort int) error {
	// connect to the manager in the pod,
	// start a stream and wait for remote connections
	// proxy remote connections to local port

	panic("not implemented") // TODO: Implement
}
