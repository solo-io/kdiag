package frwrd

import (
	"context"
	"fmt"
	"io"
	"net/http"

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
	Out          io.Writer
	ErrOut       io.Writer

	// Address       []string
	// Ports         []string

	// StopChannel   chan struct{}
	ReadyChannel chan struct{}
	StopChannel  <-chan struct{}

	fw *portforward.PortForwarder
}

func (p *PortForward) RESTClient() (*rest.RESTClient, error) {
	config := rest.CopyConfig(p.RESTConfig)
	rest.SetKubernetesDefaults(config)
	return rest.RESTClientFor(config)

}

func (p *PortForward) LocalPort() (uint16, error) {
	ports, err := p.fw.GetPorts()
	if err != nil {
		return 0, err
	}

	return ports[0].Local, nil
}

func (p *PortForward) ForwardPorts() error {
	cli, err := p.RESTClient()
	if err != nil {
		return err
	}
	if p.ReadyChannel == nil {
		p.ReadyChannel = make(chan struct{})
	}

	req := cli.Post().
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
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, method, url)
	stopChannel := p.StopChannel
	if stopChannel == nil {
		stopChannel = p.Ctx.Done()
	}
	p.fw, err = portforward.NewOnAddresses(dialer, address, ports, stopChannel, p.ReadyChannel, p.Out, p.ErrOut)

	return p.fw.ForwardPorts()
}
