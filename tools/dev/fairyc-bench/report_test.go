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
	return Receipt{Schema: receiptSchema, Started: "2026-10-20T00:00:00Z", Commit: "test-fixture", Options: RunSettings{Languages: []string{"fairy"}, Profiles: []string{"dev"}, Workloads: []string{"tokenizer"}, Repetitions: 1}, Cases: []Case{{ID: "test", Language: "fairy", Profile: "dev", Workload: "tokenizer", Status: "passed", Samples: []Sample{base, noop, {Stage: "clean", Change: "none", Footprint: Footprint{}}}}}}
}

func TestReportArtifactsAndEscaping(t *testing.T) {
	r := fixtureReceipt()
	r.Toolchains = []Toolchain{{Language: "fairy", Version: "<script>alert('x')</script>"}}
	r.Cases = append(r.Cases, Case{ID: "failure", Status: "unavailable", Reason: "<script>alert('x')</script>"})
	root := t.TempDir()
	if err := render(r, root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"footprint", "growth"} {
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
	for letter, glyph := range glyphs {
		if len(glyph) != 35 {
			t.Fatalf("bad bitmap for %c: %d", letter, len(glyph))
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
