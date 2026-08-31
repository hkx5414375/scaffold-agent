package main

import (
	"os"

	"github.com/hkx5414375/scaffold-agent/internal/app"
)

func main() {
	os.Exit(app.RunIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
