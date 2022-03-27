package redir

import (
	"net"
	"os/exec"
	"strconv"
)

type Redirection struct {
	Listener net.Listener

	fromPort  uint16
	localPort uint16
}

func NewRedirection(fromPort uint16) (*Redirection, error) {

	// connect to the manager in the pod,
	// start a stream and wait for remote connections
	// proxy remote connections to local port

	// start listening socket on a random port
	// setup iptables redirect to that random port
	// every connection received, proxy to client via the grpc connection

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	// get the local port from the listener
	//	localPort := l.Addr().(*net.TCPAddr).Port

	listenerAddress := listener.Addr().String()
	_, localPort, _ := net.SplitHostPort(listenerAddress)
	localPortUInt, err := strconv.ParseUint(localPort, 10, 16)
	if err != nil {
		return nil, err
	}
	return &Redirection{
		Listener:  listener,
		fromPort:  fromPort,
		localPort: uint16(localPortUInt)}, nil
}

func (r *Redirection) Redirect() error {
	return exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", strconv.Itoa(int(r.fromPort)), "-j", "REDIRECT", "--to-port", strconv.Itoa(int(r.localPort))).Run()
}

func (r *Redirection) Close() error {
	defer r.Listener.Close()
	err := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", strconv.Itoa(int(r.fromPort)), "-j", "REDIRECT", "--to-port", strconv.Itoa(int(r.localPort))).Run()
	return err
}
