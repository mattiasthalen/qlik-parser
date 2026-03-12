package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
	"github.com/mattiasthalen/qlik-parser/internal/ui"
)

func newExportCmd() *cobra.Command {
	var sourceDir string
	var outDir string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Extract load scripts from .qvw files",
		Long: `Recursively scans --source for .qvw files and extracts the embedded
load scripts to .qvs text files alongside or under --out.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourceDir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine working directory: %w", err)
				}
				sourceDir = cwd
			}

			info, err := os.Stat(sourceDir)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: --source %q: %v\n", sourceDir, err)
				return ExitError(1)
			}
			if !info.IsDir() {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: --source %q is a file, not a directory\n", sourceDir)
				return ExitError(1)
			}

			if outDir != "" {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: cannot create --out directory %q: %v\n", outDir, err)
					return ExitError(1)
				}
			}

			qvwPaths, walkWarns := extractor.Walk(sourceDir)
			for _, w := range walkWarns {
				log.Warn().Msg(w)
			}

			isTTY := ui.IsTTY(os.Stdout)
			printer := ui.NewPrinter(cmd.OutOrStdout(), isTTY, dryRun)

			hasErr := false

			for i, qvwPath := range qvwPaths {
				printer.UpdateSpinner(i+1, len(qvwPaths))

				relPath, err := filepath.Rel(sourceDir, qvwPath)
				if err != nil {
					relPath = filepath.Base(qvwPath)
				}

				script, extractErr := extractor.ExtractScript(qvwPath)
				if extractErr != nil {
					var noScript *extractor.NoScriptError
					if errors.As(extractErr, &noScript) {
						printer.ClearSpinner()
						printer.FileResult(ui.Result{
							Status:  ui.StatusWarn,
							QVWPath: relPath,
							Message: "no script found",
						})
						continue
					}
					hasErr = true
					printer.ClearSpinner()
					errMsg := extractErr.Error()
					if after, ok := strings.CutPrefix(errMsg, qvwPath+": "); ok {
						errMsg = after
					}
					printer.FileResult(ui.Result{
						Status:  ui.StatusErr,
						QVWPath: relPath,
						Message: errMsg,
					})
					continue
				}

				outPath := extractor.ResolveOutputPath(qvwPath, sourceDir, outDir)
				relOut, err := filepath.Rel(sourceDir, outPath)
				if err != nil {
					relOut = filepath.Base(outPath)
				}
				if outDir != "" && outDir != sourceDir {
					if r, err := filepath.Rel(outDir, outPath); err == nil {
						relOut = r
					}
				}

				writeErr := extractor.WriteScript(outPath, script, dryRun)
				if writeErr != nil {
					hasErr = true
					printer.ClearSpinner()
					printer.FileResult(ui.Result{
						Status:  ui.StatusErr,
						QVWPath: relPath,
						Message: writeErr.Error(),
					})
					continue
				}

				printer.ClearSpinner()
				printer.FileResult(ui.Result{
					Status:    ui.StatusOK,
					QVWPath:   relPath,
					QVSPath:   relOut,
					CharCount: len(script),
				})
			}

			printer.Summary()

			if hasErr {
				return ExitError(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&sourceDir, "source", "s", "", "Source directory to scan for .qvw files (default: current directory)")
	cmd.Flags().StringVarP(&outDir, "out", "o", "", "Export directory (default: alongside .qvw files)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be extracted without writing files")

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		_, _ = fmt.Fprintf(c.ErrOrStderr(), "error: %v\n", err)
		return ExitError(2)
	})

	return cmd
}

// ExitCodeError signals a specific exit code to main.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

// ExitError creates an ExitCodeError.
func ExitError(code int) error {
	return &ExitCodeError{Code: code}
}
