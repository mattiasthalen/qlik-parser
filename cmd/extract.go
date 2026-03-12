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

func newExtractCmd() *cobra.Command {
	var sourceDir string
	var outDir string
	var dryRun bool
	var script bool

	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract artifacts from .qvw and .qvf files",
		Long: `Recursively scans --source for .qvw and .qvf files and extracts embedded
artifacts to text files alongside or under --out.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !script { // expand to: if !script && !variables && !charts as flags are added
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: no artifact type selected\n")
				return ExitError(1)
			}

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

			qlikPaths, walkWarns := extractor.Walk(sourceDir)
			for _, w := range walkWarns {
				log.Warn().Msg(w)
			}

			isTTY := ui.IsTTY(os.Stdout)
			printer := ui.NewPrinter(cmd.OutOrStdout(), isTTY, dryRun)

			hasErr := false

			for i, qvwPath := range qlikPaths {
				printer.UpdateSpinner(i+1, len(qlikPaths))

				relPath, err := filepath.Rel(sourceDir, qvwPath)
				if err != nil {
					relPath = filepath.Base(qvwPath)
				}

				var scriptContent string
				var extractErr error
				if filepath.Ext(qvwPath) == ".qvf" {
					scriptContent, extractErr = extractor.ExtractScriptFromQVF(qvwPath)
				} else {
					scriptContent, extractErr = extractor.ExtractScript(qvwPath)
				}
				if extractErr != nil {
					var noScript *extractor.NoScriptError
					if errors.As(extractErr, &noScript) {
						printer.ClearSpinner()
						printer.FileResult(ui.Result{
							Status:  ui.StatusWarn,
							SrcPath: relPath,
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
						SrcPath: relPath,
						Message: errMsg,
					})
					continue
				}

				outDirPath := extractor.ResolveOutputDir(qvwPath, sourceDir, outDir)
				artifactName := "script.qvs"
				outPath := filepath.Join(outDirPath, artifactName)
				relOut, err := filepath.Rel(sourceDir, outPath)
				if err != nil {
					relOut = filepath.Join(filepath.Base(outDirPath), artifactName)
				}
				if outDir != "" && outDir != sourceDir {
					if r, err := filepath.Rel(outDir, outPath); err == nil {
						relOut = r
					}
				}

				artifacts := []extractor.Artifact{
					{Name: artifactName, Content: []byte(scriptContent)},
				}
				writeErr := extractor.WriteArtifacts(outDirPath, artifacts, dryRun)
				if writeErr != nil {
					hasErr = true
					printer.ClearSpinner()
					printer.FileResult(ui.Result{
						Status:  ui.StatusErr,
						SrcPath: relPath,
						Message: writeErr.Error(),
					})
					continue
				}

				printer.ClearSpinner()
				printer.FileResult(ui.Result{
					Status:    ui.StatusOK,
					SrcPath:   relPath,
					QVSPath:   relOut,
					CharCount: len(scriptContent),
				})
			}

			printer.Summary()

			if hasErr {
				return ExitError(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&sourceDir, "source", "s", "", "Source directory to scan for .qvw and .qvf files (default: current directory)")
	cmd.Flags().StringVarP(&outDir, "out", "o", "", "Export directory (default: alongside .qvw files)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be extracted without writing files")
	cmd.Flags().BoolVar(&script, "script", true, "Extract load scripts")

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
