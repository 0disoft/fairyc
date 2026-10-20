package main

import "testing"

func TestSummaryMeansAndWidths(t *testing.T) {
	r := fixtureReceipt()
	r.Options.Repetitions = 2
	first := r.Cases[0]
	first.Samples = append([]Sample(nil), first.Samples...)
	first.Samples[1].Footprint = Footprint{Binary: 10, SharedCache: 3, Temporary: 2, Total: 15}
	second := first
	second.Repetition = 2
	second.Samples = append([]Sample(nil), first.Samples...)
	second.Samples[1].Footprint = Footprint{Binary: 30, SharedCache: 5, Total: 35}
	r.Cases = []Case{first, second, {Language: "zig", Profile: "dev", Workload: "tokenizer", Status: "unavailable"}}
	workload, rows := reportSummary(r, "en")
	if workload != "Tokenizer" || len(rows) != 1 {
		t.Fatal("summary must use the selected workload and passed configurations")
	}
	row := rows[0]
	if row.Project != "20 B" || row.Shared != "4 B" || row.Total != "25 B" {
		t.Fatalf("summary must average the same retained bytes as charts: %+v", row)
	}
	if row.ProjectWidth != 80 || row.SharedWidth != 16 || row.TempWidth != 4 {
		t.Fatalf("summary must keep true linear proportions including retained temp: %+v", row)
	}
}
