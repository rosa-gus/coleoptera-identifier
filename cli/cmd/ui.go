package cmd

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ui",
		Short: "Abre a interface interativa",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUI(cmd.Context())
		},
	}
}

func runUI(ctx context.Context) error {
	client, err := api.NewClient(opts.apiURL, opts.timeout)
	if err != nil {
		return err
	}

	program := tea.NewProgram(tui.New(client))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("não foi possível iniciar a interface: %w", err)
	}
	return nil
}
