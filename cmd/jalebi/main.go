package main

import (
	"fmt"
	"os"

	"github.com/bhardwaj-shubham/jalebi/internal/cli"
)

func main() {
	if err := cli.CmdArgs(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
