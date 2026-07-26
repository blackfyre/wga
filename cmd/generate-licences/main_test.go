package main

import (
	"os"
	"strings"
	"testing"
)

func TestReadBunLockIncludesScopedPackage(t *testing.T) {
	packages, err := readBunLock("../../bun.lock")
	if err != nil {
		t.Fatalf("readBunLock() error = %v", err)
	}
	if got := packages["@kurkle/color"].version; got != "0.3.4" {
		t.Fatalf("@kurkle/color version = %q, want 0.3.4", got)
	}
}

func TestValidateManifestRejectsStaleComponent(t *testing.T) {
	record := component{
		Ecosystem: "npm",
		Name:      "example",
		Version:   "1.0.0",
		Targets:   []string{"browser"},
		Licence: licence{
			ID:       "MIT",
			Text:     "Licence text",
			Handling: "Include it.",
		},
		SourceEvidence: "node_modules/example/LICENSE",
	}
	loaded := manifest{Version: 1, Components: []component{record}}
	discovered := []component{record}
	discovered[0].Version = "2.0.0"

	err := validateManifest(loaded, discovered)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("validateManifest() error = %v, want stale error", err)
	}
}

func TestDiscoverBrowserComponentsReadsOnlyEmittedPackages(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("change to repository root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	metafile := `{
		"inputs": {
			"node_modules/chart.js/dist/chart.js": {"imports": [{"path": "node_modules/@kurkle/color/dist/color.esm.js"}]},
			"node_modules/@kurkle/color/dist/color.esm.js": {"imports": []},
			"node_modules/unused/index.js": {"imports": []}
		},
		"outputs": {
			"app.js": {"inputs": {
				"node_modules/chart.js/dist/chart.js": {},
				"node_modules/@kurkle/color/dist/color.esm.js": {}
			}}
		}
	}`
	metafilePath := t.TempDir() + "/browser-metafile.json"
	if err := os.WriteFile(metafilePath, []byte(metafile), 0o644); err != nil {
		t.Fatalf("write metafile fixture: %v", err)
	}

	components, err := discoverBrowserComponents(metafilePath)
	if err != nil {
		t.Fatalf("discoverBrowserComponents() error = %v", err)
	}
	byName := map[string]component{}
	for _, component := range components {
		byName[component.Name] = component
	}
	if _, ok := byName["unused"]; ok {
		t.Fatal("unused browser package was discovered")
	}
	if _, ok := byName["animate.css"]; !ok {
		t.Fatal("CSS-shipped animate.css was not discovered")
	}
	if got := byName["chart.js"].Dependencies; len(got) != 1 || got[0] != "@kurkle/color" {
		t.Fatalf("chart.js dependencies = %v, want @kurkle/color", got)
	}
}

func TestCommittedNoticesMatchManifest(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("change to repository root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	manifest, err := readManifest("internal/licences/manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	generated := t.TempDir() + "/open-source-licences.html"
	if err := writeNotices(generated, manifest.Components); err != nil {
		t.Fatalf("generate notices: %v", err)
	}
	want, err := os.ReadFile("internal/assets/views/open-source-licences.html")
	if err != nil {
		t.Fatalf("read committed notices: %v", err)
	}
	got, err := os.ReadFile(generated)
	if err != nil {
		t.Fatalf("read generated notices: %v", err)
	}
	if string(got) != string(want) {
		t.Fatal("committed notices are stale; run go run ./cmd/generate-licences")
	}
}

func TestNewSBOMContainsComponentMetadataAndDependencyGraph(t *testing.T) {
	components := []component{
		{
			Ecosystem: "npm",
			Name:      "dependency",
			Version:   "1.0.0",
			PURL:      "pkg:npm/dependency@1.0.0",
			SourceURL: "https://example.test/dependency",
			Targets:   []string{"browser"},
			Integrity: "sha512-YQ==",
			Licence:   licence{ID: "MIT"},
		},
		{
			Ecosystem:    "npm",
			Name:         "root",
			Version:      "1.0.0",
			PURL:         "pkg:npm/root@1.0.0",
			SourceURL:    "https://example.test/root",
			Targets:      []string{"browser"},
			Dependencies: []string{"dependency"},
			Licence:      licence{ID: "Apache-2.0"},
		},
	}

	document := newSBOM("2.0.0", components)
	if document.BOMFormat != "CycloneDX" || document.SpecVersion != "1.7" {
		t.Fatalf("SBOM format = %s %s, want CycloneDX 1.7", document.BOMFormat, document.SpecVersion)
	}
	if document.Metadata.Component.Version != "2.0.0" {
		t.Fatalf("application version = %q, want 2.0.0", document.Metadata.Component.Version)
	}
	if len(document.Components) != 2 || document.Components[1].PURL != "pkg:npm/root@1.0.0" {
		t.Fatalf("SBOM components = %#v", document.Components)
	}
	if got := document.Dependencies[2].DependsOn; len(got) != 1 || got[0] != "pkg:npm/dependency@1.0.0" {
		t.Fatalf("root dependencies = %v", got)
	}
	if got := document.Components[0].Hashes[0].Content; got != "61" {
		t.Fatalf("decoded hash = %q, want 61", got)
	}
}
