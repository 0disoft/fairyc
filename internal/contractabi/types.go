package contractabi

import (
	"sort"
)

var builtinTypes = map[string]struct{}{
	"void": {}, "bool": {},
	"u8": {}, "u16": {}, "u32": {}, "u64": {}, "u128": {}, "u256": {},
	"i8": {}, "i16": {}, "i32": {}, "i64": {}, "i128": {},
	"String": {}, "Bytes": {},
	"Option": {}, "Result": {},
}

type documentTypeFacts struct {
	externalTypes []string
	usesOption    bool
	usesResult    bool
	usesPayable   bool
	usesU128      bool
	usesU256      bool
	usesI128      bool
}

func inspectDocumentTypes(document Document) documentTypeFacts {
	events := make(map[string]struct{}, len(document.Events))
	for _, event := range document.Events {
		events[event.Name] = struct{}{}
	}
	external := make(map[string]struct{})
	facts := documentTypeFacts{}
	var inspect func(Type)
	inspect = func(value Type) {
		switch value.Kind {
		case TypeNamed:
			switch value.Name {
			case "Option":
				facts.usesOption = true
			case "Result":
				facts.usesResult = true
			case "u128":
				facts.usesU128 = true
			case "u256":
				facts.usesU256 = true
			case "i128":
				facts.usesI128 = true
			default:
				if _, builtin := builtinTypes[value.Name]; !builtin {
					if _, event := events[value.Name]; !event {
						external[value.Name] = struct{}{}
					}
				}
			}
			for _, argument := range value.Arguments {
				inspect(argument)
			}
		case TypeArray, TypeSlice, TypeReference, TypePointer:
			if value.Element != nil {
				inspect(*value.Element)
			}
		}
	}
	for _, contract := range document.Contracts {
		for _, function := range contract.Functions {
			facts.usesPayable = facts.usesPayable || function.Payable
			inspect(function.Result)
			for _, parameter := range function.Parameters {
				inspect(parameter.Type)
			}
		}
	}
	for _, event := range document.Events {
		for _, field := range event.Fields {
			inspect(field.Type)
		}
	}
	facts.externalTypes = make([]string, 0, len(external))
	for value := range external {
		facts.externalTypes = append(facts.externalTypes, value)
	}
	sort.Strings(facts.externalTypes)
	return facts
}
