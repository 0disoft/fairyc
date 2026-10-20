package main

import (
	"fmt"
	"html"
	"strings"
)

type point struct{ X, Y float64 }
type shape struct {
	Kind, Text, Color string
	X, Y, W, H, Size  float64
	Points            []point
}
type scene struct {
	Width, Height      int
	Title, Description string
	Shapes             []shape
}

const ink = "#16283c"
const muted = "#53677c"

var bucketColors = []string{"#087f8c", "#b45309", "#7653b5", "#54718e", "#d05b76", "#2877c7", "#858c94"}
var bucketNames = []string{"Binary", "Debug files", "Incremental", "Intermediate", "Local cache", "Shared cache", "Retained temp"}
var lineColors = []string{"#087f8c", "#5c3b9c", "#bd4b21", "#be3c6d", "#2168b0", "#4e7c20", "#857000", "#654d3f", "#586b82"}

func (s *scene) text(x, y, size float64, color, text string) {
	s.Shapes = append(s.Shapes, shape{Kind: "text", X: x, Y: y, Size: size, Color: color, Text: text})
}
func (s *scene) rect(x, y, w, h float64, color string) {
	s.Shapes = append(s.Shapes, shape{Kind: "rect", X: x, Y: y, W: w, H: h, Color: color})
}
func (s *scene) line(points []point, color string) {
	s.Shapes = append(s.Shapes, shape{Kind: "line", Points: points, Color: color})
}
func chartHeader(r Receipt, title string) scene {
	s := scene{Width: 1120, Title: title, Description: r.Method}
	s.text(32, 42, 26, ink, title)
	s.text(32, 69, 13, muted, fmt.Sprintf("%s | %s/%s | %s | %d run(s)", strings.Split(r.Started, "T")[0], r.Host.OS, r.Host.Arch, shortCommit(r), r.Options.Repetitions))
	s.Height = 95
	return s
}
func buckets(f Footprint) []int64 {
	return []int64{f.Binary, f.Debug, f.Incremental, f.Intermediate, f.LocalCache, f.SharedCache, f.Temporary}
}

func footprintChart(r Receipt) scene {
	s := chartHeader(r, fmt.Sprintf("Build footprints after %d source edits", r.Options.Edits))
	y := float64(s.Height) + 10
	for _, workload := range r.Options.Workloads {
		groups := seriesFor(r, workload)
		s.text(32, y, 19, ink, workload)
		y += 25
		s.text(32, y, 12, muted, "Components: mean bytes across repetitions. End total / binary / files / clean remainder.")
		y += 27
		maximum := 1.0
		for _, g := range groups {
			total := 0.0
			for _, c := range g.Cases {
				total += float64(finalSample(c).Footprint.Total)
			}
			maximum = max(maximum, total/float64(len(g.Cases)))
		}
		for _, g := range groups {
			values := make([]float64, 7)
			total, binary, remaining, files := 0.0, 0.0, 0.0, 0.0
			for _, c := range g.Cases {
				last := finalSample(c)
				for i, b := range buckets(last.Footprint) {
					values[i] += float64(b)
				}
				total += float64(last.Footprint.Total)
				binary += float64(last.Footprint.Binary)
				files += float64(last.Footprint.Files)
				remaining += float64(findSample(c, "clean", 0).Footprint.Total)
			}
			n := float64(len(g.Cases))
			total /= n
			binary /= n
			remaining /= n
			files /= n
			s.text(32, y+17, 14, ink, g.Language+" / "+g.Profile)
			x := 245.0
			s.rect(x, y, 510, 23, "#e9eef3")
			for i, b := range values {
				w := (b / n) / maximum * 510
				if w > 0 {
					s.rect(x, y, w, 23, bucketColors[i])
					x += w
				}
			}
			s.text(775, y+15, 13, ink, bytesLabel(total)+" total")
			s.text(775, y+33, 12, muted, fmt.Sprintf("%s binary / %.0f files / %s clean", bytesLabel(binary), files, bytesLabel(remaining)))
			y += 54
		}
		if len(groups) == 0 {
			s.text(32, y+16, 14, muted, "No verified measurements for this workload.")
			y += 40
		}
		y += 18
	}
	for i, name := range bucketNames {
		x := 32 + float64(i%4)*267
		row := float64(i/4) * 25
		s.rect(x, y+row, 12, 12, bucketColors[i])
		s.text(x+19, y+row+11, 12, muted, name)
	}
	y += 75
	s.text(32, y, 12, muted, "Logical file sizes, including shared cache. Debug defaults differ. Details: report.html")
	s.Height = int(y + 30)
	return s
}

func growthChart(r Receipt) scene {
	s := chartHeader(r, "Disk growth while editing")
	y := float64(s.Height) + 10
	for _, workload := range r.Options.Workloads {
		groups := seriesFor(r, workload)
		s.text(32, y, 19, ink, workload+" - median retained MiB")
		y += 28
		left, top, width, height := 100.0, y, 970.0, 220.0
		maximum := 1.0
		value := func(g series, edit int) float64 {
			v := []float64{}
			for _, c := range g.Cases {
				stage := "edit"
				if edit == 0 {
					stage = "cold"
				}
				v = append(v, float64(findSample(c, stage, edit).Footprint.Total)/(1<<20))
			}
			return median(v)
		}
		for _, g := range groups {
			for edit := 0; edit <= r.Options.Edits; edit++ {
				maximum = max(maximum, value(g, edit))
			}
		}
		for tick := 0; tick <= 4; tick++ {
			yy := top + height - float64(tick)*height/4
			s.line([]point{{left, yy}, {left + width, yy}}, "#dce4ec")
			s.text(32, yy+4, 12, muted, fmt.Sprintf("%.1f", maximum*float64(tick)/4))
		}
		for edit := 0; edit <= r.Options.Edits; edit++ {
			if edit != 0 && edit != r.Options.Edits && edit%5 != 0 {
				continue
			}
			x := left + float64(edit)/float64(max(1, r.Options.Edits))*width
			s.text(x-5, top+height+22, 12, muted, fmt.Sprint(edit))
		}
		for i, g := range groups {
			points := []point{}
			for edit := 0; edit <= r.Options.Edits; edit++ {
				points = append(points, point{left + float64(edit)/float64(max(1, r.Options.Edits))*width, top + height - value(g, edit)/maximum*height})
			}
			s.line(points, lineColors[i%len(lineColors)])
		}
		y += height + 54
		s.text(100, y, 12, muted, "Source edits")
		y += 28
		for i, g := range groups {
			x := 32 + float64(i%3)*360
			row := float64(i/3) * 23
			s.rect(x, y+row, 14, 4, lineColors[i%len(lineColors)])
			s.text(x+22, y+row+6, 12, ink, g.Language+" / "+g.Profile)
		}
		y += float64((len(groups)+2)/3)*23 + 32
	}
	s.text(32, y, 12, muted, "Logical file sizes, including shared cache. Details: report.html")
	s.Height = int(y + 30)
	return s
}

func (s scene) svg() string {
	var b strings.Builder
	id := "chart-" + digest([]byte(s.Title))[:8]
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="%s-title %s-desc"><title id="%s-title">%s</title><desc id="%s-desc">%s</desc><rect width="100%%" height="100%%" fill="#f8fafc"/>`, s.Width, s.Height, s.Width, s.Height, id, id, id, html.EscapeString(s.Title), id, html.EscapeString(s.Description))
	for _, p := range s.Shapes {
		switch p.Kind {
		case "text":
			fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-family="Arial,Helvetica,sans-serif" font-size="%.0f" fill="%s">%s</text>`, p.X, p.Y, p.Size, p.Color, html.EscapeString(p.Text))
		case "rect":
			fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`, p.X, p.Y, p.W, p.H, p.Color)
		case "line":
			if len(p.Points) == 1 {
				fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="3" fill="%s"/>`, p.Points[0].X, p.Points[0].Y, p.Color)
			} else {
				b.WriteString(`<polyline fill="none" stroke="` + p.Color + `" stroke-width="2" points="`)
				for _, point := range p.Points {
					fmt.Fprintf(&b, "%.2f,%.2f ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		}
	}
	b.WriteString("</svg>\n")
	return b.String()
}

func summaryChart(r Receipt) scene {
	workload := r.Options.Workloads[0]
	for _, name := range r.Options.Workloads {
		if name == "many-modules" {
			workload = name
		}
	}
	filtered := r
	filtered.Options.Workloads = []string{workload}
	filtered.Cases = nil
	for _, c := range r.Cases {
		if c.Workload == workload && (c.Profile == "dev" || c.Profile == "small-dev") {
			filtered.Cases = append(filtered.Cases, c)
		}
	}
	profiles := "development profiles"
	if len(filtered.Cases) == 0 {
		filtered.Cases = r.Cases
		profiles = "all profiles"
	}
	s := chartHeader(r, "Build footprint")
	s.text(32, 99, 13, muted, fmt.Sprintf("%s | %d source edits | %s", workload, r.Options.Edits, profiles))
	groups := seriesFor(filtered, workload)
	maximum := 1.0
	for _, g := range groups {
		total := 0.0
		for _, c := range g.Cases {
			total += float64(finalSample(c).Footprint.Total)
		}
		maximum = max(maximum, total/float64(len(g.Cases)))
	}
	y := 132.0
	for _, g := range groups {
		project, shared, temporary := 0.0, 0.0, 0.0
		for _, c := range g.Cases {
			f := finalSample(c).Footprint
			project += float64(f.Total - f.SharedCache - f.Temporary)
			shared += float64(f.SharedCache)
			temporary += float64(f.Temporary)
		}
		n := float64(len(g.Cases))
		label := map[string]string{"fairy": "Fairy C", "zig": "Zig", "rust": "Rust", "c": "C"}[g.Language]
		if g.Profile != "dev" {
			label += " / " + g.Profile
		}
		s.text(32, y+19, 16, ink, label)
		s.rect(250, y, 590, 26, "#e9eef3")
		x := 250.0
		for i, b := range []float64{project, shared, temporary} {
			w := b / n / maximum * 590
			if w > 0 {
				s.rect(x, y, w, 26, []string{bucketColors[0], bucketColors[5], bucketColors[6]}[i])
				x += w
			}
		}
		s.text(860, y+19, 16, ink, bytesLabel((project+shared+temporary)/n))
		y += 48
	}
	if len(groups) == 0 {
		s.text(32, y+19, 16, muted, "No passed measurements")
		y += 48
	}
	for i, name := range []string{"Project outputs/cache", "Shared cache", "Retained temp"} {
		x := 32 + float64(i)*300
		s.rect(x, y+4, 12, 12, []string{bucketColors[0], bucketColors[5], bucketColors[6]}[i])
		s.text(x+20, y+15, 12, muted, name)
	}
	s.text(32, y+45, 12, muted, "Logical file sizes. Toolchains and peak usage excluded. Debug defaults differ.")
	s.Height = int(y + 65)
	return s
}
