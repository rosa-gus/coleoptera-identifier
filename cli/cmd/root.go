package cmd

import (
	"os"
	"time"

	"github.com/luisgustavo/coleoptera_identifier/cli/internal/config"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/i18n"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

type options struct {
	apiURL  string
	timeout time.Duration
}

func newRootCommand(localizer *i18n.Localizer) *cobra.Command {
	opts := &options{}
	language := string(localizer.Locale())
	rootCommand := &cobra.Command{
		Use:           "coleoptera",
		Short:         localizer.Text(i18n.CLIRootShort),
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUI(cmd.Context(), opts, localizer)
		},
	}

	rootCommand.PersistentFlags().StringVar(
		&opts.apiURL,
		"api-url",
		config.APIURL(),
		localizer.Text(i18n.CLIAPIURLHelp),
	)
	rootCommand.PersistentFlags().DurationVar(
		&opts.timeout,
		"timeout",
		60*time.Second,
		localizer.Text(i18n.CLITimeoutHelp),
	)
	rootCommand.PersistentFlags().StringVar(
		&language,
		"lang",
		language,
		localizer.Text(i18n.CLILanguageHelp),
	)

	rootCommand.AddCommand(newIdentifyCommand(opts, localizer))
	rootCommand.AddCommand(newUICommand(opts, localizer))
	return rootCommand
}

// Execute starts the command-line client.
func Execute() error {
	locale, err := i18n.ResolveLocale(os.Args[1:], os.LookupEnv)
	if err != nil {
		return err
	}
	return newRootCommand(i18n.NewLocalizer(locale)).Execute()
}
