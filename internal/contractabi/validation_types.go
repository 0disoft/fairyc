package contractabi

import (
	"fmt"
	"strings"
)

func validateABIType(value Type, allowVoid bool) error {
	switch value.Kind {
	case TypeArray:
		if value.Element == nil || value.Length == nil {
			return fmt.Errorf("array type is incomplete")
		}
		if *value.Length == 0 || *value.Length > maxArrayLength {
			return fmt.Errorf("array length must be in 1..%d", maxArrayLength)
		}
		return validateABIType(*value.Element, false)
	case TypeNamed:
		if !validSourcePath(value.Name) {
			return fmt.Errorf("invalid nominal type path %q", value.Name)
		}
		if value.Name == "void" {
			if !allowVoid {
				return fmt.Errorf("void is valid only as a function result or Result success payload")
			}
			if len(value.Arguments) != 0 {
				return fmt.Errorf("void does not accept type arguments")
			}
			return nil
		}
		if value.Name == "usize" || value.Name == "isize" || value.Name == "str" || strings.HasPrefix(value.Name, "c::") {
			return fmt.Errorf("target-dependent or borrowed type %s is not chain-neutral", value.Name)
		}
		switch value.Name {
		case "Option":
			if len(value.Arguments) != 1 {
				return fmt.Errorf("Option requires exactly one type argument")
			}
			return validateABIType(value.Arguments[0], false)
		case "Result":
			if len(value.Arguments) != 2 {
				return fmt.Errorf("Result requires exactly two type arguments")
			}
			if err := validateABIType(value.Arguments[0], true); err != nil {
				return err
			}
			return validateABIType(value.Arguments[1], false)
		default:
			if len(value.Arguments) != 0 {
				return fmt.Errorf("generic ABI type %s is unsupported", value.Name)
			}
			return nil
		}
	case TypeReference:
		return fmt.Errorf("safe references cannot cross a contract boundary")
	case TypePointer:
		return fmt.Errorf("raw pointers cannot cross a contract boundary")
	case TypeSlice:
		return fmt.Errorf("unbounded slices cannot cross a contract boundary")
	default:
		return fmt.Errorf("invalid ABI type kind %q", value.Kind)
	}
}

func validateEventCycles(events []Event) error {
	byName := make(map[string]Event, len(events))
	for _, event := range events {
		byName[event.Name] = event
	}
	state := make(map[string]uint8, len(events))
	var visitEvent func(string) error
	var visitType func(Type) error
	visitType = func(value Type) error {
		switch value.Kind {
		case TypeNamed:
			if _, event := byName[value.Name]; event {
				if err := visitEvent(value.Name); err != nil {
					return err
				}
			}
			for _, argument := range value.Arguments {
				if err := visitType(argument); err != nil {
					return err
				}
			}
		case TypeArray, TypeSlice, TypeReference, TypePointer:
			if value.Element != nil {
				return visitType(*value.Element)
			}
		}
		return nil
	}
	visitEvent = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("recursive event type cycle includes %s", name)
		case 2:
			return nil
		}
		state[name] = 1
		for _, field := range byName[name].Fields {
			if err := visitType(field.Type); err != nil {
				return err
			}
		}
		state[name] = 2
		return nil
	}
	for _, event := range events {
		if err := visitEvent(event.Name); err != nil {
			return err
		}
	}
	return nil
}

func validateDeclarationName(kind, value string) error {
	if !validSourceIdentifier(value) {
		return fmt.Errorf("invalid %s name %q", kind, value)
	}
	return nil
}

func validSourceIdentifier(value string) bool {
	if value == "" || value == "_" || !isIdentifierStart(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !isIdentifierContinue(value[index]) {
			return false
		}
	}
	_, reserved := reservedWords[value]
	return !reserved
}

func validSourcePath(value string) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, "::") {
		if !validSourceIdentifier(part) {
			return false
		}
	}
	return true
}
