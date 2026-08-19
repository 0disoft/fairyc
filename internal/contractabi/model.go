// Package contractabi owns the experimental, chain-neutral Fairy C contract interface profile.
package contractabi

import (
	"fmt"
	"strings"
)

// SchemaVersion identifies the deterministic JSON contract emitted by RenderJSON.
const SchemaVersion = "fairyc.contract-idl/v0"

const externalCallEffect = "external_call"

// Document is one parsed contract-interface module.
type Document struct {
	Schema    string     `json:"schema"`
	Module    string     `json:"module"`
	Contracts []Contract `json:"contracts"`
	Events    []Event    `json:"events"`
}

// Contract names one external contract and its ordered callable interface.
type Contract struct {
	Name      string     `json:"name"`
	Functions []Function `json:"functions"`
}

// Function is one typed external call declaration.
type Function struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	Result     Type        `json:"result"`
	View       bool        `json:"view"`
	Payable    bool        `json:"payable"`
	Effects    []string    `json:"effects"`
}

// Parameter is one ordered contract-call argument.
type Parameter struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

// Event is one event record selected with @event.
type Event struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Field is one ordered event payload member.
type Field struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

// Type is the chain-neutral structural type vocabulary. A chain adapter selects the concrete wire
// encoding; the IDL never bakes EVM ABI, BCS, Borsh, SCALE, or another codec into source syntax.
type Type struct {
	Kind      string  `json:"kind"`
	Name      string  `json:"name,omitempty"`
	Arguments []Type  `json:"arguments,omitempty"`
	Element   *Type   `json:"element,omitempty"`
	Length    *uint64 `json:"length,omitempty"`

	span sourceSpan
}

const (
	TypeNamed     = "named"
	TypeArray     = "array"
	TypeSlice     = "slice"
	TypeReference = "reference"
	TypePointer   = "pointer"
)

// String returns one canonical, whitespace-free source spelling.
func (t Type) String() string {
	switch t.Kind {
	case TypeNamed:
		if len(t.Arguments) == 0 {
			return t.Name
		}
		values := make([]string, len(t.Arguments))
		for index := range t.Arguments {
			values[index] = t.Arguments[index].String()
		}
		return t.Name + "<" + strings.Join(values, ",") + ">"
	case TypeArray:
		if t.Element == nil || t.Length == nil {
			return "<invalid-array>"
		}
		return fmt.Sprintf("%s[%d]", t.Element.String(), *t.Length)
	case TypeSlice:
		if t.Element == nil {
			return "<invalid-slice>"
		}
		return t.Element.String() + "[]"
	case TypeReference:
		if t.Element == nil {
			return "ref <invalid>"
		}
		return "ref " + t.Element.String()
	case TypePointer:
		if t.Element == nil {
			return "ptr <invalid>"
		}
		return "ptr " + t.Element.String()
	default:
		return "<invalid>"
	}
}

type sourceSpan struct {
	start int
	end   int
}

type parsedAttribute struct {
	name string
	span sourceSpan
}

type parsedFunction struct {
	Function
	span     sourceSpan
	nameSpan sourceSpan
	attrs    []parsedAttribute
}

type parsedContract struct {
	Contract
	span     sourceSpan
	nameSpan sourceSpan
	methods  []parsedFunction
}

type parsedField struct {
	Field
	span     sourceSpan
	nameSpan sourceSpan
}

type parsedEvent struct {
	Event
	span     sourceSpan
	nameSpan sourceSpan
	fields   []parsedField
}
