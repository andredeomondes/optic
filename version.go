package main

import (
	_ "embed"
	"encoding/json"
)

// wailsJSON is the project file; info.productVersion is the app version,
// the same source the frontend build reads.
//
//go:embed wails.json
var wailsJSON []byte

// productVersion reads info.productVersion from wails.json ("" if missing).
func productVersion(b []byte) string {
	var w struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if json.Unmarshal(b, &w) != nil {
		return ""
	}
	return w.Info.ProductVersion
}
