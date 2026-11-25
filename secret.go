package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const secretFileLimit = 64 << 10

// Secret reads key or key+"_FILE", but not both. An unset or invalid secret
// returns a zero Secret; invalid settings also record a problem.
func (r *Reader) Secret(key string) Secret {
	fileKey := key + "_FILE"
	value := r.text(key, "")
	path := r.text(fileKey, "")
	if path == "" {
		return newSecret(value)
	}
	if value != "" {
		r.Fail(fileKey, "cannot be set together with "+key)
		return Secret{}
	}
	return r.readSecretFile(fileKey, path, openSecretFile)
}

type secretFile interface {
	io.ReadCloser
	Stat() (os.FileInfo, error)
}

func (r *Reader) readSecretFile(key, path string, open func(string) (secretFile, error)) Secret {
	// Do not include OS errors: a path may itself contain a pasted credential.
	file, err := open(path)
	switch {
	case errors.Is(err, errors.ErrUnsupported):
		r.Fail(key, "secret files are not supported on this platform")
		return Secret{}
	case err != nil:
		r.Fail(key, "cannot open secret file")
		return Secret{}
	}
	defer file.Close()

	// Check the opened descriptor rather than the path, which could change.
	info, err := file.Stat()
	if err != nil {
		r.Fail(key, "cannot inspect secret file")
		return Secret{}
	}
	if !info.Mode().IsRegular() {
		r.Fail(key, "must name a regular file")
		return Secret{}
	}

	data, err := io.ReadAll(io.LimitReader(file, secretFileLimit+1))
	if err != nil {
		r.Fail(key, "cannot read secret file")
		return Secret{}
	}
	if len(data) > secretFileLimit {
		r.Fail(key, "secret file exceeds 64 KiB")
		return Secret{}
	}
	value := strings.TrimRight(string(data), "\r\n")
	if value == "" {
		r.Fail(key, "secret file is empty")
		return Secret{}
	}
	return newSecret(value)
}

// Secret holds a credential. Its formatting, logging, and marshaling methods
// emit "[redacted]" for a set secret and an empty string for an unset secret.
type Secret struct {
	// fmt cannot invoke methods on an unexported field in a containing
	// struct. Indirection keeps its fallback formatting from printing bytes.
	value *string
}

func newSecret(value string) Secret {
	if value == "" {
		return Secret{}
	}
	return Secret{value: &value}
}

// Reveal returns the credential for the call site that needs to use it.
func (s Secret) Reveal() string {
	if s.value == nil {
		return ""
	}
	return *s.value
}

// IsSet reports whether the credential is non-empty.
func (s Secret) IsSet() bool { return s.value != nil }

// String returns the redacted display value.
func (s Secret) String() string {
	if !s.IsSet() {
		return ""
	}
	return "[redacted]"
}

// GoString returns the redacted display value.
func (s Secret) GoString() string { return s.String() }

// Format writes the redacted display value regardless of verb or flags.
func (s Secret) Format(state fmt.State, _ rune) {
	io.WriteString(state, s.String())
}

// LogValue returns the redacted display value as a slog string.
func (s Secret) LogValue() slog.Value { return slog.StringValue(s.String()) }

// MarshalJSON encodes the redacted display value as a JSON string.
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// MarshalText encodes the redacted display value.
func (s Secret) MarshalText() ([]byte, error) { return []byte(s.String()), nil }
