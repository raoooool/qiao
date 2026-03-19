package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"qiao/internal/core"
)

var errProviderResolutionNotConfigured = errors.New("provider resolution is not configured")

func configureTranslateCommand(cmd *cobra.Command, deps TranslateDependencies) {
	var from string
	var to string
	var provider string
	var jsonOutput bool
	var verbose bool

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && isTerminal(deps.Stdin) {
			return cmd.Help()
		}

		text, err := resolveInput(args, deps.Stdin)
		if err != nil {
			return err
		}

		providerName := provider
		if providerName == "" {
			providerName = deps.DefaultProvider
		}

		sourceLanguage := from
		if sourceLanguage == "" {
			sourceLanguage = deps.DefaultSource
		}

		targetLanguage := to
		if targetLanguage == "" {
			targetLanguage = deps.DefaultTarget
		}

		translator, resolveErr := deps.ResolveProvider(providerName)

		start := time.Now()

		var resp *core.TranslateResponse
		var translateErr error
		if resolveErr == nil {
			resp, translateErr = translator.Translate(cmd.Context(), core.TranslateRequest{
				Text:           text,
				SourceLanguage: sourceLanguage,
				TargetLanguage: targetLanguage,
				Provider:       providerName,
			})
		}

		elapsed := time.Since(start)

		if verbose {
			if resp != nil {
				command, _ := resp.Metadata["command"].(string)
				if command != "" {
					fmt.Fprintf(deps.Stderr, "[qiao] %s (%.2fs)\n", command, elapsed.Seconds())
				} else {
					fmt.Fprintf(deps.Stderr, "[qiao] (%.2fs)\n", elapsed.Seconds())
				}
			} else {
				fmt.Fprintf(deps.Stderr, "[qiao] (%.2fs)\n", elapsed.Seconds())
			}
		}

		if resolveErr != nil {
			return resolveErr
		}
		if translateErr != nil {
			return translateErr
		}

		if jsonOutput {
			return json.NewEncoder(deps.Stdout).Encode(resp)
		}

		_, err = fmt.Fprintln(deps.Stdout, resp.Translation)

		return err
	}

	cmd.Flags().StringVarP(&from, "from", "f", "", "source language")
	cmd.Flags().StringVarP(&to, "to", "t", "", "target language")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "translation provider")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output structured JSON")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show executed command and elapsed time")
}

func resolveInput(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		text := strings.TrimSpace(strings.Join(args, " "))
		if text == "" {
			return "", errors.New("missing text input")
		}

		return text, nil
	}

	if stdin == nil {
		return "", errors.New("missing text input")
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", errors.New("missing text input")
	}

	return text, nil
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func translate(ctx context.Context, translator core.Translator, req core.TranslateRequest) (*core.TranslateResponse, error) {
	return translator.Translate(ctx, req)
}
