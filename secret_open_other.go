//go:build !unix

package env

import (
	"errors"
	"os"
)

func openSecretFile(string) (*os.File, error) {
	// Do not fall back to an open that could block before the file is checked.
	return nil, errors.ErrUnsupported
}
