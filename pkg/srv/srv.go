package srv

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"time"

	ps "github.com/mitchellh/go-ps"
	"github.com/samber/lo"
	pb "github.com/solo-io/kdiag/pkg/api/kdiag"
	"github.com/solo-io/kdiag/pkg/log"
	"github.com/solo-io/kdiag/pkg/redir"
	"github.com/solo-io/kdiag/pkg/sockets"
	"github.com/solo-io/kdiag/pkg/tunnel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
)

type server struct {
	pb.UnimplementedManagerServer
}

func Start(ctx context.Context, logOut io.Writer, bindAddress string) error {
	if bindAddress == "" {
		bindAddress = ":0"
	}
	// new GRPC server at random port:
	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		return err
	}
	defer l.Close()

	// print the port to stdout... TODO: do this via logging?!
	// problem is we want it always outputted.
	fmt.Fprintf(logOut, "Listening on %s\n", l.Addr().String())

	logOpts := []grpc_zap.Option{}
	var opts []grpc.ServerOption
	zapLogger := log.WithContext(ctx)

	keepaliveTime := 10 * time.Second

	opts = append(
		opts, grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(zapLogger, logOpts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(zapLogger, logOpts...),
		),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime: keepaliveTime / 2,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:                  keepaliveTime,
			Timeout:               keepaliveTime,
			MaxConnectionAge:      time.Duration(math.MaxInt64),
			MaxConnectionAgeGrace: keepaliveTime,
		}),
	)

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

	redir, err := redir.NewRedirection(uint16(r.Port), r.Outgoing)
	if err != nil {
		return fmt.Errorf("could not create redirection: %w", err)
	}
	defer redir.Close()
	err = redir.Redirect()
	if err != nil {
		return fmt.Errorf("could not redirect: %w", err)
	}

	go func() {
		<-respStream.Context().Done()
		redir.Listener.Close()
	}()

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
				return fmt.Errorf("could not send response: %w", err)
			}
		}

	}
}

func (s *server) Ps(ctx context.Context, r *pb.PsRequest) (*pb.PsResponse, error) {
	//	os.proc
	processList, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("could not get process list: %w", err)
	}

	proceses, err := sockets.GetListeningPorts(ctx)
	if err != nil {
		log.WithContext(ctx).With(zap.Error(err)).Error("could not get listening ports")
	}

	procs := lo.Map(processList, func(t ps.Process, _ int) *pb.PsResponse_ProcessInfo {
		var addrs []*pb.Address
		if proceses != nil {
			if socks, ok := proceses[t.Pid()]; ok {
				addrs = lo.Map(socks.Sockets, func(sock *sockets.Socket, _ int) *pb.Address {
					sid := sock.ID
					return &pb.Address{
						Ip:   sid.Source.String(),
						Port: uint32(sid.SourcePort),
					}
				})
			}
		}

		return &pb.PsResponse_ProcessInfo{
			Pid:             uint64(t.Pid()),
			Ppid:            uint64(t.PPid()),
			Name:            t.Executable(),
			ListenAddresses: addrs,
		}
	})

	myPid := os.Getpid()
	procs = lo.Reject(procs, func(v *pb.PsResponse_ProcessInfo, i int) bool {
		return v.Pid == uint64(myPid)
	})
	// map ages
	resp := &pb.PsResponse{
		Processes: procs,
	}
	return resp, nil
}

func (s *server) Pprof(context.Context, *pb.PprofRequest) (*pb.PprofResponse, error) {

	exec.CommandContext(context.Background(), "google-pprof", "")

	return nil, fmt.Errorf("method Pprof not implemented")
}
