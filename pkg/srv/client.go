package srv

import (
	"context"
	"fmt"
	"io"
	"net"

	pb "github.com/yuval-k/kdiag/pkg/api/kdiag"
	frwrd "github.com/yuval-k/kdiag/pkg/portforward"
)

// Stream Envoy access logs as they are captured.
func Redirect(ctx context.Context, client pb.ManagerClient, podPort, localPort uint16) error {

	cli, err := client.Redirect(ctx, &pb.RedirectRequest{Port: uint32(podPort)})
	if err != nil {
		return err
	}
	var portFw *frwrd.PortForward
	var localFwPort uint16
	for {
		msg, err := cli.Recv()
		if err != nil {
			return err
		}
		podPortServer := uint16(msg.Port)

		if portFw == nil {
			portFw = newPortForward(podPortServer)
			localFwPort, err = portFw.LocalPort()
			if err != nil {
				return err
			}
		}
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", localFwPort))
		if err != nil {
			return err
		}

		conn1, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", localPort))
		if err != nil {
			return err
		}

		go func() {
			defer conn.Close()
			io.Copy(conn, conn1)
		}()
		go func() {
			defer conn1.Close()
			io.Copy(conn1, conn)
		}()

	}
}

func newPortForward(podPort uint16) *frwrd.PortForward {
	panic("TODO")
}
