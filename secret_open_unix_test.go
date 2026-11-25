//go:build unix

package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecretFileChecksOpenedDescriptor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credential")
	replacement := filepath.Join(dir, "replacement")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal("cannot create regular file fixture")
	}
	if err := os.Symlink(dir, replacement); err != nil {
		t.Fatal("cannot create symlink fixture")
	}
	r := New(nil)
	secret := r.readSecretFile("TOKEN_FILE", path, func(name string) (*os.File, error) {
		// Swap a regular path for a directory between any path check and open.
		if err := os.Rename(replacement, name); err != nil {
			t.Fatal("cannot replace regular file fixture")
		}
		return openSecretFile(name)
	})
	if err := r.Err(); secret.IsSet() || err == nil || err.Error() != "TOKEN_FILE: must name a regular file" {
		t.Fatal("reader should refuse the opened directory as a non-regular file")
	}
}

func TestSecretFileIgnoresPathReplacementAfterOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credential")
	replacement := filepath.Join(dir, "replacement")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal("cannot create regular file fixture")
	}
	if err := os.Symlink(dir, replacement); err != nil {
		t.Fatal("cannot create symlink fixture")
	}
	r := New(nil)
	secret := r.readSecretFile("TOKEN_FILE", path, func(name string) (*os.File, error) {
		file, err := openSecretFile(name)
		if err == nil {
			// The path now names a directory, but the descriptor still names the file.
			if renameErr := os.Rename(replacement, name); renameErr != nil {
				file.Close()
				t.Fatal("cannot replace open file fixture")
			}
		}
		return file, err
	})
	if err := r.Err(); err != nil {
		t.Fatal(err)
	}
	if secret.Reveal() != "fixture" {
		t.Fatal("reader did not read the original file")
	}
}
