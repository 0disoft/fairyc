package pecoff

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MSVCLibrary records a searched absolute input separately from extracted members.
// An empty Members list is not evidence that the library was unnecessary to open.
type MSVCLibrary struct {
	Name, Path string
	Members    []string
}
type MSVCLibrarySymbol struct {
	Symbol, Library, Member string
	RVA                     uint32
}
type MSVCDefaultLibraryReport struct {
	ObjectDefaults []string
	Libraries      []MSVCLibrary
	Symbols        []MSVCLibrarySymbol
}

// InspectObjectDefaultLibraries admits only basename DEFAULTLIB directives from
// structurally valid executable objects, not arbitrary linker command lines.
func InspectObjectDefaultLibraries(raw []byte) ([]string, error) {
	r, err := InspectObject(raw)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, s := range r.Sections {
		if s.Name != ".drectve" {
			continue
		}
		for _, field := range strings.Fields(string(raw[uint64(s.RawOffset) : uint64(s.RawOffset)+uint64(s.RawSize)])) {
			lower := strings.ToLower(field)
			if strings.HasPrefix(lower, "/nodefaultlib") {
				return nil, fmt.Errorf("suppressed object defaults need a separate proof")
			}
			const prefix = "/defaultlib:"
			if !strings.HasPrefix(lower, prefix) {
				continue
			}
			value := field[len(prefix):]
			if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
				value = value[1 : len(value)-1]
			}
			name, err := defaultLibraryName(value)
			if err != nil {
				return nil, err
			}
			result = append(result, name)
		}
	}
	return result, nil
}

func defaultLibraryName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty default library")
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
			return "", fmt.Errorf("default library is not a basename: %q", name)
		}
	}
	name = strings.ToLower(name)
	if !strings.HasSuffix(name, ".lib") {
		name += ".lib"
	}
	return name, nil
}

// InspectMSVCDefaultLibraries binds the ordinary /MT console startup to searched
// archive paths, loaded members, owned map symbols and executable PE addresses.
// It does not inspect every archive member or model archive/COMDAT competition.
func InspectMSVCDefaultLibraries(objects [][]byte, image, linkMap, verbose []byte) (MSVCDefaultLibraryReport, error) {
	var result MSVCDefaultLibraryReport
	if len(objects) == 0 || len(objects) > 16 {
		return result, fmt.Errorf("default library proof needs 1..16 objects")
	}
	defaults := map[string]bool{}
	for _, raw := range objects {
		names, err := InspectObjectDefaultLibraries(raw)
		if err != nil {
			return result, err
		}
		for _, name := range names {
			if !defaults[name] {
				defaults[name] = true
				result.ObjectDefaults = append(result.ObjectDefaults, name)
			}
		}
	}
	if !defaults["libcmt.lib"] || !defaults["oldnames.lib"] {
		return result, fmt.Errorf("ordinary /MT defaults are missing")
	}
	libraries, err := inspectMSVCLibraryTrace(verbose)
	if err != nil {
		return result, err
	}
	result.Libraries = libraries
	byName := map[string]MSVCLibrary{}
	for _, lib := range libraries {
		byName[lib.Name] = lib
	}
	for name := range defaults {
		if _, ok := byName[name]; !ok {
			return result, fmt.Errorf("object default %s has no searched input", name)
		}
	}
	pe, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	anchors := []MSVCLibrarySymbol{{Symbol: "mainCRTStartup", Library: "libcmt.lib", Member: "exe_main.obj"}, {Symbol: "__acrt_initialize", Library: "libucrt.lib", Member: "initialization.obj"}, {Symbol: "__vcrt_initialize", Library: "libvcruntime.lib", Member: "initialization.obj"}}
	mapped, err := inspectMSVCMapSymbols(linkMap, "mainCRTStartup", "__acrt_initialize", "__vcrt_initialize")
	if err != nil {
		return result, err
	}
	for _, anchor := range anchors {
		lib, ok := byName[anchor.Library]
		if !ok {
			return result, fmt.Errorf("missing CRT archive %s", anchor.Library)
		}
		loaded := false
		for _, member := range lib.Members {
			if strings.EqualFold(member, anchor.Member) {
				loaded = true
			}
		}
		if !loaded {
			return result, fmt.Errorf("missing loaded CRT member %s(%s)", anchor.Library, anchor.Member)
		}
		symbol, ok := mapped[anchor.Symbol]
		want := strings.TrimSuffix(anchor.Library, ".lib") + ":" + anchor.Member
		if !ok || !strings.EqualFold(symbol.Owner, want) {
			return result, fmt.Errorf("CRT symbol %s has unexpected map owner", anchor.Symbol)
		}
		anchor.RVA, err = preferredAddressRVA(pe, symbol.PreferredAddress, anchor.Symbol)
		if err != nil {
			return result, err
		}
		if err := requireExecutableRVARange(pe, anchor.RVA, anchor.RVA+1); err != nil {
			return result, err
		}
		if _, err := generatedImageBytes(image, pe, anchor.RVA, 1); err != nil {
			return result, err
		}
		if anchor.Symbol == "mainCRTStartup" && anchor.RVA != pe.EntryPointRVA {
			return result, fmt.Errorf("CRT startup does not match PE entry")
		}
		result.Symbols = append(result.Symbols, anchor)
	}
	return result, nil
}

func inspectMSVCLibraryTrace(verbose []byte) ([]MSVCLibrary, error) {
	if len(verbose) == 0 || len(verbose) > 8*1024*1024 || !utf8.Valid(verbose) || bytes.IndexByte(verbose, 0) >= 0 {
		return nil, fmt.Errorf("invalid library trace text")
	}
	var libraries []MSVCLibrary
	indexes := map[string]int{}
	for _, line := range strings.Split(string(verbose), "\n") {
		if len(line) > 64*1024 {
			return nil, fmt.Errorf("library trace line exceeds budget")
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Searching ") && strings.HasSuffix(strings.ToLower(line), ".lib:") {
			path := strings.TrimSuffix(strings.TrimPrefix(line, "Searching "), ":")
			normalized := strings.ReplaceAll(path, `\`, "/")
			at := strings.LastIndexByte(normalized, '/')
			if len(path) < 3 || path[1] != ':' || path[2] != '\\' && path[2] != '/' || at < 0 {
				return nil, fmt.Errorf("library trace input is not absolute")
			}
			name, err := defaultLibraryName(normalized[at+1:])
			if err != nil {
				return nil, err
			}
			if index, ok := indexes[name]; ok {
				if !strings.EqualFold(libraries[index].Path, path) {
					return nil, fmt.Errorf("ambiguous searched library %s", name)
				}
				continue
			}
			if len(libraries) >= 16 {
				return nil, fmt.Errorf("too many searched libraries")
			}
			indexes[name] = len(libraries)
			libraries = append(libraries, MSVCLibrary{Name: name, Path: path})
		} else if strings.HasPrefix(line, "Loaded ") && strings.Contains(strings.ToLower(line), ".lib(") {
			text := strings.TrimPrefix(line, "Loaded ")
			name, member, ok := strings.Cut(text, "(")
			if !ok || !strings.HasSuffix(member, ")") {
				return nil, fmt.Errorf("invalid loaded archive member")
			}
			member = strings.TrimSuffix(member, ")")
			name, err := defaultLibraryName(name)
			if err != nil {
				return nil, err
			}
			index, ok := indexes[name]
			if !ok || member == "" {
				return nil, fmt.Errorf("loaded archive member lacks searched input")
			}
			if len(libraries[index].Members) >= 4096 {
				return nil, fmt.Errorf("loaded member budget exceeded")
			}
			libraries[index].Members = append(libraries[index].Members, member)
		}
	}
	if len(libraries) == 0 {
		return nil, fmt.Errorf("missing searched libraries")
	}
	return libraries, nil
}
