package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.2.0"

func main() {
	root := &cobra.Command{
		Use:     "gai",
		Short:   "Gai - AI-native Go web framework CLI",
		Long:    "Gai is a CLI tool for the Gai framework, providing project scaffolding, code generation, database migration, and AI-assisted development.",
		Version: version,
	}

	root.AddCommand(
		newCmd(),
		serveCmd(),
		makeCmd(),
		generateCmd(),
		migrateCmd(),
		seedCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
