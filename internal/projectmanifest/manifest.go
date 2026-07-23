// Package projectmanifest parses and validates the human-authored local-exact Fairy C manifest.
package projectmanifest

import (
	"fmt"
	pathpkg "path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

const (
	FileName      = "fairy.toml"
	SchemaVersion = "fairyc.manifest/v1"
	MaxBytes      = 1 << 20

	ReasonInputTooLarge     = "input-too-large"
	ReasonInvalidUTF8       = "invalid-utf8"
	ReasonParse             = "parse-error"
	ReasonUnknownField      = "unknown-field"
	ReasonMissingField      = "missing-field"
	ReasonInvalidPackageUID = "invalid-package-uid"
	ReasonInvalidName       = "invalid-name"
	ReasonInvalidVersion    = "invalid-version"
	ReasonInvalidLanguage   = "invalid-language-revision"
	ReasonInvalidPath       = "invalid-path"
	ReasonNotSortedUnique   = "not-sorted-unique"
	ReasonUnsupportedSchema = "unsupported-schema"
)

var (
	kebabPattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	uidPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*(?:\.[a-z][a-z0-9]*(?:-[a-z0-9]+)*)+$`)
	languagePattern = regexp.MustCompile(`^fairyc-language/(?:0|[1-9][0-9]*)$`)
	semverPattern   = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	pathSegment     = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)
)

// ValidPackageName reports whether value has the exact manifest-v1 package-name spelling.
func ValidPackageName(value string) bool {
	return kebabPattern.MatchString(value)
}

// ValidPortablePathSegment reports whether value can name one manifest-v1 directory component on
// every admitted host, including Windows reserved-device-name rejection.
func ValidPortablePathSegment(value string) bool {
	return validPathSegment(value)
}

// ValidPackageUID reports whether value has the exact manifest-v1 package UID spelling.
func ValidPackageUID(value string) bool {
	return uidPattern.MatchString(value)
}

// ValidationError exposes fixture-stable categories without freezing final user diagnostics.
type ValidationError struct {
	Reason string
	Field  string
	Detail string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("manifest %s: %s", e.Reason, e.Detail)
	}
	return fmt.Sprintf("manifest %s %s: %s", e.Field, e.Reason, e.Detail)
}

// Manifest is the validated, deterministic semantic form of one human project manifest.
type Manifest struct {
	Schema       string
	Package      Package
	Workspace    Workspace
	Dependencies []Dependency
}

type Package struct {
	UID      string
	Name     string
	Version  string
	Language string
	Source   string
}

type Workspace struct {
	Present bool
	Members []string
}

type Dependency struct {
	Binding  string
	Path     string
	Version  string
	Features []string
}

type rawManifest struct {
	Schema       string                   `toml:"schema"`
	Package      rawPackage               `toml:"package"`
	Workspace    rawWorkspace             `toml:"workspace"`
	Dependencies map[string]rawDependency `toml:"dependencies"`
}

type rawPackage struct {
	UID      string `toml:"uid"`
	Name     string `toml:"name"`
	Version  string `toml:"version"`
	Language string `toml:"language"`
	Source   string `toml:"source"`
}

type rawWorkspace struct {
	Members []string `toml:"members"`
}

type rawDependency struct {
	Path     string   `toml:"path"`
	Version  string   `toml:"version"`
	Features []string `toml:"features"`
}

// Parse decodes one bounded TOML document, rejects unknown or omitted fields, and returns a
// deterministic dependency ordering. Filesystem containment and member-manifest consistency belong
// to the workspace loader, not this byte parser.
func Parse(contents []byte) (Manifest, error) {
	if len(contents) > MaxBytes {
		return Manifest{}, &ValidationError{Reason: ReasonInputTooLarge, Detail: fmt.Sprintf("maximum is %d bytes", MaxBytes)}
	}
	if !utf8.Valid(contents) {
		return Manifest{}, &ValidationError{Reason: ReasonInvalidUTF8, Detail: "input is not valid UTF-8"}
	}
	var raw rawManifest
	metadata, err := toml.Decode(string(contents), &raw)
	if err != nil {
		return Manifest{}, &ValidationError{Reason: ReasonParse, Detail: err.Error()}
	}
	if unknown := metadata.Undecoded(); len(unknown) != 0 {
		return Manifest{}, &ValidationError{Reason: ReasonUnknownField, Field: unknown[0].String(), Detail: "field is not part of manifest v1"}
	}
	if err := requireFields(metadata, raw); err != nil {
		return Manifest{}, err
	}
	if raw.Schema != SchemaVersion {
		return Manifest{}, &ValidationError{Reason: ReasonUnsupportedSchema, Field: "schema", Detail: fmt.Sprintf("got %q", raw.Schema)}
	}
	if !ValidPackageUID(raw.Package.UID) {
		return Manifest{}, &ValidationError{Reason: ReasonInvalidPackageUID, Field: "package.uid", Detail: "expected at least two lowercase dot-separated segments"}
	}
	if !ValidPackageName(raw.Package.Name) {
		return Manifest{}, &ValidationError{Reason: ReasonInvalidName, Field: "package.name", Detail: "expected lowercase kebab-case"}
	}
	if !validSemver(raw.Package.Version) {
		return Manifest{}, &ValidationError{Reason: ReasonInvalidVersion, Field: "package.version", Detail: "expected exact SemVer 2.0.0"}
	}
	if !languagePattern.MatchString(raw.Package.Language) {
		return Manifest{}, &ValidationError{Reason: ReasonInvalidLanguage, Field: "package.language", Detail: "expected fairyc-language/<revision>"}
	}
	if !validRelativeDirectory(raw.Package.Source) {
		return Manifest{}, &ValidationError{Reason: ReasonInvalidPath, Field: "package.source", Detail: "expected a canonical portable relative directory"}
	}
	if err := validateSortedPaths("workspace.members", raw.Workspace.Members); err != nil {
		return Manifest{}, err
	}

	bindings := make([]string, 0, len(raw.Dependencies))
	for binding := range raw.Dependencies {
		bindings = append(bindings, binding)
	}
	slices.Sort(bindings)
	dependencies := make([]Dependency, 0, len(bindings))
	for _, binding := range bindings {
		definition := raw.Dependencies[binding]
		if !kebabPattern.MatchString(binding) {
			return Manifest{}, &ValidationError{Reason: ReasonInvalidName, Field: "dependencies." + binding, Detail: "binding must be lowercase kebab-case"}
		}
		prefix := "dependencies." + binding
		if !validRelativeDirectory(definition.Path) {
			return Manifest{}, &ValidationError{Reason: ReasonInvalidPath, Field: prefix + ".path", Detail: "expected a canonical portable relative directory"}
		}
		if !validSemver(definition.Version) {
			return Manifest{}, &ValidationError{Reason: ReasonInvalidVersion, Field: prefix + ".version", Detail: "expected exact SemVer 2.0.0"}
		}
		if err := validateSortedNames(prefix+".features", definition.Features); err != nil {
			return Manifest{}, err
		}
		dependencies = append(dependencies, Dependency{
			Binding: binding, Path: definition.Path, Version: definition.Version,
			Features: slices.Clone(definition.Features),
		})
	}
	return Manifest{
		Schema: raw.Schema,
		Package: Package{
			UID: raw.Package.UID, Name: raw.Package.Name, Version: raw.Package.Version,
			Language: raw.Package.Language, Source: raw.Package.Source,
		},
		Workspace: Workspace{
			Present: metadata.IsDefined("workspace"),
			Members: slices.Clone(raw.Workspace.Members),
		},
		Dependencies: dependencies,
	}, nil
}

func requireFields(metadata toml.MetaData, raw rawManifest) error {
	required := [][]string{
		{"schema"},
		{"package"},
		{"package", "uid"},
		{"package", "name"},
		{"package", "version"},
		{"package", "language"},
		{"package", "source"},
	}
	for _, key := range required {
		if !metadata.IsDefined(key...) {
			return &ValidationError{Reason: ReasonMissingField, Field: strings.Join(key, "."), Detail: "field is required"}
		}
	}
	if metadata.IsDefined("workspace") && !metadata.IsDefined("workspace", "members") {
		return &ValidationError{Reason: ReasonMissingField, Field: "workspace.members", Detail: "field is required when workspace is present"}
	}
	for binding := range raw.Dependencies {
		for _, field := range []string{"path", "version", "features"} {
			if !metadata.IsDefined("dependencies", binding, field) {
				return &ValidationError{Reason: ReasonMissingField, Field: "dependencies." + binding + "." + field, Detail: "field is required"}
			}
		}
	}
	return nil
}

func validSemver(value string) bool {
	match := semverPattern.FindStringSubmatch(value)
	if match == nil {
		return false
	}
	if match[4] == "" {
		return true
	}
	for _, identifier := range strings.Split(match[4], ".") {
		if len(identifier) > 1 && identifier[0] == '0' {
			numeric := true
			for _, character := range identifier {
				if character < '0' || character > '9' {
					numeric = false
					break
				}
			}
			if numeric {
				return false
			}
		}
	}
	return true
}

func validRelativeDirectory(value string) bool {
	if value == "" || !utf8.ValidString(value) || strings.Contains(value, "\\") ||
		strings.Contains(value, ":") || strings.HasPrefix(value, "/") ||
		pathpkg.Clean(value) != value || value == "." || value == ".." ||
		strings.HasPrefix(value, "../") || strings.HasSuffix(value, "/") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if !ValidPortablePathSegment(segment) {
			return false
		}
	}
	return true
}

func validPathSegment(value string) bool {
	if !pathSegment.MatchString(value) || strings.HasSuffix(value, ".") {
		return false
	}
	base := strings.ToUpper(strings.SplitN(value, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	default:
		return true
	}
}

func validateSortedPaths(field string, values []string) error {
	for index, value := range values {
		if !validRelativeDirectory(value) {
			return &ValidationError{Reason: ReasonInvalidPath, Field: field, Detail: fmt.Sprintf("invalid member %q", value)}
		}
		if index > 0 && values[index-1] >= value {
			return &ValidationError{Reason: ReasonNotSortedUnique, Field: field, Detail: "values must be sorted and unique"}
		}
	}
	return nil
}

func validateSortedNames(field string, values []string) error {
	for index, value := range values {
		if !kebabPattern.MatchString(value) {
			return &ValidationError{Reason: ReasonInvalidName, Field: field, Detail: fmt.Sprintf("invalid feature %q", value)}
		}
		if index > 0 && values[index-1] >= value {
			return &ValidationError{Reason: ReasonNotSortedUnique, Field: field, Detail: "values must be sorted and unique"}
		}
	}
	return nil
}
