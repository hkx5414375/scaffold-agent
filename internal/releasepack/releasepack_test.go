package releasepack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestReleaseBuildUsesExplicitProvenanceWithoutAmbientVCSState(t *testing.T) {
	options := Options{
		Version:   "1.0.0",
		Commit:    "0123456789abcdef0123456789abcdef01234567",
		BuildDate: time.Date(2026, time.September, 2, 1, 2, 3, 0, time.UTC),
	}
	arguments := releaseBuildArguments(options, "scaffold-agent")
	joined := strings.Join(arguments, " ")
	for _, fragment := range []string{
		"-buildvcs=false",
		"internal/version.Version=1.0.0",
		"internal/version.Commit=" + options.Commit,
		"internal/version.BuildDate=2026-09-02T01:02:03Z",
	} {
		if !strings.Contains(joined, fragment) {
			t.Errorf("release build arguments do not contain %q: %s", fragment, joined)
		}
	}
	if strings.Contains(joined, "-buildvcs=true") {
		t.Fatalf("release build arguments embed ambient VCS state: %s", joined)
	}
}

func TestArchivesAreDeterministicAndContainOrderedFiles(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, time.September, 2, 1, 2, 3, 0, time.UTC)
	entries := []archiveEntry{
		{Name: "scaffold-agent", Content: []byte("binary"), Executable: true},
		{Name: "LICENSE", Content: []byte("license")},
	}
	root := t.TempDir()
	for _, extension := range []string{"tar.gz", "zip"} {
		first := filepath.Join(root, "first."+extension)
		second := filepath.Join(root, "second."+extension)
		var err error
		if extension == "zip" {
			err = writeZIP(first, timestamp, entries)
			if err == nil {
				err = writeZIP(second, timestamp, entries)
			}
		} else {
			err = writeTarGZIP(first, timestamp, entries)
			if err == nil {
				err = writeTarGZIP(second, timestamp, entries)
			}
		}
		if err != nil {
			t.Fatalf("write %s: %v", extension, err)
		}
		firstContent, _ := os.ReadFile(first)
		secondContent, _ := os.ReadFile(second)
		if !reflect.DeepEqual(firstContent, secondContent) {
			t.Fatalf("%s archives differ", extension)
		}
	}
	if got := tarNames(t, filepath.Join(root, "first.tar.gz")); !reflect.DeepEqual(got, []string{"scaffold-agent", "LICENSE"}) {
		t.Fatalf("tar names = %v", got)
	}
	if got := zipNames(t, filepath.Join(root, "first.zip")); !reflect.DeepEqual(got, []string{"scaffold-agent", "LICENSE"}) {
		t.Fatalf("zip names = %v", got)
	}
}

func TestNormalizeRejectsOutputOutsideRepository(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := normalize(Options{
		Root:      root,
		Output:    filepath.Join(filepath.Dir(root), "outside"),
		Version:   "1.0.0",
		Commit:    "abcdef0",
		BuildDate: time.Now(),
	})
	if err == nil {
		t.Fatal("normalize() error = nil, want root containment error")
	}
}

func TestWriteSBOMIncludesDeterministicCycloneDXSerialNumber(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.27.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "dist")
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	options := Options{
		Root:      root,
		Output:    output,
		Version:   "1.0.1",
		Commit:    "0123456789abcdef0123456789abcdef01234567",
		BuildDate: time.Date(2026, time.September, 2, 1, 2, 3, 0, time.UTC),
	}
	file, err := writeSBOM(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(output, file.Name))
	if err != nil {
		t.Fatal(err)
	}
	var bom struct {
		BOMFormat    string `json:"bomFormat"`
		SerialNumber string `json:"serialNumber"`
		SpecVersion  string `json:"specVersion"`
	}
	if err := json.Unmarshal(content, &bom); err != nil {
		t.Fatal(err)
	}
	if bom.BOMFormat != "CycloneDX" || bom.SpecVersion != sbomSpecVersion {
		t.Fatalf("unexpected CycloneDX identity: %+v", bom)
	}
	if want := cyclonedxSerialNumber(options.Version, options.Commit); bom.SerialNumber != want {
		t.Fatalf("serialNumber = %q, want %q", bom.SerialNumber, want)
	}
	if !regexp.MustCompile(`^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-8[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(bom.SerialNumber) {
		t.Fatalf("serialNumber is not a canonical UUIDv8 URN: %q", bom.SerialNumber)
	}
}

func tarNames(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	var names []string
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
	}
	return names
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	return names
}
