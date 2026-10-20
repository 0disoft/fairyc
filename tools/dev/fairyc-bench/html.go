package main

import (
	"fmt"
	"html/template"
	"os"
)

type reportRow struct {
	Language, Profile, Workload, Status, Reason                              string
	Repetition                                                               int
	Cold, NoOp, Private, Public, Final, Clean, Binary, Growth, Amplification string
	Files                                                                    int
	Verified                                                                 int
}
type reportView struct {
	Receipt           Receipt
	Commit            string
	Rows              []reportRow
	Footprint, Growth template.HTML
	Passed, Failed    int
}

func renderHTML(r Receipt, path string) error {
	v := reportView{Receipt: r, Commit: shortCommit(r), Footprint: template.HTML(footprintChart(r).svg()), Growth: template.HTML(growthChart(r).svg())}
	for _, c := range r.Cases {
		row := reportRow{Language: c.Language, Profile: c.Profile, Workload: c.Workload, Repetition: c.Repetition, Status: c.Status, Reason: c.Reason, Cold: "-", NoOp: "-", Private: "-", Public: "-", Final: "-", Clean: "-", Binary: "-", Growth: "-"}
		if c.Status == "passed" {
			v.Passed++
			cold := findSample(c, "cold", 0)
			last := finalSample(c)
			row.Cold = fmt.Sprintf("%.1f", cold.WallMS)
			row.NoOp = fmt.Sprintf("%.1f", findSample(c, "no-op", 0).WallMS)
			row.Final = bytesLabel(float64(last.Footprint.Total))
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
	err = reportTemplate.Execute(file, v)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Fairy C build footprint report</title>
<style>
:root{color-scheme:light;font-family:system-ui,sans-serif;color:#16283c;background:#f1f5f9}body{max-width:1240px;margin:0 auto;padding:28px}h1{font-size:clamp(26px,4vw,40px);margin:10px 0}h2{margin:0 0 18px}p{line-height:1.6}a{color:#146b98}section{background:#fff;border:1px solid #dce4ec;border-radius:12px;padding:24px;margin:24px 0}.meta{color:#53677c}.pill{display:inline-block;background:#e0f2f1;padding:6px 14px;border-radius:30px;margin:4px 12px 4px 0}.failure{background:#ffe5e5}.chart{overflow:auto}.chart svg{display:block;width:100%;min-width:760px;height:auto}.scroll{overflow:auto}table{border-collapse:collapse;width:100%;font-size:13px}th,td{text-align:left;padding:10px;border-bottom:1px solid #e4eaf0;white-space:nowrap}th{background:#eef3f7}td.reason{white-space:normal;min-width:170px}.filters{display:flex;gap:16px;flex-wrap:wrap;margin-bottom:18px}label{display:flex;gap:8px;align-items:center}select{padding:7px;border:1px solid #9caebe;border-radius:6px}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f1f5f9;padding:14px;border-radius:7px}.downloads{display:flex;gap:18px;flex-wrap:wrap}li{margin:8px 0;line-height:1.5}summary{cursor:pointer;font-weight:600}code{background:#edf2f6;padding:2px 4px}tr[hidden]{display:none}@media(max-width:640px){body{padding:12px}section{padding:14px}}
</style></head><body>
<p class="meta">FAIRY C / DEVELOPER TOOLS</p><h1>Build outputs and cache growth</h1>
<p class="meta">{{.Receipt.Started}} · {{.Receipt.Host.OS}}/{{.Receipt.Host.Arch}} · commit {{.Commit}}<br>{{.Receipt.Host.CPU}} · {{.Receipt.Host.CPUs}} logical CPUs · runner {{.Receipt.Host.GoVersion}}</p>
<span class="pill">{{.Passed}} verified sequences</span><span class="pill {{if .Failed}}failure{{end}}">{{.Failed}} failed or unavailable</span>
<section><h2>Scope</h2><p>{{.Receipt.Method}}</p><p>The tokenizer reads the same stdin and produces a token count and rolling checksum. The many-modules workload additionally calls {{.Receipt.Options.Modules}} helper modules. Every build is executed and checked against the Go reference. Every fifth source edit changes a public function signature; other edits change one private helper. The checksum changes with each edit.</p>
<p>Rust small-dev sets <code>debug = "line-tables-only"</code> and <code>incremental = false</code>. Zig release uses <code>ReleaseFast</code>. C dev requests debug information; Fairy C development does not promise equivalent debugger support. The Zig and C fixtures use libc stdin/stdout, Rust uses its standard library, and Fairy C uses its SDK. These are build workflow measurements with different runtime and debugging facilities.</p>
<div class="downloads"><a href="receipt.json">Raw JSON and exact commands</a><a href="footprint.svg">Footprint SVG</a><a href="footprint.png">Footprint PNG</a><a href="growth.svg">Growth SVG</a><a href="growth.png">Growth PNG</a></div></section>
<section class="chart">{{.Footprint}}</section><section class="chart">{{.Growth}}</section>
<section><h2>Measurements</h2><p class="meta">Times in milliseconds. Private/public edit times are medians within each sequence. End size is the final build before cleaning. Filters apply to this table.</p>
<div class="filters"><label>Language <select id="language"><option value="">All</option>{{range .Receipt.Options.Languages}}<option>{{.}}</option>{{end}}</select></label><label>Profile <select id="profile"><option value="">All</option>{{range .Receipt.Options.Profiles}}<option>{{.}}</option>{{end}}</select></label><label>Workload <select id="workload"><option value="">All</option>{{range .Receipt.Options.Workloads}}<option>{{.}}</option>{{end}}</select></label></div>
<div class="scroll"><table id="measurements"><thead><tr><th>Language</th><th>Profile</th><th>Workload</th><th>Run</th><th>Status</th><th>Cold ms</th><th>No-op ms</th><th>Private edit ms</th><th>Public edit ms</th><th>End size</th><th>Growth</th><th>Binary</th><th>Total/source</th><th>Files</th><th>After clean</th><th>Verified builds</th><th>Reason</th></tr></thead><tbody>
{{range .Rows}}<tr data-language="{{.Language}}" data-profile="{{.Profile}}" data-workload="{{.Workload}}"><td>{{.Language}}</td><td>{{.Profile}}</td><td>{{.Workload}}</td><td>{{.Repetition}}</td><td>{{.Status}}</td><td>{{.Cold}}</td><td>{{.NoOp}}</td><td>{{.Private}}</td><td>{{.Public}}</td><td>{{.Final}}</td><td>{{.Growth}}</td><td>{{.Binary}}</td><td>{{.Amplification}}</td><td>{{.Files}}</td><td>{{.Clean}}</td><td>{{.Verified}}</td><td class="reason">{{.Reason}}</td></tr>{{end}}</tbody></table></div></section>
<section><h2>Toolchains</h2>{{range .Receipt.Toolchains}}<details><summary>{{.Language}} — {{.Status}}</summary><pre>{{.Version}}
{{.Backend}}
Executable: {{.Executable}} ({{.ExecutableBytes}} bytes; this is NOT total installation size)
SHA-256: {{.ExecutableHash}}
{{.Reason}}</pre></details>{{end}}</section>
<section><h2>Installed toolchain files (separate inventory)</h2><p class="meta">Excluded from build footprint charts. Shared installations appear once. OS files and components outside these roots are not inventoried.</p><div class="scroll"><table><thead><tr><th>Installation</th><th>Used by</th><th>Logical bytes</th><th>Files</th><th>Status</th><th>Scope</th></tr></thead><tbody>{{range .Receipt.Installations}}<tr><td>{{.Name}}</td><td>{{.UsedBy}}</td><td>{{.Bytes}}</td><td>{{.Files}}</td><td>{{.Status}} {{.Reason}}</td><td class="reason">{{.Scope}}</td></tr>{{end}}</tbody></table></div></section>
<section><h2>Reading the results</h2><ul><li>Each profile, workload and repetition begins with empty dedicated project outputs, Cargo home, Zig global cache, Fairy C user cache and temporary directory. Toolchain files and OS caches remain installed.</li><li>Charts include passed sequences only. The table preserves failed and unavailable cases; they do not become zero-byte results.</li><li>File lengths count each regular pathname once across disjoint measured roots. Hard links can occupy less physical storage; sparse files, filesystem compression and allocation units are not measured. Symlinks in measured output roots are rejected.</li><li>Separate PDB/debug files have their own bucket. Debug data embedded in an executable or object remains in that file's bucket. Rust dependency artifacts are part of intermediate outputs; these fixtures have no third-party packages.</li><li>Rust cleanup uses Cargo clean. For Fairy C/C the tool removes project build output; for Zig it removes zig-out and .zig-cache. Shared experiment caches are retained and measured after cleaning. Cleanup policies differ and are recorded per sample.</li><li>Footprint components use arithmetic means across repetitions so stacked totals add up. Growth curves use medians at each edit. With one repetition, all numbers are individual observations. Build order is fixed; timing is descriptive and subject to host noise.</li><li>Steady-state disk use excludes temporary files already deleted during compilation. Runtime speed, peak disk use, peak memory, downloaded dependencies and toolchain installation sizes are outside this measurement.</li></ul></section>
<script>for(const el of document.querySelectorAll('select'))el.addEventListener('change',()=>{for(const row of document.querySelectorAll('#measurements tbody tr'))row.hidden=['language','profile','workload'].some(k=>document.getElementById(k).value&&row.dataset[k]!==document.getElementById(k).value)});</script>
</body></html>`))
