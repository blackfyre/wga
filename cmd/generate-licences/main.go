// Command generate-licences generates WGA's third-party licence notices and SBOM.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const applicationModule = "github.com/blackfyre/wga"

type manifest struct {
	Version    int         `json:"version"`
	Components []component `json:"components"`
}

type component struct {
	Ecosystem      string   `json:"ecosystem"`
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	SourceURL      string   `json:"source_url"`
	PURL           string   `json:"purl"`
	Integrity      string   `json:"integrity,omitempty"`
	Targets        []string `json:"targets"`
	Dependencies   []string `json:"dependencies"`
	Licence        licence  `json:"licence"`
	SourceEvidence string   `json:"source_evidence"`
	Direct         bool     `json:"-"`
	licenceTextURL string
}

type licence struct {
	ID         string `json:"id,omitempty"`
	Expression string `json:"expression,omitempty"`
	Text       string `json:"text"`
	Notice     string `json:"notice"`
	Handling   string `json:"handling"`
}

func (licence licence) Label() string {
	if licence.Expression != "" {
		return licence.Expression
	}
	return licence.ID
}

type goModule struct {
	Path    string
	Version string
	Sum     string
	Dir     string
}

type goPackage struct {
	ImportPath string
	Imports    []string
	Standard   bool
	Module     *goModule
}

type browserMetafile struct {
	Inputs  map[string]browserInput  `json:"inputs"`
	Outputs map[string]browserOutput `json:"outputs"`
}

type browserInput struct {
	Imports []browserImport `json:"imports"`
}

type browserOutput struct {
	Inputs map[string]json.RawMessage `json:"inputs"`
}

type browserImport struct {
	Path string `json:"path"`
}

func main() {
	var manifestPath string
	var goPackagePath string
	var metafilePath string
	var noticesPath string
	var sbomPath string
	var applicationVersion string
	var bootstrap bool

	flag.StringVar(&manifestPath, "manifest", "internal/licences/manifest.json", "reviewed licence manifest")
	flag.StringVar(&goPackagePath, "go-package", "./cmd/wga", "Go package to inspect")
	flag.StringVar(&metafilePath, "browser-metafile", "dist/browser-metafile.json", "Bun build metafile")
	flag.StringVar(&noticesPath, "notices-output", "internal/assets/views/open-source-licences.html", "generated notices HTML")
	flag.StringVar(&sbomPath, "sbom-output", "dist/wga.cdx.json", "generated CycloneDX JSON")
	flag.StringVar(&applicationVersion, "application-version", "1.0.0", "WGA application version for the SBOM")
	flag.BoolVar(&bootstrap, "bootstrap", false, "create a manifest from local dependency licence files for review")
	flag.Parse()

	discovered, err := discoverComponents(goPackagePath, metafilePath)
	if err != nil {
		fatal(err)
	}

	if bootstrap {
		if err := writeManifest(manifestPath, bootstrapManifest(discovered)); err != nil {
			fatal(err)
		}
		return
	}

	loaded, err := readManifest(manifestPath)
	if err != nil {
		fatal(err)
	}
	if err := validateManifest(loaded, discovered); err != nil {
		fatal(err)
	}
	if err := writeNotices(noticesPath, loaded.Components); err != nil {
		fatal(err)
	}
	if err := writeSBOM(sbomPath, applicationVersion, mergeDiscovery(loaded.Components, discovered)); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "generate-licences:", err)
	os.Exit(1)
}

func discoverComponents(goPackagePath string, metafilePath string) ([]component, error) {
	goComponents, err := discoverGoComponents(goPackagePath)
	if err != nil {
		return nil, err
	}
	browserComponents, err := discoverBrowserComponents(metafilePath)
	if err != nil {
		return nil, err
	}
	components := append(goComponents, browserComponents...)
	sortComponents(components)
	return components, nil
}

func discoverGoComponents(packagePath string) ([]component, error) {
	command := exec.Command("go", "list", "-deps", "-json", packagePath)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("run go list: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	packages := map[string]goPackage{}
	applicationPackages := map[string]goPackage{}
	modules := map[string]goModule{}
	for {
		var pkg goPackage
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		if pkg.Standard || pkg.Module == nil {
			continue
		}
		if pkg.Module.Path == applicationModule {
			applicationPackages[pkg.ImportPath] = pkg
			continue
		}
		packages[pkg.ImportPath] = pkg
		modules[pkg.Module.Path] = *pkg.Module
	}

	edges := map[string]map[string]struct{}{}
	directModules := map[string]struct{}{}
	for _, pkg := range packages {
		for _, imported := range pkg.Imports {
			dependency, ok := packages[imported]
			if !ok || dependency.Module.Path == pkg.Module.Path {
				continue
			}
			if edges[pkg.Module.Path] == nil {
				edges[pkg.Module.Path] = map[string]struct{}{}
			}
			edges[pkg.Module.Path][dependency.Module.Path] = struct{}{}
		}
	}
	for _, pkg := range applicationPackages {
		for _, imported := range pkg.Imports {
			if dependency, ok := packages[imported]; ok {
				directModules[dependency.Module.Path] = struct{}{}
			}
		}
	}

	components := make([]component, 0, len(modules))
	for _, module := range modules {
		components = append(components, component{
			Ecosystem:      "golang",
			Name:           module.Path,
			Version:        module.Version,
			SourceURL:      "https://pkg.go.dev/" + module.Path + "@" + module.Version,
			PURL:           "pkg:golang/" + module.Path + "@" + module.Version,
			Integrity:      module.Sum,
			Targets:        []string{"binary"},
			Dependencies:   sortedSet(edges[module.Path]),
			SourceEvidence: "https://pkg.go.dev/" + module.Path + "@" + module.Version,
			Direct:         contains(directModules, module.Path),
		})
	}
	sortComponents(components)
	return components, nil
}

func discoverBrowserComponents(metafilePath string) ([]component, error) {
	content, err := os.ReadFile(metafilePath)
	if err != nil {
		return nil, fmt.Errorf("read browser metafile: %w", err)
	}
	var metafile browserMetafile
	if err := json.Unmarshal(content, &metafile); err != nil {
		return nil, fmt.Errorf("decode browser metafile: %w", err)
	}

	packageNames := map[string]struct{}{}
	directPackages := map[string]struct{}{}
	emittedInputs := map[string]struct{}{}
	for _, output := range metafile.Outputs {
		for input := range output.Inputs {
			emittedInputs[input] = struct{}{}
			if name := nodeModuleName(input); name != "" {
				packageNames[name] = struct{}{}
			}
		}
	}
	cssPackages, err := discoverCSSPackages("resources/css/style.pcss")
	if err != nil {
		return nil, err
	}
	for name := range cssPackages {
		packageNames[name] = struct{}{}
		directPackages[name] = struct{}{}
	}
	lock, err := readBunLock("bun.lock")
	if err != nil {
		return nil, err
	}

	edges := map[string]map[string]struct{}{}
	for inputPath, input := range metafile.Inputs {
		from := nodeModuleName(inputPath)
		if from == "" {
			for _, imported := range input.Imports {
				to := nodeModuleName(imported.Path)
				if to == "" {
					to = packageNameFromSpecifier(imported.Path)
				}
				if _, ok := packageNames[to]; ok {
					directPackages[to] = struct{}{}
				}
			}
		}
		if _, ok := packageNames[from]; !ok {
			continue
		}
		for _, imported := range input.Imports {
			to := nodeModuleName(imported.Path)
			if to == "" {
				to = packageNameFromSpecifier(imported.Path)
			}
			if to != "" && to != from {
				if _, ok := packageNames[to]; ok {
					if edges[from] == nil {
						edges[from] = map[string]struct{}{}
					}
					edges[from][to] = struct{}{}
				}
			}
		}
	}

	bundledComponents, err := discoverVendoredBrowserPackages(metafile.Inputs, emittedInputs)
	if err != nil {
		return nil, err
	}
	for _, bundled := range bundledComponents {
		if _, exists := packageNames[bundled.Component.Name]; exists {
			continue
		}
		if edges[bundled.Parent] == nil {
			edges[bundled.Parent] = map[string]struct{}{}
		}
		edges[bundled.Parent][bundled.Component.Name] = struct{}{}
	}
	components := make([]component, 0, len(packageNames)+len(bundledComponents))
	for name := range packageNames {
		resolved, ok := lock[name]
		if !ok {
			return nil, fmt.Errorf("browser package %q is absent from bun.lock", name)
		}
		components = append(components, component{
			Ecosystem:      "npm",
			Name:           name,
			Version:        resolved.version,
			SourceURL:      "https://www.npmjs.com/package/" + name + "/v/" + resolved.version,
			PURL:           "pkg:npm/" + strings.ReplaceAll(name, "@", "%40") + "@" + resolved.version,
			Integrity:      resolved.integrity,
			Targets:        []string{"browser"},
			Dependencies:   sortedSet(edges[name]),
			SourceEvidence: "https://www.npmjs.com/package/" + name + "/v/" + resolved.version,
			Direct:         contains(directPackages, name),
		})
	}
	for _, bundled := range bundledComponents {
		if _, exists := packageNames[bundled.Component.Name]; !exists {
			components = append(components, bundled.Component)
		}
	}
	sortComponents(components)
	return components, nil
}

type vendoredBrowserPackage struct {
	ProductPattern       *regexp.Regexp
	LicenceSourcePattern *regexp.Regexp
}

var vendoredBrowserPackages = []vendoredBrowserPackage{
	{
		ProductPattern:       regexp.MustCompile(`(DOMPurify) ([0-9]+\.[0-9]+\.[0-9]+)`),
		LicenceSourcePattern: regexp.MustCompile(`(?:https://)?github\.com/cure53/DOMPurify/blob/([0-9]+\.[0-9]+\.[0-9]+)/LICENSE`),
	},
}

type vendoredComponent struct {
	Parent    string
	Component component
}

func discoverVendoredBrowserPackages(inputs map[string]browserInput, emittedInputs map[string]struct{}) ([]vendoredComponent, error) {
	components := []vendoredComponent{}
	for _, bundled := range vendoredBrowserPackages {
		for inputPath := range inputs {
			parent := nodeModuleName(inputPath)
			if _, emitted := emittedInputs[inputPath]; !emitted || parent == "" {
				continue
			}
			content, err := os.ReadFile(inputPath)
			if err != nil {
				return nil, fmt.Errorf("read emitted package input %q: %w", inputPath, err)
			}
			match := bundled.ProductPattern.FindSubmatch(content)
			if len(match) != 3 {
				continue
			}
			name := strings.ToLower(string(match[1]))
			version := string(match[2])
			licenceSource := bundled.LicenceSourcePattern.FindSubmatch(content)
			if len(licenceSource) != 2 || string(licenceSource[1]) != version {
				continue
			}
			evidenceURL := string(licenceSource[0])
			if !strings.HasPrefix(evidenceURL, "https://") {
				evidenceURL = "https://" + evidenceURL
			}
			licenceTextURL := strings.Replace(evidenceURL, "https://github.com/cure53/DOMPurify/blob/", "https://raw.githubusercontent.com/cure53/DOMPurify/", 1)
			components = append(components, vendoredComponent{
				Parent: parent,
				Component: component{
					Ecosystem:      "npm",
					Name:           name,
					Version:        version,
					SourceURL:      "https://www.npmjs.com/package/" + name + "/v/" + version,
					PURL:           "pkg:npm/" + name + "@" + version,
					Targets:        []string{"browser"},
					Dependencies:   []string{},
					SourceEvidence: evidenceURL,
					licenceTextURL: licenceTextURL,
				},
			})
		}
	}
	return components, nil
}

var cssPackagePattern = regexp.MustCompile(`(?:@import\s+(?:url\()?|@plugin\s+)"?([^"'\s;)]+)`)

func discoverCSSPackages(path string) (map[string]struct{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CSS source: %w", err)
	}
	packages := map[string]struct{}{}
	for _, match := range cssPackagePattern.FindAllStringSubmatch(string(content), -1) {
		name := nodeModuleName(match[1])
		if name == "" {
			name = packageNameFromSpecifier(match[1])
		}
		if name != "" {
			packages[name] = struct{}{}
		}
	}
	return packages, nil
}

type lockPackage struct {
	version   string
	integrity string
}

func readBunLock(path string) (map[string]lockPackage, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bun lockfile: %w", err)
	}
	packages := map[string]lockPackage{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, `": ["`, 2)
		if len(parts) != 2 || !strings.HasPrefix(line, `"`) || !strings.Contains(parts[1], `sha512-`) {
			continue
		}
		name := strings.TrimPrefix(parts[0], `"`)
		resolvedVersion, _, ok := strings.Cut(parts[1], `",`)
		if !ok {
			continue
		}
		resolved := strings.TrimPrefix(resolvedVersion, name+"@")
		integrityStart := strings.Index(parts[1], "sha512-")
		integrityEnd := strings.Index(parts[1][integrityStart:], `"`)
		if resolved == resolvedVersion || integrityEnd == -1 {
			continue
		}
		integrity := parts[1][integrityStart : integrityStart+integrityEnd]
		packages[name] = lockPackage{version: resolved, integrity: integrity}
	}
	return packages, nil
}

func nodeModuleName(path string) string {
	path = filepath.ToSlash(path)
	marker := "/node_modules/"
	index := strings.Index(path, marker)
	if index == -1 {
		if !strings.HasPrefix(path, "node_modules/") {
			return ""
		}
		index = -1
	}
	remainder := strings.TrimPrefix(path[index+1:], "node_modules/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	if strings.HasPrefix(parts[0], "@") && len(parts) > 1 {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

func packageNameFromSpecifier(specifier string) string {
	if strings.HasPrefix(specifier, "@") {
		parts := strings.Split(specifier, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return ""
	}
	if strings.HasPrefix(specifier, ".") || strings.Contains(specifier, "://") {
		return ""
	}
	return strings.Split(specifier, "/")[0]
}

func contains(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}

func bootstrapManifest(discovered []component) manifest {
	components := make([]component, len(discovered))
	for index, component := range discovered {
		licenceText, err := readLicenceMaterial(component)
		if err != nil {
			fatal(err)
		}
		notice, _ := readNotice(component)
		component.Licence = licence{
			ID:         licenceID(licenceText),
			Expression: licenceExpression(licenceText),
			Text:       licenceText,
			Notice:     notice,
			Handling:   "Include this licence text and any supplied NOTICE material in distributed notices.",
		}
		components[index] = component
	}
	sortComponents(components)
	return manifest{Version: 1, Components: components}
}

func readLicenceMaterial(component component) (string, error) {
	if component.licenceTextURL != "" {
		response, err := http.Get(component.licenceTextURL)
		if err != nil {
			return "", fmt.Errorf("fetch licence text for %s: %w", componentKey(component), err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return "", fmt.Errorf("fetch licence text for %s: %s", componentKey(component), response.Status)
		}
		content, err := io.ReadAll(response.Body)
		if err != nil {
			return "", fmt.Errorf("read licence text for %s: %w", componentKey(component), err)
		}
		return string(content), nil
	}
	root := componentRoot(component)
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "LICENCE.md", "LICENCE.txt", "COPYING"} {
		path := filepath.Join(root, name)
		content, err := os.ReadFile(path)
		if err == nil && len(bytes.TrimSpace(content)) > 0 {
			return string(content), nil
		}
	}
	return "", fmt.Errorf("find licence text for %s", componentKey(component))
}

func readNotice(component component) (string, error) {
	content, err := os.ReadFile(filepath.Join(componentRoot(component), "NOTICE"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func componentRoot(component component) string {
	if component.Ecosystem == "npm" {
		return filepath.Join("node_modules", filepath.FromSlash(component.Name))
	}
	command := exec.Command("go", "env", "GOMODCACHE")
	moduleCache, err := command.Output()
	if err != nil {
		return ""
	}
	return filepath.Join(strings.TrimSpace(string(moduleCache)), filepath.FromSlash(component.Name)+"@"+component.Version)
}

func licenceID(text string) string {
	if licenceExpression(text) != "" {
		return ""
	}
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "apache license") && strings.Contains(lower, "version 2.0"):
		return "Apache-2.0"
	case strings.Contains(lower, "zero-clause bsd"):
		return "0BSD"
	case strings.Contains(lower, "permission is hereby granted, free of charge"):
		return "MIT"
	case strings.Contains(lower, "redistribution and use in source and binary forms") && strings.Contains(lower, "neither the name"):
		return "BSD-3-Clause"
	case strings.Contains(lower, "redistribution and use in source and binary forms"):
		return "BSD-2-Clause"
	case strings.Contains(lower, "isc license"):
		return "ISC"
	case strings.Contains(lower, "mozilla public license"):
		return "MPL-2.0"
	default:
		return "NOASSERTION"
	}
}

func licenceExpression(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "apache license") && strings.Contains(lower, "mozilla public license") {
		return "Apache-2.0 OR MPL-2.0"
	}
	if strings.Contains(lower, "redistribution and use in source and binary forms") && strings.Contains(lower, "apache license") && strings.Contains(lower, "permission is hereby granted, free of charge") {
		return "BSD-3-Clause AND Apache-2.0 AND MIT"
	}
	return ""
}

func readManifest(path string) (manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var loaded manifest
	if err := json.Unmarshal(content, &loaded); err != nil {
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if loaded.Version != 1 {
		return manifest{}, fmt.Errorf("manifest version = %d, want 1", loaded.Version)
	}
	sortComponents(loaded.Components)
	return loaded, nil
}

func writeManifest(path string, manifest manifest) error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	return writeFile(path, append(content, '\n'))
}

func validateManifest(manifest manifest, discovered []component) error {
	recorded := map[string]component{}
	for _, component := range manifest.Components {
		key := componentKey(component)
		if _, exists := recorded[key]; exists {
			return fmt.Errorf("manifest repeats %s", key)
		}
		if (component.Licence.ID == "" && component.Licence.Expression == "") || (component.Licence.ID != "" && component.Licence.Expression != "") || component.Licence.ID == "NOASSERTION" || strings.TrimSpace(component.Licence.Text) == "" || strings.TrimSpace(component.Licence.Handling) == "" || component.SourceEvidence == "" {
			return fmt.Errorf("manifest has incomplete licence material for %s", key)
		}
		recorded[key] = component
	}
	for _, component := range discovered {
		key := componentKey(component)
		recordedComponent, ok := recorded[key]
		if !ok {
			return fmt.Errorf("shipped component %s is missing from the manifest", key)
		}
		if recordedComponent.Version != component.Version || recordedComponent.SourceURL != component.SourceURL || recordedComponent.PURL != component.PURL || recordedComponent.Integrity != component.Integrity || !slices.Equal(recordedComponent.Targets, component.Targets) || !slices.Equal(recordedComponent.Dependencies, component.Dependencies) {
			return fmt.Errorf("manifest record for %s is stale", key)
		}
	}
	if len(recorded) != len(discovered) {
		return errors.New("manifest contains components that are no longer shipped")
	}
	return nil
}

func mergeDiscovery(components []component, discovered []component) []component {
	direct := map[string]bool{}
	for _, component := range discovered {
		direct[componentKey(component)] = component.Direct
	}
	merged := append([]component(nil), components...)
	for index := range merged {
		merged[index].Direct = direct[componentKey(merged[index])]
	}
	return merged
}

func writeNotices(path string, components []component) error {
	const noticesTemplate = `<section class="container mx-auto px-4 py-8">
  <h1>Open-source licences</h1>
  <p>This application includes the third-party components listed below.</p>
  {{range .}}<article class="my-8" id="{{.Ecosystem}}-{{.Name}}-{{.Version}}">
    <h2>{{.Name}} {{.Version}}</h2>
    <dl><dt>Licence</dt><dd>{{.Licence.Label}}</dd><dt>Source</dt><dd><a href="{{.SourceURL}}">{{.SourceURL}}</a></dd></dl>
    <h3>Licence text</h3><pre class="whitespace-pre-wrap">{{.Licence.Text}}</pre>
    {{if .Licence.Notice}}<h3>NOTICE</h3><pre class="whitespace-pre-wrap">{{.Licence.Notice}}</pre>{{end}}
  </article>{{end}}
</section>
`
	template, err := template.New("notices").Parse(noticesTemplate)
	if err != nil {
		return err
	}
	var output bytes.Buffer
	output.WriteString("<!-- Code generated by cmd/generate-licences; DO NOT EDIT. -->\n")
	if err := template.Execute(&output, components); err != nil {
		return fmt.Errorf("render notices: %w", err)
	}
	return writeFile(path, output.Bytes())
}

type sbom struct {
	BOMFormat    string           `json:"bomFormat"`
	SpecVersion  string           `json:"specVersion"`
	Version      int              `json:"version"`
	Metadata     sbomMetadata     `json:"metadata"`
	Components   []sbomComponent  `json:"components"`
	Dependencies []sbomDependency `json:"dependencies"`
}

type sbomMetadata struct {
	Component sbomComponent `json:"component"`
}

type sbomComponent struct {
	Type         string         `json:"type"`
	BomRef       string         `json:"bom-ref"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	PURL         string         `json:"purl"`
	Licences     []sbomLicence  `json:"licenses,omitempty"`
	ExternalRefs []externalRef  `json:"externalReferences,omitempty"`
	Hashes       []hash         `json:"hashes,omitempty"`
	Properties   []sbomProperty `json:"properties,omitempty"`
}

type sbomLicence struct {
	Licence    *sbomLicenceDetail `json:"license,omitempty"`
	Expression string             `json:"expression,omitempty"`
}

type sbomLicenceDetail struct {
	ID string `json:"id"`
}

type externalRef struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type hash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

type sbomProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type sbomDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

func writeSBOM(path string, applicationVersion string, components []component) error {
	content, err := json.MarshalIndent(newSBOM(applicationVersion, components), "", "  ")
	if err != nil {
		return fmt.Errorf("encode SBOM: %w", err)
	}
	return writeFile(path, append(content, '\n'))
}

func newSBOM(applicationVersion string, components []component) sbom {
	components = append([]component(nil), components...)
	sortComponents(components)
	sbomComponents := make([]sbomComponent, 0, len(components))
	dependencies := make([]sbomDependency, 0, len(components)+1)
	applicationDependencies := make([]string, 0, len(components))
	for _, component := range components {
		ref := bomRef(component)
		if component.Direct {
			applicationDependencies = append(applicationDependencies, ref)
		}
		licence := sbomLicence{Expression: component.Licence.Expression}
		if component.Licence.Expression == "" {
			licence.Licence = &sbomLicenceDetail{ID: component.Licence.ID}
		}
		sbomComponent := sbomComponent{
			Type:         "library",
			BomRef:       ref,
			Name:         component.Name,
			Version:      component.Version,
			PURL:         component.PURL,
			Licences:     []sbomLicence{licence},
			ExternalRefs: []externalRef{{Type: "website", URL: component.SourceURL}},
			Properties:   []sbomProperty{{Name: "wga:distribution-targets", Value: strings.Join(component.Targets, ",")}},
		}
		if checksum := cyclonedxHash(component.Integrity); checksum.Content != "" {
			sbomComponent.Hashes = []hash{checksum}
		}
		sbomComponents = append(sbomComponents, sbomComponent)
		dependsOn := make([]string, 0, len(component.Dependencies))
		for _, dependency := range component.Dependencies {
			for _, candidate := range components {
				if candidate.Name == dependency && candidate.Ecosystem == component.Ecosystem {
					dependsOn = append(dependsOn, bomRef(candidate))
				}
			}
		}
		sort.Strings(dependsOn)
		dependencies = append(dependencies, sbomDependency{Ref: ref, DependsOn: dependsOn})
	}
	sort.Strings(applicationDependencies)
	applicationPURL := "pkg:golang/github.com/blackfyre/wga@" + applicationVersion
	dependencies = append([]sbomDependency{{Ref: applicationPURL, DependsOn: applicationDependencies}}, dependencies...)
	return sbom{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.7",
		Version:     1,
		Metadata: sbomMetadata{Component: sbomComponent{
			Type:    "application",
			BomRef:  applicationPURL,
			Name:    "Web Gallery of Art",
			Version: applicationVersion,
			PURL:    applicationPURL,
		}},
		Components:   sbomComponents,
		Dependencies: dependencies,
	}
}

func cyclonedxHash(integrity string) hash {
	parts := strings.SplitN(integrity, ":", 2)
	if len(parts) == 2 && parts[0] == "h1" {
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err == nil {
			return hash{Alg: "SHA-256", Content: fmt.Sprintf("%x", decoded)}
		}
	}
	if strings.HasPrefix(integrity, "sha512-") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(integrity, "sha512-"))
		if err == nil {
			return hash{Alg: "SHA-512", Content: fmt.Sprintf("%x", decoded)}
		}
	}
	return hash{}
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func componentKey(component component) string {
	return component.Ecosystem + ":" + component.Name
}

func bomRef(component component) string {
	return component.PURL
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortComponents(components []component) {
	sort.Slice(components, func(left, right int) bool {
		return componentKey(components[left]) < componentKey(components[right])
	})
}
