//go:build !linux

package durablefile

import "os"

func openReadFile(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}

func openReadDirectory(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}
