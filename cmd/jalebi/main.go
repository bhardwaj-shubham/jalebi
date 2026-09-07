package main

import (
	"fmt"
	"os"

	"github.com/bhardwaj-shubham/jalebi/internal/cli"
)

func main() {
	if err := cli.Greet("Jalebi"); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
