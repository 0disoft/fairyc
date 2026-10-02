package adoptionaudit

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

type checkpoint struct {
	StartedPID    int  `json:"startedPid"`
	CheckpointPID int  `json:"checkpointPid"`
	KilledPID     int  `json:"killedPid"`
	Exited        bool `json:"exited"`
	Locked        bool `json:"lockAcquiredAfterCheckpoint"`
}
type record struct {
	Kind       string          `json:"kind"`
	Executable string          `json:"executable"`
	Cwd        string          `json:"cwd"`
	Argv       []string        `json:"argv"`
	Actual     json.RawMessage `json:"actual"`
	Checkpoint checkpoint      `json:"checkpoint"`
	raw        json.RawMessage
}
type image struct {
	Variant    string `json:"variant"`
	Profile    string `json:"profile"`
	Executable string `json:"executable"`
	SHA256     string `json:"sha256"`
}
type entry struct {
	Variant     string          `json:"variant"`
	Profile     string          `json:"profile"`
	Name        string          `json:"name"`
	Cwd         string          `json:"cwd"`
	Argv        []string        `json:"argv"`
	Actual      json.RawMessage `json:"actual"`
	Expected    json.RawMessage `json:"expected"`
	InputBefore string          `json:"inputSha256Before"`
	InputAfter  string          `json:"inputSha256After"`
	Pointer     struct {
		Path string `json:"path"`
		Line int    `json:"line"`
	} `json:"rawEvidence"`
	Outputs []struct {
		Path     string `json:"path"`
		Actual   string `json:"actualSha256"`
		Expected string `json:"expectedSha256"`
	} `json:"outputs"`
	Evidence struct {
		Kind          string     `json:"kind"`
		Checkpoint    checkpoint `json:"checkpoint"`
		PowerLoss     bool       `json:"powerLossClaim"`
		ActualFailure bool       `json:"isActualSdkFailure"`
	} `json:"evidence"`
}
type receipt struct {
	Schema         string            `json:"schema"`
	Passed         int               `json:"passed"`
	Failed         int               `json:"failed"`
	Skipped        int               `json:"skipped"`
	CasesPerImage  int               `json:"casesPerImage"`
	Images         []image           `json:"images"`
	Cases          []entry           `json:"cases"`
	CompilerSHA256 string            `json:"compilerSha256"`
	Protected      map[string]string `json:"protectedFiles"`
	HarnessSHA256  string            `json:"harnessSha256"`
	AuditSHA256    string            `json:"auditHelperSha256"`
	ProcessSHA256  string            `json:"processHelperSha256"`
}

// Options allows an explicitly supplied relocated evidence tree. PathMapFrom
// replaces only the given absolute prefix; it never searches a personal cache.
type Options struct{ Kind, Receipt, Compiler, PathMapFrom, PathMapTo string }
type Result struct {
	Kind             string `json:"kind"`
	Cases            int    `json:"ordinalCases"`
	NativeExecutions int    `json:"nativeExecutions"`
	ReceiptSHA256    string `json:"receiptSha256"`
}

func digest(data []byte) string { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }
func (o Options) path(path string) string {
	p := strings.ReplaceAll(path, "\\", "/")
	from := strings.TrimRight(strings.ReplaceAll(o.PathMapFrom, "\\", "/"), "/")
	if from != "" && (p == from || strings.HasPrefix(p, from+"/")) {
		p = strings.TrimRight(filepath.ToSlash(o.PathMapTo), "/") + strings.TrimPrefix(p, from)
	}
	return filepath.FromSlash(p)
}
func (o Options) checkHash(path, want string) error {
	if len(want) != 64 {
		return fmt.Errorf("missing SHA256 for %s", path)
	}
	data, err := os.ReadFile(o.path(path))
	if err != nil {
		return err
	}
	if digest(data) != want {
		return fmt.Errorf("SHA256 changed: %s", path)
	}
	return nil
}

func Audit(o Options) (Result, error) {
	absolute, err := filepath.Abs(o.Receipt)
	if err != nil {
		return Result{}, err
	}
	o.Receipt = absolute
	count, ok := map[string]int{"ledger": 68, "log": 104, "pack": 128, "kv": 328}[o.Kind]
	if !ok {
		return Result{}, fmt.Errorf("unknown kind")
	}
	data, err := os.ReadFile(o.Receipt)
	if err != nil {
		return Result{}, err
	}
	if !utf8.Valid(data) {
		return Result{}, fmt.Errorf("receipt is not UTF-8")
	}
	var r receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return Result{}, err
	}
	if r.Schema != "fairyc."+o.Kind+"-compare-adoption/v1" || r.Passed != count || len(r.Cases) != count || len(r.Images) != 4 || r.Failed != 0 || r.Skipped != 0 || r.CasesPerImage != count/4 {
		return Result{}, fmt.Errorf("unexpected receipt inventory")
	}
	root := filepath.Dir(o.Receipt)
	images := map[string]image{}
	for _, im := range r.Images {
		key := im.Variant + "/" + im.Profile
		_, duplicate := images[key]
		if duplicate || (im.Variant != "legacy" && im.Variant != "sdk") || (im.Profile != "development" && im.Profile != "release") {
			return Result{}, fmt.Errorf("unexpected image")
		}
		images[key] = im
		if err := o.checkHash(im.Executable, im.SHA256); err != nil {
			return Result{}, err
		}
	}
	if err := o.checkHash(o.Compiler, r.CompilerSHA256); err != nil {
		return Result{}, err
	}
	for path, hash := range r.Protected {
		if err := o.checkHash(path, hash); err != nil {
			return Result{}, err
		}
	}
	for file, hash := range map[string]string{"harness-executed.ps1": r.HarnessSHA256, "adoption-output-audit-executed.ps1": r.AuditSHA256, "adoption-process-executed.ps1": r.ProcessSHA256} {
		if hash != "" {
			if err := o.checkHash(filepath.Join(root, file), hash); err != nil {
				return Result{}, err
			}
		}
	}
	flat := o.Kind == "pack" || o.Kind == "kv"
	files := []string{"raw-executions.jsonl"}
	if flat {
		files = []string{"legacy/evidence/raw-executions.jsonl", "sdk/evidence/raw-executions.jsonl"}
	}
	rawFiles := map[string][]record{}
	var programs []record
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		content, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		if !utf8.Valid(content) {
			return Result{}, fmt.Errorf("raw file is not UTF-8")
		}
		scan := bufio.NewScanner(bytes.NewReader(content))
		scan.Buffer(make([]byte, 4096), 16<<20)
		var records []record
		for scan.Scan() {
			var rec record
			if err := json.Unmarshal(scan.Bytes(), &rec); err != nil {
				return Result{}, err
			}
			if rec.Kind == "program" {
				var fields struct {
					Argv []json.RawMessage `json:"argv"`
				}
				if err := json.Unmarshal(scan.Bytes(), &fields); err != nil || fields.Argv == nil {
					return Result{}, fmt.Errorf("missing or malformed raw argv")
				}
				for _, argument := range fields.Argv {
					var value *string
					if json.Unmarshal(argument, &value) != nil || value == nil {
						return Result{}, fmt.Errorf("raw argv requires strings")
					}
				}
			}
			rec.raw = append(json.RawMessage(nil), scan.Bytes()...)
			records = append(records, rec)
			if rec.Kind == "program" {
				programs = append(programs, rec)
			}
		}
		if err := scan.Err(); err != nil {
			return Result{}, err
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return Result{}, err
		}
		rawFiles[abs] = records
	}
	if len(programs) != count {
		return Result{}, fmt.Errorf("raw program inventory mismatch")
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for i, c := range r.Cases {
		key := c.Variant + "/" + c.Profile
		im, ok := images[key]
		if !ok || seen[key+"/"+c.Name] {
			return Result{}, fmt.Errorf("unexpected or duplicate case")
		}
		seen[key+"/"+c.Name] = true
		counts[key]++
		if err := compare(c.Actual, c.Expected); err != nil {
			return Result{}, fmt.Errorf("%s: %w", c.Name, err)
		}
		raw := programs[i]
		if raw.Kind != "program" || o.path(raw.Executable) != o.path(im.Executable) || filepath.Clean(o.path(raw.Cwd)) != filepath.Join(root, c.Variant) {
			return Result{}, fmt.Errorf("raw executable/workspace mismatch: %s", c.Name)
		}
		actual := raw.Actual
		if flat {
			actual = raw.raw
			pointer, err := filepath.Abs(o.path(c.Pointer.Path))
			if err != nil {
				return Result{}, err
			}
			records := rawFiles[pointer]
			if c.Pointer.Line < 1 || c.Pointer.Line > len(records) || !jsonEqual(records[c.Pointer.Line-1].raw, raw.raw) {
				return Result{}, fmt.Errorf("raw pointer/order mismatch")
			}
			if o.path(c.Cwd) != o.path(raw.Cwd) || !slices.Equal(c.Argv, raw.Argv) {
				return Result{}, fmt.Errorf("raw cwd/argv mismatch")
			}
			for _, out := range c.Outputs {
				if out.Actual != out.Expected {
					return Result{}, fmt.Errorf("output oracle mismatch")
				}
				if err := o.checkHash(out.Path, out.Actual); err != nil {
					return Result{}, err
				}
			}
			if o.Kind == "kv" {
				cp := raw.Checkpoint
				if c.Evidence.Kind == "checkpoint-kill" && (cp.StartedPID <= 0 || cp.StartedPID != cp.CheckpointPID || cp.StartedPID != cp.KilledPID || !cp.Exited || cp.StartedPID != c.Evidence.Checkpoint.StartedPID || c.Evidence.PowerLoss) {
					return Result{}, fmt.Errorf("owned child evidence mismatch")
				}
				if c.Evidence.Kind == "actual-sdk-os-publication-failure" && (!cp.Locked || !c.Evidence.ActualFailure) {
					return Result{}, fmt.Errorf("SDK failure evidence mismatch")
				}
			}
		} else if c.InputBefore != "" {
			if len(raw.Argv) == 0 {
				return Result{}, fmt.Errorf("missing input argv")
			}
			input := filepath.Join(o.path(raw.Cwd), filepath.FromSlash(raw.Argv[0]))
			rel, err := filepath.Rel(o.path(raw.Cwd), input)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(raw.Argv[0]) || c.InputBefore != c.InputAfter {
				return Result{}, fmt.Errorf("input preservation mismatch")
			}
			if err := o.checkHash(input, c.InputBefore); err != nil {
				return Result{}, err
			}
		}
		if err := compare(actual, c.Actual); err != nil {
			return Result{}, fmt.Errorf("raw case %s: %w", c.Name, err)
		}
	}
	for key := range images {
		if counts[key] != r.CasesPerImage {
			return Result{}, fmt.Errorf("per-image count mismatch")
		}
	}
	after, err := os.ReadFile(o.Receipt)
	if err != nil {
		return Result{}, err
	}
	if !bytes.Equal(data, after) {
		return Result{}, fmt.Errorf("receipt changed during audit")
	}
	return Result{o.Kind, count, 0, digest(data)}, nil
}

func jsonEqual(a, b json.RawMessage) bool {
	var left, right any
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	x, _ := json.Marshal(left)
	y, _ := json.Marshal(right)
	return bytes.Equal(x, y)
}
