package main

import (
	"fmt"
	"html/template"
	"os"
)

type reportRow struct {
	Language, Profile, Workload, Status, Reason                                               string
	Repetition                                                                                int
	Cold, NoOp, Private, Public, Final, Project, Shared, Clean, Binary, Growth, Amplification string
	Files                                                                                     int
	Verified                                                                                  int
}
type reportView struct {
	Receipt                    Receipt
	Locale                     string
	Locales                    []reportLocale
	Commit                     string
	Rows                       []reportRow
	Summary, Footprint, Growth template.HTML
	Passed, Failed             int
}

type reportLocale struct{ Code, Name, URL string }

func renderHTML(r Receipt, path, locale string, locales []string) error {
	t := translator(locale)
	v := reportView{Receipt: r, Locale: locale, Commit: shortCommit(r), Summary: template.HTML(summaryChart(r, locale).svg()), Footprint: template.HTML(footprintChart(r, locale).svg()), Growth: template.HTML(growthChart(r, locale).svg())}
	for _, code := range locales {
		v.Locales = append(v.Locales, reportLocale{code, localeNames[code], artifactName("report.html", code)})
	}
	for _, c := range r.Cases {
		row := reportRow{Language: c.Language, Profile: c.Profile, Workload: c.Workload, Repetition: c.Repetition, Status: c.Status, Reason: c.Reason, Cold: "-", NoOp: "-", Private: "-", Public: "-", Final: "-", Project: "-", Shared: "-", Clean: "-", Binary: "-", Growth: "-", Amplification: "-"}
		if c.Status == "passed" {
			v.Passed++
			cold := findSample(c, "cold", 0)
			last := finalSample(c)
			row.Cold = fmt.Sprintf("%.1f", cold.WallMS)
			row.NoOp = fmt.Sprintf("%.1f", findSample(c, "no-op", 0).WallMS)
			row.Final = bytesLabel(float64(last.Footprint.Total))
			row.Project = bytesLabel(float64(last.Footprint.Total - last.Footprint.SharedCache - last.Footprint.Temporary))
			row.Shared = bytesLabel(float64(last.Footprint.SharedCache))
			row.Binary = bytesLabel(float64(last.Footprint.Binary))
			row.Clean = bytesLabel(float64(findSample(c, "clean", 0).Footprint.Total))
			row.Growth = bytesLabel(float64(last.Footprint.Total - cold.Footprint.Total))
			row.Files = last.Footprint.Files
			row.Amplification = fmt.Sprintf("%.1fx", float64(last.Footprint.Total)/float64(max(int64(1), last.SourceBytes)))
			private, public := []float64{}, []float64{}
			for _, s := range c.Samples {
				if s.Verified {
					row.Verified++
				}
				if s.Stage == "edit" {
					if s.Change == "public-interface" {
						public = append(public, s.WallMS)
					} else {
						private = append(private, s.WallMS)
					}
				}
			}
			if len(private) > 0 {
				row.Private = fmt.Sprintf("%.1f", median(private))
			}
			if len(public) > 0 {
				row.Public = fmt.Sprintf("%.1f", median(public))
			}
		} else {
			v.Failed++
		}
		v.Rows = append(v.Rows, row)
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	tmpl, err := template.New("report").Funcs(template.FuncMap{"t": t, "artifact": func(name string) string { return artifactName(name, locale) }}).Parse(reportTemplate)
	if err != nil {
		file.Close()
		return err
	}
	err = tmpl.Execute(file, v)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

const reportTemplate = `<!doctype html>
<html lang="{{.Locale}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{t "Build footprint"}}</title>
<style>
:root{color-scheme:light;font-family:system-ui,sans-serif;color:#16283c;background:#f5f7f9}body{max-width:1180px;margin:0 auto;padding:24px}h1{font-size:32px;margin:0 0 6px}h2{font-size:18px;margin:0 0 14px}h3{font-size:15px;margin:22px 0 10px}p{margin:6px 0;line-height:1.5}a{color:#146b98}.meta{color:#53677c}.status{font-size:14px}.downloads{display:flex;flex-wrap:wrap;gap:8px 18px;margin:14px 0 20px}.panel,.failure,details{background:#fff;border:1px solid #dce4ec;border-radius:10px;padding:18px;margin:16px 0}.failure{background:#fff2f2;border-color:#edcaca}.failure ul{margin:8px 0 0;padding-left:22px}.failure li{margin:5px 0}.chart{overflow:auto}.chart svg{display:block;width:100%;min-width:760px;height:auto}.scroll{overflow:auto}table{border-collapse:collapse;width:100%;font-size:13px}th,td{text-align:left;padding:9px;border-bottom:1px solid #e4eaf0;white-space:nowrap}th{background:#eef3f7}td.reason{white-space:normal;min-width:170px}.filters{display:flex;gap:16px;flex-wrap:wrap;margin:14px 0}label{display:flex;gap:8px;align-items:center}select{padding:7px;border:1px solid #9caebe;border-radius:6px}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f1f5f9;padding:12px;border-radius:7px}summary{cursor:pointer;font-weight:600}details details{background:#f8fafc;margin:10px 0}ul.notes{padding-left:22px}ul.notes li{margin:7px 0;line-height:1.5}code{background:#edf2f6;padding:2px 4px}tr[hidden]{display:none}@media(max-width:640px){body{padding:12px}.panel,.failure,details{padding:14px}}
</style></head><body>
<header>{{if gt (len .Locales) 1}}<label style="float:right"><select id="locale" aria-label="{{t "Language"}}" onchange="location.href=this.value">{{range .Locales}}<option value="{{.URL}}" {{if eq .Code $.Locale}}selected{{end}}>{{.Name}}</option>{{end}}</select></label>{{end}}<h1>{{t "Build footprint"}}</h1><p class="meta">{{.Receipt.Started}} · {{.Receipt.Host.OS}}/{{.Receipt.Host.Arch}} · {{.Receipt.Options.Edits}} {{t "edits"}} · {{.Receipt.Options.Repetitions}} {{t "repetition(s)"}}</p><p class="status">{{.Passed}} {{t "passed"}} · {{.Failed}} {{t "failed or unavailable"}}</p>
<p class="downloads"><a href="{{artifact "summary.svg"}}">SVG</a><a href="{{artifact "summary.png"}}">PNG</a><a href="receipt.json">JSON</a></p></header>
{{if .Failed}}<section class="failure"><strong>{{.Failed}} {{t "failed or unavailable"}}</strong><ul>{{range .Rows}}{{if ne .Status "passed"}}<li><strong>{{.Language}} / {{.Profile}} / {{.Workload}}</strong> — {{t .Status}}: {{.Reason}}</li>{{end}}{{end}}</ul></section>{{end}}
<section class="panel chart">{{.Summary}}</section>
<details><summary>{{t "Full footprint chart"}}</summary><p class="downloads"><a href="{{artifact "footprint.svg"}}">SVG</a><a href="{{artifact "footprint.png"}}">PNG</a></p><div class="chart">{{.Footprint}}</div></details>
<details><summary>{{t "Growth chart"}}</summary><p class="downloads"><a href="{{artifact "growth.svg"}}">SVG</a><a href="{{artifact "growth.png"}}">PNG</a></p><div class="chart">{{.Growth}}</div></details>
<details><summary>{{t "Measurements and filters"}}</summary><p class="meta">{{t "Times in milliseconds. Totals are logical bytes."}}</p>
<div class="filters"><label>{{t "Language"}} <select id="language"><option value="">{{t "All"}}</option>{{range .Receipt.Options.Languages}}<option value="{{.}}">{{t .}}</option>{{end}}</select></label><label>{{t "Profile"}} <select id="profile"><option value="">{{t "All"}}</option>{{range .Receipt.Options.Profiles}}<option value="{{.}}">{{t .}}</option>{{end}}</select></label><label>{{t "Workload"}} <select id="workload"><option value="">{{t "All"}}</option>{{range .Receipt.Options.Workloads}}<option value="{{.}}">{{t .}}</option>{{end}}</select></label></div>
<div class="scroll"><table id="measurements"><thead><tr><th>{{t "Language"}}</th><th>{{t "Profile"}}</th><th>{{t "Workload"}}</th><th>{{t "Run"}}</th><th>{{t "Status"}}</th><th>{{t "Cold ms"}}</th><th>{{t "No-op ms"}}</th><th>{{t "Private edit ms"}}</th><th>{{t "Public edit ms"}}</th><th>{{t "End total"}}</th><th>{{t "Project outputs/cache"}}</th><th>{{t "Shared cache"}}</th><th>{{t "Growth"}}</th><th>{{t "Binary"}}</th><th>{{t "Total/source"}}</th><th>{{t "Files"}}</th><th>{{t "After clean"}}</th><th>{{t "Verified builds"}}</th><th>{{t "Reason"}}</th></tr></thead><tbody>
{{range .Rows}}<tr data-language="{{.Language}}" data-profile="{{.Profile}}" data-workload="{{.Workload}}"><td>{{.Language}}</td><td>{{t .Profile}}</td><td>{{t .Workload}}</td><td>{{.Repetition}}</td><td>{{t .Status}}</td><td>{{.Cold}}</td><td>{{.NoOp}}</td><td>{{.Private}}</td><td>{{.Public}}</td><td>{{.Final}}</td><td>{{.Project}}</td><td>{{.Shared}}</td><td>{{.Growth}}</td><td>{{.Binary}}</td><td>{{.Amplification}}</td><td>{{.Files}}</td><td>{{.Clean}}</td><td>{{.Verified}}</td><td class="reason">{{.Reason}}</td></tr>{{end}}</tbody></table></div></details>
<details><summary>{{t "Toolchains and installed inventory"}}</summary>{{range .Receipt.Toolchains}}<details><summary>{{.Language}} — {{t .Status}}</summary><pre>{{.Version}}
{{.Backend}}
{{t "Executable"}}: {{.Executable}} ({{.ExecutableBytes}} {{t "bytes; not total install size"}})
SHA-256: {{.ExecutableHash}}
{{.Reason}}</pre></details>{{end}}
<h3>{{t "Installed toolchain files"}}</h3><p class="meta">{{t "Separate from build footprint. Shared installations appear once."}}</p><div class="scroll"><table><thead><tr><th>{{t "Installation"}}</th><th>{{t "Used by"}}</th><th>{{t "Logical bytes"}}</th><th>{{t "Files"}}</th><th>{{t "Status"}}</th><th>{{t "Scope"}}</th></tr></thead><tbody>{{range .Receipt.Installations}}<tr><td>{{.Name}}</td><td>{{.UsedBy}}</td><td>{{.Bytes}}</td><td>{{.Files}}</td><td>{{t .Status}} {{.Reason}}</td><td class="reason">{{.Scope}}</td></tr>{{end}}</tbody></table></div></details>
<details><summary>{{t "Measurement notes"}}</summary><ul class="notes"><li>{{t "Retained logical bytes include project outputs, caches, and the shared experiment cache. Toolchain installs and transient peak disk use are excluded."}}</li><li>{{t "The fixtures are small and synthetic, with no third-party dependencies."}}</li><li>{{t "Development defaults differ by toolchain. Rust small-dev uses line-tables-only debug data and disables incremental builds."}}</li><li>{{t "Footprint bars are arithmetic means across repetitions; growth points are medians at each edit."}}</li></ul></details>
<script>for(const el of document.querySelectorAll('.filters select'))el.addEventListener('change',()=>{for(const row of document.querySelectorAll('#measurements tbody tr'))row.hidden=['language','profile','workload'].some(k=>document.getElementById(k).value&&row.dataset[k]!==document.getElementById(k).value)});</script>
</body></html>`
