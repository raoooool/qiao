package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"qiao/internal/core"
)

var errProviderResolutionNotConfigured = errors.New("provider resolution is not configured")

func configureTranslateCommand(cmd *cobra.Command, deps TranslateDependencies) {
	var from string
	var to string
	var provider string
	var jsonOutput bool

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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

		translator, err := deps.ResolveProvider(providerName)
		if err != nil {
			return err
		}

		resp, err := translator.Translate(cmd.Context(), core.TranslateRequest{
			Text:           text,
			SourceLanguage: sourceLanguage,
			TargetLanguage: targetLanguage,
			Provider:       providerName,
		})
		if err != nil {
			return err
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

func translate(ctx context.Context, translator core.Translator, req core.TranslateRequest) (*core.TranslateResponse, error) {
	return translator.Translate(ctx, req)
}
