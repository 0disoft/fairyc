package foreignstructprobe

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

// Bidirectional adds fixture-only C entry bridges to generated Fairy bodies.
// It does not admit source exports or aggregate callback signatures.
type Bidirectional struct {
	Unit
	Records []string
}

func GenerateBidirectional(posix bool) (Bidirectional, error) {
	unit := generate(posix)
	unit.FairySource = strings.Replace(unit.FairySource, "fn i32 main()", "fn i32 forward_main()", 1) + pointerFairyRecords
	unit.CompanionC += pointerCRecords
	var scalarFields, scalarC strings.Builder
	scalarFields.WriteString("\n@repr(C) struct fairyc_reverse_c_scalars {")
	scalarC.WriteString("\nstruct fairyc_reverse_c_scalars {")
	for _, typ := range cArrayProbeTypes(posix) {
		fmt.Fprintf(&scalarFields, "c::%s field_%s;", typ.name, typ.name)
		fmt.Fprintf(&scalarC, "%s field_%s;", typ.c, typ.name)
	}
	scalarFields.WriteString("}\n")
	scalarC.WriteString("};\n")
	unit.FairySource += scalarFields.String()
	unit.CompanionC += scalarC.String()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(unit.FairySource)), source.DefaultLimits())
	if err != nil {
		return Bidirectional{}, err
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return Bidirectional{}, fmt.Errorf("aggregate inventory syntax: %v", parsed.Tree.Diagnostics)
	}
	records := make(map[string]ast.Struct)
	enums := make(map[string]string)
	var names []string
	for _, r := range parsed.AST.Structs {
		records[r.Name] = r
		names = append(names, r.Name)
	}
	for _, e := range parsed.AST.Enums {
		for _, a := range e.Attributes {
			if a.Name == "repr" && len(a.Arguments) == 1 {
				enums[e.Name] = a.Arguments[0].Raw
			}
		}
	}
	slices.Sort(names)
	var fy, c strings.Builder
	c.WriteString("\n#include <limits.h>\n#include <float.h>\n#include <stddef.h>\n")
	fmt.Fprintf(&c, "static unsigned fy_reverse_counts[%d];\nint32_t fairyc_reverse_mark(int32_t id){if(id<0 || (size_t)id>=sizeof fy_reverse_counts/sizeof *fy_reverse_counts){return 0;}++fy_reverse_counts[id];return 42;}\n", len(names))
	fy.WriteString("\nunsafe extern C fn i32 fairyc_reverse_mark(i32 id);\n")
	var calls []string
	for id, name := range names {
		r := records[name]
		var fields []string
		for _, f := range r.Fields {
			fields = append(fields, "."+f.Name+"=value."+f.Name)
		}
		fmt.Fprintf(&fy, "unsafe fn %s reverse_%s(%s value){let i32 seen=fairyc_reverse_mark(%d);assert(seen==42);return %s{%s};}\n", name, name, name, id, name, strings.Join(fields, ","))
		fmt.Fprintf(&fy, "unsafe extern C fn %s reverse_make_%s(i32 choice);\nunsafe extern C fn i32 reverse_check_%s(%s original,%s echoed,i32 choice);\n", name, name, name, name, name)
		fmt.Fprintf(&fy, "fn bool check_reverse_%s(i32 choice){let %s original=unsafe{reverse_make_%s(choice)};let %s echoed=unsafe{reverse_%s(original)};return unsafe{reverse_check_%s(original,echoed,choice)}==42;}\n", name, name, name, name, name, name)
		for choice := 0; choice < 3; choice++ {
			calls = append(calls, fmt.Sprintf("check_reverse_%s(%d)", name, choice))
		}
		fmt.Fprintf(&c, "extern struct %s reverse_bridge_%s(struct %s value);\nstruct %s reverse_make_%s(int32_t choice){struct %s value={0};\n", name, name, name, name, name, name)
		if err := emitReverseFields(&c, records, enums, r, "value", "", false, 0, 0); err != nil {
			return Bidirectional{}, err
		}
		c.WriteString("return value;}\n")
		fmt.Fprintf(&c, "int32_t reverse_check_%s(struct %s original,struct %s echoed,int32_t choice){struct %s expected=reverse_make_%s(choice);unsigned before=fy_reverse_counts[%d];struct %s from_c=reverse_bridge_%s(original);if(fy_reverse_counts[%d]!=before+1){return 0;}\n", name, name, name, name, name, id, name, name, id)
		for _, expression := range []string{"original", "echoed", "from_c"} {
			if err := emitReverseFields(&c, records, enums, r, expression, "expected", true, 0, 0); err != nil {
				return Bidirectional{}, err
			}
		}
		c.WriteString("return 42;}\n")
	}
	fmt.Fprintf(&fy, "fn i32 main(){return if(forward_main()==42 && %s){42}else{0};}\n", strings.Join(calls, " && "))
	unit.FairySource += fy.String()
	unit.CompanionC += c.String()
	return Bidirectional{Unit: unit, Records: names}, nil
}

// Bind appends test-only wrappers using emitted function metadata. Generated
// definitions, linkage and entry are preserved; no source export is invented.
func (p Bidirectional) Bind(emitted string, functions []string) (string, error) {
	var bridges strings.Builder
	for _, name := range p.Records {
		index := slices.Index(functions, "main::reverse_"+name)
		if index < 0 {
			return "", fmt.Errorf("missing generated reverse body for %s", name)
		}
		prototype := fmt.Sprintf("static struct %s fy_fn_%d(struct %s ", name, index, name)
		if !strings.Contains(emitted, prototype) {
			return "", fmt.Errorf("reverse signature mismatch for %s", name)
		}
		fmt.Fprintf(&bridges, "\nstruct %s reverse_bridge_%s(struct %s value){return fy_fn_%d(value);}\n", name, name, name, index)
	}
	return emitted + bridges.String(), nil
}

func emitReverseFields(out *strings.Builder, records map[string]ast.Struct, enums map[string]string, r ast.Struct, left, right string, compare bool, depth, seed int) error {
	if depth > 16 {
		return fmt.Errorf("aggregate probe nesting exceeds 16")
	}
	for index, f := range r.Fields {
		a, b := left+"."+f.Name, right+"."+f.Name
		t := f.Type
		selector := fmt.Sprintf("((choice+%d)%%3)", seed+index)
		if t.Kind == ast.TypeKindArray {
			if t.Length == nil || t.Element == nil {
				return fmt.Errorf("invalid probe array")
			}
			n, err := strconv.Atoi(t.Length.Text)
			if err != nil || n < 1 || n > 4096 {
				return fmt.Errorf("invalid probe array length")
			}
			fmt.Fprintf(out, "for(size_t i%d=0;i%d<%d;++i%d){\n", depth, depth, n, depth)
			a += fmt.Sprintf("[i%d]", depth)
			b += fmt.Sprintf("[i%d]", depth)
			t = *t.Element
			selector = fmt.Sprintf("((choice+%d+i%d)%%3)", seed+index, depth)
		}
		if nested, ok := records[t.Path]; t.Kind == ast.TypeKindNamed && ok {
			if err := emitReverseFields(out, records, enums, nested, a, b, compare, depth+1, seed+index+1); err != nil {
				return err
			}
		} else if compare {
			fmt.Fprintf(out, "if(%s!=%s){return 0;}\n", a, b)
		} else if t.Kind == ast.TypeKindRawPointer {
			if t.Element == nil {
				return fmt.Errorf("invalid pointer")
			}
			key := t.Element.Path
			if key == "void" {
				key = "i32"
			}
			fmt.Fprintf(out, "%s=%s==0?NULL:&fy_reverse_target_%s;\n", a, selector, key)
		} else {
			key := t.Path
			if base, ok := enums[key]; ok {
				key = base
			}
			values, ok := reverseValues(key)
			if !ok {
				return fmt.Errorf("unsupported probe scalar %q", key)
			}
			fmt.Fprintf(out, "%s=%s==0?(%s):%s==1?(%s):(%s);\n", a, selector, values[0], selector, values[1], values[2])
		}
		if f.Type.Kind == ast.TypeKindArray {
			out.WriteString("}\n")
		}
	}
	return nil
}

func reverseValues(name string) ([3]string, bool) {
	if strings.HasPrefix(name, "c::") {
		for _, t := range cArrayProbeTypes(true) {
			if "c::"+t.name == name {
				return [3]string{t.first, t.middle, t.last}, true
			}
		}
	}
	limits := map[string][3]string{
		"u8": {"0", "17", "UINT8_MAX"}, "i8": {"INT8_MIN", "-17", "INT8_MAX"}, "i16": {"INT16_MIN", "-17", "INT16_MAX"}, "u16": {"0", "17", "UINT16_MAX"},
		"i32": {"INT32_MIN", "-17", "INT32_MAX"}, "u32": {"0", "17", "UINT32_MAX"}, "i64": {"INT64_MIN", "-17", "INT64_MAX"}, "u64": {"0", "17", "UINT64_MAX"},
		"usize": {"0", "17", "UINTPTR_MAX"}, "isize": {"INTPTR_MIN", "-17", "INTPTR_MAX"}, "f32": {"-FLT_MAX", "0.0f", "FLT_MAX"}, "f64": {"-DBL_MAX", "0.0", "DBL_MAX"},
	}
	v, ok := limits[name]
	return v, ok
}

const pointerFairyRecords = `
@repr(C) struct fairyc_reverse_integer_scalars {i8 tiny;i16 small;i32 number;i64 wide;isize offset;u8 byte;u16 word;u32 count;u64 large;usize size;}
@repr(C) struct fairyc_reverse_pointer_record {ptr u8 byte;ptr mut i32 number;ptr i64 wide;ptr mut u32 word;ptr usize size;ptr mut void opaque;ptr fairyc_fixture_status_v0 record;}
@repr(C) struct fairyc_reverse_pointer_envelope {u8 lead;fairyc_reverse_pointer_record packet;u8 tail;}
@repr(C) struct fairyc_reverse_max_array {u8[4096] bytes;u32 tail;}
`

const pointerCRecords = `
struct fairyc_reverse_integer_scalars {int8_t tiny;int16_t small;int32_t number;int64_t wide;intptr_t offset;uint8_t byte;uint16_t word;uint32_t count;uint64_t large;uintptr_t size;};
struct fairyc_fixture_float_envelope_v0 {struct fairyc_fixture_float_packet_v0 packet;};
struct fairyc_reverse_pointer_record {const uint8_t *byte;int32_t *number;const int64_t *wide;uint32_t *word;const uintptr_t *size;void *opaque;const struct fairyc_fixture_status_v0 *record;};
struct fairyc_reverse_pointer_envelope {uint8_t lead;struct fairyc_reverse_pointer_record packet;uint8_t tail;};
struct fairyc_reverse_max_array {uint8_t bytes[4096];uint32_t tail;};
static uint8_t fy_reverse_target_u8;
static int32_t fy_reverse_target_i32;
static int64_t fy_reverse_target_i64;
static uint32_t fy_reverse_target_u32;
static uintptr_t fy_reverse_target_usize;
static struct fairyc_fixture_status_v0 fy_reverse_target_fairyc_fixture_status_v0;
`
