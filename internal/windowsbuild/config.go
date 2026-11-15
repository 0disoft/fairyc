// Package windowsbuild owns explicit native Windows build inputs, separate from
// the source-language manifest and its frontend cache/lock identity.
package windowsbuild

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const FileName = "fairy.windows.json"
const Schema = "fairyc.windows-build/v1"
const MaxConfigBytes = 65536
const MaxInputBytes = 4 << 20

type Config struct {
	Schema    string   `json:"schema"`
	Subsystem string   `json:"subsystem"`
	Libraries []string `json:"libraries,omitempty"`
	Sources   []string `json:"sources,omitempty"`
	Resources []string `json:"resources,omitempty"`
}

type Input struct {
	Name string
	Data []byte
}

type Project struct {
	Root string
	Config
	CInputs, RCInputs []Input
}

var library = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}\.lib$`)
var segment = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)

func ValidPath(name, extension string) bool {
	if len(name) > 240 || path.Ext(name) != extension || !filepath.IsLocal(name) || strings.Contains(name, "\\") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if !segment.MatchString(part) || strings.HasSuffix(part, ".") {
			return false
		}
	}
	return true
}

func Parse(data []byte) (Config, error) {
	var c Config
	if len(data) == 0 || len(data) > MaxConfigBytes || !utf8.Valid(data) {
		return c, fmt.Errorf("%s must be bounded UTF-8 JSON", FileName)
	}
	d := json.NewDecoder(bytes.NewReader(data))
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return c, fmt.Errorf("%s must be an object", FileName)
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return c, err
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return c, fmt.Errorf("duplicate/invalid Windows build field %q", name)
		}
		seen[name] = true
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return c, err
		}
		if bytes.Equal(raw, []byte("null")) {
			return c, fmt.Errorf("Windows build field %s cannot be null", name)
		}
		var value any
		switch name {
		case "schema":
			value = &c.Schema
		case "subsystem":
			value = &c.Subsystem
		case "libraries":
			value = &c.Libraries
		case "sources":
			value = &c.Sources
		case "resources":
			value = &c.Resources
		default:
			return c, fmt.Errorf("unknown Windows build field %s", name)
		}
		if err := json.Unmarshal(raw, value); err != nil {
			return c, err
		}
	}
	if _, err := d.Token(); err != nil {
		return c, err
	}
	if _, err := d.Token(); err != io.EOF {
		return c, fmt.Errorf("trailing Windows build JSON")
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if c.Schema != Schema || c.Subsystem != "console" && c.Subsystem != "windows" {
		return fmt.Errorf("Windows build requires schema %s and subsystem console/windows", Schema)
	}
	for _, group := range []struct {
		values    []string
		max       int
		extension string
	}{
		{c.Libraries, 32, ""}, {c.Sources, 16, ".c"}, {c.Resources, 8, ".rc"},
	} {
		if len(group.values) > group.max {
			return fmt.Errorf("too many Windows build inputs")
		}
		seen := map[string]bool{}
		for _, name := range group.values {
			valid := library.MatchString(name)
			if group.extension != "" {
				valid = ValidPath(name, group.extension)
			}
			key := strings.ToLower(name)
			if !valid || seen[key] {
				return fmt.Errorf("invalid/duplicate Windows build input %q", name)
			}
			seen[key] = true
		}
	}
	return nil
}

// Load uses a rooted filesystem, rejects declared symlink components, and
// snapshots declared C/RC text before compilation. Includes/assets are trusted
// native-tool inputs, not a sandbox or part of the Fairy frontend lock.
func Load(directory string) (*Project, error) {
	return LoadWithOptions(directory)
}

// LoadWithOptions preserves Load's inputs and results while allowing callers to
// supply cooperative cancellation and per-call filesystem dependencies.
func LoadWithOptions(directory string, options ...LoadOptions) (project *Project, err error) {
	policy, err := loadPolicy(options)
	if err != nil {
		return nil, err
	}
	if err := inputReadError(policy.Context, nil); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	root, err := policy.OpenRoot(absolute)
	if root != nil {
		defer func() {
			err = inputReadError(policy.Context, errors.Join(err, root.Close()))
			if err != nil {
				project = nil
			}
		}()
	}
	if err = inputReadError(policy.Context, err); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("Windows build root handle is missing")
	}
	data, err := read(root, FileName, MaxConfigBytes, policy)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	config, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	p := &Project{Root: absolute, Config: config}
	total := 0
	for _, group := range []struct {
		names  []string
		output *[]Input
	}{{config.Sources, &p.CInputs}, {config.Resources, &p.RCInputs}} {
		for _, name := range group.names {
			data, err := read(root, name, MaxInputBytes, policy)
			if err != nil {
				return nil, fmt.Errorf("Windows build input %s: %w", name, err)
			}
			total += len(data)
			if total > 32<<20 {
				return nil, fmt.Errorf("Windows build inputs exceed 32 MiB")
			}
			*group.output = append(*group.output, Input{Name: name, Data: data})
		}
	}
	if err := inputReadError(policy.Context, nil); err != nil {
		return nil, err
	}
	return p, nil
}
