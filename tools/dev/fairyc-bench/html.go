package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"
)

type reportRow struct {
	Language, Profile, Workload, Status, Reason                                               string
	Repetition                                                                                int
	Cold, NoOp, Private, Public, Final, Project, Shared, Clean, Binary, Growth, Amplification string
	Files                                                                                     int
	Verified                                                                                  int
}
type reportView struct {
	Logo              template.URL
	Icons             map[string]template.HTML
	SummaryWorkload   string
	Date              string
	SummaryRows       []summaryRow
	HasTemp           bool
	ShowModules       bool
	Receipt           Receipt
	Locale            string
	Locales           []reportLocale
	Commit            string
	Rows              []reportRow
	Footprint, Growth template.HTML
	Passed, Failed    int
}

type reportLocale struct{ Code, Name, URL string }

func renderHTML(r Receipt, path, locale string, locales []string) error {
	t := translator(locale)
	v := reportView{Receipt: r, Locale: locale, Date: strings.Split(r.Started, "T")[0], Commit: shortCommit(r), Footprint: template.HTML(footprintChart(r, locale).svg()), Growth: template.HTML(growthChart(r, locale).svg())}
	v.Logo, v.Icons = reportBrand()
	v.SummaryWorkload, v.SummaryRows = reportSummary(r, locale)
	workload, _, _ := summaryGroups(r)
	v.ShowModules = workload == "many-modules"
	for _, row := range v.SummaryRows {
		v.HasTemp = v.HasTemp || row.TempWidth > 0
	}
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
:root{color-scheme:light;font-family:Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;color:#172b3b;background:#f5f7f9;font-synthesis:none}
*{box-sizing:border-box}html,body{width:100%;height:100%;margin:0}html{overflow:hidden}body{height:100vh;height:100dvh;overflow:hidden;background:#f5f7f9;color:#172b3b}
a{color:#116d77;text-decoration:none}a:hover{text-decoration:underline}button,select{font:inherit}
.sidebar{position:fixed;inset:0 auto 0 0;width:260px;height:100vh;height:100dvh;overflow-y:auto;background:#f6f8fa;border-right:1px solid #e4e9ed;padding:20px 16px}
.brand{display:flex;align-items:center;min-height:56px;padding:2px 10px 28px;text-decoration:none}.brand-logo{display:block;width:100%;max-width:158px;height:52px;object-fit:contain;object-position:left center}
.tabs{display:flex;flex-direction:column;gap:4px;margin:0;border:0}.nav-item{display:flex;align-items:center;gap:12px;width:100%;min-height:44px;padding:10px 14px;border:0;border-radius:7px;background:transparent;color:#20384a;font-weight:650;text-align:left;line-height:1.25;cursor:pointer}
.nav-item:hover{background:#edf2f4}.nav-item[aria-selected="true"]{background:#dcf4f0;color:#075e68}.nav-icon{display:flex;align-items:center;justify-content:center;width:22px;height:22px;flex:none}.nav-icon svg{display:block;width:20px;height:20px}.nav-label{min-width:0;overflow-wrap:normal}
.nav-item:focus-visible,.export-button:focus-visible,select:focus-visible{outline:2px solid #0c7d85;outline-offset:2px}
.main{height:100vh;height:100dvh;min-width:0;margin-left:260px;overflow-y:auto;background:#fff}.topbar{position:sticky;top:0;z-index:5;display:flex;justify-content:flex-end;padding:16px 36px;background:rgba(255,255,255,.98);border-bottom:1px solid #f0f3f5}.toolbar{display:flex;align-items:center;justify-content:flex-end;flex-wrap:wrap;gap:14px;max-width:100%}
.locale-picker{display:block;min-width:120px}.locale-picker select{width:100%;height:42px;padding:0 12px;border:1px solid #d8e0e6;border-radius:6px;background:#fff;color:#172b3b;font-size:14px;cursor:pointer}
.export-wrap{position:relative}.export-button{display:flex;align-items:center;justify-content:center;gap:9px;height:42px;min-width:116px;padding:0 14px;border:1px solid #d8e0e6;border-radius:6px;background:#fff;color:#172b3b;font-size:14px;font-weight:600;cursor:pointer}.export-button:hover{background:#f8fafb}.export-icon{display:flex;align-items:center;justify-content:center;width:18px;height:18px}.export-icon svg{display:block;width:18px;height:18px}
.export-menu{position:absolute;z-index:20;top:calc(100% + 7px);right:0;display:grid;gap:2px;width:184px;padding:7px;border:1px solid #dce4e9;border-radius:8px;background:#fff;box-shadow:0 12px 28px rgba(23,43,59,.14)}.export-menu a{display:flex;align-items:center;min-height:35px;padding:7px 10px;border-radius:5px;color:#172b3b;font-size:13px;font-weight:600;text-decoration:none}.export-menu a:hover{background:#edf7f6;text-decoration:none}.export-scope{padding:4px 10px 7px;color:#667988;font-size:11px;font-weight:650;letter-spacing:.04em;text-transform:uppercase}.menu-rule{height:1px;margin:4px 3px;background:#edf1f3}
.content{width:100%;max-width:1220px;margin:0 auto;padding:14px 36px 52px}h1{margin:0;color:#172b3b;font-size:clamp(30px,3vw,39px);line-height:1.15;letter-spacing:-.025em}.meta{display:flex;flex-wrap:wrap;gap:4px 0;margin:13px 0 0;color:#647888;font-size:14px;line-height:1.45}.meta span{min-width:0;overflow-wrap:anywhere}.meta span+span::before{content:"·";padding:0 9px;color:#a2afb9}.run-status{margin:8px 0 0;color:#667988;font-size:13px}.run-status.has-failure{color:#a23838}
.failure{margin:20px 0 0;padding:15px 17px;border:1px solid #efcfcf;border-radius:8px;background:#fff5f5;color:#7d2929}.failure strong{font-size:14px}.failure ul{margin:8px 0 0;padding-left:22px}.failure li{margin:5px 0;line-height:1.45;font-size:13px}
.panel{margin-top:24px;border:1px solid #e0e6eb;border-radius:8px;background:#fff;padding:20px}.summary-panel{padding:0;overflow:hidden}.summary-panel>.scroll{margin:0}.summary-table{min-width:600px}.summary-table th{padding:14px 16px;border-right:1px solid #e6ebef;background:#f4f7f9;color:#172b3b;font-size:14px;font-weight:700}.summary-table th:last-child,.summary-table td:last-child{border-right:0}.summary-table td{padding:14px 16px;border-top:1px solid #e6ebef;border-right:1px solid #e6ebef;font-size:14px;color:#2b4354}.summary-table td:not(:first-child){text-align:right;font-variant-numeric:tabular-nums}.summary-table td:last-child{color:#087780;font-weight:750}
.summary-bars{padding:23px 20px 10px}.compare-title{margin:0 0 16px;font-size:17px;line-height:1.25;color:#172b3b}.bar-row{display:grid;grid-template-columns:minmax(140px,220px) minmax(100px,1fr) 112px;align-items:center;gap:24px;min-height:34px;font-size:14px}.bar-label{min-width:0;overflow-wrap:anywhere}.bar-total{text-align:right;color:#20384a;font-variant-numeric:tabular-nums}.bar-track{display:flex;width:100%;height:14px;overflow:hidden;border-radius:2px;background:#e9edf1}.bar-track span{display:block;height:100%;flex:none}.bar-project{background:#089b99}.bar-shared{background:#3476d1}.bar-temp{background:#b79a72}
.summary-legend{display:flex;flex-wrap:wrap;gap:12px 26px;padding:15px 20px 0;color:#526778;font-size:12px}.legend-item{display:flex;align-items:center;gap:8px}.legend-swatch{display:block;width:15px;height:15px;border-radius:2px}.legend-project{background:#089b99}.legend-shared{background:#3476d1}.footnote{margin:14px 20px 19px;color:#83919d;font-size:11px;line-height:1.45}
.chart{padding:0;border:0;overflow:auto}.chart svg{display:block;width:100%;min-width:720px;height:auto}table{border-collapse:collapse;width:100%;font-size:13px}th,td{text-align:left;padding:9px;border-bottom:1px solid #e4eaf0;white-space:nowrap}th{background:#f4f7f9;color:#20384a}td.reason{white-space:normal;min-width:170px}.scroll{width:100%;max-width:100%;overflow-x:auto}
.filters{display:flex;gap:16px;flex-wrap:wrap;margin:14px 0 18px}.filters label,.filters label>*{display:flex;gap:8px;align-items:center}.filters select{height:36px;padding:0 9px;border:1px solid #b8c5ce;border-radius:6px;background:#fff;color:#172b3b}
#measurements{border-collapse:separate;border-spacing:0;isolation:isolate}
#measurements :is(th,td):nth-child(-n+3){position:sticky;z-index:1;background:#fff}
#measurements th:nth-child(-n+3){z-index:2;background:#f4f7f9}
#measurements :is(th,td):nth-child(1){left:0}
#measurements :is(th,td):nth-child(2){left:var(--pin-profile,0px)}
#measurements :is(th,td):nth-child(3){left:var(--pin-workload,0px);box-shadow:2px 0 0 #d8e0e6}
.panel h2{margin:0 0 14px;font-size:19px}.panel h3{margin:19px 0 9px;font-size:15px}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f4f7f9;padding:12px;border-radius:7px;font-size:12px}ul.notes{padding-left:22px}ul.notes li{margin:7px 0;line-height:1.5}code{background:#edf2f6;padding:2px 4px}
.visually-hidden{position:absolute!important;width:1px!important;height:1px!important;padding:0!important;margin:-1px!important;overflow:hidden!important;clip:rect(0 0 0 0)!important;white-space:nowrap!important;border:0!important}[hidden]{display:none!important}
@media(min-width:1000px){
.sidebar{width:296px;padding:24px 14px}.brand{padding:2px 24px 32px}.brand-logo{max-width:190px;height:72px}.tabs{gap:10px}.nav-item{min-height:64px;padding:16px 24px;gap:20px;font-size:20px;font-weight:500}.nav-item[aria-selected="true"]{font-weight:700}.nav-icon,.nav-icon svg{width:26px;height:26px}
.main{margin-left:296px}.topbar{padding:27px 28px 27px 50px;border-bottom:0}.toolbar{gap:24px}.locale-picker{min-width:140px}.locale-picker select,.export-button{height:57px;font-size:18px}.export-button{min-width:160px;gap:16px}.export-icon,.export-icon svg{width:23px;height:23px}
.content{max-width:1450px;padding:10px 28px 48px 50px}h1{font-size:42px}.meta{font-size:20px}.meta span+span::before{padding:0 12px}.run-status:not(.has-failure){display:none}.summary-panel{margin-top:32px;border:0;border-radius:0;overflow:visible}.summary-panel>.scroll{border:1px solid #e0e6eb;border-radius:4px}.summary-table th{padding:20px 28px;font-size:20px}.summary-table td{padding:16px 28px;font-size:21px}.summary-table th:first-child{width:33%}.summary-table td:not(:first-child),.summary-table th:not(:first-child){text-align:center}
.summary-bars{padding:42px 0 12px}.summary-bars .compare-title{font-size:25px;margin-bottom:20px}.bar-row{grid-template-columns:160px minmax(100px,1fr) 148px;min-height:49px;gap:26px;font-size:20px}.bar-track{height:28px;border-radius:0}.summary-legend{padding:20px 0 0;gap:24px 48px;font-size:16px}.legend-swatch{width:20px;height:20px}.footnote{margin:20px 0 0;font-size:14px}
}
@media(max-width:760px){.sidebar{width:72px;padding:14px 8px}.brand{justify-content:center;min-height:42px;padding:0 0 16px}.brand-logo{width:56px;height:42px;object-position:center}.nav-item{justify-content:center;min-height:44px;padding:11px 8px}.nav-label{position:absolute!important;width:1px!important;height:1px!important;padding:0!important;margin:-1px!important;overflow:hidden!important;clip:rect(0 0 0 0)!important;white-space:nowrap!important;border:0!important}.main{margin-left:72px}.topbar{padding:10px 14px}.content{padding:12px 16px 34px}.meta{font-size:12px}.meta span+span::before{padding:0 6px}.panel{padding:15px}.summary-panel{padding:0}.bar-row{grid-template-columns:minmax(0,1fr) auto;gap:5px 12px;padding:5px 0}.bar-track{grid-column:1/-1;grid-row:2}.bar-total{grid-column:2;grid-row:1}.summary-bars{padding:18px 14px 8px}.summary-legend{padding:13px 14px 0}.footnote{margin:12px 14px 15px}}
@media(max-width:480px){.sidebar{width:64px;padding-inline:7px}.main{margin-left:64px}.toolbar{gap:8px;width:100%}.locale-picker{min-width:0;flex:1}.locale-picker select{min-width:0}.export-button{min-width:92px}.export-menu{max-width:calc(100vw - 80px)}.chart svg{min-width:620px}}
</style></head><body>
<aside class="sidebar">
<a class="brand" href="#summary" aria-label="Fairy C"><img class="brand-logo" src="{{.Logo}}" alt="Fairy C"></a>
<nav class="tabs" role="tablist" aria-orientation="vertical" aria-label="{{t "Build footprint"}}">
<button type="button" class="nav-item" id="tab-summary" role="tab" aria-controls="panel-summary" aria-selected="true" tabindex="0" data-panel="panel-summary"><span class="nav-icon" aria-hidden="true">{{index .Icons "summary"}}</span><span class="nav-label">{{t "Summary"}}</span></button>
<button type="button" class="nav-item" id="tab-footprint" role="tab" aria-controls="panel-footprint" aria-selected="false" tabindex="-1" data-panel="panel-footprint"><span class="nav-icon" aria-hidden="true">{{index .Icons "footprint"}}</span><span class="nav-label">{{t "Footprint"}}</span></button>
<button type="button" class="nav-item" id="tab-growth" role="tab" aria-controls="panel-growth" aria-selected="false" tabindex="-1" data-panel="panel-growth"><span class="nav-icon" aria-hidden="true">{{index .Icons "growth"}}</span><span class="nav-label">{{t "Growth"}}</span></button>
<button type="button" class="nav-item" id="tab-measurements" role="tab" aria-controls="panel-measurements" aria-selected="false" tabindex="-1" data-panel="panel-measurements"><span class="nav-icon" aria-hidden="true">{{index .Icons "measurements"}}</span><span class="nav-label">{{t "Measurements"}}</span></button>
<button type="button" class="nav-item" id="tab-toolchains" role="tab" aria-controls="panel-toolchains" aria-selected="false" tabindex="-1" data-panel="panel-toolchains"><span class="nav-icon" aria-hidden="true">{{index .Icons "toolchains"}}</span><span class="nav-label">{{t "Toolchains"}}</span></button>
<button type="button" class="nav-item" id="tab-notes" role="tab" aria-controls="panel-notes" aria-selected="false" tabindex="-1" data-panel="panel-notes"><span class="nav-icon" aria-hidden="true">{{index .Icons "notes"}}</span><span class="nav-label">{{t "Notes"}}</span></button>
</nav>
</aside>
<main class="main">
<header class="topbar"><div class="toolbar">
{{if gt (len .Locales) 1}}<label class="locale-picker"><span class="visually-hidden">{{t "Language"}}</span><select id="locale" aria-label="{{t "Language"}}" onchange="location.href=this.value+location.hash">{{range .Locales}}<option value="{{.URL}}" {{if eq .Code $.Locale}}selected{{end}}>{{.Name}}</option>{{end}}</select></label>{{end}}
<div class="export-wrap" id="export-wrap"><button type="button" class="export-button" id="export-button" aria-expanded="false" aria-controls="export-menu"><span class="export-icon" aria-hidden="true">{{index .Icons "download"}}</span><span>{{t "Export"}}</span></button><div class="export-menu" id="export-menu" role="menu" hidden><div id="chart-exports"><div class="export-scope" id="export-chart-name">{{t "Summary"}}</div><a href="{{artifact "summary.svg"}}" data-export="svg" role="menuitem">SVG</a><a href="{{artifact "summary.png"}}" data-export="png" role="menuitem">PNG</a><div class="menu-rule"></div></div><a href="receipt.json" role="menuitem">JSON</a></div></div>
</div></header>
<div class="content">
<h1>{{t "Build footprint"}}</h1>
<p class="meta"><span>{{.Date}}</span><span>{{.Receipt.Host.OS}}/{{.Receipt.Host.Arch}}</span><span>{{.SummaryWorkload}}</span>{{if .ShowModules}}<span>{{.Receipt.Options.Modules}} {{t "modules"}}</span>{{end}}<span>{{.Receipt.Options.Edits}} {{t "Source edits"}}</span><span>{{.Receipt.Options.Repetitions}} {{t "Run"}}</span></p>
<p class="run-status{{if .Failed}} has-failure{{end}}">{{.Passed}} {{t "passed"}} · {{.Failed}} {{t "failed or unavailable"}}</p>
{{if .Failed}}<section class="failure"><strong>{{.Failed}} {{t "failed or unavailable"}}</strong><ul>{{range .Rows}}{{if ne .Status "passed"}}<li><strong>{{.Language}} / {{.Profile}} / {{.Workload}}</strong> — {{t .Status}}: {{.Reason}}</li>{{end}}{{end}}</ul></section>{{end}}
<section id="panel-summary" role="tabpanel" tabindex="0" aria-labelledby="tab-summary" class="panel summary-panel">
<div class="scroll"><table class="summary-table"><thead><tr><th scope="col">{{t "Language"}} / {{t "Profile"}}</th><th scope="col">{{t "Project outputs/cache"}}</th><th scope="col">{{t "Shared cache"}}</th><th scope="col">{{t "End total"}}</th></tr></thead><tbody>{{range .SummaryRows}}<tr><td>{{.Label}}</td><td>{{.Project}}</td><td>{{.Shared}}</td><td>{{.Total}}</td></tr>{{end}}</tbody></table></div>
<div class="summary-bars"><h2 class="compare-title">{{t "Build footprint"}}</h2>{{range .SummaryRows}}<div class="bar-row"><span class="bar-label">{{.Label}}</span><div class="bar-track" aria-hidden="true"><span class="bar-project" style="width:{{.ProjectWidth}}%"></span><span class="bar-shared" style="width:{{.SharedWidth}}%"></span><span class="bar-temp" style="width:{{.TempWidth}}%"></span></div><span class="bar-total">{{.Total}}</span></div>{{end}}</div>
<div class="summary-legend"><span class="legend-item"><span class="legend-swatch legend-project"></span>{{t "Project outputs/cache"}}</span><span class="legend-item"><span class="legend-swatch legend-shared"></span>{{t "Shared cache"}}</span>{{if .HasTemp}}<span class="legend-item"><span class="legend-swatch bar-temp"></span>{{t "Retained temp"}}</span>{{end}}</div>
<p class="footnote">{{t "Logical file sizes. Toolchains and peak usage excluded. Debug defaults differ."}}</p>
</section>
<section id="panel-footprint" role="tabpanel" tabindex="0" aria-labelledby="tab-footprint" class="panel chart" hidden>{{.Footprint}}</section>
<section id="panel-growth" role="tabpanel" tabindex="0" aria-labelledby="tab-growth" class="panel chart" hidden>{{.Growth}}</section>
<section id="panel-measurements" role="tabpanel" tabindex="0" aria-labelledby="tab-measurements" class="panel" hidden><h2>{{t "Measurements"}}</h2><p class="meta">{{t "Times in milliseconds. Totals are logical bytes."}}</p>
<div class="filters"><label>{{t "Language"}} <select id="language"><option value="">{{t "All"}}</option>{{range .Receipt.Options.Languages}}<option value="{{.}}">{{t .}}</option>{{end}}</select></label><label>{{t "Profile"}} <select id="profile"><option value="">{{t "All"}}</option>{{range .Receipt.Options.Profiles}}<option value="{{.}}">{{t .}}</option>{{end}}</select></label><label>{{t "Workload"}} <select id="workload"><option value="">{{t "All"}}</option>{{range .Receipt.Options.Workloads}}<option value="{{.}}">{{t .}}</option>{{end}}</select></label></div>
<div class="scroll"><table id="measurements"><thead><tr><th>{{t "Language"}}</th><th>{{t "Profile"}}</th><th>{{t "Workload"}}</th><th>{{t "Run"}}</th><th>{{t "Status"}}</th><th>{{t "Cold ms"}}</th><th>{{t "No-op ms"}}</th><th>{{t "Private edit ms"}}</th><th>{{t "Public edit ms"}}</th><th>{{t "End total"}}</th><th>{{t "Project outputs/cache"}}</th><th>{{t "Shared cache"}}</th><th>{{t "Growth"}}</th><th>{{t "Binary"}}</th><th>{{t "Total/source"}}</th><th>{{t "Files"}}</th><th>{{t "After clean"}}</th><th>{{t "Verified builds"}}</th><th>{{t "Reason"}}</th></tr></thead><tbody>
{{range .Rows}}<tr data-language="{{.Language}}" data-profile="{{.Profile}}" data-workload="{{.Workload}}"><td>{{.Language}}</td><td>{{t .Profile}}</td><td>{{t .Workload}}</td><td>{{.Repetition}}</td><td>{{t .Status}}</td><td>{{.Cold}}</td><td>{{.NoOp}}</td><td>{{.Private}}</td><td>{{.Public}}</td><td>{{.Final}}</td><td>{{.Project}}</td><td>{{.Shared}}</td><td>{{.Growth}}</td><td>{{.Binary}}</td><td>{{.Amplification}}</td><td>{{.Files}}</td><td>{{.Clean}}</td><td>{{.Verified}}</td><td class="reason">{{.Reason}}</td></tr>{{end}}</tbody></table></div></section>
<section id="panel-toolchains" role="tabpanel" tabindex="0" aria-labelledby="tab-toolchains" class="panel" hidden><h2>{{t "Toolchains"}}</h2>{{range .Receipt.Toolchains}}<h3>{{.Language}} — {{t .Status}}</h3><pre>{{.Version}}
{{.Backend}}
{{t "Executable"}}: {{.Executable}} ({{.ExecutableBytes}} {{t "bytes; not total install size"}})
SHA-256: {{.ExecutableHash}}
{{.Reason}}</pre>{{end}}
<h3>{{t "Installed toolchain files"}}</h3><p class="meta">{{t "Separate from build footprint. Shared installations appear once."}}</p><div class="scroll"><table><thead><tr><th>{{t "Installation"}}</th><th>{{t "Used by"}}</th><th>{{t "Logical bytes"}}</th><th>{{t "Files"}}</th><th>{{t "Status"}}</th><th>{{t "Scope"}}</th></tr></thead><tbody>{{range .Receipt.Installations}}<tr><td>{{.Name}}</td><td>{{.UsedBy}}</td><td>{{.Bytes}}</td><td>{{.Files}}</td><td>{{t .Status}} {{.Reason}}</td><td class="reason">{{.Scope}}</td></tr>{{end}}</tbody></table></div></section>
<section id="panel-notes" role="tabpanel" tabindex="0" aria-labelledby="tab-notes" class="panel" hidden><h2>{{t "Notes"}}</h2><ul class="notes"><li>{{t "Retained logical bytes include project outputs, caches, and the shared experiment cache. Toolchain installs and transient peak disk use are excluded."}}</li><li>{{t "The fixtures are small and synthetic, with no third-party dependencies."}}</li><li>{{t "Development defaults differ by toolchain. Rust small-dev uses line-tables-only debug data and disables incremental builds."}}</li><li>{{t "Footprint bars are arithmetic means across repetitions; growth points are medians at each edit."}}</li></ul></section>
</div>
</main>
<script>
const tabs = Array.from(document.querySelectorAll('.tabs button'));
const panels = Array.from(document.querySelectorAll('.panel'));
const measurements = document.getElementById('measurements');
const identityHeaders = Array.from(measurements.tHead.rows[0].cells).slice(0, 3);
function pinMeasurementColumns() {
  const widths = identityHeaders.map(cell => cell.getBoundingClientRect().width);
  if (!widths[0]) return;
  measurements.style.setProperty('--pin-profile', widths[0] + 'px');
  measurements.style.setProperty('--pin-workload', (widths[0] + widths[1]) + 'px');
}
const identityResize = new ResizeObserver(pinMeasurementColumns);
for (const cell of identityHeaders) identityResize.observe(cell);
const exportWrap = document.getElementById('export-wrap');
const exportButton = document.getElementById('export-button');
const exportMenu = document.getElementById('export-menu');
const chartExports = document.getElementById('chart-exports');
const chartExportName = document.getElementById('export-chart-name');
const chartFiles = {
  summary: {svg: '{{artifact "summary.svg"}}', png: '{{artifact "summary.png"}}'},
  footprint: {svg: '{{artifact "footprint.svg"}}', png: '{{artifact "footprint.png"}}'},
  growth: {svg: '{{artifact "growth.svg"}}', png: '{{artifact "growth.png"}}'}
};
const chartLabels = {};
for (const tab of tabs) chartLabels[tab.dataset.panel.slice(6)] = tab.querySelector('.nav-label').textContent;
function panelName(value) {
  const name = value.replace(/^#/, '');
  return tabs.some(tab => tab.dataset.panel === 'panel-' + name) ? name : 'summary';
}
function updateExport(name) {
  const files = chartFiles[name];
  chartExports.hidden = !files;
  if (files) {
    chartExportName.textContent = chartLabels[name] || '';
    for (const link of chartExports.querySelectorAll('[data-export]')) link.href = files[link.dataset.export];
  }
}
function selectPanel(name, focus) {
  for (const tab of tabs) {
    const selected = tab.dataset.panel === 'panel-' + name;
    tab.setAttribute('aria-selected', selected ? 'true' : 'false');
    tab.tabIndex = selected ? 0 : -1;
    if (selected && focus) tab.focus();
  }
  for (const panel of panels) panel.hidden = panel.id !== 'panel-' + name;
  if (name === 'measurements') pinMeasurementColumns();
  document.querySelector('.main').scrollTop = 0;
  updateExport(name);
}
function activate(name, focus) {
  selectPanel(name, focus);
  if (location.hash !== '#' + name) location.hash = name;
}
function applyHash() {
  selectPanel(panelName(location.hash), false);
}
function setExportOpen(open) {
  exportMenu.hidden = !open;
  exportButton.setAttribute('aria-expanded', open ? 'true' : 'false');
}
tabs.forEach((tab, index) => {
  tab.addEventListener('click', () => activate(tab.dataset.panel.slice(6), false));
  tab.addEventListener('keydown', event => {
    let next = index;
    if (event.key === 'ArrowUp') next = (index - 1 + tabs.length) % tabs.length;
    else if (event.key === 'ArrowDown') next = (index + 1) % tabs.length;
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = tabs.length - 1;
    else return;
    event.preventDefault();
    activate(tabs[next].dataset.panel.slice(6), true);
  });
});
exportButton.addEventListener('click', event => {
  event.stopPropagation();
  setExportOpen(exportMenu.hidden);
});
document.addEventListener('click', event => {
  if (!exportWrap.contains(event.target)) setExportOpen(false);
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape' && !exportMenu.hidden) {
    setExportOpen(false);
    exportButton.focus();
  }
});
exportMenu.addEventListener('click', event => {
  if (event.target.closest('a')) setExportOpen(false);
});
window.addEventListener('hashchange', applyHash);
applyHash();
for (const el of document.querySelectorAll('.filters select')) el.addEventListener('change', () => { for (const row of document.querySelectorAll('#measurements tbody tr')) row.hidden = ['language','profile','workload'].some(k => document.getElementById(k).value && row.dataset[k] !== document.getElementById(k).value); });
</script>
</body></html>`
