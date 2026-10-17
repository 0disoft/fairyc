package cli

import (
	"fmt"
	"strings"
)

type flagKind uint8

const (
	flagSwitch flagKind = iota
	flagScalar
	flagSlice
)

// flagDefinition owns token shape and multiplicity. ValueFirst preserves the
// stable test command's missing-value-before-duplicate diagnostic precedence.
type flagDefinition struct {
	Name           string
	Kind           flagKind
	ValueFirst     bool
	Missing        string
	DuplicateValue string
}

type flagSchema struct {
	Command       string
	Experimental  bool
	Flags         []flagDefinition
	Required      []string
	Unknown       func(string) error
	RequiredError func(string) error
	// Evidence queries extract their own flags before parsing relation flags.
	// Forward unknown tokens individually, retaining the original two-pass order.
	Forward *[]string
}

func (s flagSchema) label() string {
	if s.Experimental {
		return fmt.Sprintf("experimental command %q", s.Command)
	}
	return fmt.Sprintf("command %q", s.Command)
}

// parseCommandFlags consumes exact flag tokens only. It never accepts --x=value,
// short aliases, implicit booleans, or a terminator unless the owning command
// explicitly handles them. Apply runs in source order before any later token or
// required-flag check, preserving domain-validation diagnostic precedence.
func parseCommandFlags(arguments []string, schema flagSchema, apply func(string, string) error) (map[string]bool, error) {
	definitions := make(map[string]flagDefinition, len(schema.Flags))
	for _, d := range schema.Flags {
		definitions[d.Name] = d
	}
	seen := make(map[string]bool, len(definitions))
	values := make(map[string]map[string]bool)
	for index := 0; index < len(arguments); {
		name := arguments[index]
		definition, found := definitions[name]
		if !found {
			if schema.Forward != nil {
				*schema.Forward = append(*schema.Forward, name)
				index++
				continue
			}
			if schema.Unknown != nil {
				return nil, schema.Unknown(name)
			}
			return nil, fmt.Errorf("%s does not accept flag %q", schema.label(), name)
		}
		missing := definition.Kind != flagSwitch && (index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--"))
		if missing && definition.ValueFirst {
			return nil, missingFlagValue(schema, definition)
		}
		if definition.Kind != flagSlice && seen[name] {
			return nil, fmt.Errorf("%s received %s more than once", schema.label(), name)
		}
		if missing {
			return nil, missingFlagValue(schema, definition)
		}
		value := ""
		if definition.Kind != flagSwitch {
			value = arguments[index+1]
			index++
		}
		if definition.DuplicateValue != "" {
			if values[name] == nil {
				values[name] = make(map[string]bool)
			}
			if values[name][value] {
				return nil, fmt.Errorf(definition.DuplicateValue, value)
			}
			values[name][value] = true
		}
		seen[name] = true
		if err := apply(name, value); err != nil {
			return nil, err
		}
		index++
	}
	for _, name := range schema.Required {
		if !seen[name] {
			if schema.RequiredError != nil {
				return nil, schema.RequiredError(name)
			}
			return nil, fmt.Errorf("%s requires %s", schema.label(), name)
		}
	}
	return seen, nil
}

func missingFlagValue(schema flagSchema, definition flagDefinition) error {
	if definition.Missing != "" {
		return fmt.Errorf("%s", definition.Missing)
	}
	return fmt.Errorf("%s requires a value for %s", schema.label(), definition.Name)
}

// Ordered rules describe mutual exclusion, dependencies and value relationships.
// Conditions are evaluated by the command after flag application; presence and
// a nonempty value intentionally remain distinct for existing empty-value rules.
type flagRule struct {
	Invalid bool
	Message string
}

func validateFlagRules(rules ...flagRule) error {
	for _, rule := range rules {
		if rule.Invalid {
			return fmt.Errorf("%s", rule.Message)
		}
	}
	return nil
}
