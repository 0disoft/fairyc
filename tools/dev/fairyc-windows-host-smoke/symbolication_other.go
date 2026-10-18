//go:build !windows

package main

import "fmt"

func symbolicateWindowsProgram([]byte, []byte, uint32) (windowsSymbolicationReceipt, error) {
	return windowsSymbolicationReceipt{}, fmt.Errorf("Windows symbolication requires Windows")
}
