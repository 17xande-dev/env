//go:build unix

package env

import (
	"os"
	"syscall"
)

func openSecretFile(path string) (*os.File, error) {
	// Opening a FIFO must not wait for a writer before Stat can refuse it.
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
