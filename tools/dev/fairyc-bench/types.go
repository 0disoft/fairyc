package main

import "time"

const receiptSchema = "fairyc.build-footprint/v1"

type Options struct {
	Output      string
	Languages   []string
	Profiles    []string
	Workloads   []string
	Edits       int
	Modules     int
	Repetitions int
	Timeout     time.Duration
	Fairy       string
	Zig         string
	CC          string
}

type Receipt struct {
	Schema     string      `json:"schema"`
	Started    string      `json:"started"`
	Finished   string      `json:"finished"`
	Commit     string      `json:"commit"`
	Dirty      bool        `json:"dirty"`
	Host       Host        `json:"host"`
	Method     string      `json:"method"`
	Options    RunSettings `json:"settings"`
	Toolchains []Toolchain `json:"toolchains"`
	Cases      []Case      `json:"cases"`
}

type Host struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPU       string `json:"cpu"`
	CPUs      int    `json:"logicalCpus"`
	GoVersion string `json:"runnerGoVersion"`
}

type RunSettings struct {
	Languages   []string `json:"languages"`
	Profiles    []string `json:"profiles"`
	Workloads   []string `json:"workloads"`
	Edits       int      `json:"edits"`
	Modules     int      `json:"modules"`
	Repetitions int      `json:"repetitions"`
	Timeout     string   `json:"commandTimeout"`
}

type Toolchain struct {
	Language        string `json:"language"`
	Version         string `json:"version"`
	Executable      string `json:"executable"`
	ExecutableHash  string `json:"executableSha256"`
	ExecutableBytes int64  `json:"executableBytes"`
	Backend         string `json:"backend,omitempty"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
}

type Case struct {
	ID         string   `json:"id"`
	Language   string   `json:"language"`
	Profile    string   `json:"profile"`
	Workload   string   `json:"workload"`
	Repetition int      `json:"repetition"`
	Status     string   `json:"status"`
	Reason     string   `json:"reason,omitempty"`
	InputHash  string   `json:"inputSha256"`
	Samples    []Sample `json:"samples"`
}

type Sample struct {
	Stage          string    `json:"stage"`
	Edit           int       `json:"edit"`
	Change         string    `json:"change"`
	SourceHash     string    `json:"sourceSha256"`
	SourceBytes    int64     `json:"sourceBytes"`
	SourceFiles    int       `json:"sourceFiles"`
	Action         string    `json:"action,omitempty"`
	Commands       []Command `json:"commands"`
	WallMS         float64   `json:"buildWallMs"`
	Expected       string    `json:"expectedStdout"`
	Actual         string    `json:"actualStdout"`
	Verified       bool      `json:"verified"`
	Footprint      Footprint `json:"footprint"`
	ReclaimedBytes int64     `json:"reclaimedBytes,omitempty"`
}

type Command struct {
	Argv     []string `json:"argv"`
	Env      []string `json:"environmentOverrides,omitempty"`
	Exit     int      `json:"exitCode"`
	WallMS   float64  `json:"wallMs"`
	Log      string   `json:"log,omitempty"`
	TimedOut bool     `json:"timedOut,omitempty"`
}

// These buckets are disjoint. Debug data embedded in an object/binary stays in
// its file's bucket; this is a directory/file inventory, not a section parser.
type Footprint struct {
	Binary       int64 `json:"binaryBytes"`
	Debug        int64 `json:"separateDebugBytes"`
	Incremental  int64 `json:"incrementalBytes"`
	Intermediate int64 `json:"intermediateBytes"`
	LocalCache   int64 `json:"localCacheBytes"`
	SharedCache  int64 `json:"sharedCacheBytes"`
	Temporary    int64 `json:"retainedTempBytes"`
	Files        int   `json:"files"`
	Total        int64 `json:"totalBytes"`
}
