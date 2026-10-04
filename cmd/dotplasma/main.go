package main

import (
	"context"
	"fmt"
	"os"

	"dot-plasma/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "dotplasma: %s\n", err.Error())
		os.Exit(2)
	}
}
