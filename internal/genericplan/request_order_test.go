package genericplan

import (
	"cmp"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"testing"
)

func TestRequestOrderingMatchesEncodedKeyComparison(t *testing.T) {
	// Include prefix ordering and delimiter-bearing malformed requests. Sorting
	// must not replace expand's validation or change the first request rejected.
	names := []string{"", "app::A", "app::A::B", "app::A_", "app::AA", "<", ">", ",", "A,B"}
	random := rand.New(rand.NewSource(7))
	var input []Request
	for range 500 {
		request := Request{Declaration: DeclarationID(names[random.Intn(len(names))])}
		for range random.Intn(9) {
			request.Arguments = append(request.Arguments, TypeID(names[random.Intn(len(names))]))
		}
		input = append(input, request, request)
	}
	input = append(input,
		Request{Declaration: "app::F", Arguments: []TypeID{"A", "B"}},
		Request{Declaration: "app::F", Arguments: []TypeID{"A,B"}},
	)
	for _, values := range [][]Request{nil, {}, input[:1], input[:2], input} {
		want := cloneRequests(values)
		compare := func(left, right Request) int { return cmp.Compare(requestKey(left), requestKey(right)) }
		slices.SortFunc(want, compare)
		want = slices.CompactFunc(want, func(left, right Request) bool { return compare(left, right) == 0 })
		working := cloneRequests(values)
		got := sortUniqueRequests(working)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%d requests: got %+v, want %+v", len(values), got, want)
		}
		for _, removed := range working[len(got):] {
			if removed.Declaration != "" || removed.Arguments != nil {
				t.Fatal("compacted tail retains discarded requests")
			}
		}
	}
}

func TestPlanRequestOrderingPreservesCallerInput(t *testing.T) {
	catalog := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	for _, arguments := range [][]TypeID{{addressType, u64Type}, {coinType, u64Type}} {
		requests := []Request{{Declaration: writeDecl, Arguments: arguments}, {Declaration: encodeDecl, Arguments: []TypeID{addressType}}}
		before := cloneRequests(requests)
		_, err := catalog.Plan(requests, DefaultPlanLimits())
		if arguments[0] == addressType && err != nil || arguments[0] == coinType && ErrorCode(err) != CodeAbilityUnsatisfied {
			t.Fatalf("unexpected planning error: %v", err)
		}
		if !reflect.DeepEqual(requests, before) {
			t.Fatal("planning changed caller-owned requests or arguments")
		}
	}
}

func benchmarkRequestCatalog(b *testing.B, count int) (*Catalog, []Request) {
	b.Helper()
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		b.Fatal(err)
	}
	parameters := make([]TypeParameter, 8)
	for index := range parameters {
		parameters[index] = TypeParameter{Name: fmt.Sprintf("T%d", index)}
	}
	for index := range 8 {
		id := DeclarationID(fmt.Sprintf("app::Generic%d", index))
		if err := builder.AddDeclaration(Declaration{ID: id, Owner: "app", Parameters: parameters, Definition: DigestText("declaration", string(id))}); err != nil {
			b.Fatal(err)
		}
	}
	for index := range (count + 7) / 8 {
		id := TypeID(fmt.Sprintf("app::Element%d", index))
		if err := builder.AddType(TypeDescriptor{ID: id, Owner: "app", Definition: DigestText("type", string(id))}); err != nil {
			b.Fatal(err)
		}
	}
	requests := make([]Request, count)
	for index := range requests {
		arguments := make([]TypeID, len(parameters))
		for i := range arguments {
			arguments[i] = TypeID(fmt.Sprintf("app::Element%d", index/8))
		}
		requests[index] = Request{Declaration: DeclarationID(fmt.Sprintf("app::Generic%d", index%8)), Arguments: arguments}
	}
	random := rand.New(rand.NewSource(7))
	random.Shuffle(len(requests), func(i, j int) { requests[i], requests[j] = requests[j], requests[i] })
	catalog, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}
	return catalog, requests
}

func BenchmarkPlanRequestOrdering(b *testing.B) {
	for _, count := range []int{32, 512, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			catalog, requests := benchmarkRequestCatalog(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				plan, err := catalog.Plan(requests, DefaultPlanLimits())
				if err != nil || len(plan.Specializations) != count || len(plan.Roots) != count {
					b.Fatalf("specializations=%d, roots=%d, error=%v", len(plan.Specializations), len(plan.Roots), err)
				}
			}
		})
	}
}
