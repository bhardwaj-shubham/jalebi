package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bhardwaj-shubham/jalebi/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := cli.CmdArgs(ctx, os.Args[1:])
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("\nOperation cancelled due to interrupt.")
			return
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
