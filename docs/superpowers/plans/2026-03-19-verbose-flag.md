# Verbose Flag (`-v`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `-v` / `--verbose` flag that prints the executed command and elapsed time to stderr.

**Architecture:** Providers populate `Metadata["command"]` in their response. The CLI layer adds a `-v` flag, times the `Translate()` call, and prints a `[qiao] <command> (<elapsed>)` line to stderr. A new `Stderr io.Writer` field in `TranslateDependencies` enables testing.

**Tech Stack:** Go, cobra, standard library (`time`, `fmt`)

**Spec:** `docs/superpowers/specs/2026-03-19-verbose-flag-design.md`

---

### Task 1: Codex provider returns command in Metadata

**Files:**
- Modify: `internal/providers/codex/provider.go:41-63`
- Test: `internal/providers/codex/provider_test.go`

- [ ] **Step 1: Write failing test — metadata contains command**

Add to `internal/providers/codex/provider_test.go`:

```go
func TestTranslateMetadataContainsCommand(t *testing.T) {
	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	provider.runCmd = fakeRunner("translated\n", nil)

	resp, err := provider.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	command, ok := resp.Metadata["command"].(string)
	if !ok || command == "" {
		t.Fatal("expected metadata to contain non-empty 'command' key")
	}
	if !strings.Contains(command, "codex") {
		t.Fatalf("expected command to contain 'codex', got %q", command)
	}
}
```

Add `"strings"` to the import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/codex/ -run TestTranslateMetadataContainsCommand -v`
Expected: FAIL — `resp.Metadata` is nil

- [ ] **Step 3: Implement — populate Metadata in Translate()**

In `internal/providers/codex/provider.go`, replace the `Translate` method body. Build a quoted command string from the args and include it in the response Metadata:

```go
func (p *Provider) Translate(ctx context.Context, req core.TranslateRequest) (*core.TranslateResponse, error) {
	prompt := buildPrompt(req)

	args := []string{"exec", prompt}
	if p.model != "" {
		args = []string{"exec", "-m", p.model, prompt}
	}

	output, err := p.runCmd(ctx, p.binary, args...)
	if err != nil {
		return nil, fmt.Errorf("codex exec failed: %w", err)
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/codex/ -run TestTranslateMetadataContainsCommand -v`
Expected: PASS

- [ ] **Step 5: Run all codex tests**

Run: `go test ./internal/providers/codex/ -v`
Expected: All PASS

---

### Task 2: Claude provider returns command in Metadata

**Files:**
- Modify: `internal/providers/claude/provider.go:41-63`
- Test: `internal/providers/claude/provider_test.go`

- [ ] **Step 1: Write failing test — metadata contains command**

Add to `internal/providers/claude/provider_test.go`:

```go
func TestTranslateMetadataContainsCommand(t *testing.T) {
	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	provider.runCmd = fakeRunner("translated\n", nil)

	resp, err := provider.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	command, ok := resp.Metadata["command"].(string)
	if !ok || command == "" {
		t.Fatal("expected metadata to contain non-empty 'command' key")
	}
	if !strings.Contains(command, "claude") {
		t.Fatalf("expected command to contain 'claude', got %q", command)
	}
}
```

`strings` is already imported.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/claude/ -run TestTranslateMetadataContainsCommand -v`
Expected: FAIL

- [ ] **Step 3: Implement — populate Metadata in Translate()**

In `internal/providers/claude/provider.go`, replace the `Translate` method body:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/claude/ -run TestTranslateMetadataContainsCommand -v`
Expected: PASS

- [ ] **Step 5: Run all claude tests**

Run: `go test ./internal/providers/claude/ -v`
Expected: All PASS

---

### Task 3: Add Stderr to TranslateDependencies

**Files:**
- Modify: `internal/cli/root.go:13-21` (struct definition)
- Modify: `internal/cli/root.go:42-73` (defaults function)
- Modify: `internal/cli/translate_test.go` (all test helpers)

- [ ] **Step 1: Add Stderr field to TranslateDependencies**

In `internal/cli/root.go`, add `Stderr io.Writer` to the struct:

```go
type TranslateDependencies struct {
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	ResolveProvider func(string) (core.Translator, error)
	ListProviders   func() []string
	DefaultProvider string
	DefaultSource   string
	DefaultTarget   string
}
```

- [ ] **Step 2: Set Stderr in both branches of defaultTranslateDependencies()**

In the error branch (line 45-58), add `Stderr: os.Stderr,`.
In the success branch (line 60-72), add `Stderr: os.Stderr,`.

- [ ] **Step 3: Add Stderr to all test TranslateDependencies**

In `internal/cli/translate_test.go`, add `Stderr: &bytes.Buffer{}` (or a new named `var stderr bytes.Buffer`) to every `TranslateDependencies` literal. There are 5 tests that construct deps:

- `TestTranslateUsesPositionalInput` — add `Stderr: &bytes.Buffer{},`
- `TestTranslateUsesStdinWhenPositionalInputMissing` — add `Stderr: &bytes.Buffer{},`
- `TestTranslatePositionalInputWinsOverStdin` — add `Stderr: &bytes.Buffer{},`
- `TestTranslateRequiresInput` — add `Stderr: &bytes.Buffer{},`
- `TestTranslateJSONOutput` — add `Stderr: &bytes.Buffer{},`

- [ ] **Step 4: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: All PASS

---

### Task 4: Implement -v flag with timing and stderr output

**Files:**
- Modify: `internal/cli/translate.go:19-78`
- Test: `internal/cli/translate_test.go`

- [ ] **Step 1: Write failing test — verbose prints command and timing to stderr**

Add to `internal/cli/translate_test.go`:

```go
func TestTranslateVerboseOutput(t *testing.T) {
	translator := &fakeTranslator{
		response: &core.TranslateResponse{
			Provider:       "fake",
			SourceLanguage: "auto",
			TargetLanguage: "zh",
			Text:           "hello",
			Translation:    "你好",
			Metadata: map[string]any{
				"command": `fake "exec" "hello"`,
			},
		},
	}
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "fake",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"-v", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "[qiao]") {
		t.Fatalf("expected stderr to contain [qiao] prefix, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, `fake "exec" "hello"`) {
		t.Fatalf("expected stderr to contain command, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "s)") {
		t.Fatalf("expected stderr to contain elapsed time, got %q", stderrStr)
	}

	if stdout.String() != "你好\n" {
		t.Fatalf("expected stdout to contain translation only, got %q", stdout.String())
	}
}
```

- [ ] **Step 2: Write failing test — no verbose output without -v flag**

Add to `internal/cli/translate_test.go`:

```go
func TestTranslateNoVerboseByDefault(t *testing.T) {
	translator := &fakeTranslator{
		response: &core.TranslateResponse{
			Provider:    "fake",
			Text:        "hello",
			Translation: "你好",
			Metadata: map[string]any{
				"command": `fake "exec" "hello"`,
			},
		},
	}
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "fake",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if stderr.String() != "" {
		t.Fatalf("expected no stderr output without -v, got %q", stderr.String())
	}
}
```

- [ ] **Step 3: Write failing test — verbose on error path shows timing**

Add to `internal/cli/translate_test.go`:

```go
func TestTranslateVerboseOnError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		ResolveProvider: func(string) (core.Translator, error) {
			return nil, errors.New("provider failed")
		},
		DefaultProvider: "fake",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"-v", "hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "[qiao]") {
		t.Fatalf("expected stderr to contain [qiao] even on error, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "s)") {
		t.Fatalf("expected stderr to contain elapsed time, got %q", stderrStr)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run "TestTranslateVerbose|TestTranslateNoVerbose" -v`
Expected: FAIL — unknown flag `-v`

- [ ] **Step 5: Implement -v flag with timing and stderr output**

In `internal/cli/translate.go`, add `"time"` to imports. Add the `verbose` flag variable and modify `RunE`:

```go
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
```

- [ ] **Step 6: Run all new tests**

Run: `go test ./internal/cli/ -run "TestTranslateVerbose|TestTranslateNoVerbose" -v`
Expected: All PASS

- [ ] **Step 7: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS
