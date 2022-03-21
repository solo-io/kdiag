package manager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/yuval-k/kdiag/pkg/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type ProcessInfo struct {
	Pid  int
	Name string
}

type Manager interface {
	GetProcesses() ([]ProcessInfo, error)
	Run(cmd *exec.Cmd) error
	StartInteractiveShell() error
	RedirectTraffic(podPort int, localPort int) error
}

func NewEmephemeralContainerManager(podGetter typedcorev1.PodsGetter) *EmephemeralContainerManager {
	return &EmephemeralContainerManager{
		podGetter: podGetter,
	}
}

type EmephemeralContainerManager struct {
	podGetter typedcorev1.PodsGetter
}

var (
	portRegexp = regexp.MustCompile(`\d+`)
)

// Create or connect to an ephemeral manager container in a pod
func (e *EmephemeralContainerManager) EnsurePodManaged(ctx context.Context, ns, pod, dbgimg, target string) (*corev1.Pod, error) {

	// name prefix is "dbg-tools-versionhash"

	podclient := e.podGetter.Pods(ns)
	podObj, err := podclient.Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	name := e.ContainerName()

	_, found := lo.Find(podObj.Spec.EphemeralContainers, func(t corev1.EphemeralContainer) bool {
		if t.Name == name {
			return true
		}
		return false
	})
	if !found {
		podObj, err = e.createContainer(ctx, name, dbgimg, target, podObj)
		if err != nil {
			return nil, err
		}
	}
	err = e.waitForReady(ctx, podclient, podObj)
	if err != nil {
		return nil, err
	}

	return podObj, nil
}

func (e *EmephemeralContainerManager) waitForReady(ctx context.Context, podclient typedcorev1.PodInterface, podObj *corev1.Pod) error {
	timeout := time.After(5 * time.Minute)
	name := e.ContainerName()
	for {
		updatedPod, err := podclient.Get(ctx, podObj.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		container, found := lo.Find(updatedPod.Status.EphemeralContainerStatuses, func(t corev1.ContainerStatus) bool {
			if t.Name == name {
				return true
			}
			return false
		})
		if found {
			if container.State.Running != nil {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-timeout:
			return fmt.Errorf("timeout waiting for pod to be ready")
		case <-time.After(1 * time.Second):
		}
	}

}

func (e *EmephemeralContainerManager) ContainerName() string {
	h := fnv.New32()
	h.Write([]byte(version.Version))

	name := fmt.Sprintf("dbg-tools-%x", h.Sum32())
	return name
}

func (e *EmephemeralContainerManager) ConnectToManager(ctx context.Context, podclient typedcorev1.PodInterface, podObj *corev1.Pod) (Manager, error) {

	name := e.ContainerName()
	port, err := getPortFromLogs(ctx, podclient, podObj, name)
	if err != nil {
		return nil, err
	}

	// port forward this port
	NewManager(podObj, name, port)

	// port forward this port
	return NewManager(podObj, name, port)
}

func getPortFromLogs(ctx context.Context, podclient typedcorev1.PodInterface, podObj *corev1.Pod, container string) (int, error) {

	// now, connect to the manager in the pod:
	currOpts := &corev1.PodLogOptions{
		Container: container,
		Follow:    false,
	}
	readCloser, err := podclient.GetLogs(podObj.Name, currOpts).Stream(ctx)
	if err != nil {
		return 0, err
	}
	defer readCloser.Close()
	r := bufio.NewReader(readCloser)
	for {
		bytes, err := r.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
			return 0, fmt.Errorf("no port found in log")
		}
		submatch := portRegexp.FindSubmatch(bytes)
		if submatch != nil {
			port, err := strconv.Atoi(string(submatch[0]))
			if err != nil {
				return 0, err
			}
			return port, nil
		}
	}
}

func (e *EmephemeralContainerManager) createContainer(ctx context.Context, containerName, dbgimg, target string, podObj *corev1.Pod) (*corev1.Pod, error) {
	if target == "" {
		target = podObj.Spec.Containers[0].Name
	}
	trueVar := true
	ephemeralContainer := corev1.EphemeralContainer{
		TargetContainerName: target,
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:                     containerName,
			Image:                    dbgimg,
			ImagePullPolicy:          corev1.PullIfNotPresent,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &trueVar,
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"SYS_PTRACE", "SYS_NET_ADMIN", "CAP_SYS_ADMIN"},
				},
			},
		},
	}
	podJS, err := json.Marshal(podObj)
	if err != nil {
		return nil, fmt.Errorf("error creating JSON for pod: %w", err)
	}

	debugPod := podObj.DeepCopy()
	debugPod.Spec.EphemeralContainers = append(debugPod.Spec.EphemeralContainers, ephemeralContainer)
	debugJS, err := json.Marshal(debugPod)
	if err != nil {
		return nil, fmt.Errorf("error creating JSON for debug container: %w", err)
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(podJS, debugJS, podObj)
	if err != nil {
		return nil, fmt.Errorf("error creating patch to add debug container: %w", err)
	}
	// use patch to update pod, that way we don't need to deal with conflicts.
	podClient := e.podGetter.Pods(podObj.Namespace)
	podObj, err = podClient.Patch(ctx, podObj.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "ephemeralcontainers")
	// _, err = podClient.UpdateEphemeralContainers(ctx, podObj.Name, podObj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	_, found := lo.Find(podObj.Spec.EphemeralContainers, func(t corev1.EphemeralContainer) bool {
		if t.Name == containerName {
			return true
		}
		return false
	})
	if !found {
		return nil, fmt.Errorf("container %s not found", containerName)
	}
	return podObj, nil
}
