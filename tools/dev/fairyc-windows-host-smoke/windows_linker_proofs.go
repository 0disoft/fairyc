package main

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/windowsarchiveprobe"
	"github.com/0disoft/fairyc/internal/windowsrelocationprobe"
	"github.com/0disoft/fairyc/internal/windowsweaksearchprobe"
)

func runWindowsRelocationProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	unit := windowsrelocationprobe.Generate()
	variantObject, err := pecoff.BuildAMD64REL32VariantProbeObject(unit.VariantSource, unit.VariantTarget)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 REL32 variants object: %w", err)
	}
	weakObject, err := pecoff.BuildAMD64WeakExternalProbeObject(
		unit.WeakFallbackEntry, unit.WeakOverrideEntry, unit.WeakMissing, unit.WeakOverride,
		unit.WeakMissingFallback, unit.WeakOverrideFallback,
	)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 weak-external object: %w", err)
	}
	overflowObject, err := pecoff.BuildAMD64RelocationOverflowProbeObject(unit.OverflowSource)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 relocation-overflow object: %w", err)
	}
	bigObject, err := pecoff.BuildAMD64BigObjectProbe(unit.BigObjectSource)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 bigobj: %w", err)
	}
	rel32Spec := pecoff.AMD64REL32LinkSpec{
		SourceSymbol: unit.SourceSymbol, TargetSymbol: unit.TargetSymbol,
		ReceiptSection: unit.ReceiptSection, ReceiptMagic: unit.ReceiptMagic,
	}
	addressSpec := pecoff.AMD64AddressLinkSpec{
		SourceSymbol: unit.SourceSymbol, TargetSymbol: unit.TargetSymbol, ReceiptSymbol: unit.ReceiptSymbol,
		SourceObject: unit.SourceObject, TargetObject: unit.TargetObject,
		ReceiptSection: unit.ReceiptSection, ReceiptMagic: unit.ReceiptMagic,
	}
	variantSpec := pecoff.AMD64REL32VariantsSpec{
		SourceSymbol: unit.VariantSource, TargetSymbol: unit.VariantTarget,
		SourceObject: unit.VariantObject, TargetObject: unit.TargetObject,
	}
	weakSpec := pecoff.AMD64WeakExternalSpec{
		ObjectName: unit.WeakObject, TargetObject: unit.TargetObject,
		FallbackEntry: unit.WeakFallbackEntry, OverrideEntry: unit.WeakOverrideEntry,
		MissingWeak: unit.WeakMissing, OverrideWeak: unit.WeakOverride,
		MissingFallback: unit.WeakMissingFallback, OverrideFallback: unit.WeakOverrideFallback,
	}
	overflowSpec := pecoff.AMD64RelocationOverflowSpec{SourceSymbol: unit.OverflowSource, SourceObject: unit.OverflowObject}
	bigObjectSpec := pecoff.AMD64BigObjectSpec{SourceSymbol: unit.BigObjectSource, SourceObject: unit.BigObject}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCRelocationProbe(toolchain, profile, unit.CallerSource, unit.TargetSource, variantObject, weakObject, overflowObject, bigObject)
		if err != nil {
			return executions, fmt.Errorf("build Windows MSVC REL32 probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 6 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "target.obj" ||
			artifact.Objects[2].Name != unit.VariantObject || artifact.Objects[3].Name != unit.WeakObject ||
			artifact.Objects[4].Name != unit.OverflowObject || artifact.Objects[5].Name != unit.BigObject {
			return executions, fmt.Errorf("Windows MSVC REL32 probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		callerCodeView, err := pecoff.InspectCodeViewRelocations(artifact.Objects[0].Bytes)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC caller CodeView relocations with %s: %w", profile.Name, err)
		}
		if !hasCodeViewRelocationPair(callerCodeView, unit.SourceSymbol) {
			return executions, fmt.Errorf("Windows MSVC caller CodeView relocations with %s omit %q; pairs=%+v", profile.Name, unit.SourceSymbol, callerCodeView.Pairs)
		}
		targetCodeView, err := pecoff.InspectCodeViewRelocations(artifact.Objects[1].Bytes)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC target CodeView relocations with %s: %w", profile.Name, err)
		}
		if !hasCodeViewRelocationPair(targetCodeView, unit.TargetSymbol) || !hasCodeViewRelocationPair(targetCodeView, unit.VariantTarget) {
			return executions, fmt.Errorf("Windows MSVC target CodeView relocations with %s omit exact target symbols; pairs=%+v", profile.Name, targetCodeView.Pairs)
		}
		if _, err := pecoff.InspectProgramDatabaseAssociation(artifact.Bytes, artifact.ProgramDatabase); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC PE/PDB association with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 probe unwind data with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64REL32Link(artifact.Objects[0].Bytes, artifact.Bytes, rel32Spec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 object-to-image link with %s: %w", profile.Name, err)
		}
		if report.ObjectSection == "" || report.LinkedDisplacement != report.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC REL32 probe with %s returned an incomplete link receipt", profile.Name)
		}
		symbolReceipt, err := symbolicateWindowsProgram(artifact.Bytes, artifact.ProgramDatabase, report.SourceRVA)
		if err != nil {
			return executions, fmt.Errorf("symbolicate Windows MSVC REL32 source with %s: %w", profile.Name, err)
		}
		if err := validateWindowsSymbolication(symbolReceipt, unit.SourceSymbol, "caller.c", unit.SourceLine); err != nil {
			return executions, fmt.Errorf("validate Windows MSVC REL32 symbolication with %s: %w", profile.Name, err)
		}
		addressReport, err := pecoff.InspectAMD64AddressLink(artifact.Objects[0].Bytes, artifact.Bytes, artifact.LinkMap, addressSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC ADDR64 and unwind ADDR32NB object-to-image link with %s: %w", profile.Name, err)
		}
		if addressReport.ADDR64LinkedValue != addressReport.ADDR64ExpectedValue ||
			addressReport.ADDR32NBLinkedBeginRVA != addressReport.ADDR32NBExpectedBeginRVA ||
			addressReport.RuntimeFunctionEndRVA != addressReport.ExpectedFunctionEndRVA ||
			addressReport.UnwindInfoRVA != addressReport.ExpectedUnwindInfoRVA {
			return executions, fmt.Errorf("Windows MSVC address-relocation probe with %s returned an incomplete link receipt", profile.Name)
		}
		variantReport, err := pecoff.InspectAMD64REL32VariantsLink(artifact.Objects[2].Bytes, artifact.Bytes, artifact.LinkMap, variantSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 variants object-to-image link with %s: %w", profile.Name, err)
		}
		if len(variantReport.Variants) != 6 || variantReport.SourceRVA == 0 || variantReport.TargetRVA == 0 {
			return executions, fmt.Errorf("Windows MSVC REL32 variants probe with %s returned an incomplete receipt", profile.Name)
		}
		for _, variant := range variantReport.Variants {
			if variant.LinkedDisplacement != variant.ExpectedDisplacement {
				return executions, fmt.Errorf("Windows MSVC REL32_%d probe with %s returned an incomplete formula receipt", variant.TrailingBytes, profile.Name)
			}
		}
		weakReport, err := pecoff.InspectAMD64WeakExternalLink(artifact.Objects[3].Bytes, artifact.Bytes, artifact.LinkMap, weakSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-external link with %s: %w", profile.Name, err)
		}
		if !weakReport.Fallback.UsedFallback || weakReport.Override.UsedFallback ||
			weakReport.Fallback.LinkedDisplacement != weakReport.Fallback.ExpectedDisplacement ||
			weakReport.Override.LinkedDisplacement != weakReport.Override.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC weak-external probe with %s returned an incomplete receipt", profile.Name)
		}
		overflowReport, err := pecoff.InspectAMD64RelocationOverflowLink(artifact.Objects[4].Bytes, artifact.Bytes, artifact.LinkMap, overflowSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC relocation-overflow link with %s: %w", profile.Name, err)
		}
		if overflowReport.Relocations != 0xffff || overflowReport.SourceRVA == 0 ||
			overflowReport.LinkedDisplacement != overflowReport.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC relocation-overflow probe with %s returned an incomplete receipt", profile.Name)
		}
		bigObjectReport, err := pecoff.InspectAMD64BigObjectLink(artifact.Objects[5].Bytes, artifact.Bytes, artifact.LinkMap, bigObjectSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC bigobj link with %s: %w", profile.Name, err)
		}
		if bigObjectReport.Version != 2 || bigObjectReport.SourceRVA == 0 ||
			bigObjectReport.LinkedDisplacement != bigObjectReport.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC bigobj probe with %s returned an incomplete receipt", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCRelocationProbe(artifact)
		if err != nil {
			return executions, fmt.Errorf("execute Windows MSVC REL32 probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != unit.ExpectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, fmt.Errorf("Windows MSVC REL32 probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, nil
}
func runWindowsWeakSearchProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	unit := windowsweaksearchprobe.Generate()
	weakObject, err := pecoff.BuildAMD64WeakSearchProbeObject(
		unit.NoLibraryEntry, unit.LibraryEntry, unit.NoLibraryWeak, unit.LibraryWeak,
		unit.NoLibraryFallback, unit.LibraryFallback,
	)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 weak-search object: %w", err)
	}
	spec := pecoff.AMD64WeakSearchSpec{
		ObjectName: unit.ObjectName, ArchiveName: unit.ArchiveName,
		NoLibraryMember: unit.NoLibraryMember, LibraryMember: unit.LibraryMember,
		NoLibraryEntry: unit.NoLibraryEntry, LibraryEntry: unit.LibraryEntry,
		NoLibraryWeak: unit.NoLibraryWeak, NoLibrarySentinel: unit.NoLibrarySentinel, LibraryWeak: unit.LibraryWeak,
		NoLibraryFallback: unit.NoLibraryFallback, LibraryFallback: unit.LibraryFallback,
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCWeakSearchProbe(
			toolchain, profile, unit.CallerSource, unit.NoLibrarySource, unit.LibrarySource, weakObject,
		)
		if err != nil {
			return executions, fmt.Errorf("build Windows MSVC weak-search probe with %s: %w", profile.Name, err)
		}
		if artifact.ArchiveName != unit.ArchiveName || len(artifact.Objects) != 4 ||
			artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != unit.NoLibraryMember ||
			artifact.Objects[2].Name != unit.LibraryMember || artifact.Objects[3].Name != unit.ObjectName {
			return executions, fmt.Errorf("Windows MSVC weak-search probe with %s returned unexpected artifact receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-search PE with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-search unwind data with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64WeakSearchLink(artifact.Objects[3].Bytes, artifact.Archive, artifact.Bytes, artifact.LinkMap, spec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-search link with %s: %w", profile.Name, err)
		}
		if report.ArchiveMembers != 2 || !report.NoLibrary.UsedFallback || report.Library.UsedFallback ||
			report.NoLibrary.LinkedDisplacement != report.NoLibrary.ExpectedDisplacement ||
			report.Library.LinkedDisplacement != report.Library.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC weak-search probe with %s returned an incomplete receipt", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCWeakSearchProbe(artifact)
		if err != nil {
			return executions, fmt.Errorf("execute Windows MSVC weak-search probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != unit.ExpectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, fmt.Errorf("Windows MSVC weak-search probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, nil
}
func hasCodeViewRelocationPair(report pecoff.CodeViewRelocationReport, symbol string) bool {
	for _, pair := range report.Pairs {
		if pair.Symbol == symbol && pair.ResolvedSection != 0 {
			return true
		}
	}
	return false
}
func runWindowsArchiveProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	unit := windowsarchiveprobe.Generate()
	spec := pecoff.ArchiveCOMDATLinkSpec{
		ArchiveName: unit.ArchiveName, SelectedMember: unit.SelectedMember,
		UnusedMember: unit.UnusedMember, DirectObject: unit.DirectObject,
		DemandSymbol: unit.DemandSymbol, UnusedSymbol: unit.UnusedSymbol,
		COMDATSymbol: unit.COMDATSymbol, ReceiptSection: unit.ReceiptSection,
		ReceiptMagic: unit.ReceiptMagic, ArchiveCandidate: unit.ArchiveCandidate,
		DirectCandidate: unit.DirectCandidate,
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifacts, err := cdriver.BuildWindowsMSVCArchiveProbe(
			toolchain, profile, unit.CallerSource, unit.ChosenSource, unit.UnusedSource, unit.DirectSource,
		)
		if err != nil {
			return executions, fmt.Errorf("build Windows MSVC archive probe with %s: %w", profile.Name, err)
		}
		if len(artifacts) != 2 || artifacts[0].Variant != "direct-first" || artifacts[1].Variant != "archive-first" {
			return executions, fmt.Errorf("Windows MSVC archive probe with %s returned unexpected input-order variants", profile.Name)
		}
		for _, artifact := range artifacts {
			if artifact.ArchiveName != unit.ArchiveName || len(artifact.Objects) != 4 ||
				artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != unit.SelectedMember ||
				artifact.Objects[2].Name != unit.UnusedMember || artifact.Objects[3].Name != unit.DirectObject {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned unexpected artifact receipts", profile.Name, artifact.Variant)
			}
			if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC archive probe PE with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC archive probe unwind data with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			report, err := pecoff.InspectArchiveCOMDATLink(
				artifact.Archive, artifact.Objects[3].Bytes, artifact.Bytes, artifact.LinkMap, spec,
			)
			if err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC archive and COMDAT selection with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if report.ArchiveMembers != 2 || report.ExtractedMember != unit.SelectedMember || report.DemandOwner != "archive" ||
				(report.COMDATOwner != "archive" && report.COMDATOwner != "direct") || report.TargetRVA == 0 || report.COMDATRVA == 0 {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned an incomplete selection receipt", profile.Name, artifact.Variant)
			}
			association, err := pecoff.InspectAssociativeCOMDATUnwindLink(
				artifact.Objects[1].Bytes, artifact.Bytes, artifact.LinkMap, pecoff.AssociativeCOMDATUnwindSpec{
					ArchiveName: unit.ArchiveName, SourceMember: unit.SelectedMember, SourceSymbol: unit.DemandSymbol,
				},
			)
			if err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC associative COMDAT unwind selection with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if association.ParentSectionIndex <= 0 || association.PDataSectionIndex <= 0 || association.XDataSectionIndex <= 0 ||
				association.FinalEndRVA <= association.FinalBeginRVA || association.FinalUnwindRVA == 0 {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned an incomplete associative COMDAT receipt", profile.Name, artifact.Variant)
			}
			discardedAssociation, err := pecoff.InspectDiscardedAssociativeCOMDATLink(
				artifact.Objects[1].Bytes, artifact.Objects[3].Bytes, artifact.Bytes, artifact.LinkMap,
				pecoff.DiscardedAssociativeCOMDATSpec{
					ArchiveName: unit.ArchiveName, ArchiveMember: unit.SelectedMember,
					DirectObject: unit.DirectObject, SourceSymbol: unit.DiscardSymbol,
				},
			)
			if err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC discarded associative COMDAT with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if discardedAssociation.SelectedOwner == discardedAssociation.DiscardedOwner ||
				discardedAssociation.SelectedBeginRVA == 0 || discardedAssociation.SelectedEndRVA <= discardedAssociation.SelectedBeginRVA ||
				discardedAssociation.SelectedUnwindRVA == 0 || discardedAssociation.DiscardedPayloadMatches != 0 || discardedAssociation.DiscardedRawMatches != 0 || discardedAssociation.OrphanPDataBytes != 0 ||
				discardedAssociation.SelectedUnwindDigest == discardedAssociation.DiscardedUnwindDigest {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned an incomplete discarded associative COMDAT receipt", profile.Name, artifact.Variant)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCArchiveProbe(artifact)
			if err != nil {
				return executions, fmt.Errorf("execute Windows MSVC archive probe with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if outcome.ExitCode != unit.ExpectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
				return executions, fmt.Errorf("Windows MSVC archive probe with %s/%s returned %d, stdout %q, and stderr %q", profile.Name, artifact.Variant, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
			}
			executions++
		}
	}
	return executions, nil
}
