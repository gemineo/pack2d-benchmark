package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("pack2d-benchmark %s\n", version)
			fmt.Printf("Go:    %s\n", runtime.Version())
			fmt.Printf("OS:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
