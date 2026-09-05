package cmd

import (
	"time"

	"github.com/luisgustavo/coleoptera_identifier/cli/internal/config"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

type options struct {
	apiURL  string
	timeout time.Duration
}

var opts options

var rootCmd = &cobra.Command{
	Use:           "coleoptera",
	Short:         "Identifique besouros diretamente do terminal",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runUI(cmd.Context())
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&opts.apiURL,
		"api-url",
		config.APIURL(),
		"URL base da API (ou COLEOPTERA_API_URL)",
	)
	rootCmd.PersistentFlags().DurationVar(
		&opts.timeout,
		"timeout",
		60*time.Second,
		"tempo máximo de uma identificação",
	)

	rootCmd.AddCommand(newIdentifyCommand())
	rootCmd.AddCommand(newUICommand())
}

// Execute inicia o cliente de linha de comando.
func Execute() error {
	return rootCmd.Execute()
}
