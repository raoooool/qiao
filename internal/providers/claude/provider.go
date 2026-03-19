package claude

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"qiao/internal/config"
	"qiao/internal/core"
)

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type Provider struct {
	model  string
	binary string
	runCmd commandRunner
}

func New(cfg config.Config) (core.Translator, error) {
	providerConfig, _ := cfg.ProviderConfig("claude")

	binary := providerConfig["binary"]
	if binary == "" {
		binary = "claude"
	}

	return &Provider{
		model:  providerConfig["model"],
		binary: binary,
		runCmd: defaultRunner,
	}, nil
}

func (p *Provider) Name() string {
	return "claude"
}

func (p *Provider) Translate(ctx context.Context, req core.TranslateRequest) (*core.TranslateResponse, error) {
	prompt := buildPrompt(req)

	args := []string{"-p", "--no-session-persistence", prompt}
	if p.model != "" {
		args = []string{"-p", "--no-session-persistence", "--model", p.model, prompt}
	}

	output, err := p.runCmd(ctx, p.binary, args...)
	if err != nil {
		return nil, fmt.Errorf("claude failed: %w", err)
	}

	quotedArgs := make([]string, len(args))
	for i, a := range args {
		quotedArgs[i] = fmt.Sprintf("%q", a)
	}
	command := fmt.Sprintf("%s %s", p.binary, strings.Join(quotedArgs, " "))

	translation := strings.TrimSpace(string(output))

	return &core.TranslateResponse{
		Provider:       p.Name(),
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		Text:           req.Text,
		Translation:    translation,
		Metadata: map[string]any{
			"command": command,
		},
	}, nil
}

func buildPrompt(req core.TranslateRequest) string {
	source := req.SourceLanguage
	if source == "" || source == "auto" {
		source = "auto-detected language"
	}

	return fmt.Sprintf(
		"Translate the following text from %s to %s. Output ONLY the translated text, nothing else.\n\n%s",
		source, req.TargetLanguage, req.Text,
	)
}

func defaultRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = nil
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
