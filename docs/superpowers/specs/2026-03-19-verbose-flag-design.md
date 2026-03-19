# Verbose Flag (`-v`) Design Spec

## Goal

Add a `-v` / `--verbose` flag that prints the actual command executed by the provider and its elapsed time to stderr, so users can see what's happening under the hood.

## Output Format

Single line to stderr:

```
[qiao] <full command> (<elapsed>)
```

Example:

```
[qiao] codex exec "Translate the following text from auto-detected language to zh..." (1.23s)
```

- Output goes to **stderr** so it doesn't interfere with translation results, pipes, or `--json`.
- Single-level verbose only (no `-vv`).

## Design

### Provider Layer

Both `codex` and `claude` providers populate `TranslateResponse.Metadata["command"]` with the full command string they executed. This uses the existing `Metadata map[string]any` field — no interface changes needed.

Arguments containing spaces or special characters are quoted with `%q` for readability. Example in codex provider:

```go
quotedArgs := make([]string, len(args))
for i, a := range args {
    quotedArgs[i] = fmt.Sprintf("%q", a)
}
command := fmt.Sprintf("%s %s", p.binary, strings.Join(quotedArgs, " "))

return &core.TranslateResponse{
    // ...existing fields...
    Metadata: map[string]any{
        "command": command,
    },
}, nil
```

Same pattern for the claude provider.

### CLI Layer

In `translate.go`:

1. Add `--verbose` / `-v` bool flag.
2. Record `start := time.Now()` before calling `translator.Translate()`.
3. After the call (success or error), compute `elapsed := time.Since(start)`.
4. If verbose is true, print elapsed time via `defer` or immediately after the call, **before** checking the error. This ensures verbose output is shown on failures too — which is when it's most useful for debugging.
5. Guard against nil metadata: if `resp` is nil or `resp.Metadata["command"]` is missing, skip the command portion or print `[qiao] (<elapsed>)` with just the timing.
   ```go
   // Print verbose info regardless of success/failure
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
   ```

### Dependency Injection

Add `Stderr io.Writer` to `TranslateDependencies`:

```go
type TranslateDependencies struct {
    Stdin           io.Reader
    Stdout          io.Writer
    Stderr          io.Writer       // new
    ResolveProvider func(string) (core.Translator, error)
    ListProviders   func() []string
    DefaultProvider string
    DefaultSource   string
    DefaultTarget   string
}
```

`defaultTranslateDependencies()` sets `Stderr: os.Stderr`. Tests use `bytes.Buffer`.

## Files Changed

| File | Change |
|------|--------|
| `internal/cli/root.go` | Add `Stderr` field to `TranslateDependencies`, set to `os.Stderr` in defaults |
| `internal/cli/translate.go` | Add `-v` flag, timing logic, stderr output |
| `internal/providers/codex/provider.go` | Populate `Metadata["command"]` in response |
| `internal/providers/claude/provider.go` | Populate `Metadata["command"]` in response |
| `internal/cli/*_test.go` | Test verbose output and non-verbose (no output) |
| `internal/providers/codex/provider_test.go` | Verify metadata contains command |
| `internal/providers/claude/provider_test.go` | Verify metadata contains command |

## Not Changed

- `core/types.go` — no interface or struct changes
- `internal/app/` — no changes
- `internal/config/` — no changes
- `internal/providers/registry/` — no changes
