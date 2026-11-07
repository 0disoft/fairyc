// Package inlinebody owns the versioned cross-package inline export class.
package inlinebody

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const Encoding = "fairyc.inline-bodies/v1"
const Prefix = "fyinline1-sha256-"

type Hash [32]byte

func (h Hash) String() string { return Prefix + hex.EncodeToString(h[:]) }

type Entry struct {
	Name string
	Body string
}

func Compute(coordinate packagegraph.CoordinateID, entries []Entry) (Hash, error) {
	if len(entries) == 0 {
		return Hash{}, nil
	}
	if coordinate == (packagegraph.CoordinateID{}) {
		return Hash{}, fmt.Errorf("inline owner has no stable coordinate")
	}
	entries = slices.Clone(entries)
	slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	for i, e := range entries {
		raw, err := hex.DecodeString(strings.TrimPrefix(e.Body, hir.InlineFunctionPrefix))
		if e.Name == "" || !strings.HasPrefix(e.Body, hir.InlineFunctionPrefix) || len(raw) != 32 || err != nil || e.Body != strings.ToLower(e.Body) || i > 0 && e.Name == entries[i-1].Name {
			return Hash{}, fmt.Errorf("invalid or duplicate inline export")
		}
	}
	data, err := json.Marshal(struct {
		Version string
		Owner   string
		Entries []Entry
	}{Encoding, coordinate.String(), entries})
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(data), nil
}

func HIR(value hir.Package, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]Hash, error) {
	entries := map[hir.PackageID][]Entry{}
	for _, m := range value.Modules {
		resolve := hir.InlineModuleResolver(m)
		for _, fn := range m.Functions {
			if fn.InlineBody != "" {
				if _, ok := coordinates[m.Package]; !ok {
					return nil, fmt.Errorf("inline owner has no coordinate")
				}
				id, err := hir.InlineFunctionID(fn, resolve)
				if err != nil || id != fn.InlineBody {
					return nil, fmt.Errorf("inline body identity mismatch for %s::%s", m.Path, fn.Name)
				}
				entries[m.Package] = append(entries[m.Package], Entry{m.Path + "::" + fn.Name, id})
			}
		}
	}
	result := make(map[hir.PackageID]Hash, len(coordinates))
	for id, c := range coordinates {
		hash, err := Compute(c, entries[id])
		if err != nil {
			return nil, err
		}
		result[id] = hash
	}
	return result, nil
}

// VerifyProgram binds a reassembled cached graph to the freshly checked export
// class. Missing, extra or changed markers cause a safe cache fallback.
func VerifyProgram(live hir.Package, program mir.Program) error {
	expected := map[string]string{}
	for _, m := range live.Modules {
		for _, fn := range m.Functions {
			if fn.InlineBody != "" {
				expected[hir.QualifiedModulePath(m.Package, m.Path)+"::"+fn.Name] = fn.InlineBody
			}
		}
	}
	for _, fn := range program.Functions {
		want := expected[fn.Name]
		if fn.InlineBody != want {
			return fmt.Errorf("cached inline body mismatch for %s", fn.Name)
		}
		delete(expected, fn.Name)
	}
	if len(expected) != 0 {
		return fmt.Errorf("cached inline exports are missing")
	}
	return nil
}
