//go:build !unix

package env

import "errors"

func openSecretFile(string) (secretFile, error) {
	// Do not fall back to an open that could block before the file is checked.
	return nil, errors.ErrUnsupported
}
