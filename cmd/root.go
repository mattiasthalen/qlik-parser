package cmd

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// NewRootCmd constructs the root cobra command.
func NewRootCmd() *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:   "qlik-script-extractor",
		Short: "Extract QlikView load scripts from .qvw files",
		Long: `qlik-script-extractor recursively scans a directory for QVW files
and extracts the embedded load scripts to .qvs text files.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				level = zerolog.Disabled
			}
			zerolog.SetGlobalLevel(level)
			return nil
		},
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newExportCmd())

	root.PersistentFlags().StringVar(&logLevel, "log-level", "disabled",
		"Log level: debug, info, warn, error, disabled")

	return root
}
