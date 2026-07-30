package atlasmap

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestQueryIndexOverviewShowAndAroundAreDeterministicAndDetached(t *testing.T) {
	t.Parallel()
	fixture := localCallGraphFixture(t)
	partition := fixture.partition
	index, err := NewQueryIndex(partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.relations) != partition.Header().Counts.Relations {
		t.Fatalf("indexed relations = %d, header = %d", len(index.relations), partition.Header().Counts.Relations)
	}

	budget := generousQueryBudget(3)
	firstOverview, err := index.Overview(budget)
	if err != nil {
		t.Fatal(err)
	}
	secondOverview, err := index.Overview(budget)
	if err != nil || !reflect.DeepEqual(firstOverview, secondOverview) {
		t.Fatalf("Overview() is nondeterministic: %v", err)
	}
	if firstOverview.Package.Header.Counts.Symbols != len(index.nodes) ||
		len(firstOverview.Completeness) != len(partition.Completeness()) ||
		len(firstOverview.Modules) != len(partition.Modules()) || len(firstOverview.Omissions) != 0 {
		t.Fatalf("overview coverage = %+v", firstOverview)
	}

	subject := functionNodeWithLocalCall(t, index)
	request := RelationRequest{Direction: DirectionBoth, Relations: AllRelationKinds(), Budget: budget}
	shownVisible, err := index.Show(subject.symbol.Visible, request)
	if err != nil {
		t.Fatal(err)
	}
	shownFull, err := index.Show(subject.symbol.Full, request)
	if err != nil || !reflect.DeepEqual(shownVisible, shownFull) {
		t.Fatalf("visible/full Show() differ: %v", err)
	}
	if shownVisible.Subject.Symbol != subject.symbol || shownVisible.Subject.Function == nil || len(shownVisible.Relations) < 2 {
		t.Fatalf("show result = %+v", shownVisible)
	}
	if !hasOmission(shownVisible.Omissions, OmissionIncomingPackagesNotJoined) {
		t.Fatalf("show omitted no cross-package incoming warning: %+v", shownVisible.Omissions)
	}
	for _, relation := range shownVisible.Relations {
		if _, found := slices.BinarySearch(request.Relations, relation.Kind); !found {
			t.Fatalf("unrequested relation returned: %+v", relation)
		}
	}

	aroundVisible, err := index.Around(subject.symbol.Visible, request)
	if err != nil {
		t.Fatal(err)
	}
	aroundFull, err := index.Around(subject.symbol.Full, request)
	if err != nil || !reflect.DeepEqual(aroundVisible, aroundFull) {
		t.Fatalf("visible/full Around() differ: %v", err)
	}
	if len(aroundVisible.Nodes) < 2 || aroundVisible.Nodes[0].Depth != 0 || aroundVisible.Usage.Depth == 0 {
		t.Fatalf("around traversal = %+v", aroundVisible)
	}
	for index := 1; index < len(aroundVisible.Nodes); index++ {
		if aroundVisible.Nodes[index].Depth < aroundVisible.Nodes[index-1].Depth {
			t.Fatalf("around nodes are not breadth-first: %+v", aroundVisible.Nodes)
		}
	}

	firstOverview.Package.Header.Features = append(firstOverview.Package.Header.Features, "corrupt")
	firstOverview.Completeness[0].Reasons = append(firstOverview.Completeness[0].Reasons, "corrupt")
	shownVisible.Subject.Declaration.Effects = append(shownVisible.Subject.Declaration.Effects, "corrupt")
	if len(shownVisible.Relations) != 0 && shownVisible.Relations[0].Target != nil {
		shownVisible.Relations[0].Target.Name = "corrupt"
	}
	if len(shownVisible.Context.MapOmissions) != 0 {
		shownVisible.Context.MapOmissions[0] = "corrupt"
	}
	freshOverview, _ := index.Overview(budget)
	freshShow, _ := index.Show(subject.symbol.Visible, request)
	if slices.Contains(freshOverview.Package.Header.Features, "corrupt") ||
		slices.Contains(freshOverview.Completeness[0].Reasons, "corrupt") ||
		slices.Contains(freshShow.Subject.Declaration.Effects, "corrupt") ||
		len(freshShow.Context.MapOmissions) != 0 && freshShow.Context.MapOmissions[0] == "corrupt" {
		t.Fatal("query result mutation escaped into immutable index")
	}
}

func TestQueryIndexCatalogSymbolsIsExactFilteredAndPageable(t *testing.T) {
	t.Parallel()
	fixture := localCallGraphFixture(t)
	index, err := NewQueryIndex(fixture.partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	request := SymbolCatalogRequest{
		Kinds:  AllDeclarationKinds(),
		Budget: SymbolCatalogBudget{MaxSymbols: 512, MaxSelectionBytes: 8 << 20},
	}
	first, err := index.CatalogSymbols(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := index.CatalogSymbols(request)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated catalog = %+v, %v", second, err)
	}
	if !first.Complete || first.Total != fixture.partition.Header().Counts.Symbols ||
		len(first.Symbols) != first.Total || first.NextAfter != "" || first.Usage.Symbols != first.Total {
		t.Fatalf("catalog = %+v", first)
	}
	if !slices.IsSortedFunc(first.Symbols, func(left, right SymbolCatalogEntry) int {
		return strings.Compare(left.Symbol.Full, right.Symbol.Full)
	}) {
		t.Fatalf("catalog symbols are not full-SymbolId ordered: %+v", first.Symbols)
	}

	pageRequest := request
	pageRequest.Budget.MaxSymbols = 1
	page, err := index.CatalogSymbols(pageRequest)
	if err != nil {
		t.Fatal(err)
	}
	if page.Complete || len(page.Symbols) != 1 || page.NextAfter != page.Symbols[0].Symbol.Full ||
		!hasOmission(page.Omissions, OmissionBudgetSymbols) {
		t.Fatalf("first catalog page = %+v", page)
	}
	pageRequest.After = page.NextAfter
	next, err := index.CatalogSymbols(pageRequest)
	if err != nil || len(next.Symbols) != 1 || next.Symbols[0].Symbol.Full == page.Symbols[0].Symbol.Full {
		t.Fatalf("next catalog page = %+v, %v", next, err)
	}

	selected := first.Symbols[0]
	filtered, err := index.CatalogSymbols(SymbolCatalogRequest{
		Module: selected.Module, Kinds: []semanticsnapshot.DeclarationKind{selected.Kind},
		Budget: request.Budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range filtered.Symbols {
		if entry.Module != selected.Module || entry.Kind != selected.Kind {
			t.Fatalf("filtered catalog leaked %+v", entry)
		}
	}

	namedRequest := request
	namedRequest.Name = "leaf"
	named, err := index.CatalogSymbols(namedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !named.Complete || named.Name != "leaf" || named.Total != 2 || len(named.Symbols) != 2 ||
		named.Symbols[0].Module == named.Symbols[1].Module {
		t.Fatalf("exact-name catalog = %+v", named)
	}
	for _, entry := range named.Symbols {
		if entry.Name != "leaf" {
			t.Fatalf("exact-name catalog leaked %+v", entry)
		}
	}
	moduleNamed := namedRequest
	moduleNamed.Module = named.Symbols[0].Module
	one, err := index.CatalogSymbols(moduleNamed)
	if err != nil || one.Total != 1 || len(one.Symbols) != 1 || one.Symbols[0].Module != moduleNamed.Module {
		t.Fatalf("module and exact-name catalog = %+v, %v", one, err)
	}
	missingName := request
	missingName.Name = "absent"
	empty, err := index.CatalogSymbols(missingName)
	if err != nil || !empty.Complete || empty.Total != 0 || len(empty.Symbols) != 0 || empty.Name != "absent" {
		t.Fatalf("missing exact-name catalog = %+v, %v", empty, err)
	}
}

func TestQueryIndexCatalogSymbolsRejectsAmbiguousOrUnpageableRequests(t *testing.T) {
	t.Parallel()
	fixture := localCallGraphFixture(t)
	index, err := NewQueryIndex(fixture.partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	valid := SymbolCatalogRequest{
		Kinds:  AllDeclarationKinds(),
		Budget: SymbolCatalogBudget{MaxSymbols: 512, MaxSelectionBytes: 8 << 20},
	}
	tests := []struct {
		name   string
		mutate func(*SymbolCatalogRequest)
		want   error
	}{
		{name: "missing kinds", mutate: func(value *SymbolCatalogRequest) { value.Kinds = nil }},
		{name: "duplicate kind", mutate: func(value *SymbolCatalogRequest) { value.Kinds = append(value.Kinds, value.Kinds[0]) }},
		{name: "unknown kind", mutate: func(value *SymbolCatalogRequest) { value.Kinds = []semanticsnapshot.DeclarationKind{"variable"} }},
		{name: "unknown module", mutate: func(value *SymbolCatalogRequest) { value.Module = "missing" }, want: ErrModuleNotFound},
		{name: "invalid exact name", mutate: func(value *SymbolCatalogRequest) { value.Name = "not::an_identifier" }},
		{name: "visible cursor", mutate: func(value *SymbolCatalogRequest) { value.After = index.nodes[0].symbol.Visible }},
		{name: "foreign full cursor", mutate: func(value *SymbolCatalogRequest) {
			value.After = "fysym1-sha256-256-" + strings.Repeat("a", 64)
		}},
		{name: "cursor outside exact-name filter", mutate: func(value *SymbolCatalogRequest) {
			value.Name = "leaf"
			for _, node := range index.nodes {
				if node.declaration.Key.Name != value.Name {
					value.After = node.symbol.Full
					return
				}
			}
		}},
		{name: "zero symbols", mutate: func(value *SymbolCatalogRequest) { value.Budget.MaxSymbols = 0 }},
		{name: "zero bytes", mutate: func(value *SymbolCatalogRequest) { value.Budget.MaxSelectionBytes = 0 }},
		{name: "mandatory too large", mutate: func(value *SymbolCatalogRequest) { value.Budget.MaxSelectionBytes = 1 }, want: ErrQueryBudgetTooSmall},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			request.Kinds = slices.Clone(valid.Kinds)
			test.mutate(&request)
			_, catalogErr := index.CatalogSymbols(request)
			if catalogErr == nil || test.want != nil && !errors.Is(catalogErr, test.want) {
				t.Fatalf("CatalogSymbols() error = %v, want %v", catalogErr, test.want)
			}
		})
	}
}

func TestQueriesEnforceRecordRelationDepthAndSelectionByteBudgets(t *testing.T) {
	t.Parallel()
	fixture := localCallGraphFixture(t)
	partition := fixture.partition
	index, err := NewQueryIndex(partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	subject := functionNodeWithLocalCall(t, index)

	overviewBudget := QueryBudget{MaxRecords: 1, MaxRelations: 0, MaxDepth: 0, MaxSelectionBytes: 1 << 20}
	overview, err := index.Overview(overviewBudget)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Completeness) != 0 || len(overview.Modules) != 0 ||
		!hasOmission(overview.Omissions, OmissionBudgetRecords) || overview.Usage.Records != 1 {
		t.Fatalf("bounded overview = %+v", overview)
	}

	showBudget := QueryBudget{MaxRecords: 1, MaxRelations: 0, MaxDepth: 0, MaxSelectionBytes: 1 << 20}
	show, err := index.Show(subject.symbol.Visible, RelationRequest{
		Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: showBudget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(show.Relations) != 0 || !hasOmission(show.Omissions, OmissionBudgetRelations) ||
		!hasOmission(show.Omissions, OmissionRelationFilter) {
		t.Fatalf("bounded show = %+v", show)
	}
	byteBudget := generousQueryBudget(1)
	byteBudget.MaxSelectionBytes = show.Usage.SelectionBytes
	byteLimited, err := index.Show(subject.symbol.Visible, RelationRequest{
		Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: byteBudget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byteLimited.Relations) != 0 || !hasOmission(byteLimited.Omissions, OmissionBudgetBytes) ||
		byteLimited.Usage.SelectionBytes != show.Usage.SelectionBytes {
		t.Fatalf("selection-byte-bounded show = %+v", byteLimited)
	}

	depthBudget := generousQueryBudget(0)
	around, err := index.Around(subject.symbol.Visible, RelationRequest{
		Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: depthBudget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(around.Nodes) != 1 || len(around.Relations) != 0 || !hasOmission(around.Omissions, OmissionDepthLimit) {
		t.Fatalf("depth-zero around = %+v", around)
	}

	tiny := QueryBudget{MaxRecords: 1, MaxRelations: 0, MaxDepth: 0, MaxSelectionBytes: 1}
	if _, err := index.Overview(tiny); !errors.Is(err, ErrQueryBudgetTooSmall) {
		t.Fatalf("tiny Overview() error = %v", err)
	}
	if _, err := index.Show(subject.symbol.Visible, RelationRequest{
		Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: tiny,
	}); !errors.Is(err, ErrQueryBudgetTooSmall) {
		t.Fatalf("tiny Show() error = %v", err)
	}
}

func TestQueriesExposeUnjoinedDependencyTargetsWithoutGuessing(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	for _, partition := range partitions {
		index, indexErr := NewQueryIndex(partition, expectations[partition.Header().PackageInstance])
		if indexErr != nil {
			t.Fatal(indexErr)
		}
		for _, relation := range index.relations {
			if relation.Kind != RelationCalls || relation.Target == nil || relation.Target.Symbol != "" {
				continue
			}
			show, showErr := index.Show(relation.Source, RelationRequest{
				Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: generousQueryBudget(1),
			})
			if showErr != nil {
				t.Fatal(showErr)
			}
			if !hasOmission(show.Omissions, OmissionDependencyMapNotJoined) || len(show.Relations) == 0 ||
				show.Relations[0].Target.Symbol != "" {
				t.Fatalf("unjoined dependency call = %+v", show)
			}
			return
		}
	}
	t.Fatal("quickstart produced no unjoined dependency call")
}

func TestQueryIndexRejectsInvalidPartitionsSelectorsRequestsAndBudgets(t *testing.T) {
	t.Parallel()
	fixture := localCallGraphFixture(t)
	partition := fixture.partition
	index, err := NewQueryIndex(partition, fixture.expectation)
	if err != nil {
		t.Fatal(err)
	}
	subject := functionNodeWithLocalCall(t, index)
	invalidPartition := partition
	invalidPartition.data.header.Counts.Relations++
	if _, err := NewQueryIndex(invalidPartition, fixture.expectation); err == nil {
		t.Fatal("invalid partition produced a query index")
	}
	stale := fixture.expectation
	stale.Current = cloneCurrentPackage(fixture.expectation.Current)
	stale.Current.Files[0].Digest[0] ^= 0xff
	if _, err := NewQueryIndex(partition, stale); !errors.Is(err, ErrStaleMap) {
		t.Fatalf("stale query expectation error = %v", err)
	}
	incompatible := fixture.expectation
	incompatible.CompilerBuild[0] ^= 0xff
	if _, err := NewQueryIndex(partition, incompatible); !errors.Is(err, ErrIncompatibleMap) {
		t.Fatalf("incompatible query expectation error = %v", err)
	}

	if _, err := index.Overview(QueryBudget{}); err == nil {
		t.Fatal("zero query budget succeeded")
	}
	validBudget := generousQueryBudget(1)
	for _, test := range []struct {
		name    string
		symbol  string
		request RelationRequest
	}{
		{name: "bad selector", symbol: "answer", request: RelationRequest{Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: validBudget}},
		{name: "missing selector", symbol: "fysym1-sha256-128-" + strings.Repeat("0", 32), request: RelationRequest{Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: validBudget}},
		{name: "bad direction", symbol: subject.symbol.Visible, request: RelationRequest{Direction: "sideways", Relations: []RelationKind{RelationCalls}, Budget: validBudget}},
		{name: "empty filter", symbol: subject.symbol.Visible, request: RelationRequest{Direction: DirectionOutgoing, Budget: validBudget}},
		{name: "duplicate filter", symbol: subject.symbol.Visible, request: RelationRequest{Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls, RelationCalls}, Budget: validBudget}},
		{name: "unknown relation", symbol: subject.symbol.Visible, request: RelationRequest{Direction: DirectionOutgoing, Relations: []RelationKind{"guesses"}, Budget: validBudget}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, showErr := index.Show(test.symbol, test.request); showErr == nil {
				t.Fatal("invalid Show() succeeded")
			}
			if _, aroundErr := index.Around(test.symbol, test.request); aroundErr == nil {
				t.Fatal("invalid Around() succeeded")
			}
		})
	}
	missing := "fysym1-sha256-128-" + strings.Repeat("0", 32)
	if _, err := index.Show(missing, RelationRequest{
		Direction: DirectionOutgoing, Relations: []RelationKind{RelationCalls}, Budget: validBudget,
	}); !errors.Is(err, ErrSymbolNotFound) {
		t.Fatalf("missing symbol error = %v", err)
	}
}

type queryFixture struct {
	partition   Partition
	expectation LoadExpectation
}

func localCallGraphFixture(t *testing.T) queryFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeQueryFixture(t, filepath.Join(root, "fairy.toml"), `schema = "fairyc.manifest/v1"

[package]
uid = "test.atlas-query"
name = "atlas-query"
version = "1.0.0"
language = "fairyc-language/6"
source = "src"
`)
	writeQueryFixture(t, filepath.Join(root, "src", "main.fy"), `module main;

fn i32 leaf() {
    return 40;
}

fn i32 middle() {
    return leaf() + 1;
}

fn i32 main() {
    return middle() + 1;
}
`)
	writeQueryFixture(t, filepath.Join(root, "src", "other.fy"), `module other;

fn i32 leaf() {
    return 7;
}
`)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil || !result.Verified {
		t.Fatalf("CheckPackages() = verified %t error %v", result.Verified, err)
	}
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range partitions {
		localCalls := 0
		for _, function := range candidate.Functions() {
			for _, call := range function.Calls {
				if call.TargetKind == semanticsnapshot.CallTargetResolved && call.Target.PackageInstance == candidate.Header().PackageInstance {
					localCalls++
				}
			}
		}
		if localCalls >= 2 {
			return queryFixture{partition: candidate, expectation: expectations[candidate.Header().PackageInstance]}
		}
	}
	t.Fatal("query workspace produced no two-edge local call graph")
	return queryFixture{}
}

func writeQueryFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func functionNodeWithLocalCall(t *testing.T, index *QueryIndex) indexedQueryNode {
	t.Helper()
	for _, node := range index.nodes {
		for _, relationIndex := range index.outgoing[node.symbol.Visible] {
			relation := index.relations[relationIndex]
			if relation.Kind == RelationCalls && relation.Target != nil && relation.Target.Symbol != "" {
				return node
			}
		}
	}
	t.Fatal("query fixture has no local call")
	return indexedQueryNode{}
}

func generousQueryBudget(depth int) QueryBudget {
	return QueryBudget{MaxRecords: 512, MaxRelations: 256, MaxDepth: depth, MaxSelectionBytes: 8 << 20}
}

func hasOmission(values []QueryOmission, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
