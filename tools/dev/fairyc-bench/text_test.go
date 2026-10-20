package main

import (
	"testing"
	"unicode"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

func TestTranslatedGlyphCoverage(t *testing.T) {
	for code, catalog := range catalogs {
		for _, text := range catalog {
			for _, r := range text {
				if unicode.IsControl(r) {
					continue
				}
				if gid, ok := chartFonts.ResolveFace(r).NominalGlyph(r); !ok || gid == 0 {
					t.Fatalf("%s: missing glyph %U; rebuild font subsets", code, r)
				}
			}
		}
	}
}

func TestHindiShaping(t *testing.T) {
	// A conjunct is shaped as a cluster, rather than drawing each code point alone.
	text := []rune("क्षि")
	var segmenter shaping.Segmenter
	var shaper shaping.HarfbuzzShaper
	inputs := segmenter.Split(shaping.Input{Text: text, RunEnd: len(text), Direction: di.DirectionLTR, Language: language.NewLanguage("hi"), Size: fixed.I(20)}, chartFonts)
	count := 0
	for _, input := range inputs {
		run := shaper.Shape(input)
		for _, glyph := range run.Glyphs {
			if glyph.GlyphID == 0 {
				t.Fatal("missing Hindi conjunct glyph")
			}
		}
		count += len(run.Glyphs)
	}
	if count == 0 || count >= len(text) {
		t.Fatalf("Hindi conjunct was not shaped: %d glyphs for %d runes", count, len(text))
	}
}
