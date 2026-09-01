// Package releasepack builds deterministic cross-platform release archives.
package releasepack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	manifestAPIVersion = "scaffold-agent.io/release-manifest/v1"
	sbomSpecVersion    = "1.6"
)

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
	commitPattern  = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
)

// Target is one supported release platform.
type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Targets is the stable v1 cross-platform release matrix.
var Targets = []Target{
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
}

// Options configures one release package set.
type Options struct {
	Root      string
	Output    string
	Version   string
	Commit    string
	BuildDate time.Time
}

// Report describes the generated release files.
type Report struct {
	APIVersion    string     `json:"api_version"`
	Version       string     `json:"version"`
	Commit        string     `json:"commit"`
	BuildDate     string     `json:"build_date"`
	Targets       []Target   `json:"targets"`
	Artifacts     []Artifact `json:"artifacts"`
	SBOM          File       `json:"sbom"`
	Manifest      File       `json:"manifest"`
	ChecksumsFile string     `json:"checksums_file"`
}

// Artifact is one archive and its target identity.
type Artifact struct {
	File
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// File records release asset integrity metadata.
type File struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Build creates all archives, the SBOM, manifest, and SHA256SUMS file.
func Build(ctx context.Context, options Options) (Report, error) {
	normalized, err := normalize(options)
	if err != nil {
		return Report{}, err
	}
	if err := createEmptyDirectory(normalized.Output); err != nil {
		return Report{}, err
	}
	working, err := os.MkdirTemp("", "scaffold-agent-release-")
	if err != nil {
		return Report{}, fmt.Errorf("create release staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(working) }()

	report := Report{
		APIVersion:    manifestAPIVersion,
		Version:       normalized.Version,
		Commit:        normalized.Commit,
		BuildDate:     normalized.BuildDate.Format(time.RFC3339),
		Targets:       append([]Target(nil), Targets...),
		ChecksumsFile: "SHA256SUMS",
	}
	for _, target := range Targets {
		artifact, buildErr := buildTarget(ctx, normalized, working, target)
		if buildErr != nil {
			return Report{}, buildErr
		}
		report.Artifacts = append(report.Artifacts, artifact)
	}
	report.SBOM, err = writeSBOM(ctx, normalized)
	if err != nil {
		return Report{}, err
	}
	report.Manifest, err = writeManifest(normalized, report)
	if err != nil {
		return Report{}, err
	}
	if err := writeChecksums(normalized.Output, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func normalize(options Options) (Options, error) {
	if !versionPattern.MatchString(options.Version) {
		return Options{}, fmt.Errorf("version %q is not release SemVer without a v prefix", options.Version)
	}
	options.Commit = strings.ToLower(strings.TrimSpace(options.Commit))
	if !commitPattern.MatchString(options.Commit) {
		return Options{}, fmt.Errorf("commit %q is not a hexadecimal Git revision", options.Commit)
	}
	if options.BuildDate.IsZero() {
		return Options{}, errors.New("build date is required")
	}
	options.BuildDate = options.BuildDate.UTC().Truncate(time.Second)
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return Options{}, fmt.Errorf("resolve repository root: %w", err)
	}
	if info, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr != nil || !info.Mode().IsRegular() {
		return Options{}, errors.New("repository root has no regular go.mod")
	}
	output, err := filepath.Abs(options.Output)
	if err != nil {
		return Options{}, fmt.Errorf("resolve output directory: %w", err)
	}
	if output == root || !isWithin(root, output) {
		return Options{}, errors.New("release output must be a child of the repository root")
	}
	options.Root = root
	options.Output = output
	return options, nil
}

func createEmptyDirectory(path string) error {
	entries, err := os.ReadDir(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create release output: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("inspect release output: %w", err)
	case len(entries) != 0:
		return errors.New("release output directory must be empty")
	default:
		return nil
	}
}

func buildTarget(ctx context.Context, options Options, working string, target Target) (Artifact, error) {
	binaryName := "scaffold-agent"
	if target.OS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(working, target.OS+"_"+target.Arch, binaryName)
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return Artifact{}, fmt.Errorf("create target staging directory: %w", err)
	}
	command := exec.CommandContext(
		ctx,
		"go",
		releaseBuildArguments(options, binaryPath)...,
	)
	command.Dir = options.Root
	command.Env = releaseEnvironment(os.Environ(), target)
	if output, err := command.CombinedOutput(); err != nil {
		return Artifact{}, fmt.Errorf("build %s/%s: %w: %s", target.OS, target.Arch, err, strings.TrimSpace(string(output)))
	}

	archiveBase := fmt.Sprintf("scaffold-agent_%s_%s_%s", options.Version, target.OS, target.Arch)
	archiveName := archiveBase + ".tar.gz"
	if target.OS == "windows" {
		archiveName = archiveBase + ".zip"
	}
	archivePath := filepath.Join(options.Output, archiveName)
	entries, err := releaseEntries(options.Root, binaryPath, binaryName)
	if err != nil {
		return Artifact{}, err
	}
	if target.OS == "windows" {
		err = writeZIP(archivePath, options.BuildDate, entries)
	} else {
		err = writeTarGZIP(archivePath, options.BuildDate, entries)
	}
	if err != nil {
		return Artifact{}, fmt.Errorf("archive %s/%s: %w", target.OS, target.Arch, err)
	}
	file, err := describeFile(archivePath)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{File: file, OS: target.OS, Arch: target.Arch}, nil
}

func releaseBuildArguments(options Options, binaryPath string) []string {
	linkerFlags := strings.Join([]string{
		"-s", "-w",
		"-X", "github.com/hkx5414375/scaffold-agent/internal/version.Version=" + options.Version,
		"-X", "github.com/hkx5414375/scaffold-agent/internal/version.Commit=" + options.Commit,
		"-X", "github.com/hkx5414375/scaffold-agent/internal/version.BuildDate=" + options.BuildDate.Format(time.RFC3339),
	}, " ")
	return []string{
		"build",
		"-trimpath",
		// The release output is intentionally inside the repository. Embedding
		// ambient VCS dirty state would make the first target differ from later
		// targets and from a second reproducibility build. Provenance is supplied
		// explicitly through the immutable linker values above and the manifest.
		"-buildvcs=false",
		"-ldflags",
		linkerFlags,
		"-o",
		binaryPath,
		"./cmd/scaffold-agent",
	}
}

type archiveEntry struct {
	Name       string
	Content    []byte
	Executable bool
}

func releaseEntries(root, binaryPath, binaryName string) ([]archiveEntry, error) {
	paths := []struct {
		Source string
		Name   string
	}{
		{Source: binaryPath, Name: binaryName},
		{Source: filepath.Join(root, "LICENSE"), Name: "LICENSE"},
		{Source: filepath.Join(root, "README.md"), Name: "README.md"},
		{Source: filepath.Join(root, "README.zh-CN.md"), Name: "README.zh-CN.md"},
		{Source: filepath.Join(root, "SECURITY.md"), Name: "SECURITY.md"},
		{Source: filepath.Join(root, "docs", "installation.md"), Name: "docs/installation.md"},
		{Source: filepath.Join(root, "docs", "installation.zh-CN.md"), Name: "docs/installation.zh-CN.md"},
		{Source: filepath.Join(root, "docs", "upgrade-policy.md"), Name: "docs/upgrade-policy.md"},
		{Source: filepath.Join(root, "docs", "upgrade-policy.zh-CN.md"), Name: "docs/upgrade-policy.zh-CN.md"},
	}
	entries := make([]archiveEntry, 0, len(paths))
	for index, path := range paths {
		content, err := os.ReadFile(path.Source)
		if err != nil {
			return nil, fmt.Errorf("read release file %s: %w", path.Name, err)
		}
		entries = append(entries, archiveEntry{Name: path.Name, Content: content, Executable: index == 0})
	}
	return entries, nil
}

func writeTarGZIP(path string, timestamp time.Time, entries []archiveEntry) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(file)
	gzipWriter.Header.ModTime = timestamp
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	writeErr := error(nil)
	for _, entry := range entries {
		mode := int64(0o644)
		if entry.Executable {
			mode = 0o755
		}
		header := &tar.Header{
			Name:    entry.Name,
			Mode:    mode,
			Size:    int64(len(entry.Content)),
			ModTime: timestamp,
			Format:  tar.FormatPAX,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			writeErr = err
			break
		}
		if _, err := tarWriter.Write(entry.Content); err != nil {
			writeErr = err
			break
		}
	}
	return closeWriters(writeErr, tarWriter.Close, gzipWriter.Close, file.Close)
}

func writeZIP(path string, timestamp time.Time, entries []archiveEntry) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	zipWriter := zip.NewWriter(file)
	writeErr := error(nil)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Name, Method: zip.Deflate}
		header.SetModTime(timestamp)
		if entry.Executable {
			header.SetMode(0o755)
		} else {
			header.SetMode(0o644)
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			writeErr = err
			break
		}
		if _, err := writer.Write(entry.Content); err != nil {
			writeErr = err
			break
		}
	}
	return closeWriters(writeErr, zipWriter.Close, file.Close)
}

func closeWriters(initial error, closers ...func() error) error {
	result := initial
	for _, closeWriter := range closers {
		if err := closeWriter(); result == nil && err != nil {
			result = err
		}
	}
	return result
}

type goModule struct {
	Path    string    `json:"Path"`
	Version string    `json:"Version"`
	Sum     string    `json:"Sum"`
	Main    bool      `json:"Main"`
	Replace *goModule `json:"Replace"`
}

type cyclonedxBOM struct {
	BOMFormat   string               `json:"bomFormat"`
	SpecVersion string               `json:"specVersion"`
	Version     int                  `json:"version"`
	Metadata    cyclonedxMetadata    `json:"metadata"`
	Components  []cyclonedxComponent `json:"components"`
}

type cyclonedxMetadata struct {
	Timestamp string             `json:"timestamp"`
	Component cyclonedxComponent `json:"component"`
}

type cyclonedxComponent struct {
	Type       string              `json:"type"`
	BOMRef     string              `json:"bom-ref"`
	Name       string              `json:"name"`
	Version    string              `json:"version,omitempty"`
	PURL       string              `json:"purl,omitempty"`
	Licenses   []cyclonedxLicense  `json:"licenses,omitempty"`
	Properties []cyclonedxProperty `json:"properties,omitempty"`
}

type cyclonedxLicense struct {
	License struct {
		ID string `json:"id"`
	} `json:"license"`
}

type cyclonedxProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func writeSBOM(ctx context.Context, options Options) (File, error) {
	command := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	command.Dir = options.Root
	output, err := command.Output()
	if err != nil {
		return File{}, fmt.Errorf("list Go modules for SBOM: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	modules := make([]goModule, 0)
	for decoder.More() {
		var module goModule
		if err := decoder.Decode(&module); err != nil {
			return File{}, fmt.Errorf("decode Go module for SBOM: %w", err)
		}
		if !module.Main {
			modules = append(modules, module)
		}
	}
	sort.Slice(modules, func(left, right int) bool { return modules[left].Path < modules[right].Path })
	mainComponent := cyclonedxComponent{
		Type:    "application",
		BOMRef:  "pkg:golang/github.com/hkx5414375/scaffold-agent@" + options.Version,
		Name:    "github.com/hkx5414375/scaffold-agent",
		Version: options.Version,
		PURL:    "pkg:golang/github.com/hkx5414375/scaffold-agent@" + options.Version,
	}
	license := cyclonedxLicense{}
	license.License.ID = "Apache-2.0"
	mainComponent.Licenses = []cyclonedxLicense{license}
	bom := cyclonedxBOM{
		BOMFormat:   "CycloneDX",
		SpecVersion: sbomSpecVersion,
		Version:     1,
		Metadata: cyclonedxMetadata{
			Timestamp: options.BuildDate.Format(time.RFC3339),
			Component: mainComponent,
		},
	}
	for _, module := range modules {
		resolved := module
		properties := make([]cyclonedxProperty, 0, 2)
		if module.Sum != "" {
			properties = append(properties, cyclonedxProperty{Name: "go:module:sum", Value: module.Sum})
		}
		if module.Replace != nil {
			resolved = *module.Replace
			properties = append(properties, cyclonedxProperty{Name: "go:module:replaces", Value: module.Path + "@" + module.Version})
		}
		version := resolved.Version
		if version == "" {
			version = "unknown"
		}
		purl := "pkg:golang/" + resolved.Path + "@" + version
		bom.Components = append(bom.Components, cyclonedxComponent{
			Type:       "library",
			BOMRef:     purl,
			Name:       resolved.Path,
			Version:    version,
			PURL:       purl,
			Properties: properties,
		})
	}
	name := fmt.Sprintf("scaffold-agent_%s_sbom.cdx.json", options.Version)
	path := filepath.Join(options.Output, name)
	if err := writeJSONFile(path, bom); err != nil {
		return File{}, fmt.Errorf("write SBOM: %w", err)
	}
	return describeFile(path)
}

func writeManifest(options Options, report Report) (File, error) {
	name := fmt.Sprintf("scaffold-agent_%s_release-manifest.json", options.Version)
	path := filepath.Join(options.Output, name)
	manifest := struct {
		APIVersion    string     `json:"api_version"`
		Version       string     `json:"version"`
		Commit        string     `json:"commit"`
		BuildDate     string     `json:"build_date"`
		Targets       []Target   `json:"targets"`
		Artifacts     []Artifact `json:"artifacts"`
		SBOM          File       `json:"sbom"`
		ChecksumsFile string     `json:"checksums_file"`
	}{
		APIVersion:    report.APIVersion,
		Version:       report.Version,
		Commit:        report.Commit,
		BuildDate:     report.BuildDate,
		Targets:       report.Targets,
		Artifacts:     report.Artifacts,
		SBOM:          report.SBOM,
		ChecksumsFile: "SHA256SUMS",
	}
	if err := writeJSONFile(path, manifest); err != nil {
		return File{}, fmt.Errorf("write release manifest: %w", err)
	}
	return describeFile(path)
}

func writeJSONFile(path string, value any) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	writeErr := encoder.Encode(value)
	return closeWriters(writeErr, file.Close)
}

func writeChecksums(output string, report Report) error {
	files := make([]File, 0, len(report.Artifacts)+2)
	for _, artifact := range report.Artifacts {
		files = append(files, artifact.File)
	}
	files = append(files, report.SBOM, report.Manifest)
	sort.Slice(files, func(left, right int) bool { return files[left].Name < files[right].Name })
	var content strings.Builder
	for _, file := range files {
		_, _ = fmt.Fprintf(&content, "%s  %s\n", file.SHA256, file.Name)
	}
	path := filepath.Join(output, report.ChecksumsFile)
	if err := os.WriteFile(path, []byte(content.String()), 0o644); err != nil {
		return fmt.Errorf("write checksums: %w", err)
	}
	return nil
}

func describeFile(path string) (File, error) {
	file, err := os.Open(path)
	if err != nil {
		return File{}, fmt.Errorf("open release file: %w", err)
	}
	hash := sha256.New()
	size, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return File{}, fmt.Errorf("hash release file: %w", copyErr)
	}
	if closeErr != nil {
		return File{}, fmt.Errorf("close release file: %w", closeErr)
	}
	return File{Name: filepath.Base(path), Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func releaseEnvironment(current []string, target Target) []string {
	filtered := make([]string, 0, len(current)+4)
	for _, value := range current {
		name, _, _ := strings.Cut(value, "=")
		switch strings.ToUpper(name) {
		case "CGO_ENABLED", "GOOS", "GOARCH", "SOURCE_DATE_EPOCH":
			continue
		default:
			filtered = append(filtered, value)
		}
	}
	return append(
		filtered,
		"CGO_ENABLED=0",
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
	)
}

func isWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == "" {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
