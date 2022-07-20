package logs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type logEntry struct {
	podName string
	err     error
	log     string
	done    bool
}

type PodAndContainerName struct {
	PodName string
	// may be empty
	ContainerName string
}

func (p *PodAndContainerName) String() string {
	podnameToPrint := p.PodName
	if p.ContainerName != "" {
		podnameToPrint += ":" + p.ContainerName
	}
	return podnameToPrint
}

type MultiLogPrinter struct {
	Out    io.Writer
	ErrOut io.Writer
	In     io.Reader
	Args   []string
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (m *MultiLogPrinter) PrintLogs(ctx context.Context, podclient typedcorev1.PodInterface, podNames []PodAndContainerName) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// exec!
	zero := int64(0)
	logEntries := make(chan logEntry)

	var wg sync.WaitGroup
	for _, podName := range podNames {
		// get the logs from the pod
		currOpts := &corev1.PodLogOptions{
			Container: podName.ContainerName,
			Follow:    true,
			TailLines: &zero,
		}
		readCloser, err := podclient.GetLogs(podName.PodName, currOpts).Stream(ctx)
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
						logEntries <- logEntry{podName: podName, err: err, done: true}
					} else {
						logEntries <- logEntry{podName: podName, done: true}
					}
					return
				}

				logline := strings.TrimSuffix(string(bytes), "\n")
				logEntries <- logEntry{podName: podName, log: logline}
			}
		}(podName.String())
	}

	printLoopDone := make(chan struct{})
	go func() {
		defer close(printLoopDone)
		for entry := range logEntries {
			if entry.err != nil && !errors.Is(entry.err, context.Canceled) {
				fmt.Fprintf(m.ErrOut, "error reading logs for %s: %v\n", entry.podName, entry.err)
			} else if entry.done {
				fmt.Fprintf(m.Out, "pod %s is done\n", entry.podName)
			} else {
				fmt.Fprintf(m.Out, "%s: %s\n", entry.podName, entry.log)
			}
		}
	}()

	if len(m.Args) > 0 {
		cmd := exec.CommandContext(ctx, m.Args[0], m.Args[1:]...)
		cmd.Stderr = m.ErrOut
		cmd.Stdout = m.Out
		cmd.Stdin = m.In
		err := cmd.Start()
		if err != nil {
			return err
		}
		// wait until user command exits
		cmd.Wait()
	} else {
		// wait until user interrupts us.
		<-ctx.Done()
	}

	// cancel the log context
	cancel()
	// drain pending logs
	wg.Wait()
	// close channel so print loop exits.
	close(logEntries)
	<-printLoopDone
	return nil
}
