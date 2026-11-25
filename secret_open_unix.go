//go:build unix

package env

import (
	"os"
	"syscall"
)

func openSecretFile(path string) (secretFile, error) {
	// Opening a FIFO must not wait for a writer before Stat can refuse it.
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return file, nil
}
