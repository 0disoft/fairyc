//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() { fmt.Fprintln(os.Stderr, "private process oracle requires Windows"); os.Exit(2) }
