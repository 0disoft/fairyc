package atlasmap

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestQueryV3ShowsStoredAggregateAndCleanupFacts(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, result := loadAggregateMapWorkspace(t, used)
		_, plans, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectations, err := CurrentWorkspaceExpectationsV3(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectation := expectations[plans[0].Header().PackageInstance]
		store, err := NewMapStoreV3(t.TempDir(), testMapStoreLimits())
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
			t.Fatal("v1 accepted v3")
		}
		if _, err := NewQueryIndexV2(partition, expectation); err == nil {
			t.Fatal("v2 accepted v3")
		}
		index, err := NewQueryIndexV3(partition, expectation)
		if err != nil {
			t.Fatal(err)
		}
		ids := map[string]string{}
		for _, symbol := range partition.Symbols() {
			ids[symbol.Declaration.Name] = symbol.Visible
		}
		request := RelationRequest{Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls, RelationBorrowsShared}, Budget: generousQueryBudget(2)}
		shown, err := index.Show(ids["Token"], request)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if used {
			want = 2
		}
		if shown.Subject.Function != nil || len(shown.Subject.Specializations) != want || len(shown.Relations) != 2*want {
			t.Fatalf("resource facts collapsed: %+v", shown)
		}
		if shown.Context.LogicalSchema != LogicalSchemaV3 || shown.Context.GenericInterfaceHash != partition.Header().GenericInterfaceHash {
			t.Fatal("context lost v3 identity")
		}
		unknown := false
		for _, fact := range shown.Context.Completeness {
			if fact.Relation == "generic-template-bodies" {
				unknown = fact.Status == semanticsnapshot.CompletenessUnavailable
			}
		}
		if !unknown {
			t.Fatal("template uncertainty disappeared")
		}
		byID := map[string]QuerySpecializationFacts{}
		for _, value := range shown.Subject.Specializations {
			if value.Function != nil || value.Aggregate == nil || value.Cleanup == nil || value.Aggregate.Key != shown.Subject.Declaration.Key {
				t.Fatal("wrong specialization kind")
			}
			field := value.Aggregate.Fields[0].Type
			if field.Name != "main::Cell" || len(field.Arguments) != 1 || field.Arguments[0].Name != value.TypeArguments[0].Name {
				t.Fatal("concrete field missing")
			}
			byID[value.ID] = value
		}
		for _, relation := range shown.Relations {
			if _, found := byID[relation.SourceSpecialization]; !found || relation.SourceBody != "cleanup" || relation.Source != ids["Token"] {
				t.Fatal("cleanup owner lost")
			}
			if relation.Kind == RelationCalls && (relation.Target == nil || relation.Target.Symbol != ids["finish"] || relation.Target.Specialization == "") {
				t.Fatal("cleanup helper binding lost")
			}
		}
		for _, name := range []string{"Cell", "Empty"} {
			value, err := index.Show(ids[name], request)
			if err != nil || value.Subject.Function != nil || len(value.Relations) != 0 {
				t.Fatalf("nonexecuting aggregate: %s %v", name, err)
			}
			for _, item := range value.Subject.Specializations {
				if item.Aggregate == nil || item.Function != nil || item.Cleanup != nil {
					t.Fatal("empty aggregate acquired execution")
				}
			}
		}
		catalog, err := index.CatalogSymbols(SymbolCatalogRequest{Kinds: []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationStruct, semanticsnapshot.DeclarationResource}, Budget: SymbolCatalogBudget{MaxSymbols: 10, MaxSelectionBytes: 1 << 20}})
		if err != nil || catalog.Total != 3 || len(catalog.Symbols) != 3 {
			t.Fatalf("original catalog: %+v %v", catalog, err)
		}
		incoming := request
		incoming.Direction = DirectionIncoming
		incoming.Relations = []RelationKind{RelationCalls}
		calls, err := index.Show(ids["finish"], incoming)
		if err != nil || len(calls.Relations) != want {
			t.Fatalf("incoming cleanup calls: %v", err)
		}
		for _, relation := range calls.Relations {
			if relation.Source != ids["Token"] || relation.SourceBody != "cleanup" {
				t.Fatal("incoming owner differs")
			}
		}
		around, err := index.Around(ids["Token"], request)
		if err != nil || len(around.Nodes) == 0 {
			t.Fatalf("around: %v", err)
		}
		mandatory := struct {
			Context QueryContext `json:"context"`
			Subject QueryNode    `json:"subject"`
		}{shown.Context, shown.Subject}
		cost, err := selectionRecordCost(mandatory)
		if err != nil {
			t.Fatal(err)
		}
		bounded := request
		bounded.Budget.MaxSelectionBytes = cost - 1
		if _, err := index.Show(ids["Token"], bounded); !errors.Is(err, ErrQueryBudgetTooSmall) {
			t.Fatalf("aggregate bytes not budgeted: %v", err)
		}
		bounded.Budget.MaxSelectionBytes = cost
		if value, err := index.Show(ids["Token"], bounded); err != nil || len(value.Relations) != 0 || !reflect.DeepEqual(value.Subject, shown.Subject) {
			t.Fatalf("exact subject budget: %v", err)
		}
		if used {
			before, err := json.Marshal(shown)
			if err != nil {
				t.Fatal(err)
			}
			shown.Subject.Specializations[0].Aggregate.Fields[0].Type.Arguments[0].Name = "corrupt"
			shown.Subject.Specializations[0].Cleanup.Relations[0].Count = -1
			shown.Relations[0].SourceBody = "corrupt"
			again, err := index.Show(ids["Token"], request)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(again)
			if string(before) != string(after) {
				t.Fatal("result aliases retained query facts")
			}
			limited := request
			limited.Budget.MaxRelations = 1
			partial, err := index.Show(ids["Token"], limited)
			if err != nil || len(partial.Relations) != 1 || !hasOmission(partial.Omissions, OmissionBudgetRelations) {
				t.Fatalf("relation budget: %v", err)
			}
		}
		wrong := expectation
		wrong.GenericInterface[0] ^= 1
		if _, err := NewQueryIndexV3(partition, wrong); !errors.Is(err, ErrIncompatibleMap) {
			t.Fatalf("interface: %v", err)
		}
	}
}
