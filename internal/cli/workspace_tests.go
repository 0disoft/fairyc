package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const stableTestListSchema = "fairyc.test-list/v1"
const stableTestReportSchema = "fairyc.test-report/v1"
const stableWorkspaceTestListSchema = "fairyc.test-list/v2"
const stableWorkspaceTestReportSchema = "fairyc.test-report/v2"
const stableLocatedTestListSchema = "fairyc.test-list/v3"
const stableLocatedTestReportSchema = "fairyc.test-report/v3"
const stableCapturedTestReportSchema = "fairyc.test-report/v4"
const stableNativeTestTimeoutSeconds = 60

type stableTestOptions struct {
	locked          bool
	native          bool
	release         bool
	stdin           bool
	stdinFile       string
	timeoutSeconds  int
	timeoutExplicit bool
	captureOutput   bool
	list            bool
	json            bool
	locations       bool
	workspace       bool
	packageUID      string
	files           []string
	directories     []string
	exact           []string
	programArgs     []string
}

type stableTestListReport struct {
	SchemaVersion string                `json:"schemaVersion"`
	Tests         []string              `json:"tests"`
	Locations     *[]stableTestLocation `json:"locations,omitempty"`
	Count         int                   `json:"count"`
	Lock          string                `json:"lock"`
}

type stableTestSource struct {
	PackageUID      string `json:"packageUid"`
	Path            string `json:"path"`
	Line            int    `json:"line"`
	ByteColumn      int    `json:"byteColumn"`
	StartByte       int    `json:"startByte"`
	EndByte         int    `json:"endByte"`
	FileIdentity    string `json:"fileIdentity"`
	LineMapIdentity string `json:"lineMapIdentity"`
}

type stableTestLocation struct {
	Identity string           `json:"identity"`
	Source   stableTestSource `json:"source"`
}

type stableTestExecutionOutcome struct {
	Identity     string            `json:"identity"`
	Outcome      string            `json:"outcome"`
	TrapCategory *string           `json:"trapCategory"`
	Source       *stableTestSource `json:"source,omitempty"`
	StdoutBase64 *string           `json:"stdoutBase64,omitempty"`
	StderrBase64 *string           `json:"stderrBase64,omitempty"`
}

type stableTestExecutionReport struct {
	SchemaVersion string                       `json:"schemaVersion"`
	Engine        string                       `json:"engine"`
	Target        *string                      `json:"target"`
	Profile       *string                      `json:"profile"`
	Tests         []stableTestExecutionOutcome `json:"tests"`
	Count         int                          `json:"count"`
	Passed        int                          `json:"passed"`
	Failed        int                          `json:"failed"`
	Status        string                       `json:"status"`
	Lock          string                       `json:"lock"`
}

type workspaceTest struct {
	display string
	entry   string
	file    string
	source  stableTestSource
}

type workspaceTestOutcome struct {
	display      string
	trapCategory string
	source       stableTestSource
	stdout       string
	stderr       string
}

type testSourceDiagnostics struct {
	path        string
	contents    []byte
	diagnostics []diagnostic.Diagnostic
}

func runStableTest(arguments []string, sourceRoot string, stdin io.Reader, stdout, stderr io.Writer) int {
	options, err := parseStableTestOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
			return code
		}
		return packageFailure(stderr, err)
	}
	return runLoadedStableTest(workspace, lockState, options, sourceRoot, stdin, stdout, stderr)
}

func runLoadedStableTest(workspace workspaceloader.Workspace, lockState string, options stableTestOptions, sourceRoot string, stdin io.Reader, stdout, stderr io.Writer) int {
	packagePaths, err := selectWorkspaceTestPackagePaths(workspace, options)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	if options.workspace {
		workspace, err = workspaceloader.WithWorkspaceTests(workspace)
	} else {
		workspace, err = workspaceloader.WithPackageTests(workspace, packagePaths[0])
	}
	if err != nil {
		if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
			return code
		}
		return packageFailure(stderr, err)
	}
	var tests []workspaceTest
	var testFiles []string
	var testDirectories []string
	for _, packagePath := range packagePaths {
		packageUID := ""
		if options.workspace {
			loaded, found := workspace.Package(packagePath)
			if !found {
				return internalError(stderr)
			}
			packageUID = loaded.Manifest.Package.UID
		}
		packageTests, packageTestFiles, prepareCode := prepareWorkspaceTests(workspace, packagePath, packageUID, options.locations, stderr)
		if prepareCode != ExitSuccess {
			return prepareCode
		}
		tests = append(tests, packageTests...)
		testFiles = append(testFiles, packageTestFiles...)
		testDirectories = append(testDirectories, workspaceTestDirectories(
			packageTestFiles,
			workspaceTestSourcePath(packagePath, "tests"),
		)...)
	}
	slices.SortFunc(tests, func(left, right workspaceTest) int { return strings.Compare(left.display, right.display) })
	slices.Sort(testFiles)
	slices.Sort(testDirectories)
	code := ExitSuccess
	tests, err = selectWorkspaceTestDirectories(tests, testDirectories, options.directories)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	tests, err = selectWorkspaceTestFiles(tests, testFiles, options.files)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	tests, err = selectWorkspaceTests(tests, options.exact)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	checked, pipelineErr := workspacecompiler.Check(workspace)
	if len(checked.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Semantic.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(checked.Semantic.Unsupported) != 0 {
		if err := writeStableUnsupportedSemantics(stderr, checked.Semantic.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if pipelineErr != nil {
		var failure *compiler.Error
		if errors.As(pipelineErr, &failure) && failure.Stage == compiler.StageMIRLower {
			return workspaceSubsetCapabilityError(
				workspaceCompilePolicy{stableDiagnostics: true}, stderr, "semantic test", failure.Cause,
			)
		}
		return pipelineFailure(stderr, pipelineErr)
	}
	if !checked.Verified {
		return internalError(stderr)
	}
	sdkStorage, err := workspaceSDKStorage(workspace)
	if err != nil {
		return workspaceSubsetCapabilityError(workspaceCompilePolicy{stableDiagnostics: true}, stderr, "native test SDK", err)
	}
	if options.stdin && len(tests) != 1 {
		return invocationError(stderr, `command "test" requires exactly one selected test when --stdin is supplied`)
	}
	if options.list {
		if options.json {
			return writeStableTestListJSON(tests, lockState, options, stdout, stderr)
		}
		return writeStableTestList(tests, lockState, options, stdout, stderr)
	}

	var outcomes []workspaceTestOutcome
	if sdkStorage && !options.native {
		return invocationError(stderr, "workspace tests using std::vec require --native on Windows or Linux AMD64")
	}
	if options.native {
		profile := cdriver.LowProfile
		if options.release {
			profile = cdriver.OptimizedProfile
		}
		nativeInput := nativeWorkspaceTestInput{}
		if options.stdin {
			nativeInput.caller = stdin
		}
		if options.stdinFile != "" {
			contents, err := readStableNativeTestInput(sourceRoot, options.stdinFile)
			if err != nil {
				return packageFailure(stderr, err)
			}
			nativeInput.replay = contents
			nativeInput.replayEach = true
		}
		outcomes, code = runNativeWorkspaceTests(checked.MIR, tests, profile, nativeInput, options.programArgs, time.Duration(options.timeoutSeconds)*time.Second, options.captureOutput, sdkStorage, stderr)
	} else {
		outcomes, code = runInterpreterWorkspaceTests(checked.MIR, tests, stderr)
	}
	if code != ExitSuccess {
		return code
	}
	if options.json {
		return writeStableTestReportJSON(outcomes, lockState, options, stdout, stderr)
	}
	return writeStableTestReport(outcomes, lockState, options, stdout, stderr)
}

func runInterpreterWorkspaceTests(program mir.Program, tests []workspaceTest, stderr io.Writer) ([]workspaceTestOutcome, int) {
	outcomes := make([]workspaceTestOutcome, 0, len(tests))
	for _, test := range tests {
		entry, found := programEntry(program, test.entry)
		if !found || len(entry.Parameters) != 0 || entry.ResultType != hir.TypeVoid {
			return nil, internalError(stderr)
		}
		outcome, err := interpreter.ExecuteVoid(program, test.entry)
		if err != nil {
			var capability interpreter.CapabilityError
			if errors.As(err, &capability) {
				return nil, workspaceSubsetCapabilityError(
					workspaceCompilePolicy{stableDiagnostics: true}, stderr, "semantic test interpreter",
					fmt.Errorf("%s: %w", test.display, err),
				)
			}
			return nil, internalError(stderr)
		}
		switch outcome.Kind {
		case interpreter.OutcomeReturn:
			outcomes = append(outcomes, workspaceTestOutcome{display: test.display, source: test.source})
		case interpreter.OutcomeTrap:
			outcomes = append(outcomes, workspaceTestOutcome{display: test.display, trapCategory: outcome.TrapCategory, source: test.source})
		default:
			return nil, internalError(stderr)
		}
	}
	return outcomes, ExitSuccess
}

func writeStableTestReport(outcomes []workspaceTestOutcome, lockState string, options stableTestOptions, stdout, stderr io.Writer) int {
	var report strings.Builder
	passed := 0
	failed := 0
	for _, outcome := range outcomes {
		location := ""
		if options.locations {
			location = fmt.Sprintf(" at %s:%d:%d", safeDisplayPath(outcome.source.Path), outcome.source.Line, outcome.source.ByteColumn)
		}
		if outcome.trapCategory == "" {
			passed++
			fmt.Fprintf(&report, "test %s%s ... ok\n", outcome.display, location)
			continue
		}
		failed++
		fmt.Fprintf(&report, "test %s%s ... FAILED (%s)\n", outcome.display, location, outcome.trapCategory)
	}
	status := "ok"
	if failed != 0 {
		status = "FAILED"
	}
	fmt.Fprintf(&report, "test result: %s. %d passed; %d failed; lock %s\n", status, passed, failed, lockState)
	if _, err := io.WriteString(stdout, report.String()); err != nil {
		return outputError(stderr)
	}
	if failed != 0 {
		return ExitProgramTrap
	}
	return ExitSuccess
}

func writeStableTestReportJSON(
	outcomes []workspaceTestOutcome,
	lockState string,
	options stableTestOptions,
	stdout, stderr io.Writer,
) int {
	tests := make([]stableTestExecutionOutcome, 0, len(outcomes))
	passed := 0
	failed := 0
	for _, outcome := range outcomes {
		result := stableTestExecutionOutcome{Identity: outcome.display, Outcome: "passed"}
		if options.locations {
			source := outcome.source
			result.Source = &source
		}
		if outcome.trapCategory == "" {
			passed++
		} else {
			failed++
			result.Outcome = "failed"
			category := outcome.trapCategory
			result.TrapCategory = &category
		}
		if options.captureOutput {
			stdout := base64.StdEncoding.EncodeToString([]byte(outcome.stdout))
			stderr := base64.StdEncoding.EncodeToString([]byte(outcome.stderr))
			result.StdoutBase64 = &stdout
			result.StderrBase64 = &stderr
		}
		tests = append(tests, result)
	}
	status := "ok"
	if failed != 0 {
		status = "failed"
	}
	report := stableTestExecutionReport{
		SchemaVersion: stableTestReportSchemaFor(options),
		Engine:        "interpreter",
		Tests:         tests,
		Count:         len(tests),
		Passed:        passed,
		Failed:        failed,
		Status:        status,
		Lock:          lockState,
	}
	if options.native {
		target := string(nativeWorkspaceTarget())
		profile := "development"
		if options.release {
			profile = "release"
		}
		report.Engine = "native"
		report.Target = &target
		report.Profile = &profile
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return internalError(stderr)
	}
	encoded = append(encoded, '\n')
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	if failed != 0 {
		return ExitProgramTrap
	}
	return ExitSuccess
}

func writeStableTestList(tests []workspaceTest, lockState string, options stableTestOptions, stdout, stderr io.Writer) int {
	var report strings.Builder
	for _, test := range tests {
		if options.locations {
			fmt.Fprintf(&report, "test %s at %s:%d:%d\n", test.display, safeDisplayPath(test.source.Path), test.source.Line, test.source.ByteColumn)
		} else {
			fmt.Fprintf(&report, "test %s\n", test.display)
		}
	}
	fmt.Fprintf(&report, "test list: %d discovered; lock %s\n", len(tests), lockState)
	if _, err := io.WriteString(stdout, report.String()); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func writeStableTestListJSON(tests []workspaceTest, lockState string, options stableTestOptions, stdout, stderr io.Writer) int {
	identities := make([]string, 0, len(tests))
	locations := make([]stableTestLocation, 0, len(tests))
	for _, test := range tests {
		identities = append(identities, test.display)
		if options.locations {
			locations = append(locations, stableTestLocation{Identity: test.display, Source: test.source})
		}
	}
	var locationReport *[]stableTestLocation
	if options.locations {
		locationReport = &locations
	}
	report, err := json.Marshal(stableTestListReport{
		SchemaVersion: stableTestListSchemaFor(options),
		Tests:         identities,
		Locations:     locationReport,
		Count:         len(identities),
		Lock:          lockState,
	})
	if err != nil {
		return internalError(stderr)
	}
	report = append(report, '\n')
	if _, err := stdout.Write(report); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func stableTestListSchemaFor(options stableTestOptions) string {
	if options.locations {
		return stableLocatedTestListSchema
	}
	if options.workspace {
		return stableWorkspaceTestListSchema
	}
	return stableTestListSchema
}

func stableTestReportSchemaFor(options stableTestOptions) string {
	if options.captureOutput {
		return stableCapturedTestReportSchema
	}
	if options.locations {
		return stableLocatedTestReportSchema
	}
	if options.workspace {
		return stableWorkspaceTestReportSchema
	}
	return stableTestReportSchema
}

func parseStableTestOptions(arguments []string) (stableTestOptions, error) {
	compilerArguments, programArguments, hasDelimiter, err := splitStableProgramArguments("test", arguments)
	if err != nil {
		return stableTestOptions{}, err
	}
	options, err := parseStableTestCompilerOptions(compilerArguments)
	if err != nil {
		return stableTestOptions{}, err
	}
	if hasDelimiter && !options.native {
		return stableTestOptions{}, fmt.Errorf(`command "test" requires --native before --`)
	}
	options.programArgs = programArguments
	return options, nil
}

func parseStableTestCompilerOptions(arguments []string) (stableTestOptions, error) {
	options := stableTestOptions{timeoutSeconds: stableNativeTestTimeoutSeconds}
	_, err := parseCommandFlags(arguments, flagSchema{Command: "test", Flags: []flagDefinition{
		{Name: "--locked", Kind: flagSwitch},
		{Name: "--native", Kind: flagSwitch},
		{Name: "--release", Kind: flagSwitch},
		{Name: "--stdin", Kind: flagSwitch},
		{Name: "--capture-output", Kind: flagSwitch},
		{Name: "--list", Kind: flagSwitch},
		{Name: "--json", Kind: flagSwitch},
		{Name: "--locations", Kind: flagSwitch},
		{Name: "--workspace", Kind: flagSwitch},
		{Name: "--package", Kind: flagScalar, ValueFirst: true, Missing: `command "test" requires a package UID after --package`},
		{Name: "--stdin-file", Kind: flagScalar, ValueFirst: true, Missing: `command "test" requires a portable workspace file after --stdin-file`},
		{Name: "--timeout-seconds", Kind: flagScalar, ValueFirst: true, Missing: `command "test" requires canonical seconds after --timeout-seconds`},
		{Name: "--exact", Kind: flagSlice, ValueFirst: true, Missing: `command "test" requires a test identity after --exact`, DuplicateValue: `command "test" received exact test %q more than once`},
		{Name: "--file", Kind: flagSlice, ValueFirst: true, Missing: `command "test" requires a test path after --file`, DuplicateValue: `command "test" received test file %q more than once`},
		{Name: "--dir", Kind: flagSlice, ValueFirst: true, Missing: `command "test" requires a test directory after --dir`, DuplicateValue: `command "test" received test directory %q more than once`},
	}, Unknown: func(string) error {
		return fmt.Errorf(`command "test" accepts only optional --list, --json, --locations, --native, --release, --stdin, --stdin-file <portable-workspace-file>, --capture-output, --workspace, --package <package-uid>, --dir <portable-test-directory>, --file <portable-test-path>, --exact <test-identity>, and --locked`)
	}}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--native":
			options.native = true
		case "--release":
			options.release = true
		case "--stdin":
			options.stdin = true
		case "--capture-output":
			options.captureOutput = true
		case "--list":
			options.list = true
		case "--json":
			options.json = true
		case "--locations":
			options.locations = true
		case "--workspace":
			options.workspace = true
		case "--package":
			options.packageUID = value
		case "--stdin-file":
			options.stdinFile = value
		case "--exact":
			options.exact = append(options.exact, value)
		case "--file":
			options.files = append(options.files, value)
		case "--dir":
			options.directories = append(options.directories, value)
		case "--timeout-seconds":
			seconds, err := strconv.Atoi(value)
			if err != nil || strconv.Itoa(seconds) != value || seconds < 1 || seconds > stableNativeTestTimeoutSeconds {
				return fmt.Errorf(`command "test" requires --timeout-seconds in canonical range 1 through %d`, stableNativeTestTimeoutSeconds)
			}
			options.timeoutSeconds, options.timeoutExplicit = seconds, true
		}
		return nil
	})
	if err != nil {
		return stableTestOptions{}, err
	}
	if err := validateFlagRules(
		flagRule{Invalid: options.workspace && options.packageUID != "", Message: `command "test" does not combine --workspace with --package`},
		flagRule{Invalid: len(options.files) != 0 && len(options.exact) != 0, Message: `command "test" does not combine --file with --exact`},
		flagRule{Invalid: len(options.directories) != 0 && (len(options.files) != 0 || len(options.exact) != 0), Message: `command "test" does not combine --dir with --file or --exact`},
		flagRule{Invalid: options.list && (options.native || options.release || options.stdin || options.stdinFile != "" || options.timeoutExplicit || options.captureOutput || len(options.exact) != 0), Message: `command "test" does not combine --list with --native, --release, --stdin, --stdin-file, --capture-output, or --exact`},
		flagRule{Invalid: options.release && !options.native, Message: `command "test" requires --native when --release is supplied`},
		flagRule{Invalid: options.stdin && !options.native, Message: `command "test" requires --native when --stdin is supplied`},
		flagRule{Invalid: options.timeoutExplicit && !options.native, Message: `command "test" requires --native when --timeout-seconds is supplied`},
		flagRule{Invalid: options.stdinFile != "" && !options.native, Message: `command "test" requires --native when --stdin-file is supplied`},
		flagRule{Invalid: options.stdin && options.stdinFile != "", Message: `command "test" does not combine --stdin with --stdin-file`},
		flagRule{Invalid: options.captureOutput && !options.native, Message: `command "test" requires --native when --capture-output is supplied`},
		flagRule{Invalid: options.captureOutput && !options.json, Message: `command "test" requires --json when --capture-output is supplied`},
	); err != nil {
		return stableTestOptions{}, err
	}
	return options, nil
}

func selectWorkspaceTestPackagePaths(workspace workspaceloader.Workspace, options stableTestOptions) ([]string, error) {
	if options.workspace {
		packagePaths, err := workspace.RootReachableLocalPackagePaths()
		if err != nil {
			return nil, fmt.Errorf(`command "test" cannot select workspace packages: %w`, err)
		}
		if len(packagePaths) == 0 {
			return nil, fmt.Errorf(`command "test" workspace has no root-reachable local packages`)
		}
		return packagePaths, nil
	}
	loaded, err := selectWorkspaceTestPackage(workspace, options.packageUID)
	if err != nil {
		return nil, err
	}
	return []string{loaded.Path}, nil
}

func selectWorkspaceTestPackage(workspace workspaceloader.Workspace, packageUID string) (workspaceloader.Package, error) {
	packagePath := "."
	if packageUID != "" {
		packagePath = ""
		for _, loaded := range workspace.Packages {
			if loaded.SourceKind == workspaceloader.SourceKindLocal && loaded.Manifest.Package.UID == packageUID {
				packagePath = loaded.Path
				break
			}
		}
		if packagePath == "" {
			return workspaceloader.Package{}, fmt.Errorf(`command "test" package %q was not found in the local workspace`, packageUID)
		}
	}
	loaded, found := workspace.Package(packagePath)
	if !found || loaded.SourceKind != workspaceloader.SourceKindLocal {
		return workspaceloader.Package{}, fmt.Errorf(`command "test" selected local package is missing`)
	}
	if _, err := workspace.PackageID(packagePath); err != nil {
		return workspaceloader.Package{}, fmt.Errorf(`command "test" cannot select package %q: %w`, loaded.Manifest.Package.UID, err)
	}
	return loaded, nil
}

func selectWorkspaceTests(tests []workspaceTest, exact []string) ([]workspaceTest, error) {
	if len(exact) == 0 {
		return tests, nil
	}
	wanted := make(map[string]struct{}, len(exact))
	for _, identity := range exact {
		wanted[identity] = struct{}{}
	}
	selected := make([]workspaceTest, 0, len(exact))
	for _, test := range tests {
		if _, found := wanted[test.display]; !found {
			continue
		}
		selected = append(selected, test)
		delete(wanted, test.display)
	}
	if len(wanted) != 0 {
		missing := make([]string, 0, len(wanted))
		for identity := range wanted {
			missing = append(missing, identity)
		}
		slices.Sort(missing)
		for index := range missing {
			missing[index] = fmt.Sprintf("%q", missing[index])
		}
		return nil, fmt.Errorf(`command "test" exact test selection was not found: %s`, strings.Join(missing, ", "))
	}
	return selected, nil
}

func selectWorkspaceTestFiles(tests []workspaceTest, available, requested []string) ([]workspaceTest, error) {
	if len(requested) == 0 {
		return tests, nil
	}
	known := make(map[string]struct{}, len(available))
	for _, path := range available {
		known[path] = struct{}{}
	}
	wanted := make(map[string]struct{}, len(requested))
	var missing []string
	for _, path := range requested {
		if _, found := known[path]; !found {
			missing = append(missing, path)
			continue
		}
		wanted[path] = struct{}{}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		for index := range missing {
			missing[index] = fmt.Sprintf("%q", missing[index])
		}
		return nil, fmt.Errorf(`command "test" test file selection was not found: %s`, strings.Join(missing, ", "))
	}
	selected := make([]workspaceTest, 0, len(tests))
	for _, test := range tests {
		if _, found := wanted[test.file]; found {
			selected = append(selected, test)
		}
	}
	return selected, nil
}

func workspaceTestDirectories(files []string, root string) []string {
	known := make(map[string]struct{})
	for _, file := range files {
		for directory := path.Dir(file); directory != "."; directory = path.Dir(directory) {
			known[directory] = struct{}{}
			if directory == root {
				break
			}
		}
	}
	directories := make([]string, 0, len(known))
	for directory := range known {
		directories = append(directories, directory)
	}
	slices.Sort(directories)
	return directories
}

func selectWorkspaceTestDirectories(tests []workspaceTest, available, requested []string) ([]workspaceTest, error) {
	if len(requested) == 0 {
		return tests, nil
	}
	known := make(map[string]struct{}, len(available))
	for _, directory := range available {
		known[directory] = struct{}{}
	}
	wanted := make(map[string]struct{}, len(requested))
	var missing []string
	for _, directory := range requested {
		if _, found := known[directory]; !found {
			missing = append(missing, directory)
			continue
		}
		wanted[directory] = struct{}{}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		for index := range missing {
			missing[index] = fmt.Sprintf("%q", missing[index])
		}
		return nil, fmt.Errorf(`command "test" test directory selection was not found: %s`, strings.Join(missing, ", "))
	}
	selected := make([]workspaceTest, 0, len(tests))
	for _, test := range tests {
		for directory := range wanted {
			if strings.HasPrefix(test.file, directory+"/") {
				selected = append(selected, test)
				break
			}
		}
	}
	return selected, nil
}

func workspaceTestSourcePath(packagePath, filePath string) string {
	if packagePath == "." {
		return filePath
	}
	return packagePath + "/" + filePath
}

func prepareWorkspaceTests(workspace workspaceloader.Workspace, packagePath, packageUID string, includeLocations bool, stderr io.Writer) ([]workspaceTest, []string, int) {
	packageID, err := workspace.PackageID(packagePath)
	if err != nil {
		return nil, nil, packageFailure(stderr, err)
	}
	testPackage, found := workspace.Package(packagePath)
	if !found {
		return nil, nil, packageFailure(stderr, fmt.Errorf("workspace test package %q is missing", packagePath))
	}
	var tests []workspaceTest
	var testFiles []string
	var sourceDiagnostics []testSourceDiagnostics
	for _, file := range testPackage.Sources {
		if !file.Test {
			continue
		}
		portablePath := workspaceTestSourcePath(packagePath, file.Path)
		testFiles = append(testFiles, portablePath)
		var diagnostics []diagnostic.Diagnostic
		var fileIdentity packagecontent.FileDigest
		var lineMapIdentity packagecontent.LineMapID
		locationPath := ""
		if includeLocations {
			fileIdentity, lineMapIdentity = packagecontent.ComputeFileIdentities(file.Bytes)
			locationPath = portablePath
		}
		inspection := semantic.InspectTestDeclarations(file.Module)
		diagnostics = append(diagnostics, inspection.Diagnostics...)
		for _, index := range inspection.Functions {
			function := file.Module.Functions[index]
			display := file.Module.Path + "::" + function.Name
			if packageUID != "" {
				display = packageUID + "::" + display
			}
			var source stableTestSource
			if includeLocations {
				line, byteColumn := bytePosition(file.Bytes, function.NameSpan.StartByte)
				source = stableTestSource{
					PackageUID: testPackage.Manifest.Package.UID, Path: locationPath,
					Line: line, ByteColumn: byteColumn,
					StartByte: function.NameSpan.StartByte, EndByte: function.NameSpan.EndByte,
					FileIdentity: fileIdentity.String(), LineMapIdentity: lineMapIdentity.String(),
				}
			}
			tests = append(tests, workspaceTest{
				display: display,
				entry:   hir.QualifiedModulePath(packageID, file.Module.Path) + "::" + function.Name,
				file:    portablePath,
				source:  source,
			})
		}
		if len(diagnostics) != 0 {
			slices.SortFunc(diagnostics, diagnostic.Compare)
			displayPath := file.Path
			if packageUID != "" && packagePath != "." {
				displayPath = packagePath + "/" + file.Path
			}
			sourceDiagnostics = append(sourceDiagnostics, testSourceDiagnostics{
				path: displayPath, contents: slices.Clone(file.Bytes), diagnostics: diagnostics,
			})
		}
	}
	if len(sourceDiagnostics) != 0 {
		for _, source := range sourceDiagnostics {
			if err := writeDiagnostics(stderr, source.path, source.contents, source.diagnostics); err != nil {
				return nil, nil, ExitInternalFailure
			}
		}
		return nil, nil, ExitSourceDiagnostics
	}
	slices.SortFunc(tests, func(left, right workspaceTest) int { return strings.Compare(left.display, right.display) })
	slices.Sort(testFiles)
	return tests, testFiles, ExitSuccess
}
