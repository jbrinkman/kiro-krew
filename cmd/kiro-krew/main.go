package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/jbrinkman/kiro-krew/cmd/kiro-krew/cmd"
	"github.com/jbrinkman/kiro-krew/internal/templates"
)

//go:embed templates
var Templates embed.FS

func main() {
	// Normalize the subcommand to lowercase for case-insensitive matching.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		os.Args[1] = strings.ToLower(os.Args[1])
	}
	cmd.SetTemplates(Templates)
	templates.SetTemplates(Templates)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
