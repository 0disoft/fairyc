package packagesource

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Deterministic fixture key only; never used by production or documented as trust.
func registryFixtureKey() ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
}
func registryJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func registryFixture(t *testing.T) (RegistryTrust, RegistryCatalog, time.Time) {
	t.Helper()
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	trust := RegistryTrust{Schema: RegistryTrustSchema, MinimumSequence: 2, Keys: []RegistryKey{{ID: "fixture", PublicKey: hex.EncodeToString(registryFixtureKey().Public().(ed25519.PublicKey))}}}
	catalog := RegistryCatalog{Schema: RegistryCatalogSchema, Sequence: 2, Expires: now.Add(time.Hour).Format(time.RFC3339)}
	for _, version := range []string{"1.0.0", "2.0.0", "1.9.0"} {
		catalog.Packages = append(catalog.Packages, RegistryRelease{PackageUID: "example.math", Version: version, URL: "https://packages.example/math-" + version + ".zip", SHA256: strings.Repeat("a", 64)})
	}
	return trust, catalog, now
}
func registryFixtureSignature(t *testing.T, data []byte) []byte {
	t.Helper()
	signature := ed25519.Sign(registryFixtureKey(), append([]byte(RegistrySignatureDomain), data...))
	return registryJSON(t, RegistrySignature{Schema: RegistrySignatureSchema, KeyID: "fixture", Signature: hex.EncodeToString(signature)})
}
func registryFixtureRequests(t *testing.T, constraint string) []byte {
	return registryJSON(t, RegistryRequests{Schema: RegistryRequestsSchema, Packages: []RegistryRequest{{Path: "vendor/math", PackageUID: "example.math", Version: constraint}}})
}

func TestVerifiedRegistryResolution(t *testing.T) {
	trust, catalog, now := registryFixture(t)
	data := registryJSON(t, catalog)
	verified, err := VerifyRegistryCatalog(registryJSON(t, trust), data, registryFixtureSignature(t, data), now)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := ResolveRegistry(verified, registryFixtureRequests(t, "^1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Sequence != 2 || resolution.CatalogSHA256 != hash(data) || len(resolution.SourcePlan.Packages) != 1 || resolution.SourcePlan.Packages[0].Version != "1.9.0" {
		t.Fatalf("resolution: %+v", resolution)
	}
	if _, err := Parse(registryJSON(t, resolution.SourcePlan)); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveRegistry(VerifiedCatalog{}, registryFixtureRequests(t, "^1.0.0")); err == nil {
		t.Fatal("unsigned catalog admitted")
	}
	for _, constraint := range []string{"^3.0.0", "*", "latest", "1.0.0-alpha"} {
		if _, err := ResolveRegistry(verified, registryFixtureRequests(t, constraint)); err == nil {
			t.Fatalf("admitted %s", constraint)
		}
	}
	requests := RegistryRequests{Schema: RegistryRequestsSchema, Packages: []RegistryRequest{{Path: "vendor/math", PackageUID: "example.math", Version: "^1.0.0"}, {Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0"}}}
	if _, err := ResolveRegistry(verified, registryJSON(t, requests)); err == nil {
		t.Fatal("duplicate request admitted")
	}
}

func TestRegistrySignatureAndStrictBoundaries(t *testing.T) {
	trust, catalog, now := registryFixture(t)
	data := registryJSON(t, catalog)
	sig := registryFixtureSignature(t, data)
	for _, tc := range []struct {
		name                      string
		trust, catalog, signature []byte
		now                       time.Time
	}{
		{"changed-bytes", registryJSON(t, trust), append(append([]byte{}, data...), ' '), sig, now},
		{"expiry", registryJSON(t, trust), data, sig, now.Add(time.Hour)},
		{"unknown-key", registryJSON(t, trust), data, []byte(strings.Replace(string(sig), "fixture", "unknown", 1)), now},
		{"duplicate-signature-field", registryJSON(t, trust), data, []byte(strings.Replace(string(sig), `"keyId":"fixture"`, `"keyId":"fixture","keyId":"fixture"`, 1)), now},
		{"catalog-byte-bound", registryJSON(t, trust), make([]byte, MaxCatalogBytes+1), sig, now},
		{"missing-trust", nil, data, sig, now},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := VerifyRegistryCatalog(tc.trust, tc.catalog, tc.signature, tc.now); err == nil {
				t.Fatal("invalid registry accepted")
			}
		})
	}
	for _, mutate := range []func(*RegistryCatalog){
		func(c *RegistryCatalog) { c.Sequence = 1 }, func(c *RegistryCatalog) { c.Schema = "unknown" }, func(c *RegistryCatalog) { c.Expires = "2026-11-14T01:00:00+00:00" }, func(c *RegistryCatalog) { c.Packages[0].URL = "http://packages.example/a.zip" },
		func(c *RegistryCatalog) { c.Packages[0].Version = "1.0.0-alpha" }, func(c *RegistryCatalog) { c.Packages = append(c.Packages, c.Packages[0]) }, func(c *RegistryCatalog) { c.Packages[0].SHA256 = strings.Repeat("A", 64) },
	} {
		_, c, _ := registryFixture(t)
		mutate(&c)
		raw := registryJSON(t, c)
		if _, err := VerifyRegistryCatalog(registryJSON(t, trust), raw, registryFixtureSignature(t, raw), now); err == nil {
			t.Fatalf("signed invalid catalog admitted: %s", raw)
		}
	}
	for _, raw := range []string{strings.Replace(string(data), `"sequence":2`, `"sequence":2,"sequence":2`, 1), strings.Replace(string(data), `"sequence":2`, `"sequence":2,"extra":true`, 1), string(data) + " {}"} {
		if _, err := VerifyRegistryCatalog(registryJSON(t, trust), []byte(raw), registryFixtureSignature(t, []byte(raw)), now); err == nil {
			t.Fatal("signed malformed JSON admitted")
		}
	}
	trust.MinimumSequence = 3
	if _, err := VerifyRegistryCatalog(registryJSON(t, trust), data, sig, now); err == nil {
		t.Fatal("rollback admitted")
	}
}
