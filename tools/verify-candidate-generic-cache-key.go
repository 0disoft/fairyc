// Existing Go ComputeGeneric is the hash oracle; the reconstructed envelope
// must first hash to that independently computed key before becoming a fixture.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packagecache"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func run() error {
	if len(os.Args) != 3 || os.Args[1] != "--output" {
		return fmt.Errorf("usage: verify-candidate-generic-cache-key.go --output <fixture>")
	}
	source, err := os.ReadFile("examples/lexer_candidate/tests/project_generic_cache_key.fy")
	if err != nil {
		return err
	}
	read := func(name string) (string, error) {
		found := regexp.MustCompile(`fn ref u8\[\] ` + name + `\(\)\{return b("(?:\\.|[^"\\])*");\}`).FindSubmatch(source)
		if len(found) != 2 {
			return "", fmt.Errorf("missing authored %s", name)
		}
		return strconv.Unquote(string(found[1]))
	}
	env, err := read("env_value")
	if err != nil {
		return err
	}
	changed, err := read("changed_env")
	if err != nil {
		return err
	}
	ids := regexp.MustCompile(`if\(id==([012])\)\{return b("(?:\\.|[^"\\])*");\}`).FindAllSubmatch(source, -1)
	if len(ids) != 3 {
		return fmt.Errorf("missing authored specialization IDs")
	}
	var specializations []genericplan.SpecializationID
	for index, row := range ids {
		if string(row[1]) != strconv.Itoa(index) {
			return fmt.Errorf("noncontiguous IDs")
		}
		value, err := strconv.Unquote(string(row[2]))
		if err != nil {
			return err
		}
		specializations = append(specializations, genericplan.SpecializationID(value))
	}
	var fixture bytes.Buffer
	binary.Write(&fixture, binary.BigEndian, uint32(8))
	var rows []map[string]any
	for mode := 0; mode < 8; mode++ {
		b := packagecache.Inputs{CompilerBuild: env, LanguageRevision: env, MIRVersion: env, TargetSchema: env, ABIProfile: env, ToolchainFingerprint: env, Backend: env, RuntimeABI: env, ProfileFlags: []string{"-O2"}}
		if mode == 7 {
			b.CompilerBuild = changed
		}
		facts := [][]byte{b.StandardLibrary[:], b.Package[:], b.Source[:], b.CurrentInterface[:], b.CurrentBody[:]}
		for i, f := range facts {
			seed := byte(i + 1)
			if mode == 6 && i == 4 {
				seed = 6
			}
			for j := range f {
				f[j] = seed
			}
		}
		if mode != 5 {
			b.Dependencies = make([]packagecache.Dependency, 1)
			for j := 0; j < 32; j++ {
				b.Dependencies[0].Coordinate[j] = 1
				b.Dependencies[0].Interface[j] = 2
			}
		}
		p := packagecache.GenericInputs{Base: b}
		for j := range p.CurrentInterface {
			p.CurrentInterface[j] = 6
			if mode == 1 {
				p.CurrentInterface[j] = 7
			}
		}
		if mode != 5 {
			p.Dependencies = make([]packagecache.GenericDependency, 1)
			for j := 0; j < 32; j++ {
				p.Dependencies[0].Coordinate[j] = 1
				p.Dependencies[0].Interface[j] = 7
				if mode == 2 {
					p.Dependencies[0].Interface[j] = 8
				}
			}
		}
		if mode != 4 && mode != 5 {
			p.Specializations = []genericplan.SpecializationID{specializations[0], specializations[2]}
			if mode == 3 {
				p.Specializations[0] = specializations[1]
			}
		}
		key, err := packagecache.ComputeGeneric(p)
		if err != nil {
			return err
		}
		base, err := packagecache.Encode(b)
		if err != nil {
			return err
		}
		var encoded bytes.Buffer
		writeString := func(value string) {
			binary.Write(&encoded, binary.BigEndian, uint64(len(value)))
			encoded.WriteString(value)
		}
		writeString("fairyc.generic-package-cache-key/v0")
		binary.Write(&encoded, binary.BigEndian, uint64(len(base)))
		encoded.Write(base)
		encoded.Write(p.CurrentInterface[:])
		binary.Write(&encoded, binary.BigEndian, uint64(len(p.Dependencies)))
		for _, dep := range p.Dependencies {
			encoded.Write(dep.Coordinate[:])
			encoded.Write(dep.Interface[:])
		}
		binary.Write(&encoded, binary.BigEndian, uint64(len(p.Specializations)))
		for _, id := range p.Specializations {
			writeString(string(id))
		}
		if sha256.Sum256(encoded.Bytes()) != [32]byte(key) {
			return fmt.Errorf("reconstructed bytes contradict Go key")
		}
		if !strings.HasPrefix(key.String(), "fycacheg0-sha256-") || len(key.String()) != 81 {
			return fmt.Errorf("unexpected key text")
		}
		binary.Write(&fixture, binary.BigEndian, uint32(encoded.Len()))
		fixture.Write(encoded.Bytes())
		fixture.WriteString(key.String())
		rows = append(rows, map[string]any{"case": mode, "bytes": encoded.Len(), "key": key.String()})
	}
	if err := os.WriteFile(os.Args[2], fixture.Bytes(), 0600); err != nil {
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
