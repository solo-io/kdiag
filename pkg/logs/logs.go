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
	"time"

	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type logEntry struct {
	podName string
	err     error
	log     string
	done    bool
	fprintf func(w io.Writer, format string, a ...interface{}) (n int, err error)
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

func pallete(i int) *color.Color {
	colors := []*color.Color{
		color.New(color.FgHiRed),
		color.New(color.FgHiGreen),
		color.New(color.FgHiYellow),
		color.New(color.FgHiBlue),
		color.New(color.FgHiMagenta),
		color.New(color.FgHiCyan),
		color.New(color.FgHiWhite),
	}
	return colors[i%len(colors)]
}

type MultiLogPrinter struct {
	Out          io.Writer
	ErrOut       io.Writer
	In           io.Reader
	Args         []string
	LogDrainTime time.Duration
}

// Run lists all available namespaces on a user's KUBECONFIG or updates the
// current context based on a provided namespace.
func (m *MultiLogPrinter) PrintLogs(ctx context.Context, podclient typedcorev1.PodInterface, podNames []PodAndContainerName) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// exec!
	zero := int64(0)
	logEntries := make(chan logEntry)
	allDone := make(chan struct{})

	var wg sync.WaitGroup
	for i, podName := range podNames {

		podNameColor := pallete(i)
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

				if len(bytes) != 0 {
					logline := strings.TrimSuffix(string(bytes), "\n")
					logEntries <- logEntry{podName: podName, fprintf: podNameColor.Fprintf, log: logline}
				}
				if err != nil {
					if err != io.EOF {
						err := fmt.Errorf("failed to read logs: %w", err)
						logEntries <- logEntry{podName: podName, fprintf: podNameColor.Fprintf, err: err, done: true}
					} else {
						logEntries <- logEntry{podName: podName, fprintf: podNameColor.Fprintf, done: true}
					}
					return
				}

			}
		}(podName.String())
	}

	go func() {
		wg.Wait()
		close(allDone)
	}()

	printLoopDone := make(chan struct{})
	go func() {
		defer close(printLoopDone)
		for entry := range logEntries {
			if entry.err != nil {
				if !errors.Is(entry.err, context.Canceled) {
					fmt.Fprintf(m.ErrOut, "error reading logs for %s: %v\n", entry.podName, entry.err)
				}
			} else if entry.done {
				fmt.Fprintf(m.Out, "pod %s is done\n", entry.podName)
			} else {
				entry.fprintf(m.Out, "%s:", entry.podName)
				fmt.Fprintf(m.Out, " %s\n", entry.log)
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
		// command done, wait the drain time
		if m.LogDrainTime != 0 {
			time.Sleep(m.LogDrainTime)
		}
	} else {
		// wait until user interrupts us.
		// or all the pods exited
		select {
		case <-ctx.Done():
		case <-allDone:
		}
	}

	// cancel the log context
	cancel()
	// drain pending logs
	wg.Wait()
	// close channel so print loop exits.
	close(logEntries)
	// wait for print loop to exit
	<-printLoopDone
	return nil
}
