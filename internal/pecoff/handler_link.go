package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// AMD64HandlerLinkSpec identifies one handler-bearing function and its exact runtime handler.
type AMD64HandlerLinkSpec struct {
	SourceSymbol  string
	SourceObject  string
	HandlerSymbol string
}

// AMD64HandlerLinkReport records the object and final-image handler relocation formula.
type AMD64HandlerLinkReport struct {
	SourceRVA           uint32
	UnwindInfoRVA       uint32
	ObjectFlags         uint8
	ObjectHandlerAddend uint32
	LinkedHandlerRVA    uint32
	ExpectedHandlerRVA  uint32
	HandlerMapOwner     string
	UnwindPrefixBytes   int
}

// InspectAMD64HandlerLink binds one function's .pdata entry to its handler-bearing .xdata record,
// then proves the handler ADDR32NB relocation as S + A - ImageBase in the final image.
func InspectAMD64HandlerLink(object, image, linkMap []byte, spec AMD64HandlerLinkSpec) (AMD64HandlerLinkReport, error) {
	if err := validateAMD64HandlerLinkSpec(spec); err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("inspect AMD64 handler object: %w", err)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("inspect AMD64 handler image: %w", err)
	}
	if _, err := InspectAMD64Unwind(image); err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("inspect AMD64 handler unwind image: %w", err)
	}
	source, err := exactObjectSymbol(objectReport.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	if err := requireDefinedExternalFunction(objectReport, spec.SourceSymbol, "handler source"); err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	objectFile, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("reparse AMD64 handler object: %w", err)
	}
	defer objectFile.Close()
	pdata, err := exactPDataEntryRelocations(objectReport, objectFile, source)
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	beginOffset, err := objectRelocationRawOffset(object, pdata.Section, pdata.Begin, 4)
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler .pdata begin: %w", err)
	}
	unwindOffset, err := objectRelocationRawOffset(object, pdata.Section, pdata.Unwind, 4)
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler .pdata unwind: %w", err)
	}
	beginAddend := binary.LittleEndian.Uint32(object[beginOffset:])
	unwindAddend := binary.LittleEndian.Uint32(object[unwindOffset:])
	xdata := objectFile.Sections[pdata.UnwindTarget.SectionNumber-1]
	recordOffset := uint64(pdata.UnwindTarget.Value) + uint64(unwindAddend)
	if recordOffset+unwindHeaderBytes > uint64(xdata.Size) || uint64(xdata.Offset)+recordOffset+unwindHeaderBytes > uint64(len(object)) {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler object unwind record is out of range")
	}
	objectRecord := uint64(xdata.Offset) + recordOffset
	versionAndFlags := object[objectRecord]
	version := versionAndFlags & 0x07
	flags := versionAndFlags >> 3
	if version != 1 && version != 2 || flags&(unwindFlagExceptionHandler|unwindFlagUnwindHandler) == 0 || flags&unwindFlagChainInfo != 0 {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler object has invalid version/flags %d/%#x", version, flags)
	}
	alignedSlots := uint64(object[objectRecord+2])
	if alignedSlots%2 != 0 {
		alignedSlots++
	}
	tailInSection := recordOffset + unwindHeaderBytes + alignedSlots*2
	if tailInSection+4 > uint64(xdata.Size) || uint64(xdata.Offset)+tailInSection+4 > uint64(len(object)) || tailInSection > math.MaxUint32 {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler object tail is out of range")
	}
	handlerRelocation, err := exactPDataSiteRelocation(
		objectReport.RelocationRecords, int(pdata.UnwindTarget.SectionNumber), objectReport.Sections[pdata.UnwindTarget.SectionNumber-1].Name,
		uint64(xdata.VirtualAddress)+tailInSection,
	)
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler ADDR32NB: %w", err)
	}
	if handlerRelocation.Type != ObjectRelocationAMD64Addr32NB {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler tail relocation is type %#x, want ADDR32NB", handlerRelocation.Type)
	}
	handler, err := exactObjectSymbol(objectReport.SymbolRecords, handlerRelocation.Symbol)
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	if handler.TableIndex != handlerRelocation.SymbolIndex {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler relocation symbol index drifted")
	}
	if handler.Name != spec.HandlerSymbol {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler tail targets %q, want %q", handler.Name, spec.HandlerSymbol)
	}
	if handler.SectionNumber != 0 {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler target %q is not undefined", handler.Name)
	}
	if err := requireREL32FunctionSymbol(handler, "handler target"); err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	handlerObjectOffset, err := objectRelocationRawOffset(object, xdata, handlerRelocation, 4)
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler ADDR32NB: %w", err)
	}
	handlerAddend := binary.LittleEndian.Uint32(object[handlerObjectOffset:])

	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.SourceSymbol, pdata.UnwindTarget.Name, spec.HandlerSymbol)
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	sourceMap, found := mapSymbols[spec.SourceSymbol]
	if !found || !msvcMapOwnerIsObject(sourceMap.Owner, spec.SourceObject) {
		return AMD64HandlerLinkReport{}, fmt.Errorf("MSVC handler source owner is %q, want %q", sourceMap.Owner, spec.SourceObject)
	}
	unwindMap, found := mapSymbols[pdata.UnwindTarget.Name]
	if !found || !msvcMapOwnerIsObject(unwindMap.Owner, spec.SourceObject) {
		return AMD64HandlerLinkReport{}, fmt.Errorf("MSVC handler unwind owner is %q, want %q", unwindMap.Owner, spec.SourceObject)
	}
	handlerMap, found := mapSymbols[spec.HandlerSymbol]
	if !found || handlerMap.Owner == "" {
		return AMD64HandlerLinkReport{}, fmt.Errorf("MSVC map has no owned handler target %q", spec.HandlerSymbol)
	}
	sourceRVA, err := preferredAddressRVA(imageReport, sourceMap.PreferredAddress, "handler source")
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	unwindBaseRVA, err := preferredAddressRVA(imageReport, unwindMap.PreferredAddress, "handler unwind")
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	handlerRVA, err := preferredAddressRVA(imageReport, handlerMap.PreferredAddress, "handler target")
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	expectedBegin, err := addRVA(sourceRVA, beginAddend, "AMD64 handler begin")
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	expectedUnwind, err := addRVA(unwindBaseRVA, unwindAddend, "AMD64 handler unwind")
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	expectedHandler, err := addRVA(handlerRVA, handlerAddend, "AMD64 handler target")
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	_, _, linkedUnwind, err := exactRuntimeFunctionByBegin(image, imageReport, expectedBegin)
	if err != nil {
		return AMD64HandlerLinkReport{}, err
	}
	if linkedUnwind != expectedUnwind {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 linked handler unwind RVA is %#x, want %#x", linkedUnwind, expectedUnwind)
	}
	finalRecord, _, err := mapImageRVA(imageReport, linkedUnwind, uint32(unwindHeaderBytes+alignedSlots*2+4))
	if err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("map final AMD64 handler record: %w", err)
	}
	prefixBytes := unwindHeaderBytes + int(alignedSlots)*2
	if !bytes.Equal(object[objectRecord:objectRecord+uint64(prefixBytes)], image[finalRecord:finalRecord+prefixBytes]) {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 handler unwind prefix drifted")
	}
	linkedHandler := binary.LittleEndian.Uint32(image[finalRecord+prefixBytes:])
	if linkedHandler != expectedHandler {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 linked handler RVA is %#x, want %#x", linkedHandler, expectedHandler)
	}
	if linkedHandler == math.MaxUint32 {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 linked handler target range overflows")
	}
	if err := requireExecutableRVARange(imageReport, linkedHandler, linkedHandler+1); err != nil {
		return AMD64HandlerLinkReport{}, fmt.Errorf("AMD64 linked handler target is not executable: %w", err)
	}
	return AMD64HandlerLinkReport{
		SourceRVA: sourceRVA, UnwindInfoRVA: linkedUnwind, ObjectFlags: flags,
		ObjectHandlerAddend: handlerAddend, LinkedHandlerRVA: linkedHandler,
		ExpectedHandlerRVA: expectedHandler, HandlerMapOwner: handlerMap.Owner,
		UnwindPrefixBytes: prefixBytes,
	}, nil
}

func validateAMD64HandlerLinkSpec(spec AMD64HandlerLinkSpec) error {
	if spec.SourceSymbol == "" || spec.HandlerSymbol == "" || spec.SourceSymbol == spec.HandlerSymbol ||
		strings.ContainsAny(spec.SourceSymbol+spec.HandlerSymbol, " \t\r\n") ||
		filepath.Base(spec.SourceObject) != spec.SourceObject || !strings.EqualFold(filepath.Ext(spec.SourceObject), ".obj") {
		return fmt.Errorf("AMD64 handler-link spec has invalid identities")
	}
	return nil
}
