package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func renderCLI(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	var input, output, locale string
	fs.StringVar(&input, "input", "", "receipt JSON")
	fs.StringVar(&output, "output", "", "report directory")
	fs.StringVar(&locale, "locale", "all", "report language: all, en, ko, zh, es, fr, hi")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if input == "" || output == "" || fs.NArg() != 0 {
		return errors.New("render requires --input FILE --output DIR")
	}
	codes := supportedLocales
	if locale != "all" {
		if !slices.Contains(supportedLocales, locale) {
			return fmt.Errorf("unsupported report locale %q (use all, en, ko, zh, es, fr, hi)", locale)
		}
		codes = []string{locale}
	}
	file, err := os.Open(input)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<20))
	decoder.DisallowUnknownFields()
	var receipt Receipt
	if err := decoder.Decode(&receipt); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("receipt contains trailing data or exceeds 64 MiB")
	}
	return renderLocales(receipt, output, codes)
}

func validateReceipt(r Receipt) error {
	if r.Schema != receiptSchema {
		return fmt.Errorf("unsupported schema %q", r.Schema)
	}
	if len(r.Cases) == 0 {
		return errors.New("receipt has no cases")
	}
	if err := validateWorkloadSelection(r.Options.Languages, r.Options.Workloads); err != nil {
		return err
	}
	if r.Finished == "" {
		return errors.New("receipt is unfinished")
	}
	if r.Options.Edits < 0 || r.Options.Edits > 1000 || r.Options.Modules < 2 || r.Options.Modules > 64 || r.Options.Repetitions < 1 || r.Options.Repetitions > 100 {
		return errors.New("invalid run settings")
	}
	for _, setting := range []struct{ values, allowed []string }{{r.Options.Languages, []string{"fairy", "zig", "rust", "c"}}, {r.Options.Profiles, []string{"dev", "release", "small-dev"}}, {r.Options.Workloads, availableWorkloads}} {
		if _, err := selection(strings.Join(setting.values, ","), setting.allowed...); err != nil {
			return err
		}
	}
	expected := 0
	for _, language := range r.Options.Languages {
		for _, profile := range r.Options.Profiles {
			if profile != "small-dev" || language == "rust" {
				expected += len(r.Options.Workloads) * r.Options.Repetitions
			}
		}
	}
	if len(r.Cases) != expected {
		return errors.New("receipt does not cover its declared matrix")
	}
	for _, i := range r.Installations {
		if i.Bytes < 0 || i.Files < 0 || i.Symlinks < 0 {
			return errors.New("invalid installation inventory")
		}
	}
	seen := map[string]bool{}
	combinations := map[string]bool{}
	for _, c := range r.Cases {
		if c.Workload == "rust-dependencies" && c.Status == "passed" && (c.DependencyLock == "" || len(c.Preparation) == 0) {
			return fmt.Errorf("missing dependency preparation for %s", c.ID)
		}
		if !slices.Contains(r.Options.Languages, c.Language) || !slices.Contains(r.Options.Profiles, c.Profile) || !slices.Contains(r.Options.Workloads, c.Workload) || (c.Profile == "small-dev" && c.Language != "rust") || c.Repetition < 1 || c.Repetition > r.Options.Repetitions {
			return errors.New("case outside declared matrix")
		}
		key := fmt.Sprintf("%s/%s/%s/%d", c.Language, c.Profile, c.Workload, c.Repetition)
		if combinations[key] {
			return errors.New("duplicate matrix configuration")
		}
		combinations[key] = true
		if c.ID == "" || seen[c.ID] {
			return errors.New("missing or duplicate case ID")
		}
		seen[c.ID] = true
		if c.Status != "passed" && c.Status != "failed" && c.Status != "unavailable" {
			return fmt.Errorf("invalid case status %q", c.Status)
		}
		if c.Status == "passed" && len(c.Samples) != r.Options.Edits+3 {
			return fmt.Errorf("incomplete passed case %s", c.ID)
		}
		for _, s := range c.Samples {
			f := s.Footprint
			var sum int64
			for _, n := range []int64{f.Binary, f.Debug, f.Incremental, f.Intermediate, f.LocalCache, f.SharedCache, f.Temporary} {
				if n < 0 || n > 1<<60 {
					return errors.New("invalid footprint byte count")
				}
				sum += n
			}
			if f.Total != sum || f.Files < 0 || s.WallMS < 0 {
				return fmt.Errorf("invalid inventory for %s", c.ID)
			}
			if c.Status == "passed" && s.Stage != "clean" && (!s.Verified || s.Actual != s.Expected || f.Binary <= 0) {
				return fmt.Errorf("unverified passed sample for %s", c.ID)
			}
		}
		if c.Status == "passed" {
			for i, s := range c.Samples {
				stage, edit := "edit", i-1
				switch i {
				case 0:
					stage = "cold"
					edit = 0
				case 1:
					stage = "no-op"
					edit = 0
				case len(c.Samples) - 1:
					stage = "clean"
					edit = r.Options.Edits
				}
				if s.Stage != stage || s.Edit != edit {
					return fmt.Errorf("invalid sample sequence for %s", c.ID)
				}
			}
		}
	}
	return nil
}

func render(r Receipt, output string) error {
	return renderLocales(r, output, supportedLocales)
}

func renderLocales(r Receipt, output string, locales []string) error {
	if err := validateReceipt(r); err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	for _, locale := range locales {
		for name, chart := range map[string]scene{"footprint": footprintChart(r, locale), "growth": growthChart(r, locale), "summary": summaryChart(r, locale)} {
			if err := os.WriteFile(filepath.Join(output, artifactName(name+".svg", locale)), []byte(chart.svg()), 0o644); err != nil {
				return err
			}
			if err := chart.png(filepath.Join(output, artifactName(name+".png", locale))); err != nil {
				return err
			}
		}
		if err := renderHTML(r, filepath.Join(output, artifactName("report.html", locale)), locale, locales); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(output, "receipt.json"), r)
}

type series struct {
	Language, Profile, Workload string
	Cases                       []Case
}

func seriesFor(r Receipt, workload string) []series {
	var result []series
	indexes := map[string]int{}
	for _, c := range r.Cases {
		if c.Status != "passed" || c.Workload != workload {
			continue
		}
		key := c.Language + "/" + c.Profile
		index, ok := indexes[key]
		if !ok {
			index = len(result)
			indexes[key] = index
			result = append(result, series{Language: c.Language, Profile: c.Profile, Workload: c.Workload})
		}
		result[index].Cases = append(result[index].Cases, c)
	}
	return result
}
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2
	}
	return values[middle]
}
func findSample(c Case, stage string, edit int) Sample {
	for _, s := range c.Samples {
		if s.Stage == stage && (stage != "edit" || s.Edit == edit) {
			return s
		}
	}
	return Sample{}
}
func finalSample(c Case) Sample {
	if len(c.Samples) < 2 {
		return Sample{}
	}
	return c.Samples[len(c.Samples)-2]
}
func bytesLabel(n float64) string {
	if n >= 1<<30 {
		return fmt.Sprintf("%.2f GiB", n/(1<<30))
	}
	if n >= 1<<20 {
		return fmt.Sprintf("%.2f MiB", n/(1<<20))
	}
	if n >= 1<<10 {
		return fmt.Sprintf("%.1f KiB", n/(1<<10))
	}
	return fmt.Sprintf("%.0f B", n)
}
func shortCommit(r Receipt) string {
	c := r.Commit
	if len(c) > 10 {
		c = c[:10]
	}
	if r.Dirty {
		c += " +dirty"
	}
	return c
}
