package main

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/yuval-k/kdiag/pkg/cmd/diag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-diag", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := diag.NewCmdDiag(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
