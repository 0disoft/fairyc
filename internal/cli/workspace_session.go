package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

const (
	workspaceSessionSchema            = "fairyc.experimental.workspace-check-session/v0"
	workspaceSessionRequestKind       = "check"
	workspaceSessionResponseKind      = "check-result"
	workspaceSessionStatusOK          = "ok"
	workspaceSessionStatusDiagnostics = "source-diagnostics"
	workspaceSessionStatusPackage     = "package-failure"
	workspaceSessionCacheNotRun       = "not-run"

	maxWorkspaceSessionRequests      = 256
	maxWorkspaceSessionRequestBytes  = 1024
	maxWorkspaceSessionCacheEntries  = 65_536
	maxWorkspaceSessionCacheBytes    = 1 << 30
	maxWorkspaceSessionArtifactBytes = 1 << 30
)

type workspaceSessionOptions struct {
	target           target.ID
	maxRequests      int
	cacheEntries     int
	cacheBytes       uint64
	maxArtifactBytes uint64
	locked           bool
}

type workspaceSessionRequest struct {
	SchemaVersion string
	Kind          string
	Sequence      int
}

type workspaceSessionResponse struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Kind          string                      `json:"kind"`
	Sequence      int                         `json:"sequence"`
	Status        string                      `json:"status"`
	ExitCode      int                         `json:"exitCode"`
	LockState     string                      `json:"lockState"`
	Instances     int                         `json:"instances"`
	SourceFiles   int                         `json:"sourceFiles"`
	MIRFunctions  int                         `json:"mirFunctions"`
	Diagnostics   int                         `json:"diagnostics"`
	Unsupported   int                         `json:"unsupported"`
	Cache         workspaceSessionCacheReport `json:"cache"`
}

type workspaceSessionCacheReport struct {
	Outcome            string `json:"outcome"`
	Packages           int    `json:"packages"`
	Hits               int    `json:"hits"`
	Misses             int    `json:"misses"`
	Rejected           int    `json:"rejected"`
	Stored             int    `json:"stored"`
	StoreSkipped       int    `json:"storeSkipped"`
	Lowered            bool   `json:"lowered"`
	ReassemblyFallback bool   `json:"reassemblyFallback"`
	RetainedEntries    int    `json:"retainedEntries"`
	RetainedBytes      uint64 `json:"retainedBytes"`
	VerifiedHits       uint64 `json:"verifiedHits"`
}

type workspaceSessionInputError struct {
	cause error
}

func (value workspaceSessionInputError) Error() string {
	return value.cause.Error()
}

func runWorkspaceCheckSession(arguments []string, sourceRoot string, stdin io.Reader, stdout, stderr io.Writer) int {
	options, err := parseWorkspaceSessionOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	session, err := workspacecompiler.NewSourceSession(workspacecompiler.Configuration{
		Target: options.target,
		Cache: packagecache.Limits{
			MaxEntries: options.cacheEntries,
			MaxBytes:   options.cacheBytes,
		},
		MaxArtifactBytes: options.maxArtifactBytes,
	})
	if err != nil {
		return invocationError(stderr, fmt.Sprintf("invalid workspace session cache policy: %v", err))
	}

	defer session.Close()
	reader := bufio.NewReaderSize(stdin, maxWorkspaceSessionRequestBytes+2)
	lastExitCode := ExitSuccess
	requests := 0
	for requests < options.maxRequests {
		line, eof, err := readWorkspaceSessionLine(reader)
		if err != nil {
			var inputFailure workspaceSessionInputError
			if errors.As(err, &inputFailure) {
				if _, writeErr := fmt.Fprintln(stderr, "fairyc: workspace session input read failed"); writeErr != nil {
					return ExitInternalFailure
				}
				return ExitInternalFailure
			}
			return invocationError(stderr, err.Error())
		}
		if eof {
			break
		}
		expectedSequence := requests + 1
		request, err := decodeWorkspaceSessionRequest(line, expectedSequence)
		if err != nil {
			return invocationError(stderr, err.Error())
		}
		response, fatalCode := executeWorkspaceSessionCheck(session, request.Sequence, sourceRoot, options.locked, stderr)
		if fatalCode != ExitSuccess {
			return fatalCode
		}
		if err := writeWorkspaceSessionResponse(stdout, response); err != nil {
			return outputError(stderr)
		}
		lastExitCode = response.ExitCode
		requests++
	}
	if requests == 0 {
		return invocationError(stderr, "experimental command \"check-workspace-session\" requires at least one NDJSON request before EOF")
	}
	return lastExitCode
}

func parseWorkspaceSessionOptions(arguments []string) (workspaceSessionOptions, error) {
	options := workspaceSessionOptions{}
	_, err := parseCommandFlags(arguments, flagSchema{Command: "check-workspace-session", Experimental: true,
		Flags: []flagDefinition{{Name: "--target", Kind: flagScalar}, {Name: "--max-requests", Kind: flagScalar}, {Name: "--cache-max-entries", Kind: flagScalar}, {Name: "--cache-max-bytes", Kind: flagScalar}, {Name: "--cache-max-artifact-bytes", Kind: flagScalar}, {Name: "--locked", Kind: flagSwitch}}, Required: []string{"--target", "--max-requests", "--cache-max-entries", "--cache-max-bytes", "--cache-max-artifact-bytes"}}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--target":
			candidate := target.ID(value)
			if _, found := target.Lookup(candidate); !found {
				return fmt.Errorf("experimental command \"check-workspace-session\" requires a known --target")
			}
			options.target = candidate
		case "--max-requests":
			parsed, err := parseWorkspaceSessionLimit(flag, value, maxWorkspaceSessionRequests)
			if err != nil {
				return err
			}
			options.maxRequests = parsed
		case "--cache-max-entries":
			parsed, err := parseWorkspaceSessionLimit(flag, value, maxWorkspaceSessionCacheEntries)
			if err != nil {
				return err
			}
			options.cacheEntries = parsed
		case "--cache-max-bytes":
			parsed, err := parseWorkspaceSessionByteLimit(flag, value, maxWorkspaceSessionCacheBytes)
			if err != nil {
				return err
			}
			options.cacheBytes = parsed
		case "--cache-max-artifact-bytes":
			parsed, err := parseWorkspaceSessionByteLimit(flag, value, maxWorkspaceSessionArtifactBytes)
			if err != nil {
				return err
			}
			options.maxArtifactBytes = parsed
		}
		return nil
	})
	if err != nil {
		return workspaceSessionOptions{}, err
	}

	if err := validateFlagRules(flagRule{Invalid: options.maxArtifactBytes > options.cacheBytes, Message: fmt.Sprintf("experimental command \"check-workspace-session\" requires --cache-max-artifact-bytes not to exceed --cache-max-bytes")}); err != nil {
		return workspaceSessionOptions{}, err
	}
	return options, nil
}

func parseWorkspaceSessionLimit(flag, value string, maximum int) (int, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || parsed > uint64(maximum) || value != strconv.FormatUint(parsed, 10) {
		return 0, fmt.Errorf("experimental command \"check-workspace-session\" requires %s as a canonical decimal from 1 through %d", flag, maximum)
	}
	return int(parsed), nil
}

func parseWorkspaceSessionByteLimit(flag, value string, maximum uint64) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || parsed > maximum || value != strconv.FormatUint(parsed, 10) {
		return 0, fmt.Errorf("experimental command \"check-workspace-session\" requires %s as a canonical decimal from 1 through %d", flag, maximum)
	}
	return parsed, nil
}

func readWorkspaceSessionLine(reader *bufio.Reader) ([]byte, bool, error) {
	line, err := reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return nil, false, fmt.Errorf("experimental workspace session request exceeds %d bytes", maxWorkspaceSessionRequestBytes)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, workspaceSessionInputError{cause: err}
	}
	eof := errors.Is(err, io.EOF)
	if eof && len(line) == 0 {
		return nil, true, nil
	}
	line = bytes.TrimSuffix(line, []byte{'\n'})
	line = bytes.TrimSuffix(line, []byte{'\r'})
	if len(line) > maxWorkspaceSessionRequestBytes {
		return nil, false, fmt.Errorf("experimental workspace session request exceeds %d bytes", maxWorkspaceSessionRequestBytes)
	}
	return bytes.Clone(line), false, nil
}

func decodeWorkspaceSessionRequest(line []byte, expectedSequence int) (workspaceSessionRequest, error) {
	if len(line) == 0 {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request line is empty")
	}
	if !utf8.Valid(line) {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request must be valid UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request must be one JSON object")
	}
	request := workspaceSessionRequest{}
	seen := make(map[string]bool, 3)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return workspaceSessionRequest{}, fmt.Errorf("decode experimental workspace session request key")
		}
		key, ok := keyToken.(string)
		if !ok {
			return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request has a non-string key")
		}
		if seen[key] {
			return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request repeats field %q", key)
		}
		seen[key] = true
		switch key {
		case "schemaVersion":
			err = decoder.Decode(&request.SchemaVersion)
		case "kind":
			err = decoder.Decode(&request.Kind)
		case "sequence":
			err = decoder.Decode(&request.Sequence)
		default:
			return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request contains unknown field %q", key)
		}
		if err != nil {
			return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request field %q has the wrong type", key)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request is incomplete")
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) || token != nil {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request has trailing JSON")
	}
	if request.SchemaVersion != workspaceSessionSchema || request.Kind != workspaceSessionRequestKind {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request requires schemaVersion %q and kind %q", workspaceSessionSchema, workspaceSessionRequestKind)
	}
	if request.Sequence != expectedSequence {
		return workspaceSessionRequest{}, fmt.Errorf("experimental workspace session request sequence is %d, want %d", request.Sequence, expectedSequence)
	}
	return request, nil
}

func executeWorkspaceSessionCheck(session *workspacecompiler.SourceSession, sequence int, sourceRoot string, locked bool, stderr io.Writer) (workspaceSessionResponse, int) {
	response := workspaceSessionResponse{
		SchemaVersion: workspaceSessionSchema,
		Kind:          workspaceSessionResponseKind,
		Sequence:      sequence,
		Status:        workspaceSessionStatusPackage,
		ExitCode:      ExitPackageFailure,
		Cache:         workspaceSessionCacheReport{Outcome: workspaceSessionCacheNotRun},
	}
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, locked)
	if err != nil {
		if code := packageFailure(stderr, err); code != ExitPackageFailure {
			return workspaceSessionResponse{}, code
		}
		response.Cache = workspaceSessionCacheResponse(workspacecompiler.Report{}, session)
		response.Cache.Outcome = workspaceSessionCacheNotRun
		return response, ExitSuccess
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		if code := packageFailure(stderr, err); code != ExitPackageFailure {
			return workspaceSessionResponse{}, code
		}
		response.LockState = lockState
		response.Instances = len(workspace.Graph.Nodes)
		response.Cache = workspaceSessionCacheResponse(workspacecompiler.Report{}, session)
		response.Cache.Outcome = workspaceSessionCacheNotRun
		return response, ExitSuccess
	}
	checked, err := session.Check(workspace)
	if err != nil {
		return workspaceSessionResponse{}, internalError(stderr)
	}
	response.LockState = lockState
	response.Instances = len(workspace.Graph.Nodes)
	response.SourceFiles = len(input.Files)
	response.MIRFunctions = len(checked.Compiler.MIR.Functions)
	response.Diagnostics = len(checked.Compiler.Semantic.Diagnostics)
	response.Unsupported = len(checked.Compiler.Semantic.Unsupported)
	response.Cache = workspaceSessionCacheResponse(checked.Cache, session)
	if response.Diagnostics != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Compiler.Semantic.Diagnostics); err != nil {
			return workspaceSessionResponse{}, ExitInternalFailure
		}
		response.Status = workspaceSessionStatusDiagnostics
		response.ExitCode = ExitSourceDiagnostics
		return response, ExitSuccess
	}
	if response.Unsupported != 0 {
		if err := writeUnsupportedSemantics(stderr, checked.Compiler.Semantic.Unsupported); err != nil {
			return workspaceSessionResponse{}, ExitInternalFailure
		}
		response.Status = workspaceSessionStatusDiagnostics
		response.ExitCode = ExitSourceDiagnostics
		return response, ExitSuccess
	}
	if !checked.Compiler.Verified {
		return workspaceSessionResponse{}, internalError(stderr)
	}
	response.Status = workspaceSessionStatusOK
	response.ExitCode = ExitSuccess
	return response, ExitSuccess
}

func workspaceSessionCacheResponse(report workspacecompiler.Report, session *workspacecompiler.SourceSession) workspaceSessionCacheReport {
	stats := session.Stats()
	return workspaceSessionCacheReport{
		Outcome: report.Outcome, Packages: report.Packages,
		Hits: report.Hits, Misses: report.Misses, Rejected: report.Rejected,
		Stored: report.Stored, StoreSkipped: report.StoreSkipped,
		Lowered: report.Lowered, ReassemblyFallback: report.ReassemblyFallback,
		RetainedEntries: stats.Store.Entries, RetainedBytes: stats.Store.Bytes, VerifiedHits: stats.VerifiedHits,
	}
}

func writeWorkspaceSessionResponse(writer io.Writer, response workspaceSessionResponse) error {
	return json.NewEncoder(writer).Encode(response)
}

func workspaceSessionResponseForSchema() workspaceSessionResponse {
	return workspaceSessionResponse{Cache: workspaceSessionCacheReport{}}
}
