package contractabi

import "fmt"

func validateGoBindingNames(document Document) error {
	facts := inspectDocumentTypes(document)
	packageNames := map[string]string{
		"ContractCallRequest": "transport request",
		"ContractTransport":   "transport interface",
	}
	add := func(name, owner string) error {
		if previous, exists := packageNames[name]; exists {
			return fmt.Errorf("generated Go identifier %s collides between %s and %s", name, previous, owner)
		}
		packageNames[name] = owner
		return nil
	}
	if facts.usesPayable {
		if err := add("PayableCallOptions", "payable options"); err != nil {
			return err
		}
	}
	for name, used := range map[string]bool{
		"FairyOption": facts.usesOption,
		"FairyResult": facts.usesResult,
		"FairyU128":   facts.usesU128,
		"FairyU256":   facts.usesU256,
		"FairyI128":   facts.usesI128,
	} {
		if used {
			if err := add(name, "built-in ABI helper"); err != nil {
				return err
			}
		}
	}
	for _, path := range facts.externalTypes {
		if err := add(externalTypeName(path), "nominal type "+path); err != nil {
			return err
		}
	}
	for _, event := range document.Events {
		name := exportedName(event.Name) + "Event"
		if err := add(name, "event "+event.Name); err != nil {
			return err
		}
		fields := make(map[string]string, len(event.Fields))
		for _, field := range event.Fields {
			generated := exportedName(field.Name)
			if previous, duplicate := fields[generated]; duplicate {
				return fmt.Errorf("generated Go field %s collides between %s and %s in event %s", generated, previous, field.Name, event.Name)
			}
			fields[generated] = field.Name
		}
	}
	for _, contract := range document.Contracts {
		client := exportedName(contract.Name) + "Client"
		if err := add(client, "contract "+contract.Name); err != nil {
			return err
		}
		if err := add("New"+client, "constructor for contract "+contract.Name); err != nil {
			return err
		}
		methods := make(map[string]string, len(contract.Functions))
		for _, function := range contract.Functions {
			generated := exportedName(function.Name)
			if previous, duplicate := methods[generated]; duplicate {
				return fmt.Errorf("generated Go method %s collides between %s and %s in contract %s", generated, previous, function.Name, contract.Name)
			}
			methods[generated] = function.Name
		}
	}
	return nil
}

func validateTypeScriptBindingNames(document Document) error {
	facts := inspectDocumentTypes(document)
	typeNames := map[string]string{
		"ContractCallRequest": "transport request",
		"ContractTransport":   "transport interface",
	}
	add := func(name, owner string) error {
		if previous, exists := typeNames[name]; exists {
			return fmt.Errorf("generated TypeScript identifier %s collides between %s and %s", name, previous, owner)
		}
		typeNames[name] = owner
		return nil
	}
	for name, used := range map[string]bool{
		"PayableCallOptions": facts.usesPayable,
		"Option":             facts.usesOption,
		"Result":             facts.usesResult,
	} {
		if used {
			if err := add(name, "built-in ABI helper"); err != nil {
				return err
			}
		}
	}
	for _, path := range facts.externalTypes {
		if err := add(externalTypeName(path), "nominal type "+path); err != nil {
			return err
		}
	}
	for _, event := range document.Events {
		if err := add(exportedName(event.Name)+"Event", "event "+event.Name); err != nil {
			return err
		}
		fields := make(map[string]string, len(event.Fields))
		for _, field := range event.Fields {
			generated := tsLocalName(field.Name)
			if previous, duplicate := fields[generated]; duplicate {
				return fmt.Errorf("generated TypeScript field %s collides between %s and %s in event %s", generated, previous, field.Name, event.Name)
			}
			fields[generated] = field.Name
		}
	}
	for _, contract := range document.Contracts {
		if err := add(exportedName(contract.Name)+"Client", "contract "+contract.Name); err != nil {
			return err
		}
		methods := make(map[string]string, len(contract.Functions))
		for _, function := range contract.Functions {
			generated := tsLocalName(function.Name)
			if previous, duplicate := methods[generated]; duplicate {
				return fmt.Errorf("generated TypeScript method %s collides between %s and %s in contract %s", generated, previous, function.Name, contract.Name)
			}
			methods[generated] = function.Name
		}
	}
	return nil
}
