package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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

	// for each .md file in ./docs, replace the explicit home dir with $HOME

	fi, err := ioutil.ReadDir("./docs")
	if err != nil {
		panic(err)
	}
	for _, fi := range fi {
		if fi.IsDir() {
			continue
		}
		if strings.HasSuffix(fi.Name(), ".md") {
			err := replaceHomeDir(filepath.Join("./docs", fi.Name()))
			if err != nil {
				panic(err)
			}
		}
	}

}

func replaceHomeDir(f string) error {
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	data = []byte(strings.Replace(string(data), home, "$HOME", -1))
	err = ioutil.WriteFile(f, data, 0644)
	return err
}
