package atlasmap

import (
	"fmt"
	"math"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type transportEnvelope struct {
	Schema            string            `json:"schema"`
	Encoding          string            `json:"encoding"`
	ChecksumAlgorithm string            `json:"checksum_algorithm"`
	Checksum          string            `json:"checksum"`
	Map               transportDocument `json:"map"`
}

type transportDocument struct {
	Specializations []transportSpecialization  `json:"specializations,omitempty"`
	Header          transportHeader            `json:"header"`
	Files           []transportFile            `json:"files"`
	Modules         []transportModule          `json:"modules"`
	Symbols         []transportSymbol          `json:"symbols"`
	Declarations    []transportDeclaration     `json:"declarations"`
	Functions       []transportFunctionSummary `json:"functions"`
	Completeness    []transportCompleteness    `json:"completeness"`
}

type transportCounts struct {
	Specializations uint64 `json:"specializations,omitempty"`
	Files           uint64 `json:"files"`
	Modules         uint64 `json:"modules"`
	Symbols         uint64 `json:"symbols"`
	Functions       uint64 `json:"functions"`
	Relations       uint64 `json:"relations"`
	Completeness    uint64 `json:"completeness"`
}

type transportHeader struct {
	LogicalSchema              string                      `json:"logical_schema"`
	SnapshotModel              string                      `json:"snapshot_model"`
	CompilerBuild              string                      `json:"compiler_build"`
	CompilerBuildAlgorithm     string                      `json:"compiler_build_algorithm"`
	Language                   string                      `json:"language"`
	PackageUID                 string                      `json:"package_uid"`
	PackageInstance            string                      `json:"package_instance"`
	PackageCoordinate          string                      `json:"package_coordinate"`
	PackageSource              string                      `json:"package_source"`
	PackageSourceKind          string                      `json:"package_source_kind"`
	RuntimeABI                 string                      `json:"runtime_abi"`
	TargetScope                string                      `json:"target_scope"`
	Target                     string                      `json:"target"`
	PackageInstanceAlgorithm   string                      `json:"package_instance_algorithm"`
	PackageCoordinateAlgorithm string                      `json:"package_coordinate_algorithm"`
	PackageSourceAlgorithm     string                      `json:"package_source_algorithm"`
	SourceFileAlgorithm        string                      `json:"source_file_algorithm"`
	LineMapAlgorithm           string                      `json:"line_map_algorithm"`
	SymbolAlgorithm            string                      `json:"symbol_algorithm"`
	InterfaceAlgorithm         string                      `json:"interface_algorithm"`
	InterfaceHash              string                      `json:"interface_hash"`
	Features                   []string                    `json:"features"`
	Dependencies               []string                    `json:"dependencies"`
	PackageReferences          []transportPackageReference `json:"package_references"`
	GenerationStatus           string                      `json:"generation_status"`
	Omissions                  []string                    `json:"omissions"`
	Counts                     transportCounts             `json:"counts"`
}

type transportPackageReference struct {
	UID        string `json:"uid"`
	Instance   string `json:"instance"`
	Coordinate string `json:"coordinate"`
}

type transportFile struct {
	PackageInstance string `json:"package_instance"`
	Path            string `json:"path"`
	Digest          string `json:"digest"`
	LineMap         string `json:"line_map"`
	Kind            string `json:"kind"`
}

type transportModule struct {
	PackageInstance string `json:"package_instance"`
	Path            string `json:"path"`
	Source          string `json:"source"`
}

type transportDeclarationKey struct {
	PackageInstance string `json:"package_instance"`
	Module          string `json:"module"`
	Kind            string `json:"kind"`
	Name            string `json:"name"`
}

type transportSpan struct {
	PackageInstance string `json:"package_instance"`
	Source          string `json:"source"`
	StartByte       uint64 `json:"start_byte"`
	EndByte         uint64 `json:"end_byte"`
}

type transportTypeRef struct {
	Kind              string             `json:"kind"`
	Name              string             `json:"name"`
	PackageCoordinate string             `json:"package_coordinate"`
	Mutable           bool               `json:"mutable"`
	Length            uint64             `json:"length"`
	Arguments         []transportTypeRef `json:"arguments"`
}

type transportParameter struct {
	Name    string           `json:"name"`
	Type    transportTypeRef `json:"type"`
	Mutable bool             `json:"mutable"`
	Span    transportSpan    `json:"span"`
}

type transportField struct {
	Name string           `json:"name"`
	Type transportTypeRef `json:"type"`
	Span transportSpan    `json:"span"`
}

type transportVariant struct {
	Name   string           `json:"name"`
	Span   transportSpan    `json:"span"`
	Fields []transportField `json:"fields"`
}

type transportDeclaration struct {
	TypeParameters []string                `json:"type_parameters,omitempty"`
	Key            transportDeclarationKey `json:"key"`
	Span           transportSpan           `json:"span"`
	Provenance     string                  `json:"provenance"`
	Public         bool                    `json:"public"`
	Unsafe         bool                    `json:"unsafe"`
	Foreign        bool                    `json:"foreign"`
	ABI            string                  `json:"abi"`
	LinkName       string                  `json:"link_name"`
	Type           transportTypeRef        `json:"type"`
	Owner          *transportTypeRef       `json:"owner"`
	Receiver       string                  `json:"receiver"`
	Parameters     []transportParameter    `json:"parameters"`
	Effects        []string                `json:"effects"`
	Fields         []transportField        `json:"fields"`
	Variants       []transportVariant      `json:"variants"`
	ReprC          bool                    `json:"repr_c"`
	DropKind       string                  `json:"drop_kind"`
	DropEffects    []string                `json:"drop_effects"`
}

type transportSymbol struct {
	Declaration transportDeclarationKey `json:"declaration"`
	Visible     string                  `json:"visible"`
	Full        string                  `json:"full"`
}

type transportCall struct {
	Specialization string                  `json:"specialization,omitempty"`
	Span           transportSpan           `json:"span"`
	TargetKind     string                  `json:"target_kind"`
	Target         transportDeclarationKey `json:"target"`
	Certainty      string                  `json:"certainty"`
	ABI            string                  `json:"abi"`
	LinkName       string                  `json:"link_name"`
}

type transportBorrow struct {
	Span    transportSpan `json:"span"`
	Kind    string        `json:"kind"`
	Escapes bool          `json:"escapes"`
}

type transportFunctionSummary struct {
	Function         transportDeclarationKey `json:"function"`
	Provenance       string                  `json:"provenance"`
	Calls            []transportCall         `json:"calls"`
	TrapCategories   []string                `json:"trap_categories"`
	UnsafeSites      []transportSpan         `json:"unsafe_sites"`
	MoveSites        []transportSpan         `json:"move_sites"`
	DropSites        []transportSpan         `json:"drop_sites"`
	AllocationSites  []transportSpan         `json:"allocation_sites"`
	ReturnErrorSites []transportSpan         `json:"return_error_sites"`
	Borrows          []transportBorrow       `json:"borrows"`
}

type transportCompleteness struct {
	Relation string   `json:"relation"`
	Status   string   `json:"status"`
	Resolved uint64   `json:"resolved"`
	External uint64   `json:"external"`
	Unknown  uint64   `json:"unknown"`
	Reasons  []string `json:"reasons"`
}

func documentFromPartition(partition Partition) transportDocument {
	header := partition.Header()
	result := transportDocument{
		Header: transportHeader{
			LogicalSchema: header.LogicalSchema, SnapshotModel: header.SnapshotModel,
			CompilerBuild: header.CompilerBuild, CompilerBuildAlgorithm: header.CompilerBuildAlgorithm,
			Language: header.Language, PackageUID: header.PackageUID, PackageInstance: header.PackageInstance,
			PackageCoordinate: header.PackageCoordinate, PackageSource: header.PackageSource,
			PackageSourceKind: header.PackageSourceKind, RuntimeABI: header.RuntimeABI,
			TargetScope: header.TargetScope, Target: header.Target,
			PackageInstanceAlgorithm:   header.PackageInstanceAlgorithm,
			PackageCoordinateAlgorithm: header.PackageCoordinateAlgorithm,
			PackageSourceAlgorithm:     header.PackageSourceAlgorithm, SourceFileAlgorithm: header.SourceFileAlgorithm,
			LineMapAlgorithm: header.LineMapAlgorithm, SymbolAlgorithm: header.SymbolAlgorithm,
			InterfaceAlgorithm: header.InterfaceAlgorithm, InterfaceHash: header.InterfaceHash,
			Features: nonNilStrings(header.Features), Dependencies: nonNilStrings(header.Dependencies),
			PackageReferences: referencesToTransport(header.PackageReferences),
			GenerationStatus:  header.GenerationStatus, Omissions: nonNilStrings(header.Omissions),
			Counts: transportCounts{
				Specializations: uint64(header.Counts.Specializations),
				Files:           uint64(header.Counts.Files), Modules: uint64(header.Counts.Modules), Symbols: uint64(header.Counts.Symbols),
				Functions: uint64(header.Counts.Functions), Relations: uint64(header.Counts.Relations),
				Completeness: uint64(header.Counts.Completeness),
			},
		},
		Files: make([]transportFile, 0), Modules: make([]transportModule, 0), Symbols: make([]transportSymbol, 0),
		Declarations: make([]transportDeclaration, 0), Functions: make([]transportFunctionSummary, 0),
		Completeness: make([]transportCompleteness, 0),
	}
	for _, value := range partition.Files() {
		result.Files = append(result.Files, transportFile{PackageInstance: value.PackageInstance, Path: value.Path, Digest: value.Digest, LineMap: value.LineMap, Kind: string(value.Kind)})
	}
	for _, value := range partition.Modules() {
		result.Modules = append(result.Modules, transportModule{PackageInstance: value.PackageInstance, Path: value.Path, Source: value.Source})
	}
	for _, value := range partition.Symbols() {
		result.Symbols = append(result.Symbols, transportSymbol{Declaration: keyToTransport(value.Declaration), Visible: value.Visible, Full: value.Full})
	}
	for _, value := range partition.Declarations() {
		result.Declarations = append(result.Declarations, declarationToTransport(value))
	}
	for _, value := range partition.Functions() {
		result.Functions = append(result.Functions, functionToTransport(value))
	}
	for _, value := range partition.Specializations() {
		result.Specializations = append(result.Specializations, specializationToTransport(value))
	}
	for _, value := range partition.Completeness() {
		result.Completeness = append(result.Completeness, transportCompleteness{
			Relation: value.Relation, Status: string(value.Status), Resolved: uint64(value.Resolved),
			External: uint64(value.External), Unknown: uint64(value.Unknown), Reasons: nonNilStrings(value.Reasons),
		})
	}
	return result
}

func keyToTransport(value semanticsnapshot.DeclarationKey) transportDeclarationKey {
	return transportDeclarationKey{PackageInstance: value.PackageInstance, Module: value.Module, Kind: string(value.Kind), Name: value.Name}
}

func spanToTransport(value semanticsnapshot.Span) transportSpan {
	return transportSpan{PackageInstance: value.PackageInstance, Source: value.Source, StartByte: uint64(value.StartByte), EndByte: uint64(value.EndByte)}
}

func typeToTransport(value semanticsnapshot.TypeRef) transportTypeRef {
	result := transportTypeRef{Kind: string(value.Kind), Name: value.Name, PackageCoordinate: value.PackageCoordinate, Mutable: value.Mutable, Length: value.Length, Arguments: make([]transportTypeRef, 0, len(value.Arguments))}
	for _, argument := range value.Arguments {
		result.Arguments = append(result.Arguments, typeToTransport(argument))
	}
	return result
}

func fieldToTransport(value semanticsnapshot.Field) transportField {
	return transportField{Name: value.Name, Type: typeToTransport(value.Type), Span: spanToTransport(value.Span)}
}

func declarationToTransport(value semanticsnapshot.Declaration) transportDeclaration {
	result := transportDeclaration{
		TypeParameters: append([]string(nil), value.TypeParameters...),
		Key:            keyToTransport(value.Key), Span: spanToTransport(value.Span), Provenance: string(value.Provenance),
		Public: value.Public, Unsafe: value.Unsafe, Foreign: value.Foreign, ABI: value.ABI, LinkName: value.LinkName,
		Type: typeToTransport(value.Type), Receiver: value.Receiver, Effects: nonNilStrings(value.Effects),
		Parameters: make([]transportParameter, 0, len(value.Parameters)), Fields: make([]transportField, 0, len(value.Fields)),
		Variants: make([]transportVariant, 0, len(value.Variants)), ReprC: value.ReprC, DropKind: value.DropKind,
		DropEffects: nonNilStrings(value.DropEffects),
	}
	if value.Owner != nil {
		owner := typeToTransport(*value.Owner)
		result.Owner = &owner
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, transportParameter{Name: parameter.Name, Type: typeToTransport(parameter.Type), Mutable: parameter.Mutable, Span: spanToTransport(parameter.Span)})
	}
	for _, field := range value.Fields {
		result.Fields = append(result.Fields, fieldToTransport(field))
	}
	for _, variant := range value.Variants {
		item := transportVariant{Name: variant.Name, Span: spanToTransport(variant.Span), Fields: make([]transportField, 0, len(variant.Fields))}
		for _, field := range variant.Fields {
			item.Fields = append(item.Fields, fieldToTransport(field))
		}
		result.Variants = append(result.Variants, item)
	}
	return result
}

func functionToTransport(value semanticsnapshot.FunctionSummary) transportFunctionSummary {
	result := transportFunctionSummary{
		Function: keyToTransport(value.Function), Provenance: string(value.Provenance),
		Calls: make([]transportCall, 0, len(value.Calls)), TrapCategories: nonNilStrings(value.TrapCategories),
		UnsafeSites: spansToTransport(value.UnsafeSites), MoveSites: spansToTransport(value.MoveSites),
		DropSites: spansToTransport(value.DropSites), AllocationSites: spansToTransport(value.AllocationSites),
		ReturnErrorSites: spansToTransport(value.ReturnErrorSites), Borrows: make([]transportBorrow, 0, len(value.Borrows)),
	}
	for _, call := range value.Calls {
		result.Calls = append(result.Calls, transportCall{Specialization: call.Specialization, Span: spanToTransport(call.Span), TargetKind: string(call.TargetKind), Target: keyToTransport(call.Target), Certainty: string(call.Certainty), ABI: call.ABI, LinkName: call.LinkName})
	}
	for _, borrow := range value.Borrows {
		result.Borrows = append(result.Borrows, transportBorrow{Span: spanToTransport(borrow.Span), Kind: borrow.Kind, Escapes: borrow.Escapes})
	}
	return result
}

func spansToTransport(values []semanticsnapshot.Span) []transportSpan {
	result := make([]transportSpan, 0, len(values))
	for _, value := range values {
		result = append(result, spanToTransport(value))
	}
	return result
}

func partitionFromDocument(value transportDocument) (Partition, error) {
	counts, err := countsFromTransport(value.Header.Counts)
	if err != nil {
		return Partition{}, err
	}
	data := partitionData{header: Header{
		LogicalSchema: value.Header.LogicalSchema, SnapshotModel: value.Header.SnapshotModel,
		CompilerBuild: value.Header.CompilerBuild, CompilerBuildAlgorithm: value.Header.CompilerBuildAlgorithm,
		Language: value.Header.Language, PackageUID: value.Header.PackageUID, PackageInstance: value.Header.PackageInstance,
		PackageCoordinate: value.Header.PackageCoordinate, PackageSource: value.Header.PackageSource,
		PackageSourceKind: value.Header.PackageSourceKind, RuntimeABI: value.Header.RuntimeABI,
		TargetScope: value.Header.TargetScope, Target: value.Header.Target,
		PackageInstanceAlgorithm:   value.Header.PackageInstanceAlgorithm,
		PackageCoordinateAlgorithm: value.Header.PackageCoordinateAlgorithm,
		PackageSourceAlgorithm:     value.Header.PackageSourceAlgorithm, SourceFileAlgorithm: value.Header.SourceFileAlgorithm,
		LineMapAlgorithm: value.Header.LineMapAlgorithm, SymbolAlgorithm: value.Header.SymbolAlgorithm,
		InterfaceAlgorithm: value.Header.InterfaceAlgorithm, InterfaceHash: value.Header.InterfaceHash,
		Features: append([]string(nil), value.Header.Features...), Dependencies: append([]string(nil), value.Header.Dependencies...),
		PackageReferences: referencesFromTransport(value.Header.PackageReferences),
		GenerationStatus:  value.Header.GenerationStatus, Omissions: append([]string(nil), value.Header.Omissions...), Counts: counts,
	}}
	for _, file := range value.Files {
		data.files = append(data.files, semanticsnapshot.File{PackageInstance: file.PackageInstance, Path: file.Path, Digest: file.Digest, LineMap: file.LineMap, Kind: semanticsnapshot.FileKind(file.Kind)})
	}
	for _, module := range value.Modules {
		data.modules = append(data.modules, semanticsnapshot.Module{PackageInstance: module.PackageInstance, Path: module.Path, Source: module.Source})
	}
	for _, symbol := range value.Symbols {
		data.symbols = append(data.symbols, Symbol{Declaration: keyFromTransport(symbol.Declaration), Visible: symbol.Visible, Full: symbol.Full})
	}
	for _, declaration := range value.Declarations {
		decoded, decodeErr := declarationFromTransport(declaration)
		if decodeErr != nil {
			return Partition{}, decodeErr
		}
		data.declarations = append(data.declarations, decoded)
	}
	for _, function := range value.Functions {
		decoded, decodeErr := functionFromTransport(function)
		if decodeErr != nil {
			return Partition{}, decodeErr
		}
		data.functions = append(data.functions, decoded)
	}
	for _, item := range value.Completeness {
		resolved, convertErr := checkedInt(item.Resolved, "completeness resolved count")
		if convertErr != nil {
			return Partition{}, convertErr
		}
		external, convertErr := checkedInt(item.External, "completeness external count")
		if convertErr != nil {
			return Partition{}, convertErr
		}
		unknown, convertErr := checkedInt(item.Unknown, "completeness unknown count")
		if convertErr != nil {
			return Partition{}, convertErr
		}
		data.completeness = append(data.completeness, semanticsnapshot.Completeness{Relation: item.Relation, Status: semanticsnapshot.CompletenessStatus(item.Status), Resolved: resolved, External: external, Unknown: unknown, Reasons: append([]string(nil), item.Reasons...)})
	}
	for _, item := range value.Specializations {
		decoded, err := specializationFromTransport(item)
		if err != nil {
			return Partition{}, err
		}
		data.specializations = append(data.specializations, decoded)
	}
	return Partition{data: data}, nil
}

func countsFromTransport(value transportCounts) (RecordCounts, error) {
	fields := []struct {
		name  string
		value uint64
	}{
		{"files", value.Files}, {"modules", value.Modules}, {"symbols", value.Symbols},
		{"functions", value.Functions}, {"relations", value.Relations}, {"completeness", value.Completeness},
		{"specializations", value.Specializations},
	}
	converted := make([]int, len(fields))
	for index, field := range fields {
		item, err := checkedInt(field.value, "header "+field.name+" count")
		if err != nil {
			return RecordCounts{}, err
		}
		converted[index] = item
	}
	return RecordCounts{Files: converted[0], Modules: converted[1], Symbols: converted[2], Functions: converted[3], Relations: converted[4], Completeness: converted[5], Specializations: converted[6]}, nil
}

func declarationFromTransport(value transportDeclaration) (semanticsnapshot.Declaration, error) {
	span, err := spanFromTransport(value.Span)
	if err != nil {
		return semanticsnapshot.Declaration{}, err
	}
	result := semanticsnapshot.Declaration{
		TypeParameters: append([]string(nil), value.TypeParameters...),
		Key:            keyFromTransport(value.Key), Span: span, Provenance: semanticsnapshot.Provenance(value.Provenance),
		Public: value.Public, Unsafe: value.Unsafe, Foreign: value.Foreign, ABI: value.ABI, LinkName: value.LinkName,
		Type: typeFromTransport(value.Type), Receiver: value.Receiver, Effects: append([]string(nil), value.Effects...),
		ReprC: value.ReprC, DropKind: value.DropKind, DropEffects: append([]string(nil), value.DropEffects...),
	}
	if value.Owner != nil {
		owner := typeFromTransport(*value.Owner)
		result.Owner = &owner
	}
	for _, parameter := range value.Parameters {
		parameterSpan, spanErr := spanFromTransport(parameter.Span)
		if spanErr != nil {
			return semanticsnapshot.Declaration{}, spanErr
		}
		result.Parameters = append(result.Parameters, semanticsnapshot.Parameter{Name: parameter.Name, Type: typeFromTransport(parameter.Type), Mutable: parameter.Mutable, Span: parameterSpan})
	}
	for _, field := range value.Fields {
		decoded, fieldErr := fieldFromTransport(field)
		if fieldErr != nil {
			return semanticsnapshot.Declaration{}, fieldErr
		}
		result.Fields = append(result.Fields, decoded)
	}
	for _, variant := range value.Variants {
		variantSpan, spanErr := spanFromTransport(variant.Span)
		if spanErr != nil {
			return semanticsnapshot.Declaration{}, spanErr
		}
		decoded := semanticsnapshot.Variant{Name: variant.Name, Span: variantSpan}
		for _, field := range variant.Fields {
			item, fieldErr := fieldFromTransport(field)
			if fieldErr != nil {
				return semanticsnapshot.Declaration{}, fieldErr
			}
			decoded.Fields = append(decoded.Fields, item)
		}
		result.Variants = append(result.Variants, decoded)
	}
	return result, nil
}

func fieldFromTransport(value transportField) (semanticsnapshot.Field, error) {
	span, err := spanFromTransport(value.Span)
	if err != nil {
		return semanticsnapshot.Field{}, err
	}
	return semanticsnapshot.Field{Name: value.Name, Type: typeFromTransport(value.Type), Span: span}, nil
}

func functionFromTransport(value transportFunctionSummary) (semanticsnapshot.FunctionSummary, error) {
	result := semanticsnapshot.FunctionSummary{Function: keyFromTransport(value.Function), Provenance: semanticsnapshot.Provenance(value.Provenance), TrapCategories: append([]string(nil), value.TrapCategories...)}
	for _, call := range value.Calls {
		span, err := spanFromTransport(call.Span)
		if err != nil {
			return semanticsnapshot.FunctionSummary{}, err
		}
		result.Calls = append(result.Calls, semanticsnapshot.Call{Specialization: call.Specialization, Span: span, TargetKind: semanticsnapshot.CallTargetKind(call.TargetKind), Target: keyFromTransport(call.Target), Certainty: semanticsnapshot.Certainty(call.Certainty), ABI: call.ABI, LinkName: call.LinkName})
	}
	var err error
	if result.UnsafeSites, err = spansFromTransport(value.UnsafeSites); err != nil {
		return semanticsnapshot.FunctionSummary{}, err
	}
	if result.MoveSites, err = spansFromTransport(value.MoveSites); err != nil {
		return semanticsnapshot.FunctionSummary{}, err
	}
	if result.DropSites, err = spansFromTransport(value.DropSites); err != nil {
		return semanticsnapshot.FunctionSummary{}, err
	}
	if result.AllocationSites, err = spansFromTransport(value.AllocationSites); err != nil {
		return semanticsnapshot.FunctionSummary{}, err
	}
	if result.ReturnErrorSites, err = spansFromTransport(value.ReturnErrorSites); err != nil {
		return semanticsnapshot.FunctionSummary{}, err
	}
	for _, borrow := range value.Borrows {
		span, spanErr := spanFromTransport(borrow.Span)
		if spanErr != nil {
			return semanticsnapshot.FunctionSummary{}, spanErr
		}
		result.Borrows = append(result.Borrows, semanticsnapshot.Borrow{Span: span, Kind: borrow.Kind, Escapes: borrow.Escapes})
	}
	return result, nil
}

func keyFromTransport(value transportDeclarationKey) semanticsnapshot.DeclarationKey {
	return semanticsnapshot.DeclarationKey{PackageInstance: value.PackageInstance, Module: value.Module, Kind: semanticsnapshot.DeclarationKind(value.Kind), Name: value.Name}
}

func typeFromTransport(value transportTypeRef) semanticsnapshot.TypeRef {
	result := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeKind(value.Kind), Name: value.Name, PackageCoordinate: value.PackageCoordinate, Mutable: value.Mutable, Length: value.Length}
	for _, argument := range value.Arguments {
		result.Arguments = append(result.Arguments, typeFromTransport(argument))
	}
	return result
}

func spanFromTransport(value transportSpan) (semanticsnapshot.Span, error) {
	start, err := checkedInt(value.StartByte, "span start")
	if err != nil {
		return semanticsnapshot.Span{}, err
	}
	end, err := checkedInt(value.EndByte, "span end")
	if err != nil {
		return semanticsnapshot.Span{}, err
	}
	return semanticsnapshot.Span{PackageInstance: value.PackageInstance, Source: value.Source, StartByte: start, EndByte: end}, nil
}

func spansFromTransport(values []transportSpan) ([]semanticsnapshot.Span, error) {
	result := make([]semanticsnapshot.Span, 0, len(values))
	for _, value := range values {
		span, err := spanFromTransport(value)
		if err != nil {
			return nil, err
		}
		result = append(result, span)
	}
	return result, nil
}

func checkedInt(value uint64, label string) (int, error) {
	if value > uint64(math.MaxInt) {
		return 0, fmt.Errorf("%s exceeds host integer range", label)
	}
	return int(value), nil
}

func nonNilStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func referencesToTransport(values []PackageReference) []transportPackageReference {
	result := make([]transportPackageReference, 0, len(values))
	for _, value := range values {
		result = append(result, transportPackageReference{UID: value.UID, Instance: value.Instance, Coordinate: value.Coordinate})
	}
	return result
}

func referencesFromTransport(values []transportPackageReference) []PackageReference {
	result := make([]PackageReference, 0, len(values))
	for _, value := range values {
		result = append(result, PackageReference{UID: value.UID, Instance: value.Instance, Coordinate: value.Coordinate})
	}
	return result
}
