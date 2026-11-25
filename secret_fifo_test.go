//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package env_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/17xande-dev/env"
)

func TestSecretFileFIFO(t *testing.T) {
	for _, name := range []string{"without_writer", "with_writer"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "credential-pipe")
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal("cannot create FIFO fixture")
			}
			if name == "with_writer" {
				// Keep a writer open without introducing a blocking fixture setup.
				keeper, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
				if err != nil {
					t.Fatal("cannot open FIFO fixture for writing")
				}
				defer keeper.Close()
			}
			r := env.New(lookup(map[string]string{fileKey: path}))
			done := make(chan env.Secret, 1)
			go func() { done <- r.Secret("TOKEN") }()
			select {
			case secret := <-done:
				assertSecretProblem(t, r.Err(), secret, "regular file", path)
			case <-time.After(time.Second):
				// Give an accidentally blocking open a writer so it can exit.
				if keeper, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0); err == nil {
					keeper.Close()
				}
				t.Fatal("reading the FIFO did not return before the deadline")
			}
		})
	}
}
