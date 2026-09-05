package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/luisgustavo/coleoptera_identifier/cli/internal/i18n"
)

func TestRootHelpUsesEnglishCatalog(t *testing.T) {
	command := newRootCommand(i18n.NewLocalizer(i18n.EnglishUS))
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetArgs([]string{"--help"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Identify beetles directly from the terminal") {
		t.Fatalf("English help was not rendered:\n%s", output.String())
	}
}

func TestRootHelpUsesPortugueseCatalog(t *testing.T) {
	command := newRootCommand(i18n.NewLocalizer(i18n.PortugueseBR))
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetArgs([]string{"--help"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Identifique besouros diretamente do terminal") {
		t.Fatalf("Portuguese help was not rendered:\n%s", output.String())
	}
}

func TestCommandErrorsRemainInEnglish(t *testing.T) {
	command := newRootCommand(i18n.NewLocalizer(i18n.PortugueseBR))
	command.SetArgs([]string{"identify", "beetle.jpg", "--output", "yaml"})

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("expected an English error, got %v", err)
	}
}
