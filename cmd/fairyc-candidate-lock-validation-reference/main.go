// Generate strict-reader observations using the owning Go lockfile implementation.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

type sample struct {
	name  string
	input []byte
}

func encoded(d lockfile.Document) []byte {
	b, e := json.MarshalIndent(d, "", "  ")
	if e != nil {
		panic(e)
	}
	return append(b, '\n')
}
func clone(d lockfile.Document) lockfile.Document {
	var result lockfile.Document
	if e := json.Unmarshal(encoded(d), &result); e != nil {
		panic(e)
	}
	return result
}
func sorted(d *lockfile.Document) {
	slices.SortFunc(d.Packages, func(a, b lockfile.PackageRecord) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(d.Dependencies, func(a, b lockfile.DependencyRecord) int {
		if v := strings.Compare(a.Owner, b.Owner); v != 0 {
			return v
		}
		if v := strings.Compare(a.Binding, b.Binding); v != 0 {
			return v
		}
		return strings.Compare(a.Target, b.Target)
	})
}
func reidentify(d *lockfile.Document) {
	ids := map[string]string{}
	for i := range d.Packages {
		p := &d.Packages[i]
		old := p.ID
		id, e := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: p.PackageUID, SourceIdentity: p.SourceIdentity, VersionIdentity: p.Version, Features: p.Features})
		if e != nil {
			panic(e)
		}
		p.ID = id.String()
		ids[old] = p.ID
	}
	if next, ok := ids[d.Root]; ok {
		d.Root = next
	}
	for i := range d.Dependencies {
		e := &d.Dependencies[i]
		if next, ok := ids[e.Owner]; ok {
			e.Owner = next
		}
		if next, ok := ids[e.Target]; ok {
			e.Target = next
		}
	}
	sorted(d)
}
func index(d lockfile.Document, role string) int {
	for i, p := range d.Packages {
		if p.Role == role {
			return i
		}
	}
	panic("missing role: " + role)
}
func synthetic(base lockfile.Document, count int) lockfile.Document {
	d := clone(base)
	root := d.Packages[index(d, packagegraph.RoleRoot)]
	d.Packages = []lockfile.PackageRecord{root}
	d.Dependencies = []lockfile.DependencyRecord{}
	for i := 1; i < count; i++ {
		p := root
		p.ID = fmt.Sprintf("before-%d", i)
		p.Role = packagegraph.RoleDependency
		p.PackageUID = fmt.Sprintf("synthetic.%d", i)
		p.LocalPath = fmt.Sprintf("member%d", i)
		d.Packages = append(d.Packages, p)
	}
	reidentify(&d)
	return d
}
func main() {
	if len(os.Args) != 3 {
		panic("usage: fairyc-candidate-lock-validation-reference GO_LOCK_FIXTURES OUTPUT")
	}
	var samples []sample
	docs := map[string]lockfile.Document{}
	add := func(name string, d lockfile.Document) { samples = append(samples, sample{name, encoded(d)}) }
	for _, name := range []string{"None", "CandidateUnused", "Core", "AliasCore", "Vector", "ChildOnly", "Both", "ChildVector", "UnreachableVector", "FeatureSplit", "InstanceAliases", "LongTrivia", "SimilarNamespaces"} {
		input, e := os.ReadFile(filepath.Join(os.Args[1], name, "lock.json"))
		if e != nil {
			panic(e)
		}
		d, e := lockfile.Parse(input)
		if e != nil {
			panic(e)
		}
		docs[name] = d
		samples = append(samples, sample{name, input})
		if name == "None" || name == "Core" || name == "FeatureSplit" {
			var raw map[string]any
			if e := json.Unmarshal(input, &raw); e != nil {
				panic(e)
			}
			b, e := json.MarshalIndent(raw, "", "    ")
			if e != nil {
				panic(e)
			}
			samples = append(samples, sample{name + "-reordered", b})
		}
	}
	mutate := func(name, base string, refresh bool, change func(*lockfile.Document)) {
		d := clone(docs[base])
		change(&d)
		if refresh {
			reidentify(&d)
		}
		add(name, d)
	}
	mutate("schema", "None", false, func(d *lockfile.Document) { d.Schema = "wrong" })
	mutate("instance-algorithm", "None", false, func(d *lockfile.Document) { d.PackageInstanceAlgorithm = "wrong" })
	mutate("source-algorithm", "None", false, func(d *lockfile.Document) { d.SourceDigestAlgorithm = "wrong" })
	mutate("empty-language", "None", false, func(d *lockfile.Document) { d.Language = "" })
	mutate("opaque-language", "None", false, func(d *lockfile.Document) { d.Language = "opaque-language" })
	mutate("unknown-root", "None", false, func(d *lockfile.Document) { d.Root = "unknown" })
	mutate("null-packages", "None", false, func(d *lockfile.Document) { d.Packages = nil })
	mutate("empty-packages", "None", false, func(d *lockfile.Document) { d.Packages = []lockfile.PackageRecord{} })
	mutate("null-dependencies", "None", false, func(d *lockfile.Document) { d.Dependencies = nil })
	var omitted map[string]json.RawMessage
	if e := json.Unmarshal(encoded(docs["None"]), &omitted); e != nil {
		panic(e)
	}
	delete(omitted, "dependencies")
	b, e := json.Marshal(omitted)
	if e != nil {
		panic(e)
	}
	samples = append(samples, sample{"omitted-dependencies", b})
	mutate("package-order", "Both", false, func(d *lockfile.Document) { slices.Reverse(d.Packages) })
	mutate("duplicate-id", "None", false, func(d *lockfile.Document) { d.Packages = append(d.Packages, d.Packages[0]) })
	for _, name := range []string{"id-prefix", "id-width", "id-uppercase", "tuple-uid", "tuple-version", "tuple-source", "digest-prefix", "digest-width", "digest-uppercase", "digest-drift", "unknown-source-kind", "local-sdk-identity", "local-runtime-abi", "unknown-role", "no-root", "root-path", "null-features", "empty-feature", "duplicate-features", "unordered-features"} {
		name := name
		mutate(name, "None", false, func(d *lockfile.Document) {
			p := &d.Packages[0]
			switch name {
			case "id-prefix":
				p.ID = "wrong" + p.ID[5:]
			case "id-width":
				p.ID = p.ID[:len(p.ID)-1]
			case "id-uppercase":
				p.ID = strings.ToUpper(p.ID)
			case "tuple-uid":
				p.PackageUID = "changed"
			case "tuple-version":
				p.Version = "changed"
			case "tuple-source":
				p.SourceIdentity = "changed"
			case "digest-prefix":
				p.SourceDigest = "wrong" + p.SourceDigest[5:]
			case "digest-width":
				p.SourceDigest = p.SourceDigest[:len(p.SourceDigest)-1]
			case "digest-uppercase":
				p.SourceDigest = strings.ToUpper(p.SourceDigest)
			case "digest-drift":
				p.SourceDigest = "fysrc1-sha256-" + strings.Repeat("0", 64)
			case "unknown-source-kind":
				p.SourceKind = "registry"
			case "local-sdk-identity":
				p.SDKIdentity = sdkstdlib.SDKIdentity
			case "local-runtime-abi":
				p.RuntimeABI = sdkstdlib.RuntimeABI
			case "unknown-role":
				p.Role = "unknown"
			case "no-root":
				p.Role = packagegraph.RoleDependency
			case "root-path":
				p.LocalPath = "member"
			case "null-features":
				p.Features = nil
			case "empty-feature":
				p.Features = []string{""}
			case "duplicate-features":
				p.Features = []string{"a", "a"}
			case "unordered-features":
				p.Features = []string{"b", "a"}
			}
		})
	}
	mutate("empty-uid", "None", false, func(d *lockfile.Document) { d.Packages[0].PackageUID = "" })
	mutate("empty-version", "None", false, func(d *lockfile.Document) { d.Packages[0].Version = "" })
	mutate("opaque-version", "None", true, func(d *lockfile.Document) { d.Packages[0].Version = "opaque-version" })
	mutate("opaque-uid", "None", true, func(d *lockfile.Document) { d.Packages[0].PackageUID = "한글\x00opaque" })
	mutate("local-source-contradiction", "None", true, func(d *lockfile.Document) { d.Packages[0].SourceIdentity = "different-source" })
	mutate("valid-root-features", "None", true, func(d *lockfile.Document) { d.Packages[0].Features = []string{"a", "한글"} })
	for i, path := range []string{"", "/absolute", "../escape", "..", "a/../b", "a/./b", "a//b", "a/", "C:/drive", "a\\b", "a..", "한글/멤버", "space ", "NUL", "a\x00b"} {
		d := synthetic(docs["None"], 2)
		d.Packages[index(d, packagegraph.RoleDependency)].LocalPath = path
		add(fmt.Sprintf("local-path-%d", i), d)
	}
	for _, name := range []string{"sdk-uid", "sdk-version", "sdk-identity", "sdk-abi", "sdk-local-path", "sdk-source-kind", "sdk-role", "sdk-language", "sdk-features", "sdk-composition", "private-sdk", "sdk-other-digest"} {
		name := name
		mutate(name, "Core", true, func(d *lockfile.Document) {
			p := &d.Packages[index(*d, packagegraph.RoleStandardLibrary)]
			switch name {
			case "sdk-uid":
				p.PackageUID = "other.sdk"
			case "sdk-version":
				p.Version = "other"
			case "sdk-identity":
				p.SDKIdentity = "other"
			case "sdk-abi":
				p.RuntimeABI = "other"
			case "sdk-local-path":
				p.LocalPath = "vendor"
			case "sdk-source-kind":
				p.SourceKind = "local"
			case "sdk-role":
				p.Role = packagegraph.RoleDependency
			case "sdk-language":
				d.Language = "other"
			case "sdk-features":
				p.Features = []string{"extra"}
			case "sdk-composition":
				p.SourceIdentity = "wrong"
			case "private-sdk":
				p.SDKIdentity = "fairyc-sdk-vector-candidate/0"
				p.SourceIdentity = p.SDKIdentity + ";" + p.RuntimeABI + ";" + p.SourceDigest
			case "sdk-other-digest":
				p.SourceDigest = "fysrc1-sha256-" + strings.Repeat("0", 64)
				p.SourceIdentity = p.SDKIdentity + ";" + p.RuntimeABI + ";" + p.SourceDigest
			}
		})
	}
	mutate("multiple-sdk", "Core", false, func(d *lockfile.Document) {
		p := d.Packages[index(*d, packagegraph.RoleStandardLibrary)]
		p.SourceDigest = "fysrc1-sha256-" + strings.Repeat("0", 64)
		p.SourceIdentity = p.SDKIdentity + ";" + p.RuntimeABI + ";" + p.SourceDigest
		id, e := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: p.PackageUID, SourceIdentity: p.SourceIdentity, VersionIdentity: p.Version, Features: p.Features})
		if e != nil {
			panic(e)
		}
		p.ID = id.String()
		d.Packages = append(d.Packages, p)
		sorted(d)
	})
	for _, name := range []string{"unknown-owner", "unknown-target", "empty-binding", "duplicate-binding", "edge-order", "self-cycle", "two-cycle", "alias-targets", "opaque-binding"} {
		name := name
		mutate(name, "Both", false, func(d *lockfile.Document) {
			switch name {
			case "unknown-owner":
				d.Dependencies[0].Owner = "unknown"
				sorted(d)
			case "unknown-target":
				d.Dependencies[0].Target = "unknown"
				sorted(d)
			case "empty-binding":
				d.Dependencies[0].Binding = ""
				sorted(d)
			case "duplicate-binding":
				d.Dependencies = append(d.Dependencies, d.Dependencies[0])
				sorted(d)
			case "edge-order":
				slices.Reverse(d.Dependencies)
			case "self-cycle":
				d.Dependencies = append(d.Dependencies, lockfile.DependencyRecord{Owner: d.Root, Binding: "self", Target: d.Root})
				sorted(d)
			case "two-cycle":
				target := d.Dependencies[0].Target
				d.Dependencies = append(d.Dependencies, lockfile.DependencyRecord{Owner: target, Binding: "return", Target: d.Dependencies[0].Owner})
				sorted(d)
			case "alias-targets":
				alias := d.Dependencies[0]
				alias.Binding = "second-alias"
				d.Dependencies = append(d.Dependencies, alias)
				sorted(d)
			case "opaque-binding":
				d.Dependencies[0].Binding = "\x00binding/한글"
				sorted(d)
			}
		})
	}
	d := synthetic(docs["None"], 3)
	dep := index(d, packagegraph.RoleDependency)
	d.Packages[dep].Role = packagegraph.RoleRoot
	add("two-roots", d)
	d = synthetic(docs["None"], 3)
	var dependencies []string
	for _, p := range d.Packages {
		if p.Role == packagegraph.RoleDependency {
			dependencies = append(dependencies, p.ID)
		}
	}
	d.Dependencies = []lockfile.DependencyRecord{{Owner: dependencies[0], Binding: "x", Target: dependencies[1]}, {Owner: dependencies[1], Binding: "y", Target: dependencies[0]}}
	sorted(&d)
	add("unreachable-cycle", d)
	d = synthetic(docs["None"], 256)
	for i := 0; i < len(d.Packages)-1; i++ {
		d.Dependencies = append(d.Dependencies, lockfile.DependencyRecord{Owner: d.Packages[i].ID, Binding: "next", Target: d.Packages[i+1].ID})
	}
	sorted(&d)
	add("long-chain", d)
	d.Dependencies = append(d.Dependencies, lockfile.DependencyRecord{Owner: d.Packages[len(d.Packages)-1].ID, Binding: "return", Target: d.Packages[0].ID})
	sorted(&d)
	add("long-cycle", d)
	var output bytes.Buffer
	output.WriteString("FYLVALID")
	number := func(value int) { var b [8]byte; binary.BigEndian.PutUint64(b[:], uint64(value)); output.Write(b[:]) }
	blob := func(value []byte) { number(len(value)); output.Write(value) }
	number(len(samples))
	accepted := 0
	var observations []map[string]any
	for _, s := range samples {
		parsed, e := lockfile.Parse(s.input)
		var canonical []byte
		reason := ""
		if e == nil {
			accepted++
			canonical, e = lockfile.Marshal(parsed)
			if e != nil {
				panic(e)
			}
		} else {
			reason = e.Error()
		}
		blob([]byte(s.name))
		blob(s.input)
		if e == nil {
			number(1)
		} else {
			number(0)
		}
		blob(canonical)
		observations = append(observations, map[string]any{"name": s.name, "accepted": e == nil, "reason": reason})
	}
	if output.Len() > 8388608 {
		panic("fixture exceeds test reader cap")
	}
	path := os.Args[2]
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		panic(e)
	}
	if e := os.WriteFile(path, output.Bytes(), 0600); e != nil {
		panic(e)
	}
	report, e := json.MarshalIndent(observations, "", "  ")
	if e != nil {
		panic(e)
	}
	if e := os.WriteFile(path+".json", append(report, '\n'), 0600); e != nil {
		panic(e)
	}
	fmt.Printf("Strict locks: %d cases, %d accepted, %d rejected; %d fixture bytes\n", len(samples), accepted, len(samples)-accepted, output.Len())
}
