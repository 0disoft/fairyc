package main

import (
	"reflect"
	"regexp"
	"testing"
)

func TestReportCatalogs(t *testing.T) {
	formats := regexp.MustCompile(`%(?:\.[0-9]+)?[sdf]`)
	for _, code := range supportedLocales {
		catalog := catalogs[code]
		if len(catalog) != len(catalogs["en"]) {
			t.Fatalf("%s: incomplete catalog", code)
		}
		for key := range catalogs["en"] {
			value := catalog[key]
			if value == "" || !reflect.DeepEqual(formats.FindAllString(key, -1), formats.FindAllString(value, -1)) {
				t.Errorf("%s: missing text or changed placeholders: %s", code, key)
			}
		}
	}
	if translator("hi")("rustc --version") != "rustc --version" {
		t.Fatal("raw identifiers must stay verbatim")
	}
}
