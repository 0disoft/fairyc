package packageartifact

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const contextArtifactLimit = 1 << 20

func TestStableContextIgnoresBuildLocalPackageAndSymbolNumbers(t *testing.T) {
	t.Parallel()
	leftContext, leftIdentities, leftBindings := stableContextFixture(3, 41, 51)
	rightContext, rightIdentities, rightBindings := stableContextFixture(9, 141, 151)

	left, err := DescribeContext(leftContext, leftIdentities, leftBindings)
	if err != nil {
		t.Fatal(err)
	}
	right, err := DescribeContext(rightContext, rightIdentities, rightBindings)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("stable contexts differ after local renumbering:\nleft:  %+v\nright: %+v", left, right)
	}

	leftBytes, err := EncodeContextDescriptor(left, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	rightBytes, err := EncodeContextDescriptor(right, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftBytes, rightBytes) {
		t.Fatal("canonical context bytes differ after local renumbering")
	}
	decoded, err := DecodeContextDescriptor(leftBytes, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, left) {
		t.Fatal("decoded stable context differs from source")
	}
	leftID, err := ComputeContextID(left, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	rightID, err := ComputeContextID(right, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if leftID != rightID {
		t.Fatalf("context IDs differ after local renumbering: %s != %s", leftID, rightID)
	}
	parsed, err := ParseContextID(leftID.String())
	if err != nil || parsed != leftID {
		t.Fatalf("ParseContextID() = %s, %v", parsed, err)
	}
}

func TestStableContextCodecRejectsBoundsUnknownFieldsAndNoncanonicalBytes(t *testing.T) {
	t.Parallel()
	context, identities, bindings := stableContextFixture(3, 41, 51)
	descriptor, err := DescribeContext(context, identities, bindings)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeContextDescriptor(descriptor, contextArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeContextDescriptor(descriptor, uint64(len(encoded)-1)); !errors.Is(err, ErrContextArtifactTooLarge) {
		t.Fatalf("small encode bound error = %v", err)
	}
	if _, err := DecodeContextDescriptor(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrContextArtifactTooLarge) {
		t.Fatalf("small decode bound error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)
	if _, err := DecodeContextDescriptor(unknown, contextArtifactLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	withWhitespace := append(append([]byte(nil), encoded...), '\n')
	if _, err := DecodeContextDescriptor(withWhitespace, contextArtifactLimit); !errors.Is(err, ErrContextArtifactEncoding) {
		t.Fatalf("noncanonical error = %v", err)
	}
	if _, err := DecodeContextDescriptor(encoded, 0); err == nil {
		t.Fatal("zero decode bound accepted")
	}
	if _, err := ParseContextID(strings.ToUpper(ContextIDPrefix) + strings.Repeat("0", 64)); err == nil {
		t.Fatal("noncanonical context ID accepted")
	}
}

func TestStableContextRejectsOrderingTypeAndDropContradictions(t *testing.T) {
	t.Parallel()
	context, identities, bindings := stableContextFixture(3, 41, 51)
	descriptor, err := DescribeContext(context, identities, bindings)
	if err != nil {
		t.Fatal(err)
	}

	unsorted := cloneContextDescriptor(descriptor)
	unsorted.Structs = append(unsorted.Structs, unsorted.Structs[0])
	slices.Reverse(unsorted.Structs)
	if _, err := EncodeContextDescriptor(unsorted, contextArtifactLimit); err == nil || !strings.Contains(err.Error(), "sorted and unique") {
		t.Fatalf("unsorted context error = %v", err)
	}
	badType := cloneContextDescriptor(descriptor)
	badType.Enums[0].Variants[0].Fields[0].Type = "$p3::types::Point"
	if _, err := EncodeContextDescriptor(badType, contextArtifactLimit); err == nil || !strings.Contains(err.Error(), "field") {
		t.Fatalf("build-local type error = %v", err)
	}
	badDrop := cloneContextDescriptor(descriptor)
	badDrop.Resources[0].DropFunctionID = FunctionID{}
	if _, err := EncodeContextDescriptor(badDrop, contextArtifactLimit); err == nil || !strings.Contains(err.Error(), "drop function") {
		t.Fatalf("drop contradiction error = %v", err)
	}
	borrowedForeign := cloneContextDescriptor(descriptor)
	borrowedForeign.ForeignFunctions[0].ParameterTypes[0] = "ref i32"
	if _, err := EncodeContextDescriptor(borrowedForeign, contextArtifactLimit); err == nil || !strings.Contains(err.Error(), "safe borrow") {
		t.Fatalf("foreign borrow error = %v", err)
	}
}

func stableContextFixture(packageID hir.PackageID, dropSymbol, foreignSymbol hir.SymbolID) (mir.Program, map[hir.PackageID]packagegraph.InstanceID, map[hir.SymbolID]FunctionID) {
	instance := packagegraph.InstanceID{1, 2, 3}
	dropID := FunctionID{9, 8, 7}
	packageText := strconv.FormatUint(uint64(packageID), 10)
	prefix := "$p" + packageText + "::types::"
	span := hir.Span{Package: packageID}
	context := mir.Program{
		StructTypes: []mir.StructType{{
			Type: hir.Type(prefix + "Point"), Name: "Point", Span: span, ReprC: true,
			Fields: []mir.StructField{{Name: "x", Type: hir.TypeI32, Span: span}},
		}},
		EnumTypes: []mir.EnumType{{
			Type: hir.Type(prefix + "Message"), Name: "Message", Span: span,
			Variants: []mir.EnumVariant{{
				Name: "Point", Span: span,
				Fields: []mir.EnumField{{Name: "value", Type: hir.Type(prefix + "Point"), Span: span}},
			}},
		}},
		ResourceTypes: []mir.ResourceType{{
			Type: hir.Type(prefix + "Handle"), Span: span,
			Fields:      []mir.ResourceField{{Name: "raw", Type: hir.TypeUSize, Span: span}},
			DropKind:    hir.ResourceDropNonEmpty,
			DropSymbol:  dropSymbol,
			DropEffects: []hir.Effect{hir.EffectFilesystem},
		}},
		ForeignFunctions: []mir.ForeignFunction{{
			Symbol: foreignSymbol, Name: "$p" + packageText + "::ffi::write",
			LinkName: "write", ABI: "C", Span: span,
			ParameterTypes: []hir.Type{hir.TypeUSize}, ResultType: hir.TypeI32,
		}},
	}
	return context, map[hir.PackageID]packagegraph.InstanceID{packageID: instance}, map[hir.SymbolID]FunctionID{dropSymbol: dropID}
}
