package srv

import (
	"context"
	"fmt"
	"math"
	"net"

	pb "github.com/yuval-k/kdiag/pkg/api/kdiag"
	"github.com/yuval-k/kdiag/pkg/redir"
	"github.com/yuval-k/kdiag/pkg/tunnel"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedManagerServer
}

func Start(ctx context.Context, bindAddress string) error {
	if bindAddress == "" {
		bindAddress = ":0"
	}
	// new GRPC server at random port:
	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		return err
	}
	defer l.Close()

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterManagerServer(grpcServer, newServer())
	reflection.Register(grpcServer)

	return grpcServer.Serve(l)
}

func newServer() pb.ManagerServer {
	return new(server)
}

// Stream Envoy access logs as they are captured.
func (s *server) Redirect(r *pb.RedirectRequest, respStream pb.Manager_RedirectServer) error {

	if r.Port > math.MaxUint16 {
		return fmt.Errorf("port number %d is too large", r.Port)
	}

	redir, err := redir.NewRedirection(uint16(r.Port))
	if err != nil {
		return err
	}
	defer redir.Close()
	err = redir.Redirect()
	if err != nil {
		return err
	}
	signal := make(chan uint16, 1)
	ctx, cancel := context.WithCancel(respStream.Context())
	defer cancel()
	go tunnel.Tunnel(ctx, redir.Listener, signal)

	for {
		select {
		case <-respStream.Context().Done():
			return nil
		case port := <-signal:
			err = respStream.Send(&pb.RedirectResponse{Port: uint32(port)})
			if err != nil {
				return err
			}
		}

	}
}
