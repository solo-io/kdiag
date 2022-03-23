package frwrd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForward struct {
	Ctx          context.Context
	Port         int
	PodName      string
	PodNamespace string
	RESTConfig   *rest.Config
	RESTClient   rest.Interface

	Out    io.Writer
	ErrOut io.Writer

	// Address       []string
	// Ports         []string

	// StopChannel   chan struct{}
	ReadyChannel chan struct{}
	StopChannel  <-chan struct{}

	fw *portforward.PortForwarder
}

func (p *PortForward) LocalPort() (uint16, error) {
	ports, err := p.fw.GetPorts()
	if err != nil {
		return 0, err
	}

	return ports[0].Local, nil
}

func (p *PortForward) ForwardPorts() error {
	if p.ReadyChannel == nil {
		p.ReadyChannel = make(chan struct{})
	}

	req := p.RESTClient.Post().Prefix("api", "v1").
		Resource("pods").
		Namespace(p.PodNamespace).
		Name(p.PodName).
		SubResource("portforward")
	method := "POST"
	url := req.URL()
	address := []string{"localhost"}

	ports := []string{fmt.Sprintf("%d:%d", 0, p.Port)}

	transport, upgrader, err := spdy.RoundTripperFor(p.RESTConfig)
	if err != nil {
		return fmt.Errorf("failed to create round tripper: %v", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, method, url)
	stopChannel := p.StopChannel
	if stopChannel == nil {
		stopChannel = p.Ctx.Done()
	}
	p.fw, err = portforward.NewOnAddresses(dialer, address, ports, stopChannel, p.ReadyChannel, p.Out, p.ErrOut)
	errchan := make(chan error, 1)

	go func() {
		errchan <- p.fw.ForwardPorts()
	}()
	select {
	case err := <-errchan:
		return err
	case <-p.ReadyChannel:
		return nil
	case <-p.Ctx.Done():
		return nil
	case <-time.After(time.Second * 10):
		return fmt.Errorf("timeout waiting for port forward to be ready")
	}
}
