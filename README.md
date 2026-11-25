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

## API

Start with `env.New(os.LookupEnv)`, or supply a test lookup. Read all settings
and check `Err` before using the configuration. A reader is not safe for
concurrent use.

| Method | Reads or records |
| --- | --- |
| `String` | A string as given. |
| `Required` | A non-empty string. |
| `Bool` | An exact boolean. |
| `Int` | A decimal integer with inclusive bounds. |
| `Duration` | A Go duration with inclusive bounds; a maximum of zero means no upper limit. |
| `List` | Comma-separated items, with surrounding whitespace and empty items removed. |
| `OneOf` | An exact match from a list of allowed strings. |
| `Fail` | A problem found by a caller's own validation. |
| `Err` | All recorded problems, or `nil`. |

## Secrets

## Configuration documentation tests
