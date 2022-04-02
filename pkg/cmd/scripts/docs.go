package main

import (
	"os"

	"github.com/solo-io/kdiag/pkg/cmd/diag"
	"github.com/spf13/cobra/doc"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {

	root := diag.NewCmdDiag(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	err := doc.GenMarkdownTree(root, "./docs")
	if err != nil {
		panic(err)
	}
}
