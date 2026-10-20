// fairyc-bench compares retained native build files in disposable, isolated roots.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := entry(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fairyc-bench:", err)
		os.Exit(1)
	}
}

func entry(ctx context.Context, args []string) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		fmt.Fprintln(os.Stdout, "usage: fairyc-bench run --output NEW_DIR [--languages fairy,zig,rust,c] [--workloads tokenizer,many-modules] [--edits 20] [--repetitions 1]\n       fairyc-bench render --input receipt.json --output REPORT_DIR [--locale all|en|ko|zh|es|fr|hi]\nUse run --help or render --help for all options.")
		return nil
	}
	if len(args) == 0 {
		return errors.New("usage: fairyc-bench run [options] | render --input receipt.json --output DIR")
	}
	if args[0] == "render" {
		return renderCLI(args[1:])
	}
	if args[0] != "run" {
		return fmt.Errorf("unknown command %q", args[0])
	}
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var o Options
	var languages, profiles, workloads string
	fs.StringVar(&o.Output, "output", "", "new output directory (required; never overwrites)")
	fs.StringVar(&languages, "languages", "fairy,zig,rust,c", "comma-separated languages")
	fs.StringVar(&profiles, "profiles", "dev,release,small-dev", "small-dev applies only to Rust")
	fs.StringVar(&workloads, "workloads", "tokenizer,many-modules", "comma-separated workloads")
	fs.IntVar(&o.Edits, "edits", 20, "number of successive source edits")
	fs.IntVar(&o.Modules, "modules", 16, "module count for many-modules")
	fs.IntVar(&o.Repetitions, "repetitions", 1, "independent cold-to-edit sequences")
	fs.DurationVar(&o.Timeout, "timeout", 10*time.Minute, "timeout per command (empty Zig build-system caches can take several minutes)")
	fs.StringVar(&o.Fairy, "fairyc", "", "prebuilt compiler path (otherwise builds current checkout before measuring)")
	fs.StringVar(&o.Zig, "zig", "zig", "Zig executable")
	fs.StringVar(&o.CC, "cc", "", "C compiler (MSVC on Windows; cc elsewhere)")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	var err error
	if o.Languages, err = selection(languages, "fairy", "zig", "rust", "c"); err != nil {
		return err
	}
	if o.Profiles, err = selection(profiles, "dev", "release", "small-dev"); err != nil {
		return err
	}
	if o.Workloads, err = selection(workloads, "tokenizer", "many-modules"); err != nil {
		return err
	}
	if o.Output == "" || o.Edits < 0 || o.Edits > 1000 || o.Modules < 2 || o.Modules > 64 || o.Repetitions < 1 || o.Repetitions > 100 || o.Timeout <= 0 {
		return errors.New("require --output, edits 0..1000, modules 2..64, repetitions 1..100, positive timeout")
	}
	applicable := false
	for _, language := range o.Languages {
		for _, profile := range o.Profiles {
			applicable = applicable || profile != "small-dev" || language == "rust"
		}
	}
	if !applicable {
		return errors.New("selected profiles do not apply to any selected language")
	}
	return run(ctx, o)
}

func selection(s string, allowed ...string) ([]string, error) {
	var result []string
	seen := map[string]bool{}
	for _, value := range strings.Split(s, ",") {
		value = strings.TrimSpace(value)
		valid := false
		for _, candidate := range allowed {
			valid = valid || candidate == value
		}
		if !valid || seen[value] {
			return nil, fmt.Errorf("invalid or duplicate selection %q (choose %s)", value, strings.Join(allowed, ","))
		}
		seen[value] = true
		result = append(result, value)
	}
	return result, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
