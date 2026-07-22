package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

const (
	workspaceCacheSchema            = "fairyc.experimental.workspace-cache-check/v0"
	maxWorkspaceCacheTempAgeSeconds = 31_536_000
)

type workspaceCacheOptions struct {
	target           target.ID
	cacheDirectory   string
	cacheEntries     int
	cacheBytes       uint64
	maxRecordBytes   uint64
	maxArtifactBytes uint64
	tempMaxAge       time.Duration
	locked           bool
}

type workspaceCacheReport struct {
	SchemaVersion  string                    `json:"schemaVersion"`
	Target         target.ID                 `json:"target"`
	CompilerBuild  string                    `json:"compilerBuild"`
	CacheDirectory string                    `json:"cacheDirectory"`
	LockState      string                    `json:"lockState"`
	Instances      int                       `json:"instances"`
	SourceFiles    int                       `json:"sourceFiles"`
	MIRFunctions   int                       `json:"mirFunctions"`
	Cache          workspaceCacheReuseReport `json:"cache"`
}

type workspaceCacheReuseReport struct {
	Storage            string `json:"storage"`
	Outcome            string `json:"outcome"`
	Packages           int    `json:"packages"`
	Hits               int    `json:"hits"`
	Misses             int    `json:"misses"`
	Rejected           int    `json:"rejected"`
	Stored             int    `json:"stored"`
	StoreSkipped       int    `json:"storeSkipped"`
	StoreFailed        int    `json:"storeFailed"`
	Lowered            bool   `json:"lowered"`
	ReassemblyFallback bool   `json:"reassemblyFallback"`
	RetainedEntries    int    `json:"retainedEntries"`
	RetainedBytes      uint64 `json:"retainedBytes"`
	RecordHits         uint64 `json:"recordHits"`
	RecordMisses       uint64 `json:"recordMisses"`
	RecordRejected     uint64 `json:"recordRejected"`
	Evictions          uint64 `json:"evictions"`
	VerifiedHits       uint64 `json:"verifiedHits"`
}

func runWorkspaceCacheCheck(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseWorkspaceCacheOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return internalError(stderr)
	}
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		return packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return packageFailure(stderr, err)
	}
	absoluteRoot, err := filepath.Abs(sourceRoot)
	if err != nil {
		return packageFailure(stderr, fmt.Errorf("resolve workspace root: %w", err))
	}
	workspaceRoot, err := os.OpenRoot(absoluteRoot)
	if err != nil {
		return packageFailure(stderr, fmt.Errorf("open workspace filesystem boundary: %w", err))
	}
	defer workspaceRoot.Close()
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target:           options.target,
		MaxArtifactBytes: options.maxArtifactBytes,
		Persistent: &workspacecompiler.PersistentConfiguration{
			Root: filepath.FromSlash(options.cacheDirectory), RootBoundary: workspaceRoot,
			Limits: packagecache.DiskLimits{
				MaxEntries: options.cacheEntries, MaxBytes: options.cacheBytes,
				MaxRecordBytes: options.maxRecordBytes, TempMaxAge: options.tempMaxAge,
			},
			CompilerBuild: compilerBuild.String(),
		},
	})
	if err != nil {
		return packageFailure(stderr, fmt.Errorf("initialize persistent workspace cache: %w", err))
	}
	defer session.Close()
	checked, err := session.Check(workspace)
	if err != nil {
		return internalError(stderr)
	}
	if len(checked.Compiler.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Compiler.Semantic.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(checked.Compiler.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, checked.Compiler.Semantic.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if !checked.Compiler.Verified {
		return internalError(stderr)
	}
	stats, err := session.PersistentStats()
	if err != nil {
		return internalError(stderr)
	}
	report := workspaceCacheReport{
		SchemaVersion: workspaceCacheSchema, Target: options.target,
		CompilerBuild: compilerBuild.String(), CacheDirectory: options.cacheDirectory,
		LockState: lockState, Instances: len(workspace.Graph.Nodes), SourceFiles: len(input.Files),
		MIRFunctions: len(checked.Compiler.MIR.Functions),
		Cache: workspaceCacheReuseReport{
			Storage: checked.Cache.Storage, Outcome: checked.Cache.Outcome, Packages: checked.Cache.Packages,
			Hits: checked.Cache.Hits, Misses: checked.Cache.Misses, Rejected: checked.Cache.Rejected,
			Stored: checked.Cache.Stored, StoreSkipped: checked.Cache.StoreSkipped, StoreFailed: checked.Cache.StoreFailed,
			Lowered: checked.Cache.Lowered, ReassemblyFallback: checked.Cache.ReassemblyFallback,
			RetainedEntries: stats.Store.Entries, RetainedBytes: stats.Store.Bytes,
			RecordHits: stats.Store.Hits, RecordMisses: stats.Store.Misses,
			RecordRejected: stats.Store.Rejected, Evictions: stats.Store.Evictions,
			VerifiedHits: stats.VerifiedHits,
		},
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseWorkspaceCacheOptions(arguments []string) (workspaceCacheOptions, error) {
	options := workspaceCacheOptions{}
	seen := make(map[string]bool, 8)
	for index := 0; index < len(arguments); {
		flag := arguments[index]
		if flag == "--locked" {
			if seen[flag] {
				return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" received %s more than once", flag)
			}
			seen[flag], options.locked = true, true
			index++
			continue
		}
		switch flag {
		case "--target", "--cache-dir", "--cache-max-entries", "--cache-max-bytes", "--cache-max-record-bytes", "--cache-max-artifact-bytes", "--cache-temp-max-age-seconds":
		default:
			return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" does not accept flag %q", flag)
		}
		if seen[flag] {
			return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" received %s more than once", flag)
		}
		if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
			return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" requires a value for %s", flag)
		}
		seen[flag] = true
		value := arguments[index+1]
		switch flag {
		case "--target":
			candidate := target.ID(value)
			if _, found := target.Lookup(candidate); !found {
				return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" requires a known --target")
			}
			options.target = candidate
		case "--cache-dir":
			parsed, err := validateWorkspaceCacheDirectory(value)
			if err != nil {
				return workspaceCacheOptions{}, err
			}
			options.cacheDirectory = parsed
		case "--cache-max-entries":
			parsed, err := parseWorkspaceCacheLimit(flag, value, uint64(maxWorkspaceSessionCacheEntries))
			if err != nil {
				return workspaceCacheOptions{}, err
			}
			options.cacheEntries = int(parsed)
		case "--cache-max-bytes":
			parsed, err := parseWorkspaceCacheLimit(flag, value, maxWorkspaceSessionCacheBytes)
			if err != nil {
				return workspaceCacheOptions{}, err
			}
			options.cacheBytes = parsed
		case "--cache-max-record-bytes":
			parsed, err := parseWorkspaceCacheLimit(flag, value, maxWorkspaceSessionCacheBytes)
			if err != nil {
				return workspaceCacheOptions{}, err
			}
			options.maxRecordBytes = parsed
		case "--cache-max-artifact-bytes":
			parsed, err := parseWorkspaceCacheLimit(flag, value, maxWorkspaceSessionArtifactBytes)
			if err != nil {
				return workspaceCacheOptions{}, err
			}
			options.maxArtifactBytes = parsed
		case "--cache-temp-max-age-seconds":
			parsed, err := parseWorkspaceCacheLimit(flag, value, maxWorkspaceCacheTempAgeSeconds)
			if err != nil {
				return workspaceCacheOptions{}, err
			}
			options.tempMaxAge = time.Duration(parsed) * time.Second
		}
		index += 2
	}
	for _, flag := range []string{"--target", "--cache-dir", "--cache-max-entries", "--cache-max-bytes", "--cache-max-record-bytes", "--cache-max-artifact-bytes", "--cache-temp-max-age-seconds"} {
		if !seen[flag] {
			return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" requires %s", flag)
		}
	}
	if options.maxRecordBytes > options.cacheBytes {
		return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" requires --cache-max-record-bytes not to exceed --cache-max-bytes")
	}
	if options.maxArtifactBytes > options.maxRecordBytes {
		return workspaceCacheOptions{}, fmt.Errorf("experimental command \"check-workspace-cache\" requires --cache-max-artifact-bytes not to exceed --cache-max-record-bytes")
	}
	return options, nil
}

func parseWorkspaceCacheLimit(flag, value string, maximum uint64) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || parsed > maximum || value != strconv.FormatUint(parsed, 10) {
		return 0, fmt.Errorf("experimental command \"check-workspace-cache\" requires %s as a canonical decimal from 1 through %d", flag, maximum)
	}
	return parsed, nil
}

func validateWorkspaceCacheDirectory(value string) (string, error) {
	label := "experimental workspace cache directory"
	if value == "" || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("%s must be valid non-empty UTF-8", label)
	}
	if len([]byte(value)) > maxBuildPathBytes {
		return "", fmt.Errorf("%s exceeds %d UTF-8 bytes", label, maxBuildPathBytes)
	}
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf("%s must use forward slashes", label)
	}
	if path.IsAbs(value) || strings.HasPrefix(value, "//") || isWindowsVolumePath(value) {
		return "", fmt.Errorf("%s must be relative to the workspace root", label)
	}
	if path.Clean(value) != value {
		return "", fmt.Errorf("%s must be a canonical relative path", label)
	}
	components := strings.Split(value, "/")
	if len(components) < 2 || components[0] != "build" {
		return "", fmt.Errorf("%s must be inside the ignored build/ directory", label)
	}
	for _, component := range components {
		if err := validatePortablePathComponent(label, component); err != nil {
			return "", err
		}
	}
	return value, nil
}

func workspaceCacheReportForSchema() workspaceCacheReport {
	return workspaceCacheReport{Cache: workspaceCacheReuseReport{}}
}
