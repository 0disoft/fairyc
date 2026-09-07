package atlasmap

import (
	"errors"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestQueryV2ReadsStoredGenericDeclarationsAndImplementations(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, compiled := loadGenericPlanWorkspace(t, used)
		_, plans, err := PrepareWorkspaceV2(workspace, compiled, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectations, err := CurrentWorkspaceExpectationsV2(workspace, compiled, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectation := expectations[plans[0].Header().PackageInstance]
		store, err := NewMapStoreV2(t.TempDir(), testMapStoreLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		if _, err := store.Publish(plans[0]); err != nil {
			t.Fatal(err)
		}
		partition, found, err := store.LoadCurrent(expectation)
		if err != nil || !found {
			t.Fatalf("load: %v %v", found, err)
		}
		if _, err := NewQueryIndex(partition, expectation); err == nil {
			t.Fatal("v1 query admitted v2 map")
		}
		index, err := NewQueryIndexV2(partition, expectation)
		if err != nil {
			t.Fatal(err)
		}
		ids := map[string]string{}
		for _, symbol := range partition.Symbols() {
			ids[symbol.Declaration.Name] = symbol.Visible
		}
		request := RelationRequest{Direction: DirectionOutgoing, Relations: AllRelationKinds(), Budget: generousQueryBudget(2)}
		shown, err := index.Show(ids["id"], request)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if used {
			want = 2
		}
		if shown.Subject.Function != nil || len(shown.Subject.Specializations) != want || len(shown.Relations) != want || len(shown.Subject.Declaration.TypeParameters) != 1 || !reflect.DeepEqual(shown.Subject.TypeParameters, shown.Subject.Declaration.TypeParameters) {
			t.Fatalf("template and implementation facts collapsed: %+v", shown.Subject)
		}
		if shown.Context.GenericInterfaceHash != partition.Header().GenericInterfaceHash {
			t.Fatal("query lost source interface")
		}
		bodiesUnknown := false
		for _, record := range shown.Context.Completeness {
			if record.Relation == "generic-template-bodies" {
				bodiesUnknown = record.Status == semanticsnapshot.CompletenessUnavailable
			}
		}
		if !bodiesUnknown {
			t.Fatal("query lost template-body uncertainty")
		}
		for _, relation := range shown.Relations {
			if relation.SourceSpecialization == "" || relation.Kind != RelationBorrowsShared {
				t.Fatal("concrete borrow lost implementation identity")
			}
		}
		unused, err := index.Show(ids["unused"], request)
		if err != nil || unused.Subject.Function != nil || len(unused.Subject.Specializations) != 0 || len(unused.Relations) != 0 {
			t.Fatalf("unused template acquired executable facts: %v", err)
		}
		catalog, err := index.CatalogSymbols(SymbolCatalogRequest{Name: "id", Kinds: []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationFunction}, Budget: SymbolCatalogBudget{MaxSymbols: 10, MaxSelectionBytes: 1 << 20}})
		if err != nil || catalog.Total != 1 || len(catalog.Symbols) != 1 {
			t.Fatalf("catalog duplicated original declaration: %v", err)
		}
		if used {
			pass, err := index.Show(ids["pass"], request)
			if err != nil || len(pass.Relations) != 2 {
				t.Fatalf("specialization helper calls: %v", err)
			}
			for _, relation := range pass.Relations {
				if relation.SourceSpecialization == "" || relation.Target == nil || relation.Target.Specialization == "" || relation.Target.Symbol != ids["id"] {
					t.Fatal("helper call erased a concrete endpoint")
				}
			}
			if compareQueryRelation(pass.Relations[0], pass.Relations[1]) == 0 {
				t.Fatal("distinct implementation calls compare equal")
			}
			original, err := index.Show(ids["id"], request)
			if err != nil {
				t.Fatal(err)
			}
			shown.Subject.Specializations[0].TypeArguments[0].Name = "changed"
			shown.Subject.TypeParameters[0] = "changed"
			shown.Subject.Specializations[0].Function.Relations[0].Count = -1
			again, err := index.Show(ids["id"], request)
			if err != nil || !reflect.DeepEqual(original, again) {
				t.Fatal("query result aliases index")
			}
			limited := request
			limited.Budget.MaxRelations = 1
			short, err := index.Show(ids["id"], limited)
			if err != nil || len(short.Relations) != 1 || !hasOmission(short.Omissions, OmissionBudgetRelations) {
				t.Fatalf("bounded concrete relations: %v", err)
			}
		}
		tiny := request
		tiny.Budget.MaxSelectionBytes = 1
		if _, err := index.Show(ids["id"], tiny); !errors.Is(err, ErrQueryBudgetTooSmall) {
			t.Fatalf("required generic subject byte bound: %v", err)
		}
		around, err := index.Around(ids["main"], request)
		if err != nil || len(around.Nodes) == 0 {
			t.Fatalf("around generic declarations: %v", err)
		}
		wrong := expectation
		wrong.GenericInterface[0] ^= 1
		if _, err := NewQueryIndexV2(partition, wrong); !errors.Is(err, ErrIncompatibleMap) {
			t.Fatalf("query admitted wrong interface: %v", err)
		}
	}
}
