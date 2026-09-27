package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rosa-gus/coleoptera_identifier/curator/internal/curation"
	"github.com/rosa-gus/coleoptera_identifier/curator/internal/ui"
)

func main() {
	dataDir := flag.String("data-dir", "", "evaluation data directory (contains manifest.csv)")
	observation := flag.String("observation", "", "iNaturalist observation ID to fetch at startup")
	flag.Parse()

	directory := *dataDir
	if directory == "" {
		var err error
		directory, err = findDataDirectory()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if _, err := os.Stat(filepath.Join(directory, "manifest.csv")); err != nil {
		fmt.Fprintf(os.Stderr, "evaluation manifest: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(ui.New(curation.NewClient(), directory, *observation))
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func findDataDirectory() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(directory, "data", "evaluation")
		if _, err := os.Stat(filepath.Join(candidate, "manifest.csv")); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("manifest.csv not found; use --data-dir")
		}
		directory = strings.TrimSpace(parent)
	}
}
