package hir

import (
	"encoding/json"
	"strconv"
	"strings"
	"unique"
)

// TypeID is an immutable, comparable, interned type identity. Handles are opaque
// process-local implementation details; only type spellings cross text/JSON boundaries.
// unique releases entries when no live HIR/MIR value references them.
type TypeID struct{ handle unique.Handle[typeNode] }

// Type is the type identity shared by semantic analysis, HIR, MIR and backends.
type Type = TypeID

type TypeKind uint8

const (
	TypeKindUnset TypeKind = iota
	TypeKindBuiltin
	TypeKindNominal
	TypeKindBorrow
	TypeKindRawPointer
	TypeKindSlice
	TypeKindFixedArray
	TypeKindOption
	TypeKindResult
	// TypeKindOpaque preserves malformed spellings for stage-owned validation.
	TypeKindOpaque
)

// TypeDescriptor contains direct child identities, never reparsed type text.
// Name is used only for builtins, nominal identities and opaque invalid inputs.
// Descriptor returns a copy; modifying it cannot mutate an interned type.
type TypeDescriptor struct {
	Element TypeID
	Failure TypeID
	Length  uint64
	Name    string
	Kind    TypeKind
	Mutable bool
}

type typeNode struct {
	element TypeID
	failure TypeID
	length  uint64
	// Preserve noncanonical input bytes, so artifact canonicalization and hashes
	// cannot silently accept a spelling rewritten during decoding. For leaf
	// kinds this is the builtin/nominal/opaque name instead.
	text    string
	kind    TypeKind
	mutable bool
}

func internType(d TypeDescriptor) Type {
	return Type{unique.Make(typeNode{element: d.Element, failure: d.Failure, length: d.Length, text: d.Name, kind: d.Kind, mutable: d.Mutable})}
}
func builtin(name string) Type { return internType(TypeDescriptor{Kind: TypeKindBuiltin, Name: name}) }

func (t TypeID) Descriptor() TypeDescriptor {
	n := t.node()
	d := TypeDescriptor{Kind: n.kind, Element: n.element, Failure: n.failure, Length: n.length, Mutable: n.mutable}
	if n.kind == TypeKindBuiltin || n.kind == TypeKindNominal || n.kind == TypeKindOpaque {
		d.Name = n.text
	}
	return d
}

func (t TypeID) node() typeNode {
	if t == (Type{}) {
		return typeNode{}
	}
	return t.handle.Value()
}

func (t TypeID) canonicalCompound() bool { return t != (Type{}) && t.handle.Value().text == "" }

// Slice constructs an unsized element shape; semantic/MIR rules decide whether
// a particular slice can be borrowed or executed.
func Slice(element Type) Type {
	return internType(TypeDescriptor{Kind: TypeKindSlice, Element: element})
}

func (t TypeID) String() string {
	if t == (Type{}) {
		return ""
	}
	n := t.handle.Value()
	if n.text != "" {
		return n.text
	}
	var b strings.Builder
	t.writeTo(&b)
	return b.String()
}

func (t TypeID) writeTo(b *strings.Builder) {
	if t == (Type{}) {
		return
	}
	n := t.handle.Value()
	d := t.Descriptor()
	if n.text != "" {
		b.WriteString(n.text)
		return
	}
	switch d.Kind {
	case TypeKindBorrow:
		b.WriteByte('&')
		if d.Mutable {
			b.WriteString("mut ")
		}
		d.Element.writeTo(b)
	case TypeKindRawPointer:
		b.WriteByte('*')
		if d.Mutable {
			b.WriteString("mut ")
		}
		d.Element.writeTo(b)
	case TypeKindSlice:
		b.WriteByte('[')
		d.Element.writeTo(b)
		b.WriteByte(']')
	case TypeKindFixedArray:
		b.WriteByte('[')
		d.Element.writeTo(b)
		b.WriteString("; ")
		b.WriteString(strconv.FormatUint(d.Length, 10))
		b.WriteByte(']')
	case TypeKindOption:
		b.WriteString("Option<")
		d.Element.writeTo(b)
		b.WriteByte('>')
	case TypeKindResult:
		b.WriteString("Result<")
		d.Element.writeTo(b)
		b.WriteString(", ")
		d.Failure.writeTo(b)
		b.WriteByte('>')
	default:
		b.WriteString(d.Name)
	}
}

func (t TypeID) MarshalText() ([]byte, error)     { return []byte(t.String()), nil }
func (t *TypeID) UnmarshalText(data []byte) error { *t = ParseType(string(data)); return nil }
func (t TypeID) MarshalJSON() ([]byte, error)     { return json.Marshal(t.String()) }
func (t *TypeID) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	// The previous string representation treated null as a no-op.
	if string(data) != "null" {
		*t = ParseType(text)
	}
	return nil
}

// ParseType is a text boundary for fixtures, persisted artifacts and nominal
// names. Compiler code constructs compounds with the structural constructors.
// This is deliberately not a language validator: malformed input remains
// distinguishable and is rejected by the semantic/MIR boundary that owns it.
func ParseType(text string) Type { return parseType(text, 0) }

func parseType(text string, depth int) Type {
	if text == "" {
		return Type{}
	}
	if t, ok := builtinTypesByName[text]; ok {
		return t
	}
	opaque := func() Type { return internType(TypeDescriptor{Kind: TypeKindOpaque, Name: text}) }
	// Untrusted serialized spellings must not create an unbounded recursive parser.
	if depth >= 256 {
		return opaque()
	}
	child := func(s string) Type { return parseType(s, depth+1) }
	var t Type
	switch {
	case strings.HasPrefix(text, "&mut ") && len(text) > 5:
		t = MutableBorrow(child(text[5:]))
	case strings.HasPrefix(text, "&") && len(text) > 1:
		t = SharedBorrow(child(text[1:]))
	case strings.HasPrefix(text, "*mut ") && len(text) > 5:
		t = RawPointer(child(text[5:]), true)
	case strings.HasPrefix(text, "*") && len(text) > 1:
		t = RawPointer(child(text[1:]), false)
	case strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]"):
		parts, ok := typeParts(text[1:len(text)-1], ';')
		if !ok {
			return opaque()
		}
		switch len(parts) {
		case 1:
			if parts[0] == "" {
				return opaque()
			}
			t = Slice(child(parts[0]))
		case 2:
			count, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil || parts[0] == "" {
				return opaque()
			}
			t = FixedArray(child(parts[0]), count)
		default:
			return opaque()
		}
	case strings.HasPrefix(text, "Option<") && strings.HasSuffix(text, ">"):
		parts, ok := typeParts(text[7:len(text)-1], ',')
		if !ok || len(parts) != 1 || parts[0] == "" {
			return opaque()
		}
		t = Option(child(parts[0]))
	case strings.HasPrefix(text, "Result<") && strings.HasSuffix(text, ">"):
		parts, ok := typeParts(text[7:len(text)-1], ',')
		if !ok || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return opaque()
		}
		t = Result(child(parts[0]), child(parts[1]))
	default:
		if strings.Contains(text, "::") && !strings.ContainsAny(text, "[];&<> *\t\r\n,") {
			return internType(TypeDescriptor{Kind: TypeKindNominal, Name: text})
		}
		return opaque()
	}
	if t.String() != text {
		n := t.handle.Value()
		n.text = text
		return Type{unique.Make(n)}
	}
	return t
}

// typeParts splits only at balanced top-level separators. It is used once at
// text admission; decomposition of an existing Type never calls this parser.
func typeParts(text string, separator byte) ([]string, bool) {
	var stack []byte
	var parts []string
	start := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '[', '<':
			stack = append(stack, text[i])
		case ']', '>':
			if len(stack) == 0 {
				return nil, false
			}
			open := stack[len(stack)-1]
			if open == '[' && text[i] != ']' || open == '<' && text[i] != '>' {
				return nil, false
			}
			stack = stack[:len(stack)-1]
		default:
			if text[i] == separator && len(stack) == 0 {
				parts = append(parts, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}
	if len(stack) != 0 {
		return nil, false
	}
	return append(parts, strings.TrimSpace(text[start:])), true
}

var builtinTypesByName = func() map[string]Type {
	values := append(CScalarTypes(), TypeInvalid, TypeVoid, TypeBool, TypeU8, TypeI32, TypeI64, TypeU32, TypeUSize, TypeI8, TypeI16, TypeU16, TypeU64, TypeISize, TypeF32, TypeF64, TypeStr, TypeCStr, TypeRegion)
	result := make(map[string]Type, len(values))
	for _, t := range values {
		result[t.String()] = t
	}
	return result
}()
