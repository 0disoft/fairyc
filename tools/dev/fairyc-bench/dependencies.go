package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// This opt-in Rust CLI exercises derive macros, JSON parsing, and regex search.
// It is intentionally separate from the equivalent cross-language workloads.
func dependencyInput() []byte {
	var input bytes.Buffer
	for i := 0; i < 4096; i++ {
		message := "info: ready 한국어"
		if i%3 == 0 {
			message = "error: retry"
		}
		line, _ := json.Marshal(struct {
			Message string `json:"message"`
		}{message})
		input.Write(line)
		input.WriteByte('\n')
	}
	return input.Bytes()
}

func dependencyReference(input []byte, revision int) string {
	lines, matches, checksum := 0, 0, 0
	for _, b := range input {
		checksum = (checksum*31 + int(b)) % 1000003
	}
	for _, line := range bytes.Split(input, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var record struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(line, &record) != nil {
			return "" // An invalid fixture cannot produce a verified sample.
		}
		lines++
		if bytes.Contains([]byte(record.Message), []byte("error")) {
			matches++
		}
	}
	return fmt.Sprintf("{\"lines\":%d,\"matches\":%d,\"checksum\":%d}\n", lines, matches, (checksum+revision+1)%1000003)
}

func dependencySources(revision int) map[string]string {
	files := sources("rust", 1, revision)
	files["Cargo.toml"] += `
[dependencies]
clap = { version = "4", features = ["derive"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
regex = "1"
`
	argument := ""
	if revision/5%2 == 1 {
		argument = ", 0"
	}
	files["src/main.rs"] = `mod m000;
use clap::Parser;
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::{fs::File, io::{BufRead, BufReader}, path::PathBuf};

#[derive(Parser)]
struct Args {
    file: PathBuf,
    #[arg(long, default_value = "error")]
    pattern: String,
}
#[derive(Deserialize)]
struct Record { message: String }
#[derive(Serialize)]
struct Summary { lines: usize, matches: usize, checksum: i32 }

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();
    let regex = Regex::new(&args.pattern)?;
    let mut reader = BufReader::new(File::open(args.file)?);
    let mut line = Vec::new();
    let mut result = Summary { lines: 0, matches: 0, checksum: 0 };
    while reader.read_until(b'\n', &mut line)? != 0 {
        for &b in &line {
            result.checksum = (result.checksum * 31 + b as i32) % 1000003;
        }
        let record: Record = serde_json::from_slice(&line)?;
        result.lines += 1;
        if regex.is_match(&record.message) { result.matches += 1; }
        line.clear();
    }
    result.checksum = m000::mix(result.checksum` + argument + `);
    println!("{}", serde_json::to_string(&result)?);
    Ok(())
}
`
	return files
}

func resolveDependencyLock(ctx context.Context, o Options, a adapter) (string, []Command, error) {
	root := filepath.Join(o.Output, "preparation")
	project, state := filepath.Join(root, "project"), filepath.Join(root, "state")
	if err := writeSources(project, dependencySources(0)); err != nil {
		return "", nil, err
	}
	if err := makeState(state); err != nil {
		return "", nil, err
	}
	_, env, err := a.build(project, state, "dev")
	if err != nil {
		return "", nil, err
	}
	env = append(env, "CARGO_NET_OFFLINE=false")
	command, out, err := execute(ctx, o.Timeout, project, env, nil, a.executable, "generate-lockfile", "--color", "never")
	command.Log = out + command.Log
	commands := []Command{portable(command, o.Output)}
	if err != nil {
		return "", commands, fmt.Errorf("resolve Cargo.lock: %w (see preparation log)", err)
	}
	lock, err := os.ReadFile(filepath.Join(project, "Cargo.lock"))
	return string(lock), commands, err
}

func prepareDependencies(ctx context.Context, o Options, a adapter, c *Case, project, state string) error {
	if o.dependencyLock == "" {
		return fmt.Errorf("dependency workload requires a resolved Cargo.lock")
	}
	c.DependencyLock = o.dependencyLock
	files := dependencySources(0)
	files["Cargo.lock"] = o.dependencyLock
	if err := writeSources(project, files); err != nil {
		return err
	}
	_, env, err := a.build(project, state, c.Profile)
	if err != nil {
		return err
	}
	env = append(env, "CARGO_NET_OFFLINE=false")
	command, out, err := execute(ctx, o.Timeout, project, env, nil, a.executable, "fetch", "--locked", "--color", "never")
	command.Log = out + command.Log
	c.Preparation = append(c.Preparation, portable(command, o.Output))
	if err != nil {
		return fmt.Errorf("fetch dependencies: %w (see preparation log)", err)
	}
	return nil
}
