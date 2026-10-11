// Package lockfile defines the deterministic generated Fairy C lockfile v1.
package lockfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	FileName      = "fairy.lock"
	SchemaVersion = "fairyc.lock/v1"
	MaxBytes      = 8 << 20
)

type Document struct {
	Schema                   string             `json:"schema"`
	PackageInstanceAlgorithm string             `json:"packageInstanceAlgorithm"`
	SourceDigestAlgorithm    string             `json:"sourceDigestAlgorithm"`
	Language                 string             `json:"language"`
	Root                     string             `json:"root"`
	Packages                 []PackageRecord    `json:"packages"`
	Dependencies             []DependencyRecord `json:"dependencies"`
}

type PackageRecord struct {
	ID             string   `json:"id"`
	Role           string   `json:"role"`
	PackageUID     string   `json:"packageUid"`
	SourceKind     string   `json:"sourceKind"`
	LocalPath      string   `json:"localPath"`
	SDKIdentity    string   `json:"sdkIdentity"`
	RuntimeABI     string   `json:"runtimeAbi"`
	SourceIdentity string   `json:"sourceIdentity"`
	SourceDigest   string   `json:"sourceDigest"`
	Version        string   `json:"version"`
	Features       []string `json:"features"`
}

type DependencyRecord struct {
	Owner   string `json:"owner"`
	Binding string `json:"binding"`
	Target  string `json:"target"`
}

// FromWorkspace projects a validated workspace into a host-path-free lock document.
func FromWorkspace(workspace workspaceloader.Workspace) (Document, error) {
	if len(workspace.Graph.Nodes) != len(workspace.Instances) {
		return Document{}, fmt.Errorf("workspace graph and instance map differ")
	}
	ids := make(map[string]string, len(workspace.Graph.Nodes))
	for _, class := range workspace.Resolution.InstanceClasses {
		for _, member := range class.Members {
			ids[member] = class.ID.String()
		}
	}
	document := Document{
		Schema: SchemaVersion, PackageInstanceAlgorithm: packagegraph.InstanceIDEncoding,
		SourceDigestAlgorithm: packagecontent.DigestEncoding,
		Packages:              []PackageRecord{},
		Dependencies:          []DependencyRecord{},
	}
	if len(workspace.Packages) != 0 {
		document.Language = workspace.Packages[0].Manifest.Package.Language
	}
	for index, node := range workspace.Graph.Nodes {
		mapping := workspace.Instances[index]
		if mapping.NodeID != node.ID {
			return Document{}, fmt.Errorf("workspace node mapping differs at %d", index)
		}
		loaded, found := workspace.Package(mapping.PackagePath)
		if !found {
			return Document{}, fmt.Errorf("workspace package %q is missing", mapping.PackagePath)
		}
		id, found := ids[node.ID]
		if !found {
			return Document{}, fmt.Errorf("workspace node %q has no instance id", node.ID)
		}
		localPath := ""
		if loaded.SourceKind == workspaceloader.SourceKindLocal {
			localPath = mapping.PackagePath
		}
		record := PackageRecord{
			ID: id, Role: node.Role, PackageUID: node.Identity.PackageUID,
			SourceKind: loaded.SourceKind, LocalPath: localPath,
			SDKIdentity: loaded.SDKIdentity, RuntimeABI: loaded.RuntimeABI,
			SourceIdentity: node.Identity.SourceIdentity, SourceDigest: loaded.Digest.String(),
			Version: node.Identity.VersionIdentity, Features: cloneStrings(node.Identity.Features),
		}
		document.Packages = append(document.Packages, record)
		if node.Role == packagegraph.RoleRoot {
			document.Root = id
		}
	}
	slices.SortFunc(document.Packages, func(left, right PackageRecord) int { return strings.Compare(left.ID, right.ID) })
	for _, edge := range workspace.Graph.Dependencies {
		document.Dependencies = append(document.Dependencies, DependencyRecord{Owner: ids[edge.Owner], Binding: edge.Binding, Target: ids[edge.Target]})
	}
	slices.SortFunc(document.Dependencies, compareDependencies)
	if err := Validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func Marshal(document Document) ([]byte, error) {
	if err := Validate(document); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func Parse(contents []byte) (Document, error) {
	if len(contents) > MaxBytes {
		return Document{}, fmt.Errorf("lockfile exceeds %d bytes", MaxBytes)
	}
	if err := rejectDuplicateKeys(contents); err != nil {
		return Document{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("decode lockfile: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Document{}, fmt.Errorf("decode lockfile: trailing JSON value")
	}
	if err := Validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

// Read loads one bounded regular lockfile without following a final symlink.
func Read(path string) (Document, []byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Document{}, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > MaxBytes {
		return Document{}, nil, fmt.Errorf("lockfile is not a bounded regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return Document{}, nil, err
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err != nil {
		return Document{}, nil, fmt.Errorf("read bounded lockfile: %w", err)
	}
	if len(contents) > MaxBytes {
		return Document{}, nil, fmt.Errorf("lockfile exceeds %d bytes", MaxBytes)
	}
	document, err := Parse(contents)
	return document, contents, err
}

func Validate(document Document) error {
	if document.Schema != SchemaVersion || document.PackageInstanceAlgorithm != packagegraph.InstanceIDEncoding || document.SourceDigestAlgorithm != packagecontent.DigestEncoding || document.Language == "" {
		return fmt.Errorf("invalid lockfile header")
	}
	if document.Dependencies == nil {
		return fmt.Errorf("lockfile has null dependencies")
	}
	if len(document.Packages) == 0 || !slices.IsSortedFunc(document.Packages, func(left, right PackageRecord) int { return strings.Compare(left.ID, right.ID) }) {
		return fmt.Errorf("lockfile packages must be nonempty and sorted by id")
	}
	known := make(map[string]struct{}, len(document.Packages))
	rootCount := 0
	standardLibraryCount := 0
	validationGraph := packagegraph.Graph{Nodes: make([]packagegraph.Node, 0, len(document.Packages))}
	for index, record := range document.Packages {
		if index > 0 && document.Packages[index-1].ID == record.ID {
			return fmt.Errorf("duplicate lock package id %q", record.ID)
		}
		parsedID, err := packagegraph.ParseInstanceID(record.ID)
		if err != nil {
			return err
		}
		if record.Features == nil {
			return fmt.Errorf("lock package %q has null features", record.ID)
		}
		identity := packagegraph.Identity{PackageUID: record.PackageUID, SourceIdentity: record.SourceIdentity, VersionIdentity: record.Version, Features: record.Features}
		computed, err := packagegraph.ComputeInstanceID(identity)
		if err != nil || computed != parsedID {
			return fmt.Errorf("lock package %q has contradictory identity", record.ID)
		}
		digest, err := packagecontent.ParseDigest(record.SourceDigest)
		if err != nil {
			return fmt.Errorf("lock package %q has invalid source digest", record.ID)
		}
		switch record.SourceKind {
		case workspaceloader.SourceKindLocal:
			if digest.String() != record.SourceIdentity {
				return fmt.Errorf("lock package %q has contradictory local source identity", record.ID)
			}
			if !validLocalPath(record.LocalPath) {
				return fmt.Errorf("lock package %q has invalid local path", record.ID)
			}
			if record.SDKIdentity != "" || record.RuntimeABI != "" || record.Role == packagegraph.RoleStandardLibrary {
				return fmt.Errorf("lock package %q has contradictory local source origin", record.ID)
			}
		case workspaceloader.SourceKindSDK:
			standardLibraryCount++
			if record.Role != packagegraph.RoleStandardLibrary || record.LocalPath != "" ||
				record.PackageUID != sdkstdlib.PackageUID || record.Version != sdkstdlib.Version ||
				record.SDKIdentity != sdkstdlib.SDKIdentity || record.RuntimeABI != sdkstdlib.RuntimeABI ||
				document.Language != sdkstdlib.LanguageRevision || len(record.Features) != 0 ||
				record.SourceIdentity != sdkstdlib.SourceIdentity(record.SDKIdentity, record.RuntimeABI, digest) {
				return fmt.Errorf("lock package %q has contradictory SDK source origin", record.ID)
			}
		default:
			return fmt.Errorf("lock package %q has invalid source kind", record.ID)
		}
		switch record.Role {
		case packagegraph.RoleRoot:
			rootCount++
			if record.ID != document.Root || record.LocalPath != "." {
				return fmt.Errorf("lock root record is contradictory")
			}
		case packagegraph.RoleDependency, packagegraph.RoleStandardLibrary:
		default:
			return fmt.Errorf("lock package %q has invalid role", record.ID)
		}
		known[record.ID] = struct{}{}
		validationGraph.Nodes = append(validationGraph.Nodes, packagegraph.Node{ID: record.ID, Role: record.Role, Identity: identity, Modules: []packagegraph.Module{{Path: fmt.Sprintf("lockpkg%d", index)}}})
	}
	if rootCount != 1 {
		return fmt.Errorf("lockfile must contain exactly one root package")
	}
	if standardLibraryCount > 1 {
		return fmt.Errorf("lockfile contains multiple SDK standard-library packages")
	}
	if !slices.IsSortedFunc(document.Dependencies, compareDependencies) {
		return fmt.Errorf("lock dependencies are not sorted")
	}
	bindings := make(map[string]struct{}, len(document.Dependencies))
	for _, edge := range document.Dependencies {
		if _, found := known[edge.Owner]; !found {
			return fmt.Errorf("lock dependency owner is unknown")
		}
		if _, found := known[edge.Target]; !found {
			return fmt.Errorf("lock dependency target is unknown")
		}
		if edge.Binding == "" {
			return fmt.Errorf("lock dependency binding is empty")
		}
		key := edge.Owner + "\x00" + edge.Binding
		if _, duplicate := bindings[key]; duplicate {
			return fmt.Errorf("duplicate lock dependency binding")
		}
		bindings[key] = struct{}{}
		validationGraph.Dependencies = append(validationGraph.Dependencies, packagegraph.Dependency{Owner: edge.Owner, Binding: edge.Binding, Target: edge.Target})
	}
	if _, rejection, err := packagegraph.Resolve(validationGraph); err != nil {
		return fmt.Errorf("invalid lock graph: %w", err)
	} else if rejection != nil {
		return fmt.Errorf("invalid lock graph: %s", rejection.Reason)
	}
	return nil
}

func MatchesWorkspace(document Document, workspace workspaceloader.Workspace) error {
	expected, err := FromWorkspace(workspace)
	if err != nil {
		return err
	}
	actualBytes, err := Marshal(document)
	if err != nil {
		return err
	}
	expectedBytes, err := Marshal(expected)
	if err != nil {
		return err
	}
	if !bytes.Equal(actualBytes, expectedBytes) {
		return fmt.Errorf("lockfile is stale for the current workspace")
	}
	return nil
}

func WriteAtomic(path string, contents []byte) error {
	document, err := Parse(contents)
	if err != nil {
		return err
	}
	canonical, err := Marshal(document)
	if err != nil {
		return err
	}
	if !bytes.Equal(contents, canonical) {
		return fmt.Errorf("refuse to write noncanonical lockfile bytes")
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".fairy-lock-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	remove = false
	return nil
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return slices.Clone(values)
}

func compareDependencies(left, right DependencyRecord) int {
	if result := strings.Compare(left.Owner, right.Owner); result != 0 {
		return result
	}
	if result := strings.Compare(left.Binding, right.Binding); result != 0 {
		return result
	}
	return strings.Compare(left.Target, right.Target)
}

func validLocalPath(value string) bool {
	if value == "." {
		return true
	}
	return value != "" && !strings.Contains(value, "\\") && !strings.Contains(value, ":") && !strings.HasPrefix(value, "/") && filepath.ToSlash(filepath.Clean(value)) == value && value != ".." && !strings.HasPrefix(value, "../")
}

func rejectDuplicateKeys(contents []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	return scanValue(decoder)
}

func scanValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	if delimiter == '[' {
		for decoder.More() {
			if err := scanValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	}
	if delimiter != '{' {
		return fmt.Errorf("unexpected JSON delimiter")
	}
	seen := map[string]struct{}{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("invalid JSON object key")
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate JSON key %q", key)
		}
		seen[key] = struct{}{}
		if err := scanValue(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
