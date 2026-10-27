package genericplan

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestImplementationBuildPreservesDigestAndInput(t *testing.T) {
	builder := implementationBuildFixture(t, 32)
	before := make([]Implementation, len(builder.implementations))
	for i, value := range builder.implementations {
		before[i] = cloneImplementation(value)
	}
	catalog, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	const wantDigest = "723e0a9c64e5f6d9b162daaad6722b9a1ff3b8640ecaf7dc4b2c2e27b76ffc41"
	if got := catalog.Digest().String(); got != wantDigest {
		t.Fatalf("catalog digest = %s, want %s", got, wantDigest)
	}
	if !reflect.DeepEqual(builder.implementations, before) {
		t.Fatal("Build changed builder-owned implementation order or arguments")
	}
	key := implementationKey(before[0].Capability, before[0].Arguments)
	builder.implementations[0].Arguments[0] = "app::Changed"
	if !reflect.DeepEqual(catalog.implementations[key], before[0]) {
		t.Fatal("catalog retained mutable implementation arguments")
	}
	if _, err := builder.Build(); ErrorCode(err) != CodeUnknownCatalogItem {
		t.Fatalf("next Build reused stale implementation keys or values: %v", err)
	}
}

func TestImplementationBuildPreservesDuplicateProviderOrder(t *testing.T) {
	builder := implementationBuildFixture(t, 1)
	value := builder.implementations[0]
	builder.implementations = nil
	for _, provider := range []ProviderID{"app::provider_Z", "app::provider_A", "app::provider_M"} {
		value.Provider = provider
		if err := builder.AddImplementation(value); err != nil {
			t.Fatal(err)
		}
	}
	_, err := builder.Build()
	if ErrorCode(err) != CodeDuplicateCatalogItem || !strings.Contains(err.Error(), `implementations "app::provider_A" and "app::provider_M" overlap`) {
		t.Fatalf("duplicate error order changed: %v", err)
	}
}

func TestImplementationBuildPreservesFirstMalformedEntry(t *testing.T) {
	builder := implementationBuildFixture(t, 1)
	value := builder.implementations[0]
	// ':' sorts before '<': encoded keys must retain their full string order.
	for _, capability := range []CapabilityID{"app::A", "app::A::B"} {
		value.Capability = capability
		if err := builder.AddImplementation(value); err != nil {
			t.Fatal(err)
		}
	}
	_, err := builder.Build()
	if ErrorCode(err) != CodeUnknownCatalogItem || !strings.Contains(err.Error(), `implementation references unknown capability "app::A::B"`) {
		t.Fatalf("first malformed implementation changed: %v", err)
	}
}

func implementationBuildFixture(tb testing.TB, count int) *Builder {
	tb.Helper()
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		tb.Fatal(err)
	}
	capability := CapabilityID("app::Encode")
	if err := builder.AddCapability(CapabilityDescriptor{ID: capability, Owner: "app", Arity: 8, Definition: DigestText("capability", string(capability))}); err != nil {
		tb.Fatal(err)
	}
	for i := range 16 {
		id := TypeID(fmt.Sprintf("app::Type_%02d", i))
		if err := builder.AddType(TypeDescriptor{ID: id, Owner: "app", Definition: DigestText("type", string(id))}); err != nil {
			tb.Fatal(err)
		}
	}
	for i := range count {
		arguments := make([]TypeID, 8)
		for j := range arguments {
			arguments[j] = TypeID(fmt.Sprintf("app::Type_%02d", (i>>(4*j))&15))
		}
		provider := ProviderID(fmt.Sprintf("app::encode_%05d", i))
		if err := builder.AddImplementation(Implementation{Capability: capability, Arguments: arguments, Owner: "app", Provider: provider, ProviderDigest: DigestText("provider", string(provider))}); err != nil {
			tb.Fatal(err)
		}
	}
	rand.New(rand.NewPCG(1, 2)).Shuffle(len(builder.implementations), func(i, j int) {
		builder.implementations[i], builder.implementations[j] = builder.implementations[j], builder.implementations[i]
	})
	return builder
}

func BenchmarkBuildImplementationCatalog(b *testing.B) {
	for _, count := range []int{32, 1024, 16384} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			builder := implementationBuildFixture(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				catalog, err := builder.Build()
				if err != nil || len(catalog.implementations) != count {
					b.Fatalf("build %d implementations: %v", count, err)
				}
			}
		})
	}
}
