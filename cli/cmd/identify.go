package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luisgustavo/coleoptera_identifier/cli/internal/api"
	"github.com/luisgustavo/coleoptera_identifier/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newIdentifyCommand() *cobra.Command {
	var output string

	command := &cobra.Command{
		Use:   "identify <imagem>",
		Short: "Identifica uma imagem sem abrir a interface interativa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format := strings.ToLower(output)
			if format != "text" && format != "json" {
				return fmt.Errorf("formato de saída %q inválido: use text ou json", output)
			}

			client, err := api.NewClient(opts.apiURL, opts.timeout)
			if err != nil {
				return err
			}

			result, err := client.Identify(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if format == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(result)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), tui.RenderResult(result, 64))
			return err
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "text", "formato da saída: text ou json")
	return command
}
