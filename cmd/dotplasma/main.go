package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/A-Murchison/dot-plasma/internal/cli"
	"github.com/A-Murchison/dot-plasma/internal/profile"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		code := cli.ExitCode(err)
		if !errors.Is(err, profile.ErrDifferencesFound) {
			fmt.Fprintf(os.Stderr, "dotplasma: %s\n", err.Error())
		}
		os.Exit(code)
	}
}
