package srv

import (
	"context"
	"os"

	"github.com/go-logr/zapr"
	"github.com/yuval-k/kdiag/pkg/log"
	srvimpl "github.com/yuval-k/kdiag/pkg/srv"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
	"k8s.io/klog/v2"
)

func Run() {
	ctx := log.InitialContext(context.Background())
	grpclog.SetLoggerV2(zapgrpc.NewLogger(log.WithContext(ctx)))
	klog.SetLogger(zapr.NewLogger(log.WithContext(ctx)))
	srvimpl.Start(ctx, os.Stdout, "")
}
