package genericsource

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
)

// PackageModule supplies resolved ownership and exact import providers. Imports
// maps source module paths to package owners, never aliases or search prefixes.
type PackageModule struct {
	Module  *ast.Module
	Owner   genericplan.PackageID
	Imports map[string]genericplan.PackageID
}

// ExpandPackages specializes one closed, already-resolved source graph in a
// single bounded plan. Generated functions stay in their defining module and
// retain its visibility and imports. Callers must verify the resulting modules
// through ordinary package-aware semantics; this is not an interface or cache.
func ExpandPackages(inputs []PackageModule, limits genericplan.PlanLimits) ([]*ast.Module, genericplan.Plan, error) {
	result, err := PreparePackages(inputs, limits)
	return result.Modules, result.Plan, err
}

// PreparePackages also retains public template identities before specialization
// erases their declarations. No template summary is returned on expansion failure.
func PreparePackages(inputs []PackageModule, limits genericplan.PlanLimits) (PackageExpansion, error) {
	return PrepareWithPolicy(inputs, limits, admission.Policy[genericplan.PackageID]{})
}

// PrepareStoragePackages enables symbolic queries across an exact resolved graph.
// Arguments remain scalar; nominal arguments need cross-owner type binding.
func PrepareStoragePackages(inputs []PackageModule, limits genericplan.PlanLimits) (PackageExpansion, error) {
	return PrepareWithPolicy(inputs, limits, admission.Policy[genericplan.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageQueries}})
}

// PrepareNominalStoragePackages is a private, closed-graph nominal and layout lane.
func PrepareNominalStoragePackages(inputs []PackageModule, limits genericplan.PlanLimits) (PackageExpansion, error) {
	return PrepareWithPolicy(inputs, limits, admission.Policy[genericplan.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageQueries, Arguments: admission.NominalArguments}})
}

// PrepareStorageTransferPackages admits explicit unsafe transfers with exact
// nominal bindings. Ordinary and query-only package preparation remain closed.
func PrepareStorageTransferPackages(inputs []PackageModule, limits genericplan.PlanLimits) (PackageExpansion, error) {
	return PrepareWithPolicy(inputs, limits, admission.Policy[genericplan.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments}})
}

// PrepareOwnedStorageTransferPackages restricts private storage operations to
// explicitly selected package owners. The caller must establish those owners'
// source authority; module names and import aliases never grant permission.
// Ordinary generic/nominal use and calls to the provider remain available.
func PrepareOwnedStorageTransferPackages(inputs []PackageModule, limits genericplan.PlanLimits, owners []genericplan.PackageID) (PackageExpansion, error) {
	return prepareOwnedStorageTransferPackages(inputs, limits, owners, admission.OrdinaryTemplates)
}

// PrepareSDKStoragePackages admits aggregate and nominal-argument templates only
// in authenticated SDK owners. Consumer templates retain scalar-only arguments.
func PrepareSDKStoragePackages(inputs []PackageModule, limits genericplan.PlanLimits, owners []genericplan.PackageID) (PackageExpansion, error) {
	return prepareOwnedStorageTransferPackages(inputs, limits, owners, admission.SDKOwnerTemplates)
}

func prepareOwnedStorageTransferPackages(inputs []PackageModule, limits genericplan.PlanLimits, owners []genericplan.PackageID, templates admission.TemplateMode) (PackageExpansion, error) {
	if len(owners) == 0 {
		return PackageExpansion{}, fmt.Errorf("storage operation owners must be explicit")
	}
	return PrepareWithPolicy(inputs, limits, admission.Policy[genericplan.PackageID]{
		Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: templates}, StorageOwners: owners,
	})
}

// PrepareWithPolicy is the shared resolved-graph expansion boundary. Owner
// authentication belongs to the caller; validation prevents malformed scopes.
func PrepareWithPolicy(inputs []PackageModule, limits genericplan.PlanLimits, policy admission.Policy[genericplan.PackageID]) (PackageExpansion, error) {
	if err := policy.Validate(); err != nil {
		return PackageExpansion{}, err
	}
	capabilities := policy.Capabilities
	var storageOwners map[genericplan.PackageID]bool
	if policy.StorageOwners != nil {
		storageOwners = make(map[genericplan.PackageID]bool, len(policy.StorageOwners))
		graphOwners := make(map[genericplan.PackageID]bool, len(inputs))
		for _, input := range inputs {
			graphOwners[input.Owner] = true
		}
		for _, owner := range policy.StorageOwners {
			if owner == "" {
				return PackageExpansion{}, fmt.Errorf("storage operation owner is empty or duplicated")
			}
			if !graphOwners[owner] {
				return PackageExpansion{}, fmt.Errorf("storage operation owner is absent from source graph")
			}
			storageOwners[owner] = true
		}
	}
	type moduleKey struct {
		owner genericplan.PackageID
		path  string
	}
	modules := make(map[moduleKey]*expander, len(inputs))
	ordered := make([]*expander, len(inputs))
	for i, input := range inputs {
		if input.Module == nil {
			return PackageExpansion{}, fmt.Errorf("generic source module is nil")
		}
		key := moduleKey{input.Owner, input.Module.Path}
		if _, exists := modules[key]; exists {
			return PackageExpansion{}, fmt.Errorf("duplicate generic module %q", input.Module.Path)
		}
		allowStorage := storageOwners == nil || storageOwners[input.Owner]
		x := &expander{module: input.Module, owner: input.Owner, templates: map[string]ast.Function{}, names: map[string]bool{}, imports: map[string]*expander{}, allowPublic: true, allowStorage: capabilities.AllowsStorageQueries() && allowStorage, allowNominal: capabilities.AllowsNominalArguments(), packageNominal: capabilities.AllowsNominalArguments(), allowStorageTransfers: capabilities.AllowsStorageTransfers() && allowStorage}
		x.scalarTemplatesOnly = capabilities.SDKTemplates() && !allowStorage
		modules[key], ordered[i] = x, x
	}
	for i, input := range inputs {
		for _, imported := range input.Module.Imports {
			owner, bound := input.Imports[imported.Path]
			if !bound {
				return PackageExpansion{}, fmt.Errorf("unbound generic module import %q", imported.Path)
			}
			target := modules[moduleKey{owner, imported.Path}]
			if target == nil {
				return PackageExpansion{}, fmt.Errorf("missing generic module provider %q", imported.Path)
			}
			ordered[i].imports[imported.Alias] = target
		}
	}
	if capabilities.SDKTemplates() {
		for _, module := range ordered {
			if err := module.activateSDKVectorLoops(); err != nil {
				return PackageExpansion{}, err
			}
		}
	}
	var templates []PublicTemplate
	if capabilities.AllowsNominalArguments() {
		if err := bindNominalContext(ordered); err != nil {
			return PackageExpansion{}, err
		}
	}
	var declarations *genericplan.Catalog
	expanded, plan, err := expandAll(ordered, limits, func(catalog *genericplan.Catalog) error {
		declarations = catalog
		var err error
		templates, err = publicTemplates(ordered, catalog)
		return err
	})
	if err != nil {
		return PackageExpansion{}, err
	}
	result := PackageExpansion{Modules: expanded, Plan: plan, Templates: templates, Specializations: map[genericplan.PackageID][]genericplan.SpecializationID{}}
	if capabilities.AllowsNominalArguments() && len(ordered) != 0 {
		for _, binding := range ordered[0].boundNominals {
			result.NominalBindings = append(result.NominalBindings, binding)
		}
		slices.SortFunc(result.NominalBindings, func(a, b NominalBinding) int { return strings.Compare(string(a.ID), string(b.ID)) })
	}
	result.Declarations, result.Origins, err = sourceOrigins(ordered, plan)
	if err != nil {
		return PackageExpansion{}, err
	}
	for _, module := range ordered {
		result.Specializations[module.owner] = nil
	}
	for _, specialization := range plan.Specializations {
		result.GeneratedNames = append(result.GeneratedNames, generatedPrefix+string(specialization.ID))
		declaration, exists := declarations.Declaration(specialization.Declaration)
		if !exists {
			return PackageExpansion{}, fmt.Errorf("missing specialization owner")
		}
		result.Specializations[declaration.Owner] = append(result.Specializations[declaration.Owner], specialization.ID)
	}
	for _, ids := range result.Specializations {
		slices.Sort(ids)
	}
	return result, nil
}

func expandAll(modules []*expander, limits genericplan.PlanLimits, observers ...func(*genericplan.Catalog) error) ([]*ast.Module, genericplan.Plan, error) {
	builder, err := genericplan.NewBuilder(genericplan.DefaultCatalogLimits())
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	for _, name := range scalarTypes {
		if err := builder.AddType(genericplan.TypeDescriptor{ID: genericplan.TypeID(name), Owner: "builtin", Abilities: []genericplan.Ability{genericplan.AbilityScalar}, Definition: genericplan.DigestText("source-scalar-v0", name)}); err != nil {
			return nil, genericplan.Plan{}, err
		}
	}
	for _, module := range modules {
		if err := module.collect(); err != nil {
			return nil, genericplan.Plan{}, module.at(module.module.Span, FailureDeclaration, err)
		}
		if err := module.collectNominalTypes(builder); err != nil {
			return nil, genericplan.Plan{}, module.at(module.module.Span, FailureDeclaration, err)
		}
	}
	var requests []genericplan.Request
	bindNominalRegistry(modules)
	for _, module := range modules {
		roots, err := module.declare(builder)
		if err != nil {
			return nil, genericplan.Plan{}, err
		}
		requests = append(requests, roots...)
	}
	catalog, err := builder.Build()
	if err != nil {
		return nil, genericplan.Plan{}, err
	}
	for _, observe := range observers {
		if err := observe(catalog); err != nil {
			return nil, genericplan.Plan{}, err
		}
	}
	plan, err := catalog.Plan(requests, limits)
	if err != nil {
		return nil, genericplan.Plan{}, locatePlanningFailure(modules, err)
	}
	names := map[string]string{}
	for _, specialization := range plan.Specializations {
		arguments := make([]genericplan.TypeID, len(specialization.Bindings))
		for i, binding := range specialization.Bindings {
			arguments[i] = binding.Type
		}
		names[requestKey(specialization.Declaration, arguments)] = generatedPrefix + string(specialization.ID)
	}
	result := make([]*ast.Module, len(modules))
	for i, module := range modules {
		result[i], err = module.emit(plan, names)
		if err != nil {
			return nil, genericplan.Plan{}, err
		}
	}
	return result, plan, nil
}
