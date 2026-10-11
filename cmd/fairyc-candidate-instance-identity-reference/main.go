// Package-instance fixture bytes and local workspace identities from Go.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func write(root, path string, value []byte) error {
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, value, 0600)
}
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: instance-identity-reference <output>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type fixture struct {
		ID       string
		Identity packagegraph.Identity
		Expect   struct{ Status, EncodingHex, InstanceID, Reason string }
	}
	var normative struct{ Cases []fixture }
	data, err := os.ReadFile("spec/fixtures/package-identity/manifest-v1.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &normative); err != nil {
		return err
	}
	cases := normative.Cases
	add := func(name, uid, source, version string, features []string) {
		cases = append(cases, fixture{ID: name, Identity: packagegraph.Identity{PackageUID: uid, SourceIdentity: source, VersionIdentity: version, Features: features}})
	}
	add("composed", "p\u00e4ck", "src", "1", []string{"\u00e1", "\u03b2"})
	add("decomposed", "pa\u0308ck", "src", "1", []string{"\u00e1", "\u03b2"})
	add("nul-bytes", "pkg\x00x", "src", "1", []string{})
	add("empty-uid", "", "src", "1", []string{})
	add("empty-source", "pkg", "", "1", []string{})
	add("empty-version", "pkg", "src", "", []string{})
	add("utf8-uid", "\xff", "src", "1", []string{})
	add("utf8-source", "pkg", "\xff", "1", []string{})
	add("utf8-version", "pkg", "src", "\xff", []string{})
	add("utf8-feature", "pkg", "src", "1", []string{"\xff"})
	add("mixed-empty-order", "pkg", "src", "1", []string{"x", ""})
	add("mixed-utf8-order", "pkg", "src", "1", []string{"\xff", "a"})
	add("large", "pkg", "src", strings.Repeat("v", 1052672), []string{})
	type row struct {
		Name                           string
		Features                       int
		Accepted                       bool
		Reason, Digest, EncodingSHA256 string
		EncodingBytes                  int
	}
	rows := []row{}
	for _, f := range cases {
		dir := filepath.Join(root, f.ID)
		if err := write(dir, "uid.bin", []byte(f.Identity.PackageUID)); err != nil {
			return err
		}
		if err := write(dir, "source.bin", []byte(f.Identity.SourceIdentity)); err != nil {
			return err
		}
		if err := write(dir, "version.bin", []byte(f.Identity.VersionIdentity)); err != nil {
			return err
		}
		for i, value := range f.Identity.Features {
			if err := write(dir, fmt.Sprintf("feature%d.bin", i), []byte(value)); err != nil {
				return err
			}
		}
		encoded, err := packagegraph.EncodeIdentity(f.Identity)
		r := row{Name: f.ID, Features: len(f.Identity.Features), Accepted: err == nil}
		oracle := ""
		if err != nil {
			e, ok := err.(*packagegraph.IdentityValidationError)
			if !ok {
				return err
			}
			r.Reason = e.Reason
			oracle = "reject\n" + r.Reason + "\n"
			if f.Expect.Status != "" && (f.Expect.Status != "reject" || f.Expect.Reason != r.Reason) {
				return fmt.Errorf("normative rejection mismatch: %s", f.ID)
			}
		} else {
			digest, err := packagegraph.ComputeInstanceID(f.Identity)
			if err != nil {
				return err
			}
			r.Digest = digest.String()
			r.EncodingBytes = len(encoded)
			sum := sha256.Sum256(encoded)
			r.EncodingSHA256 = hex.EncodeToString(sum[:])
			oracle = "pass\n" + r.Digest + "\n"
			if f.Expect.Status != "" && (f.Expect.Status != "pass" || f.Expect.InstanceID != r.Digest || f.Expect.EncodingHex != hex.EncodeToString(encoded)) {
				return fmt.Errorf("normative encoding mismatch: %s", f.ID)
			}
			if err := write(dir, "encoding.bin", encoded); err != nil {
				return err
			}
		}
		if err := write(dir, "oracle.txt", []byte(oracle)); err != nil {
			return err
		}
		rows = append(rows, r)
	}
	local := filepath.Join(root, "Local")
	rootManifest := "schema='fairyc.manifest/v1'\n[package]\nuid='example.root'\nname='root'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n[workspace]\nmembers=['a']\n"
	for _, binding := range []string{"alpha", "beta", "twin"} {
		feature := "alpha"
		if binding == "beta" {
			feature = "beta"
		}
		rootManifest += fmt.Sprintf("[dependencies.%s]\npath='a'\nversion='1.0.0'\nfeatures=['%s']\n", binding, feature)
	}
	memberManifest := "schema='fairyc.manifest/v1'\n[package]\nuid='example.member'\nname='member'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
	for path, value := range map[string]string{"fairy.toml": rootManifest, "src/main.fy": "module main;\n", "a/fairy.toml": memberManifest, "a/src/lib.fy": "module lib;\n"} {
		if err := write(local, path, []byte(value)); err != nil {
			return err
		}
	}
	workspace, err := workspaceloader.Load(local)
	if err != nil {
		return err
	}
	localRows := []row{}
	for i := 0; i < 4; i++ {
		p := workspace.Packages[0]
		features := []string{}
		name := "root"
		if i > 0 {
			d := p.Manifest.Dependencies[i-1]
			p = workspace.Packages[1]
			features = d.Features
			name = fmt.Sprintf("edge%d", i-1)
		}
		identity := packagegraph.Identity{PackageUID: p.Manifest.Package.UID, SourceIdentity: p.Digest.String(), VersionIdentity: p.Manifest.Package.Version, Features: features}
		encoded, err := packagegraph.EncodeIdentity(identity)
		if err != nil {
			return err
		}
		digest, err := packagegraph.ComputeInstanceID(identity)
		if err != nil {
			return err
		}
		if err := write(local, name+".id", []byte(digest.String())); err != nil {
			return err
		}
		if err := write(local, name+".encoding.bin", encoded); err != nil {
			return err
		}
		localRows = append(localRows, row{Name: name, Features: len(features), Accepted: true, Digest: digest.String(), EncodingBytes: len(encoded)})
	}
	report := struct{ Cases, Local []row }{rows, localRows}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := write(root, "reference.json", append(encoded, '\n')); err != nil {
		return err
	}
	return write(root, "root.bin", []byte(root))
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
