package packagesource

import (
	"strings"
	"testing"
)

func TestRegistryUpdateProposals(t *testing.T) {
	trust, catalog, now := registryFixture(t)
	data := registryJSON(t, catalog)
	verified, err := VerifyRegistryCatalog(registryJSON(t, trust), data, registryFixtureSignature(t, data), now)
	if err != nil {
		t.Fatal(err)
	}
	old := Pin{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: catalog.Packages[0].URL, SHA256: catalog.Packages[0].SHA256}
	current := Plan{Schema: Schema, Packages: []Pin{old}}
	proposal, err := ProposeRegistryUpdate(current, verified, registryFixtureRequests(t, "^1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Changes) != 1 || proposal.Changes[0].Before != old || proposal.Changes[0].After.Version != "1.9.0" || current.Packages[0] != old || proposal.CurrentPlanSHA256 != hash(registryJSON(t, current)) {
		t.Fatalf("proposal or source mutation: %+v", proposal)
	}
	unchanged, err := ProposeRegistryUpdate(current, verified, registryFixtureRequests(t, "1.0.0"))
	if err != nil || len(unchanged.Changes) != 0 {
		t.Fatalf("no-op: %+v %v", unchanged, err)
	}
	for _, mutate := range []func(*Plan){
		func(p *Plan) { p.Packages[0].Version = "2.0.0" }, func(p *Plan) { p.Packages[0].SHA256 = strings.Repeat("b", 64) }, func(p *Plan) { p.Packages[0].PackageUID = "example.other" }, func(p *Plan) { p.Packages[0].Path = "vendor/other" },
	} {
		candidate := Plan{Schema: Schema, Packages: []Pin{old}}
		mutate(&candidate)
		if _, err := ProposeRegistryUpdate(candidate, verified, registryFixtureRequests(t, "1.0.0")); err == nil {
			t.Fatal("unsafe update admitted")
		}
	}
}
