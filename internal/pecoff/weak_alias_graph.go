package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

// InspectCOFFAlternateDirectives admits ordinary AMD64 metadata-only members
// such as CRT initializers.obj. It does not relax executable-object admission.
func InspectCOFFAlternateDirectives(raw []byte) ([]WeakAliasDirective, error) {
	if err := VerifyObjectParserBudget(raw); err != nil {
		return nil, err
	}
	f, err := pe.NewFile(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if f.Machine != machineAMD64 || f.OptionalHeader != nil {
		return nil, fmt.Errorf("alternate directives require ordinary AMD64 COFF")
	}
	var result []WeakAliasDirective
	for _, s := range f.Sections {
		if s.Name != ".drectve" {
			continue
		}
		if s.Size > 1024*1024 || uint64(s.Offset)+uint64(s.Size) > uint64(len(raw)) {
			return nil, fmt.Errorf("invalid directive section range")
		}
		found, err := parseWeakDirectives(string(raw[s.Offset : uint64(s.Offset)+uint64(s.Size)]))
		if err != nil {
			return nil, err
		}
		result = append(result, found...)
	}
	return result, nil
}

func parseWeakDirectives(text string) ([]WeakAliasDirective, error) {
	var result []WeakAliasDirective
	for _, field := range strings.Fields(text) {
		field = strings.Trim(field, "\"")
		const prefix = "/alternatename:"
		if !strings.HasPrefix(strings.ToLower(field), prefix) {
			continue
		}
		name, fallback, ok := strings.Cut(field[len(prefix):], "=")
		if !ok || name == "" || fallback == "" || strings.ContainsAny(name+fallback, "\x00\"") || strings.Contains(fallback, "=") {
			return nil, fmt.Errorf("invalid alternate-name directive")
		}
		result = append(result, WeakAliasDirective{Name: name, Fallback: fallback})
	}
	return result, nil
}

type WeakAliasObject struct {
	Name  string
	Bytes []byte
}
type WeakAliasDirective struct{ Name, Fallback string }
type WeakAliasInventory struct {
	WeakExternals []ObjectSymbol
	Directives    []WeakAliasDirective
}

// InspectWeakAliases distinguishes COFF class-105 records from linker alias
// directives. A directive is evidence of an offered alternative, not selection.
func InspectWeakAliases(raw []byte) (WeakAliasInventory, error) {
	var result WeakAliasInventory
	r, err := InspectObject(raw)
	if err != nil {
		return result, err
	}
	for _, s := range r.SymbolRecords {
		if s.StorageClass == coffStorageClassWeakExternal {
			result.WeakExternals = append(result.WeakExternals, s)
		}
	}
	for _, s := range r.Sections {
		if s.Name != ".drectve" {
			continue
		}
		text := string(raw[uint64(s.RawOffset) : uint64(s.RawOffset)+uint64(s.RawSize)])
		found, err := parseWeakDirectives(text)
		if err != nil {
			return result, err
		}
		result.Directives = append(result.Directives, found...)
	}
	return result, nil
}

type DirectWeakAliasPlan struct {
	Path            []string
	Selected, Owner string
}

// PlanDirectWeakAlias models only explicitly supplied objects: a strong symbol
// wins before fallback, ALIAS/NOLIBRARY chains terminate or fail on a cycle.
// Archive search, COMDAT competition and absent external definitions are not
// guessed. LIBRARY mode needs the separate archive-selection verifier.
func PlanDirectWeakAlias(inputs []WeakAliasObject, rootObject, root string) (DirectWeakAliasPlan, error) {
	var result DirectWeakAliasPlan
	if len(inputs) == 0 || len(inputs) > 16 {
		return result, fmt.Errorf("weak alias proof requires 1..16 explicit objects")
	}
	edges := map[string]string{}
	definitions := map[string]string{}
	objects := map[string]bool{}
	rootFound := false
	for _, input := range inputs {
		if input.Name == "" || strings.ContainsAny(input.Name, "/\\:") || filepath.Base(input.Name) != input.Name || filepath.Ext(input.Name) != ".obj" || objects[input.Name] {
			return result, fmt.Errorf("invalid or duplicate weak object identity")
		}
		objects[input.Name] = true
		r, err := InspectObject(input.Bytes)
		if err != nil {
			return result, err
		}
		inventory, err := InspectWeakAliases(input.Bytes)
		if err != nil {
			return result, err
		}
		for _, s := range r.SymbolRecords {
			if s.StorageClass == coffStorageClassExternal && s.SectionNumber > 0 {
				if _, exists := definitions[s.Name]; exists {
					return result, fmt.Errorf("ambiguous strong definition %q", s.Name)
				}
				definitions[s.Name] = input.Name
			}
		}
		add := func(name, fallback string) error {
			if previous, ok := edges[name]; ok && previous != fallback {
				return fmt.Errorf("contradictory weak alternatives for %q", name)
			}
			edges[name] = fallback
			if input.Name == rootObject && name == root {
				rootFound = true
			}
			return nil
		}
		for _, s := range inventory.WeakExternals {
			if s.WeakSearch == ObjectWeakSearchLibrary {
				return result, fmt.Errorf("LIBRARY weak mode requires archive evidence")
			}
			if err := add(s.Name, s.WeakFallback); err != nil {
				return result, err
			}
		}
		for _, d := range inventory.Directives {
			if err := add(d.Name, d.Fallback); err != nil {
				return result, err
			}
		}
	}
	if !rootFound {
		return result, fmt.Errorf("missing offered weak root %s:%s", rootObject, root)
	}
	seen := map[string]bool{}
	for name := root; ; {
		if len(result.Path) >= 4096 {
			return result, fmt.Errorf("weak chain exceeds 4096 symbols")
		}
		result.Path = append(result.Path, name)
		if owner, ok := definitions[name]; ok {
			result.Selected = name
			result.Owner = owner
			return result, nil
		}
		if seen[name] {
			return result, fmt.Errorf("unresolved weak alias cycle at %q", name)
		}
		seen[name] = true
		next, ok := edges[name]
		if !ok {
			return result, fmt.Errorf("unresolved weak alias terminal %q", name)
		}
		name = next
	}
}

type DirectWeakAliasLink struct {
	Plan                         DirectWeakAliasPlan
	EntryRVA, SiteRVA, TargetRVA uint32
	Addend, Expected, Actual     int32
	Kind                         uint16
	TargetDataValue              uint32
}

// InspectDirectWeakAliasLink binds an exact function call/jump relocation to the
// explicit-object plan, selected map owner and final displacement bytes.
func InspectDirectWeakAliasLink(inputs []WeakAliasObject, objectName, entryName, root string, image, linkMap []byte) (DirectWeakAliasLink, error) {
	return inspectDirectWeakAliasLink(inputs, objectName, entryName, root, image, linkMap, false)
}

// InspectDirectWeakDataAliasLink additionally checks a four-byte data target;
// RIP-relative data operations may use any REL32 trailing-byte variant.
func InspectDirectWeakDataAliasLink(inputs []WeakAliasObject, objectName, entryName, root string, image, linkMap []byte) (DirectWeakAliasLink, error) {
	return inspectDirectWeakAliasLink(inputs, objectName, entryName, root, image, linkMap, true)
}

func inspectDirectWeakAliasLink(inputs []WeakAliasObject, objectName, entryName, root string, image, linkMap []byte, dataTarget bool) (DirectWeakAliasLink, error) {
	var result DirectWeakAliasLink
	plan, err := PlanDirectWeakAlias(inputs, objectName, root)
	if err != nil {
		return result, err
	}
	result.Plan = plan
	var raw []byte
	for _, input := range inputs {
		if input.Name == objectName {
			raw = input.Bytes
		}
	}
	r, err := InspectObject(raw)
	if err != nil {
		return result, err
	}
	entry, err := exactObjectSymbol(r.SymbolRecords, entryName)
	if err != nil {
		return result, err
	}
	if err := requireDefinedExternalFunction(r, entryName, "weak chain entry"); err != nil {
		return result, err
	}
	var rel ObjectRelocation
	matches := 0
	end := r.Sections[entry.SectionNumber-1].RawSize
	for _, s := range r.SymbolRecords {
		if s.SectionNumber == entry.SectionNumber && s.Type == coffSymbolTypeFunction && s.Value > entry.Value && s.Value < end {
			end = s.Value
		}
	}
	for _, candidate := range r.RelocationRecords {
		if candidate.SectionIndex == int(entry.SectionNumber) && candidate.VirtualAddress > entry.Value && uint64(candidate.VirtualAddress)+4 <= uint64(end) && candidate.Symbol == root && candidate.Type >= ObjectRelocationAMD64Rel32 && candidate.Type <= ObjectRelocationAMD64Rel32_5 {
			rel = candidate
			matches++
		}
	}
	if matches != 1 {
		return result, fmt.Errorf("weak chain lacks exact entry relocation")
	}
	s := r.Sections[rel.SectionIndex-1]
	at := uint64(s.RawOffset) + uint64(rel.VirtualAddress)
	if uint64(rel.VirtualAddress)+4 > uint64(s.RawSize) || rel.VirtualAddress == 0 || !dataTarget && (rel.Type != ObjectRelocationAMD64Rel32 || raw[at-1] != 0xe9 && raw[at-1] != 0xe8) {
		return result, fmt.Errorf("weak chain site is not a direct call/jump")
	}
	pe, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	mapped, err := inspectMSVCMapSymbols(linkMap, entryName, plan.Selected)
	if err != nil {
		return result, err
	}
	start, ok := mapped[entryName]
	if !ok || !msvcMapOwnerIsObject(start.Owner, objectName) {
		return result, fmt.Errorf("weak chain source owner mismatch")
	}
	target, ok := mapped[plan.Selected]
	if !ok || !msvcMapOwnerIsObject(target.Owner, plan.Owner) {
		return result, fmt.Errorf("weak chain selected owner mismatch")
	}
	result.EntryRVA, err = preferredAddressRVA(pe, start.PreferredAddress, "weak chain entry")
	if err != nil {
		return result, err
	}
	result.TargetRVA, err = preferredAddressRVA(pe, target.PreferredAddress, "weak chain target")
	if err != nil {
		return result, err
	}
	if dataTarget {
		data, err := generatedImageBytes(image, pe, result.TargetRVA, 4)
		if err != nil {
			return result, err
		}
		result.TargetDataValue = binary.LittleEndian.Uint32(data)
	} else if err := requireExecutableRVARange(pe, result.TargetRVA, result.TargetRVA+1); err != nil {
		return result, err
	}
	if uint64(result.EntryRVA)+uint64(rel.VirtualAddress-entry.Value)+4 >= uint64(pe.SizeOfImage) {
		return result, fmt.Errorf("weak entry outside image")
	}
	result.SiteRVA = result.EntryRVA + rel.VirtualAddress - entry.Value
	if err := requireExecutableRVARange(pe, result.EntryRVA, result.SiteRVA+4); err != nil {
		return result, err
	}
	data, err := generatedImageBytes(image, pe, result.SiteRVA-1, 5)
	if err != nil {
		return result, err
	}
	if !dataTarget && data[0] != raw[at-1] {
		return result, fmt.Errorf("linked weak entry opcode mismatch")
	}
	result.Addend = int32(binary.LittleEndian.Uint32(raw[at:]))
	result.Actual = int32(binary.LittleEndian.Uint32(data[1:]))
	result.Kind = rel.Type
	expected, err := generatedRelocationExpected(rel.Type, pe.ImageBase, result.TargetRVA, result.SiteRVA, uint64(uint32(result.Addend)))
	result.Expected = int32(uint32(expected))
	if err != nil {
		return result, err
	}
	if result.Expected != result.Actual {
		return result, fmt.Errorf("weak chain final displacement mismatch")
	}
	return result, nil
}

// BuildAMD64WeakChainProbeObject keeps the existing leaf code but connects the
// two real COFF weak records as A -> B -> fallback, or A -> B -> A.
func BuildAMD64WeakChainProbeObject(cycle bool) ([]byte, error) {
	raw, err := BuildAMD64WeakExternalProbeObject("fy_ca", "fy_cb", "fy_wa", "fy_wb", "fy_fa", "fy_fb")
	if err != nil {
		return nil, err
	}
	table := binary.LittleEndian.Uint32(raw[8:])
	binary.LittleEndian.PutUint32(raw[table+3*18:], 4)
	next := uint32(6)
	if cycle {
		next = 2
	}
	binary.LittleEndian.PutUint32(raw[table+5*18:], next)
	if _, err := InspectObject(raw); err != nil {
		return nil, err
	}
	return raw, nil
}
