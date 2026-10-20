package main

type summaryRow struct {
	Label, Project, Shared, Total        string
	ProjectWidth, SharedWidth, TempWidth float64
}

func reportSummary(r Receipt, locale string) (string, []summaryRow) {
	t := translator(locale)
	workload, groups, _ := summaryGroups(r)
	maximum := 1.0
	for _, g := range groups {
		total := 0.0
		for _, c := range g.Cases {
			total += float64(finalSample(c).Footprint.Total)
		}
		maximum = max(maximum, total/float64(len(g.Cases)))
	}
	var rows []summaryRow
	for _, g := range groups {
		project, shared, temporary := 0.0, 0.0, 0.0
		for _, c := range g.Cases {
			f := finalSample(c).Footprint
			project += float64(f.Total - f.SharedCache - f.Temporary)
			shared += float64(f.SharedCache)
			temporary += float64(f.Temporary)
		}
		n := float64(len(g.Cases))
		project, shared, temporary = project/n, shared/n, temporary/n
		label := map[string]string{"fairy": "Fairy C", "zig": "Zig", "rust": "Rust", "c": "C"}[g.Language]
		if g.Language == "rust" || g.Profile != "dev" {
			label += " / " + t(g.Profile)
		}
		rows = append(rows, summaryRow{Label: label, Project: bytesLabel(project), Shared: bytesLabel(shared), Total: bytesLabel(project + shared + temporary), ProjectWidth: project / maximum * 100, SharedWidth: shared / maximum * 100, TempWidth: temporary / maximum * 100})
	}
	return t(workload), rows
}
