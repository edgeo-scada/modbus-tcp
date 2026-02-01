// Package main provides a comprehensive Modbus TCP CLI client.
package main

import (
	"fmt"
	"os"
)

var version = "1.0.0"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
