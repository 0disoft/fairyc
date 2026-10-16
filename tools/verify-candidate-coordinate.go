//go:build ignore

// Run from the repository root: bun tools/run-go.ts run tools/verify-candidate-coordinate.go
// Read the authored Fairy inputs; the maintained Go encoder supplies the oracle.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/0disoft/fairyc/internal/packagegraph"
)

type observation struct {
	Case       int    `json:"case"`
	Bytes      int    `json:"bytes"`
	EncodedHex string `json:"encodedHex"`
	ID         string `json:"id"`
}

func run() error {
	source, err := os.ReadFile("examples/lexer_candidate/tests/project_coordinate_identity.fy")
	if err != nil {
		return err
	}
	literal := `("(?:\\.|[^"\\])*")`
	read := func(name string, mode int) (string, error) {
		pattern := `fn ref u8\[\] ` + name + `\(usize mode\)\{[^\n]*?if\(mode==` + strconv.Itoa(mode) + `\)\{return b` + literal + `;\}`
		found := regexp.MustCompile(pattern).FindSubmatch(source)
		if len(found) != 2 {
			return "", fmt.Errorf("missing %s fixture %d", name, mode)
		}
		return strconv.Unquote(string(found[1]))
	}
	var observations []observation
	for mode := 0; mode < 5; mode++ {
		uid, e := read("uid", mode)
		if e != nil {
			return e
		}
		version, e := read("version", mode)
		if e != nil {
			return e
		}
		coordinate := packagegraph.Coordinate{PackageUID: uid, VersionIdentity: version}
		pattern := `if\(mode==` + strconv.Itoa(mode) + ` && i==([0-9]+)\)\{return b` + literal + `;\}`
		for _, found := range regexp.MustCompile(pattern).FindAllSubmatch(source, -1) {
			index, e := strconv.Atoi(string(found[1]))
			if e != nil || index != len(coordinate.Features) {
				return fmt.Errorf("noncontiguous feature index")
			}
			feature, e := strconv.Unquote(string(found[2]))
			if e != nil {
				return e
			}
			coordinate.Features = append(coordinate.Features, feature)
		}
		encoded, e := packagegraph.EncodeCoordinate(coordinate)
		if e != nil {
			return e
		}
		id, e := packagegraph.ComputeCoordinateID(coordinate)
		if e != nil {
			return e
		}
		row := observation{mode, len(encoded), hex.EncodeToString(encoded), id.String()}
		if len(os.Args) == 1 {
			expected, e := read("golden_hex", mode)
			if e != nil {
				return e
			}
			if expected != row.EncodedHex {
				return fmt.Errorf("encoding mismatch in fixture %d", mode)
			}
			expected, e = read("golden_id", mode)
			if e != nil {
				return e
			}
			if expected != row.ID {
				return fmt.Errorf("digest mismatch in fixture %d", mode)
			}
		} else if len(os.Args) != 2 || os.Args[1] != "--goldens" {
			return fmt.Errorf("usage: verify-candidate-coordinate.go [--goldens]")
		}
		observations = append(observations, row)
	}
	return json.NewEncoder(os.Stdout).Encode(observations)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
