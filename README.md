# env

`env` reads typed settings from the environment and reports configuration
problems together. It uses only the Go standard library.

## Rules

| Rule | Behavior |
| --- | --- |
| Defaults | An unset or empty setting uses its default. `Required` reports a problem instead. |
| Exact input | Values are not trimmed or case-folded. `List` trims whitespace from individual items after splitting. |
| Booleans | Only `true` and `false` are accepted. |
| Errors | Invalid settings return their defaults. `Err` joins all problems as `KEY: problem` in the order they were recorded. |
| No value disclosure | Getters report what is wanted without echoing the input. Pass fixed, value-free descriptions to `Fail`. |
| Secret files | Set either `KEY` or `KEY_FILE`, not both. File errors name `KEY_FILE` without echoing the path or contents. |

## API

Start with `env.New(os.LookupEnv)`, or supply a test lookup. Read all settings
and check `Err` before using the configuration. A reader is not safe for
concurrent use.

| Method | Reads or records |
| --- | --- |
| `String` | A string as given. |
| `Required` | A non-empty string. |
| `Bool` | An exact boolean. |
| `Int` | A decimal integer with inclusive bounds. Use `math.MaxInt` for no upper limit. |
| `Duration` | A Go duration with inclusive bounds; a maximum of zero means no upper limit. |
| `List` | Comma-separated items, with surrounding whitespace and empty items removed. An unset or empty setting returns the default slice as-is. A non-empty setting with no items returns `nil` without an error. |
| `OneOf` | An exact match from a list of allowed strings. |
| `Secret` | A credential from a setting or file, or an unset `Secret`. |
| `Fail` | A problem found by a caller's own validation. |
| `Err` | All recorded problems, or `nil`. |

## Secrets

`r.Secret("PUBLISH_TOKEN")` always looks up `PUBLISH_TOKEN` and
`PUBLISH_TOKEN_FILE`. Empty settings count as unset. A direct value is kept
as-is. A file must be regular and at most 64 KiB, including any line endings.
On Unix, symlinks are followed, relative paths resolve against the process
working directory, and the opened file is checked without waiting for a
FIFO writer. On other platforms the reader reports
`secret files are not supported on this platform` rather than risk a blocking
open. There is no file-permission check.

All trailing CR and LF bytes are stripped from file contents. Leading and
interior newlines, spaces, and other bytes are preserved. A file that is empty
after stripping is an error.

Use `IsSet()` for a presence check and `Reveal()` only at the point where the
credential is needed. The formatting, slog, JSON, and text marshaling methods
display `[redacted]` for a set secret and an empty string for an unset one.
This does not protect the plain string returned by `Reveal()`.

`Secret` is immutable and safe to share across goroutines. The `==` operator
compares identity, not contents: two independent reads of the same non-empty
credential compare unequal, and unset secrets equal the zero value. For
credential comparisons, use `crypto/subtle` on the revealed value where a
constant-time comparison is needed.

Keep `Secret` fields exported if they should display `[redacted]` when
formatted with `fmt`: an unexported field may display an address instead.
Reflection-based dumpers that bypass methods can expose the credential.
`encoding/gob` refuses to encode a `Secret`, which has no exported fields.

## Worked example

Keep application-specific validation in the loader. Load configuration
before opening resources or starting workers.

```go
package config

import (
	"os"

	"github.com/17xande-dev/env"
)

type Config struct {
	Label   string
	Publish bool
	Token   env.Secret
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup env.Lookup) (Config, error) {
	r := env.New(lookup)
	cfg := Config{
		Label:   r.String("LABEL", "worker"),
		Publish: r.Bool("PUBLISH", false),
		Token:   r.Secret("PUBLISH_TOKEN"),
	}
	if cfg.Publish && !cfg.Token.IsSet() {
		r.Fail("PUBLISH_TOKEN", "is required when publishing is enabled (or set PUBLISH_TOKEN_FILE)")
	}
	if err := r.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
```

## Configuration

For the loader above:

| Setting | Purpose |
| --- | --- |
| `LABEL` | Worker label; defaults to `worker`. |
| `PUBLISH` | Enable publishing; defaults to `false`. |
| `PUBLISH_TOKEN` | Credential supplied directly. |
| `PUBLISH_TOKEN_FILE` | File containing the credential instead. |

The corresponding `.env.example`:

```dotenv
LABEL=worker
PUBLISH=false
# PUBLISH_TOKEN=
# PUBLISH_TOKEN_FILE=
```

## Configuration documentation tests

In a test in the loader's package, compare the keys read with the example
file and, if present, the README table. Use paths relative to the test package.
Read every key **unconditionally**, then validate conditionally. A recorder
cannot see keys on an execution path the test never takes.

```go
package config

import (
	"testing"

	"github.com/17xande-dev/env/envtest"
)

func TestConfigDocumentation(t *testing.T) {
	recorder := envtest.NewRecorder(nil)
	if _, err := load(recorder.Lookup); err != nil {
		t.Fatal(err)
	}
	read := recorder.Keys()
	envtest.Documented(t, ".env.example", read, envtest.EnvExample(t, ".env.example"))
	envtest.Documented(t, "README", read, envtest.MarkdownTable(t, "README.md", "## Configuration"))
}
```

`Recorder.Lookup` records every requested key, including unset keys, and
`Keys` returns the sorted, unique names. `EnvExample` accepts `KEY=` and
`# KEY=` lines, with no spaces between the key and `=`. Values are not parsed,
so quoted values are fine. Use each key exactly once: `KEY=` and `# KEY=`
for the same key are a duplicate. Do not use `export KEY=`, `## KEY=`, or
`KEY = value`: these lines are ignored. Save the file without a UTF-8 BOM.

`MarkdownTable` reads backticked keys in the first column beneath the exact
named heading. It skips fenced code blocks, stops at the next heading of the
same or higher level, and rejects duplicate keys. `Documented` flags missing
documentation, documented keys that were not read, and vacuous tests that
read nothing.

## Changes

- **v0.1.0:** Typed environment settings, aggregated value-free
  configuration errors, redacted credentials, and documentation parity tests.

## License

[MIT No Attribution (MIT-0)](LICENSE).
