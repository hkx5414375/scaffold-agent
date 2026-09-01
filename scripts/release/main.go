// Command release builds deterministic Scaffold Agent release archives.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hkx5414375/scaffold-agent/internal/releasepack"
)

func main() {
	root := flag.String("root", ".", "repository root")
	output := flag.String("output", "dist", "empty repository-relative output directory")
	version := flag.String("version", "", "release SemVer without v prefix")
	commit := flag.String("commit", "", "release Git commit")
	date := flag.String("date", "", "release build date in RFC3339")
	flag.Parse()
	if flag.NArg() != 0 {
		_, _ = fmt.Fprintln(os.Stderr, "release does not accept positional arguments")
		os.Exit(2)
	}
	buildDate, err := time.Parse(time.RFC3339, *date)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "invalid release date: %v\n", err)
		os.Exit(2)
	}
	report, err := releasepack.Build(context.Background(), releasepack.Options{
		Root:      *root,
		Output:    *output,
		Version:   *version,
		Commit:    *commit,
		BuildDate: buildDate,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "build release: %v\n", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "encode release report: %v\n", err)
		os.Exit(1)
	}
}
