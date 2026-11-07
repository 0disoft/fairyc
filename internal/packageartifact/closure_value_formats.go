package packageartifact

import (
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Closure envelopes are feature-exact: neither a legacy envelope containing a
// copied closure nor a closure envelope without that feature is compatible.
// Selection never replaces semantic, MIR, identity, or canonical validation.
const (
	ClosureProcessCodecSchema           = "fairyc.package-mir-closure-process/v0"
	ClosureProcessCodecVersion          = "fairyc-package-mir-closure-process/1"
	ClosureStorageProcessCodecSchema    = "fairyc.package-mir-storage-closure-process/v0"
	ClosureStorageProcessCodecVersion   = "fairyc-package-mir-storage-closure-process/1"
	ClosureBodySchema                   = "fairyc.package-function-closure-body/v0"
	ClosureBodyVersion                  = "fairyc-package-function-closure-body/1"
	ClosureStableArtifactSchema         = "fairyc.stable-package-closure-mir/v0"
	ClosureStableArtifactVersion        = "fairyc-stable-package-closure-mir/1"
	ClosureStorageStableArtifactSchema  = "fairyc.stable-package-storage-closure-mir/v0"
	ClosureStorageStableArtifactVersion = "fairyc-stable-package-storage-closure-mir/1"
	ClosureContextSchema                = "fairyc.package-type-closure-context/v0"
	ClosureContextVersion               = "fairyc-package-type-closure-context/1"
)

func bodyFormat(body StableBodyArtifact) (schema, version string) {
	if mir.HasCopiedClosureValues(mir.Program{Functions: []mir.Function{body.Function}}) {
		return ClosureBodySchema, ClosureBodyVersion
	}
	return BodySchema, BodyVersion
}

func stableArtifactFormat(artifact StableArtifact, transfers bool) (schema, version string) {
	closure := contextHasClosureValues(artifact.Context)
	for _, body := range artifact.Functions {
		if schema, _ := bodyFormat(body); schema == ClosureBodySchema {
			closure = true
			break
		}
	}
	if closure {
		if transfers {
			return ClosureStorageStableArtifactSchema, ClosureStorageStableArtifactVersion
		}
		return ClosureStableArtifactSchema, ClosureStableArtifactVersion
	}
	return stableArtifactContract(transfers)
}

func processArtifactHasClosureValues(artifact Artifact) bool {
	if mir.HasCopiedClosureValues(artifact.Context) || contextHasClosureValues(artifact.StableContext) {
		return true
	}
	for _, record := range artifact.Functions {
		if mir.HasCopiedClosureValues(mir.Program{Functions: []mir.Function{record.Body}}) || descriptorHasClosureValues(record.Descriptor) {
			return true
		}
	}
	for _, imported := range artifact.Imports {
		if mir.HasCopiedClosureValues(mir.Program{Functions: []mir.Function{imported.Signature.stub()}}) || descriptorHasClosureValues(imported.Descriptor) {
			return true
		}
	}
	return false
}

func descriptorHasClosureValues(value FunctionDescriptor) bool {
	if stableTypeHasClosureValues(value.ResultType) || stableTypeHasClosureValues(value.DropGlueFor) {
		return true
	}
	for _, typ := range value.ParameterTypes {
		if stableTypeHasClosureValues(typ) {
			return true
		}
	}
	return false
}

func contextFormat(value ContextDescriptor) (schema, version string) {
	if contextHasClosureValues(value) {
		return ClosureContextSchema, ClosureContextVersion
	}
	return ContextSchema, ContextVersion
}

func contextHasClosureValues(value ContextDescriptor) bool {
	fieldsContain := func(fields []ContextField) bool {
		for _, field := range fields {
			if stableTypeHasClosureValues(field.Type) {
				return true
			}
		}
		return false
	}
	for _, item := range value.Structs {
		if stableTypeHasClosureValues(item.Type) || fieldsContain(item.Fields) {
			return true
		}
	}
	for _, item := range value.Enums {
		if stableTypeHasClosureValues(item.Type) {
			return true
		}
		for _, variant := range item.Variants {
			if fieldsContain(variant.Fields) {
				return true
			}
		}
	}
	for _, item := range value.Resources {
		if stableTypeHasClosureValues(item.Type) || fieldsContain(item.Fields) {
			return true
		}
	}
	for _, item := range value.ForeignFunctions {
		if stableTypeHasClosureValues(item.ResultType) {
			return true
		}
		for _, typ := range item.ParameterTypes {
			if stableTypeHasClosureValues(typ) {
				return true
			}
		}
	}
	return false
}

// Context types use private stable ref/ptr and coordinate-bound nominal
// spellings. Walk parsed children and unwrap private indirection explicitly;
// arbitrary substrings, names, and callback spellings do not confer a feature.
// Wrappers are detected for version gating, but MIR still rejects their use.
func stableTypeHasClosureValues(value string) bool {
	pending := []string{value}
	for len(pending) != 0 {
		value := stripStableIndirection(pending[len(pending)-1])
		pending = pending[:len(pending)-1]
		parsed := hir.ParseType(value)
		if parsed == hir.TypeClosureI32 {
			return true
		}
		d := parsed.Descriptor()
		// Stable arrays use T[N], whereas HIR parses [T; N]. In
		// particular a spelling starting with fn remains opaque in HIR.
		// Match the private grammar's positive uint64 length admission.
		if d.Kind == hir.TypeKindOpaque {
			if strings.HasSuffix(value, "[]") {
				pending = append(pending, strings.TrimSuffix(value, "[]"))
			} else if strings.HasSuffix(value, "]") {
				if separator := strings.LastIndexByte(value, '['); separator > 0 {
					if length, err := strconv.ParseUint(value[separator+1:len(value)-1], 10, 64); err == nil && length > 0 {
						pending = append(pending, value[:separator])
					}
				}
			}
		}
		if d.Element != (hir.Type{}) {
			pending = append(pending, d.Element.String())
		}
		if d.Failure != (hir.Type{}) {
			pending = append(pending, d.Failure.String())
		}
	}
	return false
}
