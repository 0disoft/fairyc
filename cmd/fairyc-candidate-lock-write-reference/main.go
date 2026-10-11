// Fixture/oracle setup only. The candidate owns the actual save protocol.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/lockfile"
)

func write(root, leaf string, data []byte) {
	path := filepath.Join(root, filepath.FromSlash(leaf))
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		panic(e)
	}
	if e := os.WriteFile(path, data, 0600); e != nil {
		panic(e)
	}
}
func main() {
	if len(os.Args) != 3 {
		panic("usage: fairyc-candidate-lock-write-reference SDK_FIXTURES OUTPUT")
	}
	root, e := filepath.Abs(os.Args[2])
	if e != nil {
		panic(e)
	}
	for _, name := range []string{"None", "Core"} {
		input, e := os.ReadFile(filepath.Join(os.Args[1], name, "lock.json"))
		if e != nil {
			panic(e)
		}
		if _, e := lockfile.Parse(input); e != nil {
			panic(e)
		}
		write(root, name+".json", input)
		parent := filepath.Join(root, "go", name)
		if e := os.MkdirAll(parent, 0700); e != nil {
			panic(e)
		}
		path := filepath.Join(parent, lockfile.FileName)
		if e := lockfile.WriteAtomic(path, input); e != nil {
			panic(e)
		}
		write(root, "go/"+name+"/fairy.lock", []byte("OLD"))
		if e := lockfile.WriteAtomic(path, input); e != nil {
			panic(e)
		}
		actual, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(actual, input) {
			panic("owning Go save mismatch")
		}
		var raw map[string]any
		if e := json.Unmarshal(input, &raw); e != nil {
			panic(e)
		}
		noncanonical, e := json.Marshal(raw)
		if e != nil {
			panic(e)
		}
		if e := lockfile.WriteAtomic(path, noncanonical); e == nil {
			panic("Go accepted noncanonical bytes")
		}
		actual, e = os.ReadFile(path)
		if e != nil || !bytes.Equal(actual, input) {
			panic("rejected Go save changed live lock")
		}
		if name == "None" {
			write(root, "noncanonical.json", noncanonical)
		}
	}
	for _, name := range []string{"Existing", "Missing", "한글-λ", "Collision", "Sharing", "Invalid", "Noncanonical", "Bounds", "Namespace", "fault-probe"} {
		if e := os.MkdirAll(filepath.Join(root, name), 0700); e != nil {
			panic(e)
		}
	}
	write(root, "Collision/.fairy-lock-0.tmp", []byte("COLLISION"))
	write(root, "Collision/.fairy-lock-1.tmp/sentinel", []byte("DIRECTORY"))
	for i := 0; i < 4096; i++ {
		write(root, fmt.Sprintf("Namespace/.fairy-lock-%d.tmp", i), []byte("OCCUPIED"))
	}
	write(root, "Directory/fairy.lock/sentinel", []byte("DIRECTORY"))
	write(root, "invalid.json", []byte("{}"))
	write(root, "root.bin", []byte(root))
	fmt.Println("Go canonical create/replace and noncanonical-preservation observations passed; candidate fixture roots prepared")
}
