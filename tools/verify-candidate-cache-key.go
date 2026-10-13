// Run with tools/run-go.ts run tools/verify-candidate-cache-key.go --output <fixture>.
// Existing Go packagecache owns every oracle byte; Fairy owns the implementation.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/0disoft/fairyc/internal/packagecache"
)

func run() error {
	if len(os.Args) != 3 || os.Args[1] != "--output" {
		return fmt.Errorf("usage: verify-candidate-cache-key.go --output <fixture>")
	}
	source, err := os.ReadFile("examples/lexer_candidate/tests/project_cache_key.fy")
	if err != nil {
		return err
	}
	var environment [8]string
	for index := range environment {
		pattern := `if\(index==` + strconv.Itoa(index) + `\)\{return b("(?:\\.|[^"\\])*");\}`
		found := regexp.MustCompile(pattern).FindSubmatch(source)
		if len(found) != 2 {
			return fmt.Errorf("missing authored environment %d", index)
		}
		environment[index], err = strconv.Unquote(string(found[1]))
		if err != nil {
			return err
		}
	}
	var output bytes.Buffer
	binary.Write(&output, binary.BigEndian, uint32(20))
	var rows []map[string]any
	for mode := 0; mode < 20; mode++ {
		e := environment
		if mode >= 1 && mode <= 8 {
			e[mode-1] += "!"
		}
		if mode == 19 {
			e[0] += "\x00제로디"
		}
		p := packagecache.Inputs{CompilerBuild: e[0], LanguageRevision: e[1], MIRVersion: e[2], TargetSchema: e[3], ABIProfile: e[4], ToolchainFingerprint: e[5], Backend: e[6], RuntimeABI: e[7]}
		facts := [][]byte{p.StandardLibrary[:], p.Package[:], p.Source[:], p.CurrentInterface[:], p.CurrentBody[:]}
		for i, fact := range facts {
			for j := range fact {
				fact[j] = byte(i + 1)
			}
			if mode == i+9 {
				fact[0] = 9
			}
		}
		if mode != 18 {
			p.ProfileFlags = []string{"-O2"}
			p.Features = []string{"a"}
			p.Dependencies = make([]packagecache.Dependency, 2)
			for i := range p.Dependencies {
				for j := 0; j < 32; j++ {
					p.Dependencies[i].Coordinate[j] = byte(1 + 2*i)
					p.Dependencies[i].Interface[j] = byte(2 + 2*i)
				}
			}
		}
		if mode == 14 {
			p.ProfileFlags = append(p.ProfileFlags, "-g")
		}
		if mode == 15 {
			p.Features = append(p.Features, "z")
		}
		if mode == 16 {
			for j := range p.Dependencies[0].Coordinate {
				p.Dependencies[0].Coordinate[j] = 2
			}
		}
		if mode == 17 {
			for j := range p.Dependencies[0].Interface {
				p.Dependencies[0].Interface[j] = 8
			}
		}
		encoded, err := packagecache.Encode(p)
		if err != nil {
			return err
		}
		key, err := packagecache.Compute(p)
		if err != nil {
			return err
		}
		binary.Write(&output, binary.BigEndian, uint32(len(encoded)))
		output.Write(encoded)
		output.WriteString(key.String())
		rows = append(rows, map[string]any{"case": mode, "bytes": len(encoded), "key": key.String()})
	}
	if err = os.WriteFile(os.Args[2], output.Bytes(), 0600); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(rows)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
