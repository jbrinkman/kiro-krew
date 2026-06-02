package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/jbrinkman/kiro-krew/cmd/kiro-krew/cmd"
	"github.com/jbrinkman/kiro-krew/internal/templates"
)

//go:embed templates
var Templates embed.FS

func main() {
	cmd.SetTemplates(Templates)
	templates.SetTemplates(Templates)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}


