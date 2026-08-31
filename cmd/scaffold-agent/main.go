package main

import (
	"os"

	"github.com/hkx5414375/scaffold-agent/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
