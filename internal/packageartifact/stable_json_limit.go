package packageartifact

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

var (
	operandType       = reflect.TypeOf(mir.Operand{})
	hirType           = reflect.TypeOf(hir.Type{})
	base64CountType   = reflect.TypeOf(stableBase64Count(0))
	operationKindType = reflect.TypeOf(mir.OperationKind(0))
	functionIDType    = reflect.TypeOf(FunctionID{})
	contextIDType     = reflect.TypeOf(ContextID{})
	instanceIDType    = reflect.TypeOf(packagegraph.InstanceID{})
	coordinateIDType  = reflect.TypeOf(packagegraph.CoordinateID{})
	stateHashType     = reflect.TypeOf(contractstatemir.StateBodyHash{})
)

var invalidUTF8Extra = func() uint64 { data, _ := json.Marshal("\xff"); return uint64(len(data) - 3) }()

type stableBase64Count int

// stableOperandJSON mirrors mir.Operand's closed custom JSON representation,
// substituting a byte count for StaticDataBase64 so no base64 buffer is built.
type stableOperandJSON struct {
	Kind             mir.OperandKind
	Local            mir.LocalID
	I32              int32
	I64              int64 `json:",omitempty"`
	U64              uint64
	Type             hir.Type
	StaticDataBase64 stableBase64Count
}

// stableJSONExceedsLimit counts canonical JSON bytes for the supported closed
// schemas without constructing JSON. Unsupported schema shapes return an error.
// Invalid operation kinds are checked separately to preserve marshal errors.
func stableJSONExceedsLimit(value any, limit uint64) (bool, error) {
	typ := reflect.TypeOf(value)
	if err := stableJSONSchema(typ); err != nil {
		return false, err
	}
	remaining := limit
	return !stableJSONCount(reflect.ValueOf(value), &remaining), nil
}

var stableJSONSchemaCache sync.Map
var stableJSONMarshaler = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
var stableTextMarshaler = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

// Reject unrecognized wire shapes explicitly; the counter is a closed-schema
// guard, not a generic JSON implementation.
func stableJSONSchema(t reflect.Type) error {
	if cached, ok := stableJSONSchemaCache.Load(t); ok {
		return cached.(stableSchemaResult).err
	}
	err := stableJSONCheckType(t, make(map[reflect.Type]bool))
	stableJSONSchemaCache.Store(t, stableSchemaResult{err})
	return err
}

type stableSchemaResult struct{ err error }

func stableJSONCheckType(t reflect.Type, seen map[reflect.Type]bool) error {
	if t == nil || seen[t] {
		return nil
	}
	seen[t] = true
	switch t {
	case operandType, hirType, base64CountType, operationKindType, functionIDType, contextIDType, instanceIDType, coordinateIDType, stateHashType:
		return nil
	}
	if t.Implements(stableJSONMarshaler) || t.Implements(stableTextMarshaler) || reflect.PointerTo(t).Implements(stableJSONMarshaler) || reflect.PointerTo(t).Implements(stableTextMarshaler) {
		return fmt.Errorf("unsupported stable JSON marshaler %v", t)
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return stableJSONCheckType(t.Elem(), seen)
	case reflect.Struct:
		names := make(map[string]bool)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Anonymous {
				return fmt.Errorf("unsupported anonymous stable JSON field %v.%s", t, field.Name)
			}
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, opts, _ := strings.Cut(tag, ",")
			if name == "" {
				name = field.Name
			}
			for _, opt := range strings.Split(opts, ",") {
				if opt != "" && opt != "omitempty" {
					return fmt.Errorf("unsupported stable JSON option %q", opt)
				}
			}
			if names[name] {
				return fmt.Errorf("duplicate stable JSON field %v.%s", t, name)
			}
			names[name] = true
			if err := stableJSONCheckType(field.Type, seen); err != nil {
				return err
			}
		}
		return nil
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil
	default:
		return fmt.Errorf("unsupported stable JSON type %v", t)
	}
}

func stableJSONTake(remaining *uint64, n uint64) bool {
	if n > *remaining {
		return false
	}
	*remaining -= n
	return true
}

// Count an encoding/json string, including quotes when requested. Invalid UTF-8,
// U+2028/U+2029, HTML characters and short control escapes follow json.Marshal.
func stableJSONString(s string, quoted bool, remaining *uint64) bool {
	if quoted && !stableJSONTake(remaining, 2) {
		return false
	}
	if !stableJSONTake(remaining, uint64(len(s))) {
		return false
	}
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			extra := uint64(0)
			switch c {
			case '\\', '"', '\b', '\f', '\n', '\r', '\t':
				extra = 1
			case '<', '>', '&':
				extra = 5
			default:
				if c < 0x20 {
					extra = 5
				}
			}
			if !stableJSONTake(remaining, extra) {
				return false
			}
			i++
		} else {
			r, n := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && n == 1 {
				if !stableJSONTake(remaining, invalidUTF8Extra) {
					return false
				}
			} else if r == '\u2028' || r == '\u2029' {
				if !stableJSONTake(remaining, 3) {
					return false
				}
			}
			i += n
		}
	}
	return true
}

func stableJSONCount(v reflect.Value, remaining *uint64) bool {
	if !v.IsValid() {
		return stableJSONTake(remaining, 4)
	}
	typ := v.Type()
	switch typ {
	case operandType:
		op := v.Interface().(mir.Operand)
		return stableJSONCount(reflect.ValueOf(stableOperandJSON{op.Kind, op.Local, op.I32, op.I64, op.U64, op.Type, stableBase64Count(len(op.StaticData))}), remaining)
	case hirType:
		return stableJSONTake(remaining, 2) && stableTypeJSONCount(v.Interface().(hir.Type), remaining)
	case base64CountType:
		n := uint64(v.Int())
		return stableJSONTake(remaining, 2) && stableJSONTake(remaining, (n/3)*4+((n%3+2)/3)*4)
	case functionIDType:
		return stableJSONTake(remaining, uint64(2+len(FunctionIDPrefix)+64))
	case contextIDType:
		return stableJSONTake(remaining, uint64(2+len(ContextIDPrefix)+64))
	case instanceIDType:
		return stableJSONTake(remaining, uint64(2+len(packagegraph.InstanceIDPrefix)+64))
	case coordinateIDType:
		return stableJSONTake(remaining, uint64(2+len(packagegraph.CoordinateIDPrefix)+64))
	case stateHashType:
		return stableJSONTake(remaining, uint64(2+len(contractstatemir.StateBodyHashPrefix)+64))
	case operationKindType:
		k := v.Interface().(mir.OperationKind)
		if k != mir.OperationInvalid && !k.Valid() {
			return true
		} // error is checked independently
		return stableJSONString(k.String(), true, remaining)
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return stableJSONTake(remaining, 4)
		}
		return stableJSONCount(v.Elem(), remaining)
	case reflect.String:
		return stableJSONString(v.String(), true, remaining)
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return stableJSONTake(remaining, 4)
		}
		n := v.Len()
		if v.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 {
			return stableJSONCount(reflect.ValueOf(stableBase64Count(n)), remaining)
		}
		if !stableJSONTake(remaining, 2) {
			return false
		}
		if n > 0 && !stableJSONTake(remaining, uint64(n-1)) {
			return false
		}
		for i := 0; i < n; i++ {
			if !stableJSONCount(v.Index(i), remaining) {
				return false
			}
		}
	case reflect.Struct:
		if !stableJSONTake(remaining, 2) {
			return false
		}
		fields := 0
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, options, _ := strings.Cut(tag, ",")
			if name == "" {
				name = field.Name
			}
			value := v.Field(i)
			if strings.Contains(","+options+",", ",omitempty,") && stableJSONEmpty(value) {
				continue
			}
			if fields > 0 && !stableJSONTake(remaining, 1) {
				return false
			}
			fields++
			if !stableJSONString(name, true, remaining) || !stableJSONTake(remaining, 1) || !stableJSONCount(value, remaining) {
				return false
			}
		}
	case reflect.Bool:
		if v.Bool() {
			return stableJSONTake(remaining, 4)
		}
		return stableJSONTake(remaining, 5)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var buf [32]byte
		return stableJSONTake(remaining, uint64(len(strconv.AppendInt(buf[:0], v.Int(), 10))))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var buf [32]byte
		return stableJSONTake(remaining, uint64(len(strconv.AppendUint(buf[:0], v.Uint(), 10))))
	default:
		// The schema check rejects all other kinds before counting.
		return false
	}
	return true
}

func stableJSONEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Interface, reflect.Pointer:
		return v.IsZero()
	}
	return false
}

// Count the exact type spelling, including retained noncanonical text.
func stableTypeJSONCount(t hir.Type, remaining *uint64) bool {
	return t.VisitStringChunks(func(chunk string) bool { return stableJSONString(chunk, false, remaining) })
}

// OperationKind is the only fallible custom marshaler in the stable schemas.
func stableJSONOperationError(functions []StableBodyArtifact) error {
	for _, body := range functions {
		for _, block := range body.Function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind != mir.OperationInvalid && !operation.Kind.Valid() {
					_, err := json.Marshal(operation.Kind)
					return err
				}
			}
		}
	}
	return nil
}
