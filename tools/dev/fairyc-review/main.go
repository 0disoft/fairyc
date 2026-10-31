// fairyc-review is opt-in, read-only developer review over the existing compiler facts.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/0disoft/fairyc/internal/interfacereview"
	"github.com/0disoft/fairyc/internal/sourcenotice"
)

const usage = `usage:
  fairyc-review diff --before WORKSPACE --after WORKSPACE [--format text|json] [--fail-on-change]
  fairyc-review notice [--root DIR] [--threshold-bytes 40000] [--format text|json]

diff compares the root package's verified ordinary public interface and declared effects.
notice reports source files strictly above the byte threshold without reading their contents.
Exit: 0 complete review; 1 changed with --fail-on-change; 2 invalid/incomplete review.
No compilation inputs, lockfiles or caches are written; reviewed programs are not built or run.
`

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancel()
	os.Exit(code)
}

func run(ctx context.Context, args []string, out, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		if _, err := fmt.Fprint(out, usage); err != nil {
			return 2
		}
		return 0
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "text", "text or json")
	switch args[0] {
	case "diff":
		before := flags.String("before", "", "existing before workspace")
		after := flags.String("after", "", "existing after workspace")
		failChanged := flags.Bool("fail-on-change", false, "exit 1 only for a complete comparison with changes")
		if code := parse(flags, args[1:]); code >= 0 {
			return code
		}
		if *before == "" || *after == "" || !validFormat(*format) {
			fmt.Fprintln(stderr, "diff requires --before, --after and --format text|json")
			return 2
		}
		report := interfacereview.Compare(interfacereview.Capture(ctx, *before), interfacereview.Capture(ctx, *after))
		var err error
		if *format == "json" {
			err = writeJSON(out, report)
		} else {
			err = writeDiffText(out, report)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if report.State != interfacereview.Bounded {
			return 2
		}
		if *failChanged && report.Changed != nil && *report.Changed {
			return 1
		}
		return 0
	case "notice":
		root := flags.String("root", ".", "directory to scan")
		threshold := flags.Int64("threshold-bytes", sourcenotice.DefaultThresholdBytes, "advisory byte threshold (strictly greater)")
		if code := parse(flags, args[1:]); code >= 0 {
			return code
		}
		if *threshold <= 0 || !validFormat(*format) {
			fmt.Fprintln(stderr, "notice requires a positive threshold and --format text|json")
			return 2
		}
		report := sourcenotice.Scan(ctx, *root, *threshold)
		var err error
		if *format == "json" {
			err = writeJSON(out, report)
		} else {
			err = writeNoticeText(out, report)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if report.State != "bounded" {
			return 2
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unknown review command:", args[0])
		return 2
	}
}

func parse(flags *flag.FlagSet, args []string) int {
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(flags.Output(), "unexpected positional arguments")
		return 2
	}
	return -1
}

func validFormat(format string) bool { return format == "text" || format == "json" }

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func writeDiffText(out io.Writer, report interfacereview.Report) error {
	var text strings.Builder
	fmt.Fprintf(&text, "interface review: %s\nroot package: %s\n", report.State, report.Before.Package.UID)
	if report.Changed == nil {
		text.WriteString("observed changes: unknown outside the available facts\n")
	} else {
		fmt.Fprintf(&text, "observed changes: %t\n", *report.Changed)
	}
	for _, change := range report.Changes {
		name := change.Name
		if change.Module != "" && change.Kind != "module" {
			name = change.Module + "::" + name
		}
		fmt.Fprintf(&text, "%s %s %s", change.Action, change.Kind, name)
		if len(change.Fields) != 0 {
			fmt.Fprintf(&text, " [%s]", strings.Join(change.Fields, ", "))
		}
		for _, effect := range change.EffectsAdded {
			fmt.Fprintf(&text, " +%s", effect)
		}
		for _, effect := range change.EffectsRemoved {
			fmt.Fprintf(&text, " -%s", effect)
		}
		text.WriteByte('\n')
	}
	for _, side := range []struct {
		name     string
		snapshot interfacereview.Snapshot
	}{{"before", report.Before}, {"after", report.After}} {
		for _, issue := range side.snapshot.Issues {
			fmt.Fprintf(&text, "%s unavailable %s: %s", side.name, issue.Module, issue.Reason)
			if issue.Detail != "" {
				fmt.Fprintf(&text, " (%s)", issue.Detail)
			}
			text.WriteByte('\n')
		}
	}
	for _, issue := range report.Issues {
		fmt.Fprintf(&text, "comparison unavailable: %s\n", issue.Reason)
	}
	text.WriteString("behavior compatibility: unknown; root ordinary v5 interface facts only\n")
	_, err := io.WriteString(out, text.String())
	return err
}

func writeNoticeText(out io.Writer, report sourcenotice.Report) error {
	var text strings.Builder
	fmt.Fprintf(&text, "large source notice: %s; threshold > %d bytes; %d source files\n", report.State, report.ThresholdBytes, report.SourceFiles)
	for _, file := range report.Files {
		fmt.Fprintf(&text, "%s: %d bytes\n", file.Path, file.Bytes)
	}
	for _, path := range report.SkippedSymlinks {
		fmt.Fprintf(&text, "skipped symlink: %s\n", path)
	}
	for _, issue := range report.Issues {
		fmt.Fprintf(&text, "unavailable %s: %s (%s)\n", issue.Path, issue.Reason, issue.Detail)
	}
	_, err := io.WriteString(out, text.String())
	return err
}
