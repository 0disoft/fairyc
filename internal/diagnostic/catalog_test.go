package diagnostic

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var (
	catalogCodePattern = regexp.MustCompile(`^[EAV][0-9]{4}$`)
	catalogRowPattern  = regexp.MustCompile(`^\| \x60([EAV][0-9]{4})\x60 \| (.+) \|$`)
)

func TestCatalogEntriesAreCanonicalUniqueAndLookupable(t *testing.T) {
	entries := CatalogEntries()
	if len(entries) == 0 {
		t.Fatal("CatalogEntries() is empty")
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !catalogCodePattern.MatchString(entry.Code) {
			t.Fatalf("catalog code %q is not canonical", entry.Code)
		}
		if entry.Category == "" || entry.Meaning == "" {
			t.Fatalf("catalog entry is incomplete: %+v", entry)
		}
		if _, duplicate := seen[entry.Code]; duplicate {
			t.Fatalf("catalog code %q is duplicated", entry.Code)
		}
		seen[entry.Code] = struct{}{}
		got, found := Lookup(entry.Code)
		if !found || got != entry {
			t.Fatalf("Lookup(%q) = %+v, %t; want %+v, true", entry.Code, got, found, entry)
		}
	}
	entries[0].Meaning = "mutated"
	got, found := Lookup(entries[0].Code)
	if !found || got.Meaning == "mutated" {
		t.Fatal("CatalogEntries() exposed mutable catalog storage")
	}
	for _, code := range []string{"", "e1101", "E110", "E11010", "X1101", "E11A1"} {
		if _, found := Lookup(code); found {
			t.Fatalf("Lookup(%q) unexpectedly succeeded", code)
		}
	}
}

func TestCatalogMatchesOwningMarkdownTables(t *testing.T) {
	want := make(map[string]string)
	for _, path := range []string{
		filepath.Join("..", "..", "docs", "diagnostics", "catalog-v0.md"),
		filepath.Join("..", "..", "docs", "mir", "mir-v0.md"),
	} {
		for code, meaning := range readCatalogRows(t, path) {
			if previous, duplicate := want[code]; duplicate {
				t.Fatalf("diagnostic code %q is documented twice with %q and %q", code, previous, meaning)
			}
			want[code] = meaning
		}
	}
	got := make(map[string]string)
	for _, entry := range CatalogEntries() {
		got[entry.Code] = entry.Meaning
	}
	if len(got) != len(want) {
		t.Fatalf("catalog entry count = %d, documented exact-code rows = %d", len(got), len(want))
	}
	for code, meaning := range want {
		if got[code] != meaning {
			t.Fatalf("catalog %s meaning = %q, want documented %q", code, got[code], meaning)
		}
	}
}

func readCatalogRows(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := catalogRowPattern.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			rows[matches[1]] = matches[2]
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}
