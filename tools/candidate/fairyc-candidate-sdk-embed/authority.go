package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

// Generated literals alone cannot authenticate a changed SDK. Checking also
// requires the separately reviewed candidate authority to pin those exact bytes.
func checkAuthority(dir string) error {
	definition, err := sdkstdlib.LoadVector()
	if err != nil {
		return err
	}
	authority, err := os.ReadFile(filepath.Join(dir, "sdk_authority.fy"))
	if err != nil {
		return err
	}
	for index, source := range definition.Sources {
		pin := fmt.Sprintf("id==%d){hash::matches(source,b\"%x\")", index+1, sha256.Sum256(source.Bytes))
		if !strings.Contains(string(authority), pin) {
			return fmt.Errorf("candidate SDK authority differs from bundled %s; review the source and update its exact pin", source.Path)
		}
	}
	return nil
}
