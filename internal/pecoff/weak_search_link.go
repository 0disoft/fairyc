package pecoff

import (
	"bytes"
	"debug/pe"
	"fmt"
	"path/filepath"
	"strings"
)

// AMD64WeakSearchSpec identifies one archive-suppressed and one archive-enabled weak path.
type AMD64WeakSearchSpec struct {
	ObjectName        string
	ArchiveName       string
	NoLibraryMember   string
	LibraryMember     string
	NoLibraryEntry    string
	LibraryEntry      string
	NoLibraryWeak     string
	NoLibrarySentinel string
	LibraryWeak       string
	NoLibraryFallback string
	LibraryFallback   string
}

// AMD64WeakSearchReport binds both weak search characteristics to archive extraction outcomes.
type AMD64WeakSearchReport struct {
	ArchiveMembers int
	NoLibrary      AMD64WeakExternalValue
	Library        AMD64WeakExternalValue
}

// InspectAMD64WeakSearchLink proves NOLIBRARY selects its local fallback even though a matching
// archive symbol exists, while LIBRARY extracts its separately indexed strong definition.
func InspectAMD64WeakSearchLink(object, archive, image, linkMap []byte, spec AMD64WeakSearchSpec) (AMD64WeakSearchReport, error) {
	if err := validateAMD64WeakSearchSpec(spec); err != nil {
		return AMD64WeakSearchReport{}, err
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64WeakSearchReport{}, fmt.Errorf("inspect AMD64 weak-search object: %w", err)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64WeakSearchReport{}, fmt.Errorf("inspect AMD64 weak-search image: %w", err)
	}
	archiveReport, err := InspectArchive(archive)
	if err != nil {
		return AMD64WeakSearchReport{}, fmt.Errorf("inspect AMD64 weak-search archive: %w", err)
	}
	noLibraryMember, err := exactArchiveMember(archiveReport.Members, spec.NoLibraryMember)
	if err != nil {
		return AMD64WeakSearchReport{}, err
	}
	libraryMember, err := exactArchiveMember(archiveReport.Members, spec.LibraryMember)
	if err != nil {
		return AMD64WeakSearchReport{}, err
	}
	if noLibraryMember.HeaderOffset == libraryMember.HeaderOffset {
		return AMD64WeakSearchReport{}, fmt.Errorf("weak-search archive members are not distinct")
	}
	for _, candidate := range []struct {
		name   string
		member ArchiveMember
	}{
		{name: spec.NoLibraryWeak, member: noLibraryMember},
		{name: spec.LibraryWeak, member: libraryMember},
	} {
		index, err := exactArchiveSymbol(archiveReport.Symbols, candidate.name)
		if err != nil {
			return AMD64WeakSearchReport{}, err
		}
		if !strings.EqualFold(index.Member, candidate.member.Name) {
			return AMD64WeakSearchReport{}, fmt.Errorf("weak-search symbol %q indexes member %q, want %q", candidate.name, index.Member, candidate.member.Name)
		}
		if err := requireDefinedExternalNonCOMDATFunction(candidate.member.Object, candidate.name, "weak-search archive candidate"); err != nil {
			return AMD64WeakSearchReport{}, err
		}
	}
	noLibrarySentinelIndex, err := exactArchiveSymbol(archiveReport.Symbols, spec.NoLibrarySentinel)
	if err != nil {
		return AMD64WeakSearchReport{}, err
	}
	if !strings.EqualFold(noLibrarySentinelIndex.Member, noLibraryMember.Name) {
		return AMD64WeakSearchReport{}, fmt.Errorf("weak-search sentinel %q indexes member %q, want %q", spec.NoLibrarySentinel, noLibrarySentinelIndex.Member, noLibraryMember.Name)
	}
	if err := requireDefinedExternalNonCOMDATFunction(noLibraryMember.Object, spec.NoLibrarySentinel, "weak-search archive sentinel"); err != nil {
		return AMD64WeakSearchReport{}, err
	}
	noLibraryStrong, err := exactObjectSymbol(noLibraryMember.Object.SymbolRecords, spec.NoLibraryWeak)
	if err != nil {
		return AMD64WeakSearchReport{}, err
	}
	noLibrarySentinel, err := exactObjectSymbol(noLibraryMember.Object.SymbolRecords, spec.NoLibrarySentinel)
	if err != nil {
		return AMD64WeakSearchReport{}, err
	}
	if noLibraryStrong.SectionNumber <= 0 || noLibraryStrong.SectionNumber != noLibrarySentinel.SectionNumber ||
		noLibraryMember.Object.Sections[noLibraryStrong.SectionNumber-1].Characteristics&sectionLinkCOMDAT != 0 {
		return AMD64WeakSearchReport{}, fmt.Errorf("weak NOLIBRARY candidate and sentinel are not in one indivisible non-COMDAT section")
	}
	objectFile, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64WeakSearchReport{}, fmt.Errorf("reparse AMD64 weak-search object: %w", err)
	}
	defer objectFile.Close()
	mapSymbols, err := inspectMSVCMapSymbols(linkMap,
		spec.NoLibraryEntry, spec.LibraryEntry, spec.NoLibraryFallback, spec.LibraryFallback,
		spec.NoLibraryWeak, spec.NoLibrarySentinel, spec.LibraryWeak,
	)
	if err != nil {
		return AMD64WeakSearchReport{}, err
	}
	if _, retained := mapSymbols[spec.NoLibrarySentinel]; retained {
		return AMD64WeakSearchReport{}, fmt.Errorf("MSVC weak NOLIBRARY unexpectedly extracted archive sentinel %q", spec.NoLibrarySentinel)
	}
	noLibrary, err := inspectAMD64WeakExternalPath(object, objectReport, objectFile, image, imageReport, mapSymbols, spec.ObjectName,
		spec.NoLibraryEntry, spec.NoLibraryWeak, spec.NoLibraryFallback, spec.NoLibraryFallback, coffWeakSearchNoLibrary,
		func(owner string) bool { return msvcMapOwnerIsObject(owner, spec.ObjectName) }, spec.ObjectName, true)
	if err != nil {
		return AMD64WeakSearchReport{}, fmt.Errorf("AMD64 weak NOLIBRARY path: %w", err)
	}
	library, err := inspectAMD64WeakExternalPath(object, objectReport, objectFile, image, imageReport, mapSymbols, spec.ObjectName,
		spec.LibraryEntry, spec.LibraryWeak, spec.LibraryFallback, spec.LibraryWeak, coffWeakSearchLibrary,
		func(owner string) bool {
			return msvcMapOwnerIsArchiveMember(owner, spec.ArchiveName, spec.LibraryMember)
		}, spec.ArchiveName+":"+spec.LibraryMember, false)
	if err != nil {
		return AMD64WeakSearchReport{}, fmt.Errorf("AMD64 weak LIBRARY path: %w", err)
	}
	return AMD64WeakSearchReport{ArchiveMembers: len(archiveReport.Members), NoLibrary: noLibrary, Library: library}, nil
}

func validateAMD64WeakSearchSpec(spec AMD64WeakSearchSpec) error {
	names := []string{spec.NoLibraryEntry, spec.LibraryEntry, spec.NoLibraryWeak, spec.NoLibrarySentinel, spec.LibraryWeak, spec.NoLibraryFallback, spec.LibraryFallback}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" || strings.ContainsAny(name, " \t\r\n") {
			return fmt.Errorf("AMD64 weak-search spec has invalid symbol identities")
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("AMD64 weak-search spec repeats symbol %q", name)
		}
		seen[name] = struct{}{}
	}
	for _, name := range []string{spec.ObjectName, spec.NoLibraryMember, spec.LibraryMember} {
		if filepath.Base(name) != name || !strings.EqualFold(filepath.Ext(name), ".obj") {
			return fmt.Errorf("AMD64 weak-search spec has invalid object identity")
		}
	}
	if filepath.Base(spec.ArchiveName) != spec.ArchiveName || !strings.EqualFold(filepath.Ext(spec.ArchiveName), ".lib") ||
		strings.EqualFold(spec.ObjectName, spec.NoLibraryMember) || strings.EqualFold(spec.ObjectName, spec.LibraryMember) ||
		strings.EqualFold(spec.NoLibraryMember, spec.LibraryMember) {
		return fmt.Errorf("AMD64 weak-search spec has invalid or repeated archive identities")
	}
	return nil
}

func msvcMapOwnerIsArchiveMember(owner, archive, member string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(owner), "\\", "/"))
	if separator := strings.LastIndexByte(normalized, '/'); separator >= 0 {
		normalized = normalized[separator+1:]
	}
	archive = strings.ToLower(archive)
	archiveStem := strings.TrimSuffix(archive, ".lib")
	member = strings.ToLower(member)
	return normalized == archive+":"+member || normalized == archiveStem+":"+member ||
		normalized == archive+"("+member+")" || normalized == archiveStem+"("+member+")"
}
