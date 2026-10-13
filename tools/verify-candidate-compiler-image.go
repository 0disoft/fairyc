//go:build ignore

// Derive exact executable identity strings using production Go identity code.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/buildidentity"
)

func run(root string) error {
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	var out bytes.Buffer
	out.WriteString("FYCIMAG1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(s string) { number(len(s)); out.WriteString(s) }
	sizes := []int{1, 55, 56, 63, 64, 65, 32767, 32768, 32769, 65539, 65539}
	number(len(sizes))
	var observations []map[string]any
	for i, size := range sizes {
		data := make([]byte, size)
		for j := range data {
			data[j] = byte((j*37 + j/251) % 256)
		}
		name := fmt.Sprintf("size-%d.bin", size)
		if i == len(sizes)-1 {
			name = "한글 이미지 복사.bin"
		}
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
		identity := buildidentity.Identity(sha256.Sum256(data)).String()
		if _, err := buildidentity.Parse(identity); err != nil {
			return err
		}
		field(name)
		number(size)
		field(identity)
		observations = append(observations, map[string]any{"path": name, "bytes": size, "identity": identity})
	}
	if err := os.WriteFile(filepath.Join(root, "empty.bin"), nil, 0644); err != nil {
		return err
	}
	oversized, err := os.Create(filepath.Join(root, "oversized.bin"))
	if err != nil {
		return err
	}
	truncateErr := oversized.Truncate(100663297)
	closeErr := oversized.Close()
	if truncateErr != nil {
		return truncateErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.WriteFile(filepath.Join(root, "oracle.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	current, err := buildidentity.Current()
	if err != nil {
		return err
	}
	path, err := os.Executable()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if current != buildidentity.Identity(sha256.Sum256(data)) {
		return fmt.Errorf("running Go executable identity differs")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "fixture": root, "cases": observations, "currentGoImageBytes": len(data), "currentGoImage": current.String(), "currentGoIdentityVerified": true})
}
func main() {
	root := flag.String("output", ".cache/compiler-identity-20261014/fixture", "fixture directory")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
