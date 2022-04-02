package redir

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
)

type Redirection struct {
	Listener net.Listener

	fromPort  uint16
	localPort uint16
	outgoing  bool
}

func NewRedirection(fromPort uint16, outgoing bool) (*Redirection, error) {

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
		outgoing:  outgoing,
		localPort: uint16(localPortUInt)}, nil
}

func (r *Redirection) Redirect() error {
	if r.outgoing {
		return execute("iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", strconv.Itoa(int(r.fromPort)), "-j", "DNAT", "--to-destination", "127.0.0.1:"+strconv.Itoa(int(r.localPort)))
	}
	return execute("iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", strconv.Itoa(int(r.fromPort)), "-j", "REDIRECT", "--to-port", strconv.Itoa(int(r.localPort)))
}

func (r *Redirection) Close() error {
	defer r.Listener.Close()

	if r.outgoing {
		return execute("iptables", "-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", strconv.Itoa(int(r.fromPort)), "-j", "DNAT", "--to-destination", "127.0.0.1:"+strconv.Itoa(int(r.localPort)))
	}

	return execute("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", strconv.Itoa(int(r.fromPort)), "-j", "REDIRECT", "--to-port", strconv.Itoa(int(r.localPort)))
}

func execute(cmd string, args ...string) error {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		if len(out) > 1024 {
			out = out[:1024]
		}
		return fmt.Errorf("%s %v failed: %w. output: %s", cmd, args, err, string(out))
	}
	return nil
}
