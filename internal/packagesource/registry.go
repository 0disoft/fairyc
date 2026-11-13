package packagesource

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const (
	RegistryCatalogSchema   = "fairyc.registry/v1"
	RegistryTrustSchema     = "fairyc.registry-trust/v1"
	RegistrySignatureSchema = "fairyc.registry-signature/v1"
	RegistrySignatureDomain = "fairyc.registry/v1\x00"
	MaxCatalogBytes         = 256 << 10
	MaxTrustBytes           = 16 << 10
	MaxSignatureBytes       = 1024
	MaxCatalogReleases      = 256
)

type RegistryRelease struct {
	PackageUID string `json:"packageUid"`
	Version    string `json:"version"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
}
type RegistryCatalog struct {
	Schema   string            `json:"schema"`
	Sequence uint64            `json:"sequence"`
	Expires  string            `json:"expires"`
	Packages []RegistryRelease `json:"packages"`
}
type RegistryKey struct {
	ID        string `json:"id"`
	PublicKey string `json:"publicKey"`
}
type RegistryTrust struct {
	Schema          string        `json:"schema"`
	MinimumSequence uint64        `json:"minimumSequence"`
	Keys            []RegistryKey `json:"keys"`
}
type RegistrySignature struct {
	Schema    string `json:"schema"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

// Unexported contents prevent resolution from accepting an unverified catalog.
type VerifiedCatalog struct {
	catalog RegistryCatalog
	digest  string
}

func boundedObject(data []byte, max int, fields ...string) (map[string]json.RawMessage, error) {
	if len(data) == 0 || len(data) > max || !utf8.Valid(data) {
		return nil, fmt.Errorf("bounded UTF-8 JSON required")
	}
	return object(data, fields...)
}

func canonicalHex(value string, size int) ([]byte, error) {
	if len(value) != size*2 || strings.ToLower(value) != value {
		return nil, fmt.Errorf("noncanonical hexadecimal bytes")
	}
	data, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func parseRegistryTrust(data []byte) (RegistryTrust, error) {
	fields, err := boundedObject(data, MaxTrustBytes, "schema", "minimumSequence", "keys")
	if err != nil {
		return RegistryTrust{}, err
	}
	var trust RegistryTrust
	if err = json.Unmarshal(data, &trust); err != nil {
		return trust, err
	}
	if trust.Schema != RegistryTrustSchema || trust.MinimumSequence == 0 || len(trust.Keys) == 0 || len(trust.Keys) > 8 {
		return trust, fmt.Errorf("invalid registry trust header or key bound")
	}
	var entries []json.RawMessage
	if err = json.Unmarshal(fields["keys"], &entries); err != nil {
		return trust, err
	}
	seen := map[string]bool{}
	for i, entry := range entries {
		if _, err = object(entry, "id", "publicKey"); err != nil {
			return trust, err
		}
		key := trust.Keys[i]
		if len(key.ID) > 64 || !projectmanifest.ValidPackageName(key.ID) || seen[key.ID] {
			return trust, fmt.Errorf("invalid or duplicate registry key ID")
		}
		seen[key.ID] = true
		if _, err = canonicalHex(key.PublicKey, ed25519.PublicKeySize); err != nil {
			return trust, err
		}
	}
	return trust, nil
}

func VerifyRegistryCatalog(trustData, catalogData, signatureData []byte, now time.Time) (VerifiedCatalog, error) {
	trust, err := parseRegistryTrust(trustData)
	if err != nil {
		return VerifiedCatalog{}, fmt.Errorf("registry trust: %w", err)
	}
	if len(catalogData) == 0 || len(catalogData) > MaxCatalogBytes || !utf8.Valid(catalogData) {
		return VerifiedCatalog{}, fmt.Errorf("catalog UTF-8 byte bound")
	}
	if _, err = boundedObject(signatureData, MaxSignatureBytes, "schema", "keyId", "signature"); err != nil {
		return VerifiedCatalog{}, err
	}
	var signature RegistrySignature
	if err = json.Unmarshal(signatureData, &signature); err != nil {
		return VerifiedCatalog{}, err
	}
	if signature.Schema != RegistrySignatureSchema {
		return VerifiedCatalog{}, fmt.Errorf("invalid signature schema")
	}
	var publicKey []byte
	for _, key := range trust.Keys {
		if key.ID == signature.KeyID {
			publicKey, _ = canonicalHex(key.PublicKey, ed25519.PublicKeySize)
		}
	}
	if publicKey == nil {
		return VerifiedCatalog{}, fmt.Errorf("untrusted registry key")
	}
	sig, err := canonicalHex(signature.Signature, ed25519.SignatureSize)
	if err != nil {
		return VerifiedCatalog{}, err
	}
	message := append([]byte(RegistrySignatureDomain), catalogData...)
	if !ed25519.Verify(publicKey, message, sig) {
		return VerifiedCatalog{}, fmt.Errorf("registry signature mismatch")
	}
	fields, err := boundedObject(catalogData, MaxCatalogBytes, "schema", "sequence", "expires", "packages")
	if err != nil {
		return VerifiedCatalog{}, err
	}
	var catalog RegistryCatalog
	if err = json.Unmarshal(catalogData, &catalog); err != nil {
		return VerifiedCatalog{}, err
	}
	expires, err := time.Parse(time.RFC3339, catalog.Expires)
	if err != nil || !strings.HasSuffix(catalog.Expires, "Z") || expires.Format(time.RFC3339) != catalog.Expires || !expires.After(now) {
		return VerifiedCatalog{}, fmt.Errorf("expired or noncanonical registry expiry")
	}
	if catalog.Schema != RegistryCatalogSchema || catalog.Sequence < trust.MinimumSequence || len(catalog.Packages) == 0 || len(catalog.Packages) > MaxCatalogReleases {
		return VerifiedCatalog{}, fmt.Errorf("registry schema, rollback floor or release bound")
	}
	var entries []json.RawMessage
	if err = json.Unmarshal(fields["packages"], &entries); err != nil {
		return VerifiedCatalog{}, err
	}
	seen := map[string]bool{}
	for i, entry := range entries {
		if _, err = object(entry, "packageUid", "version", "url", "sha256"); err != nil {
			return VerifiedCatalog{}, err
		}
		r := catalog.Packages[i]
		key := r.PackageUID + "@" + r.Version
		if seen[key] {
			return VerifiedCatalog{}, fmt.Errorf("duplicate registry release")
		}
		seen[key] = true
		if _, err = parseStableVersion(r.Version); err != nil {
			return VerifiedCatalog{}, err
		}
		pin := Pin{Path: "vendor/catalog", PackageUID: r.PackageUID, Version: r.Version, URL: r.URL, SHA256: r.SHA256}
		if err = Validate(Plan{Schema: Schema, Packages: []Pin{pin}}); err != nil {
			return VerifiedCatalog{}, err
		}
	}
	return VerifiedCatalog{catalog: catalog, digest: hash(catalogData)}, nil
}
