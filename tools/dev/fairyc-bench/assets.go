package main

import (
	"embed"
	"encoding/base64"
	"html/template"
)

//go:embed assets/fairyc-logo.png assets/icons/*.svg
var reportAssets embed.FS

func reportBrand() (template.URL, map[string]template.HTML) {
	logo, err := reportAssets.ReadFile("assets/fairyc-logo.png")
	if err != nil {
		panic(err)
	}
	icons := make(map[string]template.HTML)
	for _, key := range []string{"summary", "footprint", "growth", "measurements", "toolchains", "notes", "download"} {
		data, err := reportAssets.ReadFile("assets/icons/" + key + ".svg")
		if err != nil {
			panic(err)
		}
		icons[key] = template.HTML(data) // Trusted, bundled Feather assets, never receipt data.
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(logo)), icons
}
