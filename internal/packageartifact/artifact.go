package packageartifact

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// Artifact is one independently verified, process-local package MIR slice. Context deliberately
// retains the exact workspace-wide type and foreign-function snapshot for this first boundary; it
// is not a persistent or cross-process format.
type Artifact struct {
	PackageID         hir.PackageID
	Package           packagegraph.InstanceID
	PackageIdentities map[hir.PackageID]packagegraph.InstanceID
	Context           mir.Program
	Functions         []FunctionRecord
	Imports           []FunctionImport
}

// FunctionRecord binds one package-owned body to its package-stable descriptor and identity.
type FunctionRecord struct {
	ID         FunctionID
	Descriptor FunctionDescriptor
	Body       mir.Function
}

// FunctionImport binds one external package function signature to the build-local symbol used by
// this artifact's MIR body and shared type context.
type FunctionImport struct {
	ID         FunctionID
	Descriptor FunctionDescriptor
	Signature  FunctionSignature
}

// FunctionSignature is the body-free MIR signature needed to verify imported calls and resource
// drop-glue links inside the current process.
type FunctionSignature struct {
	Symbol         hir.SymbolID
	Name           string
	Span           hir.Span
	ParameterTypes []hir.Type
	ResultType     hir.Type
	Effects        []hir.Effect
	DropGlueFor    hir.Type
}

// ExtractAll slices every function-owning package from one closed, verified whole program.
func ExtractAll(program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID) ([]Artifact, error) {
	if failures := mir.Verify(program); len(failures) != 0 {
		return nil, fmt.Errorf("verify whole program before package extraction: %w", failures[0])
	}
	packages := make(map[hir.PackageID]struct{})
	for _, function := range program.Functions {
		packages[function.Span.Package] = struct{}{}
	}
	packageIDs := make([]hir.PackageID, 0, len(packages))
	for packageID := range packages {
		packageIDs = append(packageIDs, packageID)
	}
	slices.Sort(packageIDs)
	artifacts := make([]Artifact, 0, len(packageIDs))
	for _, packageID := range packageIDs {
		artifact, err := extractVerified(program, packageID, identities)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

// Extract creates one package-owned function slice with explicit internal imports. The program is
// verified before slicing, and the resulting artifact is independently verified before return.
func Extract(program mir.Program, packageID hir.PackageID, identities map[hir.PackageID]packagegraph.InstanceID) (Artifact, error) {
	if failures := mir.Verify(program); len(failures) != 0 {
		return Artifact{}, fmt.Errorf("verify whole program before package extraction: %w", failures[0])
	}
	return extractVerified(program, packageID, identities)
}

func extractVerified(program mir.Program, packageID hir.PackageID, identities map[hir.PackageID]packagegraph.InstanceID) (Artifact, error) {
	packageIdentity, found := identities[packageID]
	if !found {
		return Artifact{}, fmt.Errorf("package %d has no PackageInstanceId", packageID)
	}

	type functionMetadata struct {
		id         FunctionID
		descriptor FunctionDescriptor
		signature  FunctionSignature
		body       mir.Function
	}
	bySymbol := make(map[hir.SymbolID]functionMetadata, len(program.Functions))
	for _, function := range program.Functions {
		descriptor, err := DescribeFunction(function, identities)
		if err != nil {
			return Artifact{}, err
		}
		id, err := ComputeFunctionID(descriptor)
		if err != nil {
			return Artifact{}, fmt.Errorf("compute FunctionID for %q: %w", function.Name, err)
		}
		if _, duplicate := bySymbol[function.Symbol]; duplicate {
			return Artifact{}, fmt.Errorf("whole program function symbol %d is duplicated", function.Symbol)
		}
		bySymbol[function.Symbol] = functionMetadata{
			id: id, descriptor: cloneDescriptor(descriptor), signature: signatureOf(function), body: function,
		}
	}

	artifact := Artifact{
		PackageID: packageID, Package: packageIdentity,
		PackageIdentities: cloneIdentities(identities), Context: contextOf(program),
	}
	ownedSymbols := make(map[hir.SymbolID]struct{})
	for _, function := range program.Functions {
		if function.Span.Package != packageID {
			continue
		}
		metadata := bySymbol[function.Symbol]
		artifact.Functions = append(artifact.Functions, FunctionRecord{
			ID: metadata.id, Descriptor: cloneDescriptor(metadata.descriptor), Body: cloneFunction(metadata.body),
		})
		ownedSymbols[function.Symbol] = struct{}{}
	}
	if len(artifact.Functions) == 0 {
		return Artifact{}, fmt.Errorf("package %d owns no MIR functions", packageID)
	}

	importSymbols := make(map[hir.SymbolID]struct{})
	for _, record := range artifact.Functions {
		for _, block := range record.Body.Blocks {
			terminator := block.Terminator
			if terminator.Kind != mir.TerminatorCall || terminator.ForeignABI != "" {
				continue
			}
			if _, owned := ownedSymbols[terminator.Callee]; !owned {
				importSymbols[terminator.Callee] = struct{}{}
			}
		}
	}
	// The first process-local boundary carries one shared workspace type context. Every non-empty
	// resource drop link in that context therefore needs an explicit body or import signature.
	for _, resource := range artifact.Context.ResourceTypes {
		if resource.DropSymbol == hir.NoSymbol {
			continue
		}
		if _, owned := ownedSymbols[resource.DropSymbol]; !owned {
			importSymbols[resource.DropSymbol] = struct{}{}
		}
	}
	for symbol := range importSymbols {
		metadata, found := bySymbol[symbol]
		if !found {
			return Artifact{}, fmt.Errorf("package %d import symbol %d has no internal function", packageID, symbol)
		}
		artifact.Imports = append(artifact.Imports, FunctionImport{
			ID: metadata.id, Descriptor: cloneDescriptor(metadata.descriptor), Signature: cloneSignature(metadata.signature),
		})
	}
	sortRecords(artifact.Functions)
	sortImports(artifact.Imports)
	if err := Verify(artifact); err != nil {
		return Artifact{}, fmt.Errorf("verify extracted package %d: %w", packageID, err)
	}
	return artifact, nil
}

// Verify checks stable identities, ownership, explicit bindings, and the full MIR verifier by
// materializing unreachable body stubs for imports. Import stubs can never escape this function.
func Verify(artifact Artifact) error {
	if artifact.Package == (packagegraph.InstanceID{}) {
		return fmt.Errorf("package artifact has an empty PackageInstanceId")
	}
	if identity, found := artifact.PackageIdentities[artifact.PackageID]; !found || identity != artifact.Package {
		return fmt.Errorf("package artifact owner mapping is missing or inconsistent")
	}
	if len(artifact.Context.Functions) != 0 {
		return fmt.Errorf("package artifact context contains executable functions")
	}
	if len(artifact.Functions) == 0 {
		return fmt.Errorf("package artifact has no owned functions")
	}
	if !recordsSorted(artifact.Functions) || !importsSorted(artifact.Imports) {
		return fmt.Errorf("package artifact bindings are not sorted and unique")
	}

	seenSymbols := make(map[hir.SymbolID]string, len(artifact.Functions)+len(artifact.Imports))
	seenIDs := make(map[FunctionID]string, len(artifact.Functions)+len(artifact.Imports))
	program := mir.Clone(artifact.Context)
	for _, record := range artifact.Functions {
		if record.Body.Span.Package != artifact.PackageID {
			return fmt.Errorf("owned function %q belongs to package %d, want %d", record.Body.Name, record.Body.Span.Package, artifact.PackageID)
		}
		if err := verifyBinding(record.ID, record.Descriptor, record.Body, artifact.PackageIdentities); err != nil {
			return err
		}
		if err := recordBinding(seenSymbols, seenIDs, record.Body.Symbol, record.ID, record.Body.Name); err != nil {
			return err
		}
		program.Functions = append(program.Functions, cloneFunction(record.Body))
	}
	for _, imported := range artifact.Imports {
		stub := imported.Signature.stub()
		if err := verifyBinding(imported.ID, imported.Descriptor, stub, artifact.PackageIdentities); err != nil {
			return fmt.Errorf("import %q: %w", imported.Signature.Name, err)
		}
		if err := recordBinding(seenSymbols, seenIDs, imported.Signature.Symbol, imported.ID, imported.Signature.Name); err != nil {
			return err
		}
		program.Functions = append(program.Functions, stub)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return fmt.Errorf("verify package MIR closure: %w", failures[0])
	}
	return nil
}

// Reassemble combines a complete set of compatible process-local package artifacts, allocates a
// fresh collision-free internal function-symbol space, rewrites calls and drop links, and reruns
// the ordinary closed-world MIR verifier.
func Reassemble(artifacts []Artifact) (mir.Program, error) {
	if len(artifacts) == 0 {
		return mir.Program{}, fmt.Errorf("no package artifacts to reassemble")
	}
	for index := range artifacts {
		if err := Verify(artifacts[index]); err != nil {
			return mir.Program{}, fmt.Errorf("artifact %d: %w", index, err)
		}
	}
	base := artifacts[0]
	seenPackages := make(map[packagegraph.InstanceID]struct{}, len(artifacts))
	owned := make(map[FunctionID]FunctionRecord)
	owners := make(map[FunctionID]int)
	for artifactIndex, artifact := range artifacts {
		if !reflect.DeepEqual(artifact.Context, base.Context) || !reflect.DeepEqual(artifact.PackageIdentities, base.PackageIdentities) {
			return mir.Program{}, fmt.Errorf("package artifacts do not share one exact process-local context")
		}
		if _, duplicate := seenPackages[artifact.Package]; duplicate {
			return mir.Program{}, fmt.Errorf("package artifact %s is duplicated", artifact.Package)
		}
		seenPackages[artifact.Package] = struct{}{}
		for _, record := range artifact.Functions {
			if _, duplicate := owned[record.ID]; duplicate {
				return mir.Program{}, fmt.Errorf("owned FunctionID %s is duplicated", record.ID)
			}
			owned[record.ID] = record
			owners[record.ID] = artifactIndex
		}
	}
	for _, artifact := range artifacts {
		for _, imported := range artifact.Imports {
			record, found := owned[imported.ID]
			if !found {
				return mir.Program{}, fmt.Errorf("import FunctionID %s is unresolved", imported.ID)
			}
			if !reflect.DeepEqual(record.Descriptor, imported.Descriptor) {
				return mir.Program{}, fmt.Errorf("import FunctionID %s has a conflicting descriptor", imported.ID)
			}
		}
	}

	ids := make([]FunctionID, 0, len(owned))
	for id := range owned {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(left, right FunctionID) int { return bytes.Compare(left[:], right[:]) })
	maxSymbol := hir.SymbolID(0)
	for _, foreign := range base.Context.ForeignFunctions {
		maxSymbol = max(maxSymbol, foreign.Symbol)
	}
	for _, record := range owned {
		maxSymbol = max(maxSymbol, record.Body.Symbol)
	}
	for _, artifact := range artifacts {
		for _, imported := range artifact.Imports {
			maxSymbol = max(maxSymbol, imported.Signature.Symbol)
		}
	}
	firstFresh := maxSymbol + 1
	if firstFresh <= maxSymbol {
		return mir.Program{}, fmt.Errorf("internal function symbol space is exhausted")
	}
	fresh := make(map[FunctionID]hir.SymbolID, len(ids))
	for index, id := range ids {
		fresh[id] = firstFresh + hir.SymbolID(index)
		if fresh[id] < firstFresh {
			return mir.Program{}, fmt.Errorf("internal function symbol space overflow")
		}
	}

	program := mir.Clone(base.Context)
	originalToID := make(map[hir.SymbolID]FunctionID, len(owned))
	for id, record := range owned {
		if previous, duplicate := originalToID[record.Body.Symbol]; duplicate && previous != id {
			return mir.Program{}, fmt.Errorf("original function symbol %d maps to multiple identities", record.Body.Symbol)
		}
		originalToID[record.Body.Symbol] = id
	}
	for _, id := range ids {
		record := owned[id]
		artifact := artifacts[owners[id]]
		bindings := artifactBindings(artifact)
		function := cloneFunction(record.Body)
		function.Symbol = fresh[id]
		for blockIndex := range function.Blocks {
			terminator := &function.Blocks[blockIndex].Terminator
			if terminator.Kind != mir.TerminatorCall || terminator.ForeignABI != "" {
				continue
			}
			targetID, found := bindings[terminator.Callee]
			if !found {
				return mir.Program{}, fmt.Errorf("function %q call symbol %d has no artifact binding", function.Name, terminator.Callee)
			}
			newSymbol, found := fresh[targetID]
			if !found {
				return mir.Program{}, fmt.Errorf("function %q call target %s is unresolved", function.Name, targetID)
			}
			terminator.Callee = newSymbol
		}
		program.Functions = append(program.Functions, function)
	}
	for index := range program.ResourceTypes {
		resource := &program.ResourceTypes[index]
		if resource.DropSymbol == hir.NoSymbol {
			continue
		}
		id, found := originalToID[resource.DropSymbol]
		if !found {
			return mir.Program{}, fmt.Errorf("resource %q drop symbol %d has no owned function", resource.Type, resource.DropSymbol)
		}
		resource.DropSymbol = fresh[id]
	}
	slices.SortFunc(program.Functions, func(left, right mir.Function) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return 0
	})
	if failures := mir.Verify(program); len(failures) != 0 {
		return mir.Program{}, fmt.Errorf("verify reassembled whole program: %w", failures[0])
	}
	return program, nil
}

func verifyBinding(id FunctionID, descriptor FunctionDescriptor, function mir.Function, identities map[hir.PackageID]packagegraph.InstanceID) error {
	actual, err := DescribeFunction(function, identities)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, descriptor) {
		return fmt.Errorf("function %q descriptor does not match its MIR signature", function.Name)
	}
	computed, err := ComputeFunctionID(descriptor)
	if err != nil {
		return err
	}
	if computed != id {
		return fmt.Errorf("function %q identity does not match its descriptor", function.Name)
	}
	return nil
}

func recordBinding(symbols map[hir.SymbolID]string, ids map[FunctionID]string, symbol hir.SymbolID, id FunctionID, name string) error {
	if previous, duplicate := symbols[symbol]; duplicate {
		return fmt.Errorf("functions %q and %q share symbol %d", previous, name, symbol)
	}
	if previous, duplicate := ids[id]; duplicate {
		return fmt.Errorf("functions %q and %q share FunctionID %s", previous, name, id)
	}
	symbols[symbol] = name
	ids[id] = name
	return nil
}

func signatureOf(function mir.Function) FunctionSignature {
	return FunctionSignature{
		Symbol: function.Symbol, Name: function.Name, Span: function.Span,
		ParameterTypes: slices.Clone(function.ParameterTypes), ResultType: function.ResultType,
		Effects: slices.Clone(function.Effects), DropGlueFor: function.DropGlueFor,
	}
}

func (signature FunctionSignature) stub() mir.Function {
	function := mir.Function{
		Symbol: signature.Symbol, Name: signature.Name, Span: signature.Span,
		ParameterTypes: slices.Clone(signature.ParameterTypes), ResultType: signature.ResultType,
		Effects: slices.Clone(signature.Effects), DropGlueFor: signature.DropGlueFor,
		Entry: 0, Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorUnreachable, UnwindSuccessor: mir.NoBlock}}},
	}
	if signature.DropGlueFor != "" {
		function.Parameters = []mir.LocalID{0}
	}
	return function
}

func contextOf(program mir.Program) mir.Program {
	context := mir.Clone(program)
	context.Functions = nil
	return context
}

func cloneFunction(function mir.Function) mir.Function {
	return mir.Clone(mir.Program{Functions: []mir.Function{function}}).Functions[0]
}

func cloneSignature(signature FunctionSignature) FunctionSignature {
	result := signature
	result.ParameterTypes = slices.Clone(signature.ParameterTypes)
	result.Effects = slices.Clone(signature.Effects)
	return result
}

func cloneDescriptor(descriptor FunctionDescriptor) FunctionDescriptor {
	result := descriptor
	result.ParameterTypes = slices.Clone(descriptor.ParameterTypes)
	result.Effects = slices.Clone(descriptor.Effects)
	return result
}

func cloneIdentities(identities map[hir.PackageID]packagegraph.InstanceID) map[hir.PackageID]packagegraph.InstanceID {
	result := make(map[hir.PackageID]packagegraph.InstanceID, len(identities))
	for packageID, identity := range identities {
		result[packageID] = identity
	}
	return result
}

func artifactBindings(artifact Artifact) map[hir.SymbolID]FunctionID {
	result := make(map[hir.SymbolID]FunctionID, len(artifact.Functions)+len(artifact.Imports))
	for _, record := range artifact.Functions {
		result[record.Body.Symbol] = record.ID
	}
	for _, imported := range artifact.Imports {
		result[imported.Signature.Symbol] = imported.ID
	}
	return result
}

func sortRecords(records []FunctionRecord) {
	slices.SortFunc(records, func(left, right FunctionRecord) int { return bytes.Compare(left.ID[:], right.ID[:]) })
}

func sortImports(imports []FunctionImport) {
	slices.SortFunc(imports, func(left, right FunctionImport) int { return bytes.Compare(left.ID[:], right.ID[:]) })
}

func recordsSorted(records []FunctionRecord) bool {
	for index := 1; index < len(records); index++ {
		if bytes.Compare(records[index-1].ID[:], records[index].ID[:]) >= 0 {
			return false
		}
	}
	return true
}

func importsSorted(imports []FunctionImport) bool {
	for index := 1; index < len(imports); index++ {
		if bytes.Compare(imports[index-1].ID[:], imports[index].ID[:]) >= 0 {
			return false
		}
	}
	return true
}
