package srv

import (
	"context"

	srvimpl "github.com/yuval-k/kdiag/pkg/srv"
)

func Run() {
	srvimpl.Start(context.Background(), "")
}
