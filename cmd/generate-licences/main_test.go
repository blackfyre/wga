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

func TestValidateManifestRejectsLicenceIDAndExpression(t *testing.T) {
	record := component{
		Ecosystem: "npm",
		Name:      "example",
		Version:   "1.0.0",
		Targets:   []string{"browser"},
		Licence: licence{
			ID:         "MIT",
			Expression: "MIT OR Apache-2.0",
			Text:       "Licence text",
			Handling:   "Include it.",
		},
		SourceEvidence: "https://example.test/licence",
	}

	err := validateManifest(manifest{Version: 1, Components: []component{record}}, []component{record})
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("validateManifest() error = %v, want invalid licence error", err)
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

func TestDiscoverVendoredBrowserPackages(t *testing.T) {
	inputPath := t.TempDir() + "/node_modules/trix/dist/trix.esm.min.js"
	if err := os.MkdirAll(strings.TrimSuffix(inputPath, "/trix.esm.min.js"), 0o755); err != nil {
		t.Fatalf("create vendored fixture directory: %v", err)
	}
	fixture := "/*! @license DOMPurify 3.2.7 | https://github.com/cure53/DOMPurify/blob/3.2.7/LICENSE */"
	if err := os.WriteFile(inputPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write vendored fixture: %v", err)
	}
	components, err := discoverVendoredBrowserPackages(
		map[string]browserInput{inputPath: {}},
		map[string]struct{}{inputPath: {}},
	)
	if err != nil {
		t.Fatalf("discover vendored packages: %v", err)
	}
	if len(components) != 1 || components[0].Parent != "trix" || components[0].Component.Name != "dompurify" || components[0].Component.Version != "3.2.7" || components[0].Component.Integrity != "" || components[0].Component.SourceEvidence != "https://github.com/cure53/DOMPurify/blob/3.2.7/LICENSE" {
		t.Fatalf("vendored components = %#v, want DOMPurify 3.2.7", components)
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
			Direct:       true,
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
	if got := document.Dependencies[0].DependsOn; len(got) != 1 || got[0] != "pkg:npm/root@1.0.0" {
		t.Fatalf("application dependencies = %v, want direct root dependency", got)
	}
	if got := document.Components[0].Hashes[0].Content; got != "61" {
		t.Fatalf("decoded hash = %q, want 61", got)
	}
}

func TestNewSBOMUsesLicenceExpression(t *testing.T) {
	document := newSBOM("1.0.0", []component{{
		Ecosystem: "golang",
		Name:      "example.test/component",
		Version:   "1.0.0",
		PURL:      "pkg:golang/example.test/component@1.0.0",
		Targets:   []string{"binary"},
		Licence:   licence{Expression: "BSD-3-Clause AND Apache-2.0 AND MIT"},
	}})
	if got := document.Components[0].Licences[0].Expression; got != "BSD-3-Clause AND Apache-2.0 AND MIT" {
		t.Fatalf("licence expression = %q", got)
	}
	if document.Components[0].Licences[0].Licence != nil {
		t.Fatal("licence expression must not include a license object")
	}
}
