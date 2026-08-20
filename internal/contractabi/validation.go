package contractabi

import "fmt"

func validateDocument(document Document) error {
	if document.Schema != SchemaVersion {
		return fmt.Errorf("unsupported contract IDL schema %q", document.Schema)
	}
	if !validSourcePath(document.Module) {
		return fmt.Errorf("invalid contract IDL module %q", document.Module)
	}
	if len(document.Contracts) == 0 && len(document.Events) == 0 {
		return fmt.Errorf("contract IDL contains no contracts or events")
	}

	topLevel := make(map[string]string, len(document.Contracts)+len(document.Events))
	for _, contract := range document.Contracts {
		if err := validateDeclarationName("contract", contract.Name); err != nil {
			return err
		}
		if previous, duplicate := topLevel[contract.Name]; duplicate {
			return fmt.Errorf("duplicate top-level name %q used by %s and contract", contract.Name, previous)
		}
		topLevel[contract.Name] = "contract"
		functions := make(map[string]struct{}, len(contract.Functions))
		for _, function := range contract.Functions {
			if err := validateDeclarationName("contract function", function.Name); err != nil {
				return err
			}
			if _, duplicate := functions[function.Name]; duplicate {
				return fmt.Errorf("duplicate contract function %s::%s", contract.Name, function.Name)
			}
			functions[function.Name] = struct{}{}
			if function.View && function.Payable {
				return fmt.Errorf("contract function %s::%s is both view and payable", contract.Name, function.Name)
			}
			if len(function.Effects) != 1 || function.Effects[0] != externalCallEffect {
				return fmt.Errorf("contract function %s::%s must declare exactly the %s effect", contract.Name, function.Name, externalCallEffect)
			}
			if err := validateABIType(function.Result, true); err != nil {
				return fmt.Errorf("contract function %s::%s result: %w", contract.Name, function.Name, err)
			}
			parameters := make(map[string]struct{}, len(function.Parameters))
			for _, parameter := range function.Parameters {
				if err := validateDeclarationName("parameter", parameter.Name); err != nil {
					return fmt.Errorf("contract function %s::%s: %w", contract.Name, function.Name, err)
				}
				if _, duplicate := parameters[parameter.Name]; duplicate {
					return fmt.Errorf("duplicate parameter %s in contract function %s::%s", parameter.Name, contract.Name, function.Name)
				}
				parameters[parameter.Name] = struct{}{}
				if err := validateABIType(parameter.Type, false); err != nil {
					return fmt.Errorf("contract function %s::%s parameter %s: %w", contract.Name, function.Name, parameter.Name, err)
				}
			}
		}
	}

	for _, event := range document.Events {
		if err := validateDeclarationName("event", event.Name); err != nil {
			return err
		}
		if previous, duplicate := topLevel[event.Name]; duplicate {
			return fmt.Errorf("duplicate top-level name %q used by %s and event", event.Name, previous)
		}
		topLevel[event.Name] = "event"
		fields := make(map[string]struct{}, len(event.Fields))
		for _, field := range event.Fields {
			if err := validateDeclarationName("event field", field.Name); err != nil {
				return fmt.Errorf("event %s: %w", event.Name, err)
			}
			if _, duplicate := fields[field.Name]; duplicate {
				return fmt.Errorf("duplicate field %s in event %s", field.Name, event.Name)
			}
			fields[field.Name] = struct{}{}
			if err := validateABIType(field.Type, false); err != nil {
				return fmt.Errorf("event %s field %s: %w", event.Name, field.Name, err)
			}
		}
	}
	return validateEventCycles(document.Events)
}
