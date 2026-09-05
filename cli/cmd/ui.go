package cmd

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/i18n"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newUICommand(opts *options, localizer *i18n.Localizer) *cobra.Command {
	return &cobra.Command{
		Use:   "ui",
		Short: localizer.Text(i18n.CLIUIShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUI(cmd.Context(), opts, localizer)
		},
	}
}

func runUI(ctx context.Context, opts *options, localizer *i18n.Localizer) error {
	client, err := api.NewClient(opts.apiURL, opts.timeout)
	if err != nil {
		return err
	}

	program := tea.NewProgram(tui.New(client, localizer))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("could not start the interface: %w", err)
	}
	return nil
}
