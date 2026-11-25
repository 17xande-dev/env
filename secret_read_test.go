package env

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

type secretFileFixture struct {
	io.ReadCloser
	info os.FileInfo
}

func (f secretFileFixture) Stat() (os.FileInfo, error) { return f.info, nil }

func emptyRegularFileInfo(t *testing.T) os.FileInfo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestSecretFileReadLimitIgnoresReportedSize(t *testing.T) {
	const limit = 64 << 10
	input := strings.NewReader(strings.Repeat("x", limit+2))
	file := secretFileFixture{io.NopCloser(input), emptyRegularFileInfo(t)}
	r := New(nil)
	s := r.readSecretFile("TOKEN_FILE", "credential", func(string) (secretFile, error) {
		return file, nil
	})
	if err := r.Err(); s.IsSet() || err == nil || err.Error() != "TOKEN_FILE: secret file exceeds 64 KiB" {
		t.Fatal("reader should refuse an oversized stream despite its zero reported size")
	}
	if input.Len() != 1 {
		t.Fatal("reader should stop after one byte beyond the limit")
	}
}

func TestSecretFileReadErrorDoesNotEchoPath(t *testing.T) {
	const path = "opaque-path-fragment"
	readErr := &fs.PathError{Op: "read", Path: path, Err: io.ErrUnexpectedEOF}
	file := secretFileFixture{io.NopCloser(iotest.ErrReader(readErr)), emptyRegularFileInfo(t)}
	r := New(nil)
	s := r.readSecretFile("TOKEN_FILE", path, func(string) (secretFile, error) {
		return file, nil
	})
	if err := r.Err(); s.IsSet() || err == nil || err.Error() != "TOKEN_FILE: cannot read secret file" {
		t.Fatal("reader should record only the fixed read error without disclosing the path")
	}
}

func TestSecretFileUnsupportedPlatform(t *testing.T) {
	const path = "opaque-path-fragment"
	r := New(nil)
	s := r.readSecretFile("TOKEN_FILE", path, func(name string) (secretFile, error) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.ErrUnsupported}
	})
	if err := r.Err(); s.IsSet() || err == nil || err.Error() != "TOKEN_FILE: secret files are not supported on this platform" {
		t.Fatal("reader should explain that secret files are unsupported without disclosing the path")
	}
}
