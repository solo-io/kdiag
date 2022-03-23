package tunnel

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/yuval-k/kdiag/pkg/log"
	"go.uber.org/zap"
)

// Tunnels this listener to the remote port.
func Tunnel(ctx context.Context, l net.Listener, signal chan<- uint16) error {

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	// get the local port from the listener
	//	localPort := l.Addr().(*net.TCPAddr).Port

	listenerAddress := listener.Addr().String()
	_, localPort, _ := net.SplitHostPort(listenerAddress)
	localPortUInt, err := strconv.ParseUint(localPort, 10, 16)
	if err != nil {
		return err
	}
	logger := log.WithContext(ctx)
	if logger == nil {
		logger = ctxzap.Extract(ctx)
	}
	logger = logger.With(zap.String("component", "tunnel"))
	for {
		logger.Debug("waiting for new connection")
		// wait for local connection
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		logger.Debug("signaling to client")
		// signal that we received a connection
		signal <- uint16(localPortUInt)
		// wait for remote connection
		logger.Debug("waiting for client connection")
		conn1, err := listener.Accept()
		if err != nil {
			return err
		}

		logger.Debug("proxying connection")
		// tunnel them!
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
