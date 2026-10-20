package main

import (
	"encoding/xml"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureReceipt() Receipt {
	base := Sample{Stage: "cold", Change: "none", Verified: true, Expected: "ok\n", Actual: "ok\n", WallMS: 1, Footprint: Footprint{Binary: 10, Intermediate: 5, Total: 15, Files: 2}}
	noop := base
	noop.Stage = "no-op"
	return Receipt{Schema: receiptSchema, Started: "2026-10-20T00:00:00Z", Finished: "2026-10-20T00:01:00Z", Commit: "test-fixture", Options: RunSettings{Languages: []string{"fairy"}, Profiles: []string{"dev"}, Workloads: []string{"tokenizer"}, Modules: 16, Repetitions: 1}, Cases: []Case{{ID: "test", Language: "fairy", Profile: "dev", Workload: "tokenizer", Repetition: 1, Status: "passed", Samples: []Sample{base, noop, {Stage: "clean", Change: "none", Footprint: Footprint{}}}}}}
}

func TestReportArtifactsAndEscaping(t *testing.T) {
	r := fixtureReceipt()
	r.Toolchains = []Toolchain{{Language: "fairy", Version: "<script>alert('x')</script>"}}
	r.Options.Languages = append(r.Options.Languages, "zig")
	r.Cases = append(r.Cases, Case{ID: "failure", Language: "zig", Profile: "dev", Workload: "tokenizer", Repetition: 1, Status: "unavailable", Reason: "<script>alert('x')</script>"})
	root := t.TempDir()
	if err := render(r, root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"footprint", "growth", "summary"} {
		b, err := os.ReadFile(filepath.Join(root, name+".svg"))
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(strings.NewReader(string(b)))
		for {
			_, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if strings.Contains(string(b), "<script>") {
			t.Fatal("SVG text injection")
		}
		f, err := os.Open(filepath.Join(root, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 2240 || img.Bounds().Dy() < 300 {
			t.Fatal("incorrect PNG dimensions")
		}
	}
	b, err := os.ReadFile(filepath.Join(root, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "<script>alert") || !strings.Contains(string(b), "&lt;script&gt;") {
		t.Fatal("HTML injection or missing failure reason")
	}
	if !strings.Contains(string(b), "data-language=") || !strings.Contains(string(b), "addEventListener") {
		t.Fatal("report filters missing")
	}
	if strings.Count(string(b), `id="chart-`) != 6 {
		t.Fatal("inline SVG titles must have distinct IDs")
	}
	for _, code := range supportedLocales {
		html, err := os.ReadFile(filepath.Join(root, artifactName("report.html", code)))
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range []string{`lang="` + code + `"`, translator(code)("Build footprint"), `href="` + artifactName("summary.png", code) + `"`, `id="locale"`} {
			if !strings.Contains(string(html), text) {
				t.Errorf("%s: missing localized report content: %s", code, text)
			}
		}
		if strings.Contains(string(html), "<details") || strings.Count(string(html), `role="tab"`) != 6 || strings.Count(string(html), `role="tabpanel"`) != 6 {
			t.Errorf("%s: report must use six menu panels without disclosures", code)
		}
		for _, name := range []string{"summary.svg", "summary.png", "footprint.svg", "footprint.png", "growth.svg", "growth.png"} {
			if info, err := os.Stat(filepath.Join(root, artifactName(name, code))); err != nil || info.Size() == 0 {
				t.Fatalf("%s: missing artifact %s", code, name)
			}
		}
	}
}

func TestRejectMisleadingReceipts(t *testing.T) {
	r := fixtureReceipt()
	r.Cases[0].Samples[0].Footprint.Total = 999
	if err := validateReceipt(r); err == nil {
		t.Fatal("accepted bucket total mismatch")
	}
	r = fixtureReceipt()
	r.Cases[0].Samples[0].Actual = "wrong\n"
	if err := validateReceipt(r); err == nil {
		t.Fatal("accepted unverified output")
	}
	r = fixtureReceipt()
	r.Cases[0].Samples[1].Stage = "cold"
	if err := validateReceipt(r); err == nil {
		t.Fatal("accepted mislabeled no-op")
	}
	r = fixtureReceipt()
	r.Cases[0].Samples = r.Cases[0].Samples[:2]
	if err := validateReceipt(r); err == nil {
		t.Fatal("accepted missing clean")
	}
}

func TestSingleLocaleRender(t *testing.T) {
	root := t.TempDir()
	receipt := filepath.Join(root, "receipt.json")
	if err := writeJSON(receipt, fixtureReceipt()); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "report")
	if err := renderCLI([]string{"--input", receipt, "--output", output, "--locale", "fr"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(output, "report.fr.html"))
	if err != nil || !strings.Contains(string(b), `<html lang="fr">`) || strings.Contains(string(b), `id="locale"`) {
		t.Fatal("single-locale report must not link to missing language variants")
	}
	if _, err := os.Stat(filepath.Join(output, "summary.png")); !os.IsNotExist(err) {
		t.Fatal("single-locale render unexpectedly generated English")
	}
	invalid := filepath.Join(root, "invalid")
	if err := renderCLI([]string{"--input", receipt, "--output", invalid, "--locale", "de"}); err == nil {
		t.Fatal("unsupported locale was accepted")
	}
	if _, err := os.Stat(invalid); !os.IsNotExist(err) {
		t.Fatal("unsupported locale created output")
	}
}
