package main

import (
	"context"
	"os"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"github.com/yuval-k/kdiag/pkg/cmd/diag"
	"github.com/yuval-k/kdiag/pkg/log"
	"k8s.io/klog/v2"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	ctx := log.InitialCmdContext(context.Background())
	klog.SetLogger(zapr.NewLogger(log.WithContext(ctx)))
	flags := pflag.NewFlagSet("kubectl-diag", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := diag.NewCmdDiag(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
