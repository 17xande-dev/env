//go:build unix

package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/17xande-dev/env"
)

func TestSecretFileValue(t *testing.T) {
	const limit = 64 << 10
	for _, tt := range []struct {
		name, contents, want string
	}{
		{"lf", "v\n", "v"},
		{"crlf", "v\r\n", "v"},
		{"repeated_lf", "v\n\n", "v"},
		{"mixed_endings", "v\r\n\r\r\n\n", "v"},
		{"preserve_spaces", "  v  \n", "  v  "},
		{"preserve_other_bytes", "\n\r\tv\nx\t\r\n", "\n\r\tv\nx\t"},
		{"only_spaces", " \t\r\n", " \t"},
		{"exact_limit", strings.Repeat("v", limit-1) + "\n", strings.Repeat("v", limit-1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := writeSecretFixture(t, tt.contents)
			r := env.New(lookup(map[string]string{"TOKEN": "", fileKey: file}))
			assertSecretValue(t, r.Secret("TOKEN"), tt.want)
			if r.Err() != nil {
				t.Fatal("unexpected secret file error")
			}
		})
	}
}

func TestSecretFileRefusals(t *testing.T) {
	for _, tt := range []struct {
		name, value, contents, problem string
	}{
		{"both_set", credential, "opaque-file-content", "cannot be set together"},
		{"directory", "", "", "regular file"},
		{"too_large", "", strings.Repeat("v", 64<<10) + "\n", "exceeds 64 KiB"},
		{"empty", "", "", "secret file is empty"},
		{"only_line_endings", "", "\n\r\r\n", "secret file is empty"},
		{"missing", "", "", "cannot open secret file"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var file string
			switch tt.name {
			case "directory":
				file = t.TempDir()
			case "missing":
				file = filepath.Join(t.TempDir(), "opaque-path-fragment")
			default:
				file = writeSecretFixture(t, tt.contents)
			}
			r := env.New(lookup(map[string]string{"TOKEN": tt.value, fileKey: file}))
			secret := r.Secret("TOKEN")
			assertSecretProblem(t, r.Err(), secret, tt.problem, file, tt.contents, tt.value)
		})
	}
}

func assertSecretProblem(t *testing.T, problem error, got env.Secret, want string, forbidden ...string) {
	t.Helper()
	if problem == nil || !strings.HasPrefix(problem.Error(), fileKey+": ") {
		t.Fatal("expected an error naming TOKEN_FILE")
	}
	for _, value := range forbidden {
		if value != "" && (strings.Contains(problem.Error(), value) || strings.Contains(problem.Error(), filepath.Base(value))) {
			t.Fatal("secret file error disclosed input")
		}
	}
	if !strings.Contains(problem.Error(), want) {
		t.Fatalf("expected a TOKEN_FILE problem containing %q", want)
	}
	assertSecretValue(t, got, "")
}

func writeSecretFixture(t *testing.T, contents string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "opaque-path-fragment")
	if err := os.WriteFile(file, []byte(contents), 0o444); err != nil {
		t.Fatal("cannot create fixture")
	}
	return file
}
