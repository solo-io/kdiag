package srv

import (
	"context"
	"fmt"
	"io"
	"net"

	pb "github.com/solo-io/kdiag/pkg/api/kdiag"
	"github.com/solo-io/kdiag/pkg/log"
	frwrd "github.com/solo-io/kdiag/pkg/portforward"
	"go.uber.org/zap"
)

// Stream Envoy access logs as they are captured.
func Redirect(ctx context.Context, client pb.ManagerClient, outgoing bool, podPort, localPort uint16, newPortForward func(ctx context.Context, podPort uint16) (*frwrd.PortForward, error)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cli, err := client.Redirect(ctx, &pb.RedirectRequest{Port: uint32(podPort), Outgoing: outgoing})
	if err != nil {
		return err
	}
	var portFw *frwrd.PortForward
	var localFwPort uint16
	logger := log.WithContext(ctx)
	for {
		msg, err := cli.Recv()
		if err != nil {
			return err
		}
		podPortServer := uint16(msg.Port)

		if portFw == nil {
			portFw, err = newPortForward(ctx, podPortServer)
			if err != nil {
				return err
			}
			localFwPort, err = portFw.LocalPort()
			if err != nil {
				return err
			}
		}
		var d net.Dialer
		conn1, err := d.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", localPort))
		if err != nil {
			// if we can't connect to the local port, assume it is a transient error.
			// log the error and continue
			logger.With(zap.Error(err)).Debug("error connecting to local port")
		}

		conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", localFwPort))
		if err != nil {
			return err
		}

		proxy(conn1, conn)
	}
}

func proxy(localConn, remoteConn net.Conn) {
	// if we couldn't connect to the local pod, close the remote one after connecting
	// to propagate connection state upstream
	if localConn == nil {
		remoteConn.Close()
		return
	}

	go func() {
		defer remoteConn.Close()
		io.Copy(remoteConn, localConn)
	}()
	go func() {
		defer localConn.Close()
		io.Copy(localConn, remoteConn)
	}()
}
