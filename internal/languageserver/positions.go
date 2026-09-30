package languageserver

import (
	"errors"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

func bytePosition(text []byte, offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	p := Position{}
	for i := 0; i < offset; {
		r, n := utf8.DecodeRune(text[i:])
		if i+n > offset {
			break
		}
		i += n
		if r == '\n' {
			p.Line++
			p.Character = 0
		} else if r != '\r' {
			p.Character++
			if r > 0xffff {
				p.Character++
			}
		}
	}
	return p
}
func positionByte(text []byte, p Position) (int, bool) {
	if p.Line < 0 || p.Character < 0 {
		return 0, false
	}
	line := 0
	i := 0
	for line < p.Line && i < len(text) {
		if text[i] == '\n' {
			line++
		}
		i++
	}
	if line != p.Line {
		return 0, false
	}
	column := 0
	for i < len(text) && text[i] != '\r' && text[i] != '\n' {
		if column == p.Character {
			return i, true
		}
		r, n := utf8.DecodeRune(text[i:])
		units := 1
		if r > 0xffff {
			units = 2
		}
		if column+units > p.Character {
			return 0, false
		}
		column += units
		i += n
	}
	return i, column == p.Character
}
func textRange(text []byte, start, end int) Range {
	return Range{bytePosition(text, start), bytePosition(text, end)}
}
func filePath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("expected local file URI")
	}
	p := u.Path
	if runtime.GOOS == "windows" && len(p) > 2 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	p = filepath.FromSlash(p)
	if !filepath.IsAbs(p) {
		return "", errors.New("expected absolute file URI")
	}
	return filepath.Clean(p), nil
}
func fileURI(filename string) string {
	p := filepath.ToSlash(filename)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}
func contained(root, filename string) bool {
	rel, err := filepath.Rel(root, filename)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
