package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func CmdArgs(args []string) error {
	if len(args) == 0 {
		return errors.New("no project directory provided")
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") || arg == "-v" || arg == "-h" {
			return flags(arg)
		}
	}

	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path %q: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %q", path)
	}

	fmt.Println("Project:", info.Name())

	return nil
}

func flags(flag string) error {
	switch flag {
	case "--version", "-v":
		fmt.Println("Jalebi: 0.1")
		return nil

	case "--help", "-h":
		fmt.Println("Jalebi is made for developer to understand project.\nJalebi CLI Version: 0.1")
		return nil

	default:
		return fmt.Errorf("unknown flag: %s", flag)
	}
}
