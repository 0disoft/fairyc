package contractstatelayoutfixture

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/0disoft/fairyc/internal/contractstate"
)

const fixtureSchemaV0 = "fairyc.contract-state-layout-fixtures/v0"

const (
	expectAccepted = "accepted"
	expectRejected = "rejected"
)

type fixtureLayout struct {
	layout contractstate.Layout
	cells  map[string]contractstate.EncodedCell
	maps   map[string]contractstate.EncodedMap
}

// Run executes every candidate layout against a fresh store while one
// transaction remains open on the base layout.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if err := validateManifest(definition); err != nil {
		return Report{}, err
	}

	report := Report{CasesRun: len(definition.Cases)}
	for _, fixture := range definition.Cases {
		if failure := runCase(definition, fixture); failure != nil {
			report.Failures = append(report.Failures, *failure)
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	return definition, nil
}

func validateManifest(definition manifest) error {
	if definition.Schema != "./schema-v0.json" || definition.SchemaVersion != fixtureSchemaV0 || len(definition.Contracts) == 0 {
		return fmt.Errorf("invalid contract-state layout fixture header")
	}
	if definition.Limits.MaxWrites <= 0 || definition.Limits.MaxWriteBytes == 0 || definition.Limits.MaxValueBytes <= 0 || definition.Limits.MaxMapKeyBytes <= 0 {
		return fmt.Errorf("contract-state layout fixture limits must be positive")
	}
	if _, err := buildLayout(definition.BaseLayout); err != nil {
		return fmt.Errorf("base layout: %w", err)
	}
	if err := validateWrites(definition.BaseLayout, definition.Seed); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	if err := validateWrites(definition.BaseLayout, definition.Pending); err != nil {
		return fmt.Errorf("pending: %w", err)
	}
	if len(definition.Seed) == 0 || len(definition.Pending) == 0 {
		return fmt.Errorf("contract-state layout fixture requires seed and pending writes")
	}

	seenCases := make(map[string]struct{}, len(definition.Cases))
	accepted := 0
	for _, fixture := range definition.Cases {
		if fixture.ID == "" || (fixture.Expect != expectAccepted && fixture.Expect != expectRejected) {
			return fmt.Errorf("contract-state layout fixture has an invalid case")
		}
		if _, duplicate := seenCases[fixture.ID]; duplicate {
			return fmt.Errorf("contract-state layout fixture case %q is duplicated", fixture.ID)
		}
		seenCases[fixture.ID] = struct{}{}
		if _, err := buildLayout(fixture.Candidate); err != nil {
			return fmt.Errorf("case %q candidate: %w", fixture.ID, err)
		}
		if fixture.Expect == expectAccepted {
			accepted++
		}
	}
	if len(definition.Cases) == 0 || accepted == 0 {
		return fmt.Errorf("contract-state layout fixture requires cases and one accepted upgrade")
	}
	return nil
}

func validateWrites(layout layoutDefinition, writes []writeDefinition) error {
	fields := make(map[string]fieldDefinition, len(layout.Fields))
	for _, field := range layout.Fields {
		fields[field.Name] = field
	}
	seen := make(map[string]struct{}, len(writes))
	for _, write := range writes {
		field, found := fields[write.Field]
		if !found {
			return fmt.Errorf("write names unknown field %q", write.Field)
		}
		key, err := decodeHex(write.KeyHex)
		if err != nil {
			return fmt.Errorf("field %q key: %w", write.Field, err)
		}
		value, err := decodeRequiredHex(write.ValueHex)
		if err != nil {
			return fmt.Errorf("field %q value: %w", write.Field, err)
		}
		if field.Kind == "cell" {
			if len(key) != 0 {
				return fmt.Errorf("cell %q cannot declare a key", write.Field)
			}
		} else {
			if len(key) == 0 {
				return fmt.Errorf("map %q requires a key", write.Field)
			}
			if err := contractstate.ValidateCanonicalEncoding(field.KeyTypeID, key); err != nil {
				return fmt.Errorf("map %q key: %w", write.Field, err)
			}
		}
		if err := contractstate.ValidateCanonicalEncoding(field.ValueTypeID, value); err != nil {
			return fmt.Errorf("field %q value: %w", write.Field, err)
		}
		identity := operationIdentity(write)
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf("write %q is duplicated", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func runCase(definition manifest, fixture caseDefinition) *Failure {
	base, err := buildLayout(definition.BaseLayout)
	if err != nil {
		return failure(fixture.ID, "build base layout: %v", err)
	}
	candidate, err := buildLayout(fixture.Candidate)
	if err != nil {
		return failure(fixture.ID, "build candidate layout: %v", err)
	}
	limits := contractstate.Limits{
		MaxWrites:      definition.Limits.MaxWrites,
		MaxWriteBytes:  definition.Limits.MaxWriteBytes,
		MaxValueBytes:  definition.Limits.MaxValueBytes,
		MaxMapKeyBytes: definition.Limits.MaxMapKeyBytes,
	}
	store, err := contractstate.NewMemoryStore(base.layout)
	if err != nil {
		return failure(fixture.ID, "construct store: %v", err)
	}
	if _, err := contractstate.Execute(store, base.layout, limits, func(transaction *contractstate.Transaction) error {
		return applyWrites(transaction, base, definition.Seed)
	}); err != nil {
		return failure(fixture.ID, "seed store: %v", err)
	}
	baseVersion := store.Version()
	baseDump := store.Dump()

	openTransaction, err := store.Begin(base.layout, limits)
	if err != nil {
		return failure(fixture.ID, "begin old transaction: %v", err)
	}
	if err := applyWrites(openTransaction, base, definition.Pending); err != nil {
		return failure(fixture.ID, "prepare old transaction: %v", err)
	}

	extendsErr := candidate.layout.Extends(base.layout)
	upgradeErr := store.Upgrade(candidate.layout)
	if fixture.Expect == expectAccepted {
		return verifyAcceptedUpgrade(
			fixture.ID,
			store,
			base,
			candidate,
			openTransaction,
			limits,
			definition.Seed,
			definition.Pending,
			baseVersion,
			baseDump,
			extendsErr,
			upgradeErr,
		)
	}
	return verifyRejectedUpgrade(
		fixture.ID,
		store,
		base,
		openTransaction,
		limits,
		definition.Seed,
		definition.Pending,
		baseVersion,
		baseDump,
		extendsErr,
		upgradeErr,
	)
}

func verifyAcceptedUpgrade(
	caseID string,
	store *contractstate.MemoryStore,
	base fixtureLayout,
	candidate fixtureLayout,
	openTransaction *contractstate.Transaction,
	limits contractstate.Limits,
	seed []writeDefinition,
	pending []writeDefinition,
	baseVersion uint64,
	baseDump []contractstate.Entry,
	extendsErr error,
	upgradeErr error,
) *Failure {
	if extendsErr != nil || upgradeErr != nil {
		return failure(caseID, "append-only upgrade rejected: Extends=%v Upgrade=%v", extendsErr, upgradeErr)
	}
	if candidate.layout.Hash() == base.layout.Hash() {
		return failure(caseID, "accepted candidate did not change the layout hash")
	}
	baseFields := base.layout.Fields()
	candidateFields := candidate.layout.Fields()
	if len(candidateFields) <= len(baseFields) {
		return failure(caseID, "accepted candidate did not append a field")
	}
	for index, field := range baseFields {
		if candidateFields[index] != field {
			return failure(caseID, "field %d identity changed from %+v to %+v", index, field, candidateFields[index])
		}
	}
	if store.Version() != baseVersion+1 {
		return failure(caseID, "upgrade version = %d, want %d", store.Version(), baseVersion+1)
	}
	if store.Layout().Hash() != candidate.layout.Hash() {
		return failure(caseID, "store did not publish the candidate layout")
	}
	if !reflect.DeepEqual(store.Dump(), baseDump) {
		return failure(caseID, "upgrade changed existing state bytes")
	}

	if receipt, err := openTransaction.Commit(); !errors.Is(err, contractstate.ErrLayoutMismatch) {
		return failure(caseID, "old transaction commit = %+v, %v; want ErrLayoutMismatch", receipt, err)
	}
	if store.Version() != baseVersion+1 || !reflect.DeepEqual(store.Dump(), baseDump) {
		return failure(caseID, "failed old commit changed version or state")
	}
	if err := verifyWrites(store, candidate, limits, seed); err != nil {
		return failure(caseID, "fresh reader: %v", err)
	}
	if err := verifyUnseededWritesAbsent(store, candidate, limits, seed, pending); err != nil {
		return failure(caseID, "stale overlay: %v", err)
	}
	return nil
}

func verifyRejectedUpgrade(
	caseID string,
	store *contractstate.MemoryStore,
	base fixtureLayout,
	openTransaction *contractstate.Transaction,
	limits contractstate.Limits,
	seed []writeDefinition,
	pending []writeDefinition,
	baseVersion uint64,
	baseDump []contractstate.Entry,
	extendsErr error,
	upgradeErr error,
) *Failure {
	if !errors.Is(extendsErr, contractstate.ErrIncompatibleLayout) || !errors.Is(upgradeErr, contractstate.ErrIncompatibleLayout) {
		return failure(caseID, "breaking upgrade result: Extends=%v Upgrade=%v", extendsErr, upgradeErr)
	}
	if store.Version() != baseVersion || store.Layout().Hash() != base.layout.Hash() || !reflect.DeepEqual(store.Dump(), baseDump) {
		return failure(caseID, "rejected upgrade mutated layout, version, or bytes")
	}
	if _, err := openTransaction.Commit(); err != nil {
		return failure(caseID, "old transaction after rejected upgrade: %v", err)
	}
	if store.Version() != baseVersion+1 {
		return failure(caseID, "old transaction version = %d, want %d", store.Version(), baseVersion+1)
	}
	if err := verifyWrites(store, base, limits, mergeWrites(seed, pending)); err != nil {
		return failure(caseID, "committed old transaction: %v", err)
	}
	return nil
}

func buildLayout(definition layoutDefinition) (fixtureLayout, error) {
	builder, err := contractstate.NewLayoutBuilder(definition.ContractUID)
	if err != nil {
		return fixtureLayout{}, err
	}
	cells := make(map[string]contractstate.EncodedCell, len(definition.Fields))
	maps := make(map[string]contractstate.EncodedMap, len(definition.Fields))
	for _, field := range definition.Fields {
		if field.Name == "" || field.ValueTypeID == "" {
			return fixtureLayout{}, fmt.Errorf("layout field has an empty identity")
		}
		switch field.Kind {
		case "cell":
			if field.KeyTypeID != "" {
				return fixtureLayout{}, fmt.Errorf("cell %q cannot declare a key type", field.Name)
			}
			cell, err := contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
			if err != nil {
				return fixtureLayout{}, err
			}
			cells[field.Name] = cell
		case "map":
			if field.KeyTypeID == "" {
				return fixtureLayout{}, fmt.Errorf("map %q requires a key type", field.Name)
			}
			mapped, err := contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
			if err != nil {
				return fixtureLayout{}, err
			}
			maps[field.Name] = mapped
		default:
			return fixtureLayout{}, fmt.Errorf("field %q has unsupported kind %q", field.Name, field.Kind)
		}
	}
	if len(definition.Fields) == 0 {
		return fixtureLayout{}, fmt.Errorf("layout has no fields")
	}
	layout, err := builder.Build()
	if err != nil {
		return fixtureLayout{}, err
	}
	return fixtureLayout{layout: layout, cells: cells, maps: maps}, nil
}

func applyWrites(transaction *contractstate.Transaction, layout fixtureLayout, writes []writeDefinition) error {
	for _, write := range writes {
		value, err := decodeRequiredHex(write.ValueHex)
		if err != nil {
			return err
		}
		if cell, found := layout.cells[write.Field]; found {
			if err := cell.Set(transaction, value); err != nil {
				return err
			}
			continue
		}
		mapped, found := layout.maps[write.Field]
		if !found {
			return fmt.Errorf("unknown field %q", write.Field)
		}
		key, err := decodeRequiredHex(write.KeyHex)
		if err != nil {
			return err
		}
		if err := mapped.Set(transaction, key, value); err != nil {
			return err
		}
	}
	return nil
}

func verifyWrites(store *contractstate.MemoryStore, layout fixtureLayout, limits contractstate.Limits, writes []writeDefinition) error {
	return contractstate.View(store, layout.layout, limits, func(transaction *contractstate.Transaction) error {
		for _, write := range writes {
			want, err := decodeRequiredHex(write.ValueHex)
			if err != nil {
				return err
			}
			var got []byte
			var found bool
			if cell, ok := layout.cells[write.Field]; ok {
				got, found, err = cell.Get(transaction)
			} else if mapped, ok := layout.maps[write.Field]; ok {
				var key []byte
				key, err = decodeRequiredHex(write.KeyHex)
				if err == nil {
					got, found, err = mapped.Get(transaction, key)
				}
			} else {
				return fmt.Errorf("unknown field %q", write.Field)
			}
			if err != nil {
				return err
			}
			if !found || !bytes.Equal(got, want) {
				return fmt.Errorf("%s = %x, %v; want %x", operationIdentity(write), got, found, want)
			}
		}
		return nil
	})
}

func verifyUnseededWritesAbsent(
	store *contractstate.MemoryStore,
	layout fixtureLayout,
	limits contractstate.Limits,
	seed []writeDefinition,
	pending []writeDefinition,
) error {
	seeded := make(map[string]struct{}, len(seed))
	for _, write := range seed {
		seeded[operationIdentity(write)] = struct{}{}
	}
	return contractstate.View(store, layout.layout, limits, func(transaction *contractstate.Transaction) error {
		for _, write := range pending {
			if _, existed := seeded[operationIdentity(write)]; existed {
				continue
			}
			var found bool
			var err error
			if cell, ok := layout.cells[write.Field]; ok {
				_, found, err = cell.Get(transaction)
			} else if mapped, ok := layout.maps[write.Field]; ok {
				key, decodeErr := decodeRequiredHex(write.KeyHex)
				if decodeErr != nil {
					return decodeErr
				}
				_, found, err = mapped.Get(transaction, key)
			} else {
				return fmt.Errorf("unknown field %q", write.Field)
			}
			if err != nil {
				return err
			}
			if found {
				return fmt.Errorf("%s was partially published", operationIdentity(write))
			}
		}
		return nil
	})
}

func mergeWrites(first, second []writeDefinition) []writeDefinition {
	positions := make(map[string]int, len(first)+len(second))
	merged := make([]writeDefinition, 0, len(first)+len(second))
	for _, writes := range [][]writeDefinition{first, second} {
		for _, write := range writes {
			identity := operationIdentity(write)
			if position, found := positions[identity]; found {
				merged[position] = write
				continue
			}
			positions[identity] = len(merged)
			merged = append(merged, write)
		}
	}
	return merged
}

func operationIdentity(write writeDefinition) string {
	return write.Field + "\x00" + write.KeyHex
}

func decodeHex(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}
	return decoded, nil
}

func decodeRequiredHex(value string) ([]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("hex value is empty")
	}
	return decodeHex(value)
}

func failure(caseID, format string, arguments ...any) *Failure {
	return &Failure{CaseID: caseID, Detail: fmt.Sprintf(format, arguments...)}
}
