package main

import (
	"bytes"
	"embed"
	"fmt"
	"math"
	"strings"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// Bundled subsets make exported text independent of system fonts and network access.
//
//go:embed fonts/*.ttf
var fontFiles embed.FS

type fontSet []*font.Face

func (fonts fontSet) ResolveFace(r rune) *font.Face {
	for _, face := range fonts {
		if _, ok := face.NominalGlyph(r); ok {
			return face
		}
	}
	return fonts[0]
}

var chartFonts = loadChartFonts()

func loadChartFonts() fontSet {
	var result fontSet
	for _, name := range []string{"BenchSans.ttf", "BenchDevanagari.ttf", "BenchChinese.ttf", "BenchKorean.ttf"} {
		data, err := fontFiles.ReadFile("fonts/" + name)
		if err != nil {
			panic(err)
		}
		face, err := font.ParseTTF(bytes.NewReader(data))
		if err != nil {
			panic(err)
		}
		result = append(result, face)
	}
	return result
}

// textOutline shapes whole script runs before extracting outlines, including Hindi
// conjuncts and vowel marks. Both SVG and PNG use these exact positions.
func textOutline(p shape, locale string, maxWidth float64) []font.Segment {
	text := []rune(p.Text)
	var segmenter shaping.Segmenter
	var shaper shaping.HarfbuzzShaper
	inputs := segmenter.Split(shaping.Input{Text: text, RunEnd: len(text), Direction: di.DirectionLTR, Size: fixed.Int26_6(math.Round(p.Size * 64)), Language: language.NewLanguage(locale)}, chartFonts)
	var paths []font.Segment
	x := 0.0
	for _, input := range inputs {
		run := shaper.Shape(input)
		scale := p.Size / float64(run.Face.Upem())
		for _, glyph := range run.Glyphs {
			outline, ok := run.Face.GlyphDataOutline(glyph.GlyphID)
			if ok {
				for _, segment := range outline.Segments {
					for i := range segment.ArgsSlice() {
						pt := &segment.Args[i]
						pt.X = float32(x + float64(glyph.XOffset)/64 + float64(pt.X)*scale)
						pt.Y = float32(-float64(glyph.YOffset)/64 - float64(pt.Y)*scale)
					}
					paths = append(paths, segment)
				}
			}
			x += float64(glyph.Advance) / 64
		}
	}
	fit := 1.0
	if maxWidth > 0 && x > maxWidth {
		fit = maxWidth / x
	}
	for i := range paths {
		for j := range paths[i].ArgsSlice() {
			pt := &paths[i].Args[j]
			pt.X = float32(p.X + float64(pt.X)*fit)
			pt.Y = float32(p.Y + float64(pt.Y)*fit)
		}
	}
	return paths
}

func outlineSVG(paths []font.Segment) string {
	var b strings.Builder
	for _, segment := range paths {
		b.WriteByte(map[opentype.SegmentOp]byte{opentype.SegmentOpMoveTo: 'M', opentype.SegmentOpLineTo: 'L', opentype.SegmentOpQuadTo: 'Q', opentype.SegmentOpCubeTo: 'C'}[segment.Op])
		for _, pt := range segment.ArgsSlice() {
			fmt.Fprintf(&b, "%.2f %.2f ", pt.X, pt.Y)
		}
	}
	return b.String()
}
