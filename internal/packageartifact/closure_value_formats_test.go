package packageartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const closureFormatSource = `module main;
struct Box{fn i32(i32) action;}
fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}
fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
fn i32 main(){let Box box=Box{.action=make(40)};let fn i32(i32) action=box.action;return apply(action,2);}`

const callbackFormatSource = `module main;
unsafe extern C fn i32 visit(i32 value,unsafe extern C fn i32(i32,i32,ptr mut void) cb,ptr mut void data);
unsafe fn i32 handle(i32 value,i32 extra,ptr mut void data){return value+1;}
fn i32 main(){return unsafe{visit(41,handle,null)};}`

func closureFormatFixture(t *testing.T, text string) Artifact {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("syntax: %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked.Semantic)
	}
	artifacts, err := ExtractAll(checked.MIR,
		map[hir.PackageID]packagegraph.InstanceID{hir.DefaultPackage: {1}},
		map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: {2}})
	if err != nil || len(artifacts) != 1 {
		t.Fatalf("extract: count=%d, %v", len(artifacts), err)
	}
	return artifacts[0]
}

// Exercise the public codecs, not just their selectors: format/feature mismatch
// must fail before a valid body can be returned, including storage envelopes.
func TestClosureValueFormatsExactVersionsAndBounds(t *testing.T) {
	for _, fixture := range []struct {
		name, text string
		closure    bool
	}{
		{"plain", "module main; fn i32 main(){return 42;}", false},
		{"callback", callbackFormatSource, false},
		{"closure", closureFormatSource, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			artifact := closureFormatFixture(t, fixture.text)
			stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
			if err != nil {
				t.Fatal(err)
			}
			body := stable.Functions[0]
			for _, candidate := range stable.Functions {
				if schema, _ := bodyFormat(candidate); schema == ClosureBodySchema {
					body = candidate
					break
				}
			}
			type formatCase struct {
				name, schema, version, alternateSchema, alternateVersion string
				envelope                                                 any
				encode                                                   func(uint64) ([]byte, error)
				decode                                                   func([]byte, uint64) error
				tooLarge, noncanonical                                   error
			}
			selectFormat := func(legacySchema, legacyVersion, closureSchema, closureVersion string) (string, string, string, string) {
				if fixture.closure {
					return closureSchema, closureVersion, legacySchema, legacyVersion
				}
				return legacySchema, legacyVersion, closureSchema, closureVersion
			}
			var cases []formatCase
			for _, storage := range []bool{false, true} {
				codec := fixedProcessCodec(t, 1)
				codec.storageTransfers = storage
				schema, version, alternateSchema, alternateVersion := selectFormat(ProcessCodecSchema, ProcessCodecVersion, ClosureProcessCodecSchema, ClosureProcessCodecVersion)
				if storage {
					schema, version, alternateSchema, alternateVersion = selectFormat(StorageProcessCodecSchema, StorageProcessCodecVersion, ClosureStorageProcessCodecSchema, ClosureStorageProcessCodecVersion)
				}
				cases = append(cases, formatCase{
					name: fmt.Sprintf("process/storage=%t", storage), schema: schema, version: version, alternateSchema: alternateSchema, alternateVersion: alternateVersion,
					envelope: processEnvelope{schema, version, codec.Identity(), artifact},
					encode: func(limit uint64) ([]byte, error) {
						copy := *codec
						copy.maxBytes = limit
						return copy.Encode(artifact)
					},
					decode: func(data []byte, limit uint64) error {
						copy := *codec
						copy.maxBytes = limit
						_, err := copy.Decode(data)
						return err
					},
					tooLarge: ErrProcessArtifactTooLarge, noncanonical: ErrProcessArtifactEncoding,
				})
				schema, version, alternateSchema, alternateVersion = selectFormat(StableArtifactSchema, StableArtifactVersion, ClosureStableArtifactSchema, ClosureStableArtifactVersion)
				if storage {
					schema, version, alternateSchema, alternateVersion = selectFormat(StorageStableArtifactSchema, StorageStableArtifactVersion, ClosureStorageStableArtifactSchema, ClosureStorageStableArtifactVersion)
				}
				cases = append(cases, formatCase{
					name: fmt.Sprintf("stable/storage=%t", storage), schema: schema, version: version, alternateSchema: alternateSchema, alternateVersion: alternateVersion,
					envelope: stableArtifactEnvelope{schema, version, stable},
					encode:   func(limit uint64) ([]byte, error) { return encodeStableArtifact(stable, limit, storage) },
					decode: func(data []byte, limit uint64) error {
						_, err := decodeStableArtifact(data, limit, storage)
						return err
					},
					tooLarge: ErrStableArtifactTooLarge, noncanonical: ErrStableArtifactEncoding,
				})
			}
			schema, version, alternateSchema, alternateVersion := selectFormat(BodySchema, BodyVersion, ClosureBodySchema, ClosureBodyVersion)
			cases = append(cases, formatCase{
				name: "body", schema: schema, version: version, alternateSchema: alternateSchema, alternateVersion: alternateVersion,
				envelope: bodyEnvelope{schema, version, body},
				encode:   func(limit uint64) ([]byte, error) { return EncodeStableBody(body, limit) },
				decode:   func(data []byte, limit uint64) error { _, err := DecodeStableBody(data, limit); return err },
				tooLarge: ErrBodyArtifactTooLarge, noncanonical: ErrBodyArtifactEncoding,
			})
			schema, version, alternateSchema, alternateVersion = selectFormat(ContextSchema, ContextVersion, ClosureContextSchema, ClosureContextVersion)
			cases = append(cases, formatCase{
				name: "context", schema: schema, version: version, alternateSchema: alternateSchema, alternateVersion: alternateVersion,
				envelope: contextEnvelope{schema, version, stable.Context},
				encode:   func(limit uint64) ([]byte, error) { return EncodeContextDescriptor(stable.Context, limit) },
				decode:   func(data []byte, limit uint64) error { _, err := DecodeContextDescriptor(data, limit); return err },
				tooLarge: ErrContextArtifactTooLarge, noncanonical: ErrContextArtifactEncoding,
			})
			for _, item := range cases {
				t.Run(item.name, func(t *testing.T) {
					want, err := json.Marshal(item.envelope)
					if err != nil {
						t.Fatal(err)
					}
					exact := uint64(len(want))
					encoded, err := item.encode(exact)
					if err != nil || !bytes.Equal(encoded, want) {
						t.Fatalf("exact canonical bytes changed: %v", err)
					}
					if err := item.decode(encoded, exact); err != nil {
						t.Fatal(err)
					}
					if data, err := item.encode(exact - 1); data != nil || !errors.Is(err, item.tooLarge) {
						t.Fatalf("encode one byte over: %d bytes, %v", len(data), err)
					}
					if err := item.decode(encoded, exact-1); !errors.Is(err, item.tooLarge) {
						t.Fatalf("decode one byte over: %v", err)
					}
					if _, err := item.encode(0); err == nil {
						t.Fatal("zero encode limit accepted")
					}
					if err := item.decode(encoded, 0); err == nil {
						t.Fatal("zero decode limit accepted")
					}
					if err := item.decode(append(bytes.Clone(encoded), '\n'), exact+1); !errors.Is(err, item.noncanonical) {
						t.Fatalf("noncanonical whitespace: %v", err)
					}
					for name, raw := range map[string][]byte{
						"alternate-schema":   replaceFormatMember(encoded, "schema", item.schema, item.alternateSchema),
						"alternate-version":  replaceFormatMember(encoded, "version", item.version, item.alternateVersion),
						"alternate-envelope": replaceFormatMember(replaceFormatMember(encoded, "schema", item.schema, item.alternateSchema), "version", item.version, item.alternateVersion),
						"future-version":     replaceFormatMember(encoded, "version", item.version, "unsupported/99"),
						"unknown":            bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1),
						"trailing":           append(bytes.Clone(encoded), []byte(`{}`)...),
					} {
						if err := item.decode(raw, stablePackageArtifactLimit); err == nil {
							t.Fatalf("accepted %s", name)
						}
					}
				})
			}
		})
	}
}

func replaceFormatMember(data []byte, member, old, replacement string) []byte {
	from, _ := json.Marshal(old)
	to, _ := json.Marshal(replacement)
	prefix := []byte(`"` + member + `":`)
	return bytes.Replace(data, append(bytes.Clone(prefix), from...), append(bytes.Clone(prefix), to...), 1)
}

func TestClosureValueFormatsFeatureDetection(t *testing.T) {
	for _, typ := range []string{
		"fn i32(i32)", "ref fn i32(i32)", "ptr mut fn i32(i32)",
		"Option<fn i32(i32)>", "Result<i32, fn i32(i32)>", "fn i32(i32)[2]",
		"Option<ref fn i32(i32)>", "ref mut fn i32(i32)[2][3]",
		"Option<fn i32(i32)[18446744073709551615]>",
	} {
		if !stableTypeHasClosureValues(typ) {
			t.Errorf("lost closure type %q", typ)
		}
	}
	for _, typ := range []string{
		"i32", "fn i32(i64)", "unsafe extern C fn i32(i32,i32,ptr mut void)",
		"prefix fn i32(i32) suffix", "closure_create", "main::closure",
		"fn i32(i32)[0]", "fn i32(i32)[-1]", "fn i32(i32)[18446744073709551616]",
	} {
		if stableTypeHasClosureValues(typ) {
			t.Errorf("false closure type %q", typ)
		}
	}
	leaf := "fn i32(i32)"
	for name, context := range map[string]ContextDescriptor{
		"struct-identity":   {Structs: []ContextStruct{{Type: leaf}}},
		"struct-field":      {Structs: []ContextStruct{{Fields: []ContextField{{Type: leaf}}}}},
		"enum-identity":     {Enums: []ContextEnum{{Type: leaf}}},
		"enum-field":        {Enums: []ContextEnum{{Variants: []ContextVariant{{Fields: []ContextField{{Type: leaf}}}}}}},
		"resource-identity": {Resources: []ContextResource{{Type: leaf}}},
		"resource-field":    {Resources: []ContextResource{{Fields: []ContextField{{Type: leaf}}}}},
		"foreign-parameter": {ForeignFunctions: []ContextForeignFunction{{ParameterTypes: []string{leaf}}}},
		"foreign-result":    {ForeignFunctions: []ContextForeignFunction{{ResultType: leaf}}},
	} {
		t.Run(name, func(t *testing.T) {
			if !contextHasClosureValues(context) || !processArtifactHasClosureValues(Artifact{StableContext: context}) {
				t.Fatal("dormant context lost feature")
			}
			if schema, _ := stableArtifactFormat(StableArtifact{Context: context}, false); schema != ClosureStableArtifactSchema {
				t.Fatal("context-only stable artifact lost feature")
			}
		})
	}
	for name, artifact := range map[string]Artifact{
		"owned-target":      {Functions: []FunctionRecord{{Body: mir.Function{ClosureTarget: true}}}},
		"owned-captures":    {Functions: []FunctionRecord{{Body: mir.Function{ClosureCaptures: 1}}}},
		"import-target":     {Imports: []FunctionImport{{Signature: FunctionSignature{ClosureTarget: true}}}},
		"import-captures":   {Imports: []FunctionImport{{Signature: FunctionSignature{ClosureCaptures: 1}}}},
		"import-type":       {Imports: []FunctionImport{{Signature: FunctionSignature{ParameterTypes: []hir.Type{hir.TypeClosureI32}}}}},
		"import-descriptor": {Imports: []FunctionImport{{Descriptor: FunctionDescriptor{ResultType: leaf}}}},
		"mir-context":       {Context: mir.Program{StructTypes: []mir.StructType{{Fields: []mir.StructField{{Type: hir.TypeClosureI32}}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if !processArtifactHasClosureValues(artifact) {
				t.Fatal("lost artifact feature")
			}
			codec := fixedProcessCodec(t, 1)
			if schema, _, _ := codec.artifactContract(artifact); schema != ClosureProcessCodecSchema {
				t.Fatal("lost process format")
			}
			codec.storageTransfers = true
			if schema, _, _ := codec.artifactContract(artifact); schema != ClosureStorageProcessCodecSchema {
				t.Fatal("lost storage process format")
			}
		})
	}
}

func TestClosureValueFormatsKeepIdentityAndValidation(t *testing.T) {
	artifact := closureFormatFixture(t, closureFormatSource)
	for _, storage := range []bool{false, true} {
		codec := fixedProcessCodec(t, 1)
		codec.storageTransfers = storage
		identity := codec.Identity()
		encoded, err := codec.Encode(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if codec.Identity() != identity {
			t.Fatal("artifact feature changed process identity")
		}
		other := fixedProcessCodec(t, 2)
		other.storageTransfers = storage
		if _, err := other.Decode(encoded); !errors.Is(err, ErrProcessArtifactForeign) {
			t.Fatalf("foreign closure artifact: %v", err)
		}
		opposite := *codec
		opposite.storageTransfers = !storage
		if _, err := opposite.Decode(encoded); err == nil {
			t.Fatal("cross-storage closure artifact accepted")
		}
		bad := cloneArtifactForTest(artifact)
		bad.Functions[0].Body.ClosureCaptures = 255
		if data, err := codec.Encode(bad); err == nil || data != nil {
			t.Fatalf("format bypassed MIR: bytes=%d err=%v", len(data), err)
		}
		schema, version, _ := codec.artifactContract(bad)
		forged, err := json.Marshal(processEnvelope{schema, version, codec.Identity(), bad})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := codec.Decode(forged); err == nil {
			t.Fatal("closure envelope bypassed decoded MIR validation")
		}
	}
}
