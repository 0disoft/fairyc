package contractstatefixture

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

// HIRCase exposes independent authored inputs, not oracle execution outputs, to
// native acceptance consumers. Every call owns fresh HIR and encoded bytes.
type HIRCase struct {
	Name   string
	Input  hir.Package
	Limits contractstate.Limits
}

func LoadHIRCases(path string) ([]HIRCase, error) {
	definition, err := loadManifest(path)
	if err != nil {
		return nil, err
	}
	if err := validateManifest(definition); err != nil {
		return nil, err
	}
	result := make([]HIRCase, 0, len(definition.Cases))
	for _, fixture := range definition.Cases {
		input, err := buildHIRPackage(definition.Layout, fixture)
		if err != nil {
			return nil, fmt.Errorf("fixture %s: %w", fixture.ID, err)
		}
		limits := definition.Limits
		if fixture.Limits != nil {
			limits = *fixture.Limits
		}
		result = append(result, HIRCase{Name: fixture.ID, Input: input, Limits: contractstate.Limits{
			MaxWrites: limits.MaxWrites, MaxWriteBytes: limits.MaxWriteBytes, MaxValueBytes: limits.MaxValueBytes, MaxMapKeyBytes: limits.MaxMapKeyBytes,
		}})
	}
	return result, nil
}
