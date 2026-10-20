package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS

var supportedLocales = []string{"en", "ko", "zh", "es", "fr", "hi"}
var localeNames = map[string]string{"en": "English", "ko": "한국어", "zh": "中文", "es": "Español", "fr": "Français", "hi": "हिन्दी"}
var catalogs = loadCatalogs()

func loadCatalogs() map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, code := range supportedLocales {
		data, err := localeFiles.ReadFile("locales/" + code + ".json")
		if err != nil {
			panic(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic(err)
		}
		result[code] = catalog
	}
	return result
}

func translator(code string) func(string, ...any) string {
	return func(key string, args ...any) string {
		value := catalogs[code][key]
		if value == "" {
			value = key // Raw toolchain identifiers and diagnostics remain verbatim.
		}
		if len(args) > 0 {
			return fmt.Sprintf(value, args...)
		}
		return value
	}
}

func chartLocale(codes []string) string {
	if len(codes) == 0 {
		return "en"
	}
	return codes[0]
}

func artifactName(name, code string) string {
	if code == "en" {
		return name
	}
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + "." + code + ext
}
